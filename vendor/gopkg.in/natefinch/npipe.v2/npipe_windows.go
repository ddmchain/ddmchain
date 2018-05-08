package npipe

import (
	"fmt"
	"io"
	"net"
	"sync"
	"syscall"
	"time"
)

const (

	pipe_access_duplex   = 0x3
	pipe_access_inbound  = 0x1
	pipe_access_outbound = 0x2

	file_flag_first_pipe_instance = 0x00080000
	file_flag_write_through       = 0x80000000
	file_flag_overlapped          = 0x40000000

	write_dac              = 0x00040000
	write_owner            = 0x00080000
	access_system_security = 0x01000000

	pipe_type_byte    = 0x0
	pipe_type_message = 0x4

	pipe_readmode_byte    = 0x0
	pipe_readmode_message = 0x2

	pipe_wait   = 0x0
	pipe_nowait = 0x1

	pipe_accept_remote_clients = 0x0
	pipe_reject_remote_clients = 0x8

	pipe_unlimited_instances = 255

	nmpwait_wait_forever = 0xFFFFFFFF

	error_no_data        syscall.Errno = 0xE8
	error_pipe_connected syscall.Errno = 0x217
	error_pipe_busy      syscall.Errno = 0xE7
	error_sem_timeout    syscall.Errno = 0x79

	error_bad_pathname syscall.Errno = 0xA1
	error_invalid_name syscall.Errno = 0x7B

	error_io_incomplete syscall.Errno = 0x3e4
)

var _ net.Conn = (*PipeConn)(nil)
var _ net.Listener = (*PipeListener)(nil)

var ErrClosed = PipeError{"Pipe has been closed.", false}

type PipeError struct {
	msg     string
	timeout bool
}

func (e PipeError) Error() string {
	return e.msg
}

func (e PipeError) Timeout() bool {
	return e.timeout
}

func (e PipeError) Temporary() bool {
	return false
}

func Dial(address string) (*PipeConn, error) {
	for {
		conn, err := dial(address, nmpwait_wait_forever)
		if err == nil {
			return conn, nil
		}
		if isPipeNotReady(err) {
			<-time.After(100 * time.Millisecond)
			continue
		}
		return nil, err
	}
}

func DialTimeout(address string, timeout time.Duration) (*PipeConn, error) {
	deadline := time.Now().Add(timeout)

	now := time.Now()
	for now.Before(deadline) {
		millis := uint32(deadline.Sub(now) / time.Millisecond)
		conn, err := dial(address, millis)
		if err == nil {
			return conn, nil
		}
		if err == error_sem_timeout {

			return nil, PipeError{fmt.Sprintf(
				"Timed out waiting for pipe '%s' to come available", address), true}
		}
		if isPipeNotReady(err) {
			left := deadline.Sub(time.Now())
			retry := 100 * time.Millisecond
			if left > retry {
				<-time.After(retry)
			} else {
				<-time.After(left - time.Millisecond)
			}
			now = time.Now()
			continue
		}
		return nil, err
	}
	return nil, PipeError{fmt.Sprintf(
		"Timed out waiting for pipe '%s' to come available", address), true}
}

func isPipeNotReady(err error) bool {

	return err == syscall.ERROR_FILE_NOT_FOUND || err == error_pipe_busy
}

func newOverlapped() (*syscall.Overlapped, error) {
	event, err := createEvent(nil, true, true, nil)
	if err != nil {
		return nil, err
	}
	return &syscall.Overlapped{HEvent: event}, nil
}

func waitForCompletion(handle syscall.Handle, overlapped *syscall.Overlapped) (uint32, error) {
	_, err := syscall.WaitForSingleObject(overlapped.HEvent, syscall.INFINITE)
	if err != nil {
		return 0, err
	}
	var transferred uint32
	err = getOverlappedResult(handle, overlapped, &transferred, true)
	return transferred, err
}

func dial(address string, timeout uint32) (*PipeConn, error) {
	name, err := syscall.UTF16PtrFromString(string(address))
	if err != nil {
		return nil, err
	}

	if err := waitNamedPipe(name, timeout); err != nil {
		if err == error_bad_pathname {

			return nil, badAddr(address)
		}
		return nil, err
	}
	pathp, err := syscall.UTF16PtrFromString(address)
	if err != nil {
		return nil, err
	}
	handle, err := syscall.CreateFile(pathp, syscall.GENERIC_READ|syscall.GENERIC_WRITE,
		uint32(syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE), nil, syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_OVERLAPPED, 0)
	if err != nil {
		return nil, err
	}
	return &PipeConn{handle: handle, addr: PipeAddr(address)}, nil
}

func Listen(address string) (*PipeListener, error) {
	handle, err := createPipe(address, true)
	if err == error_invalid_name {
		return nil, badAddr(address)
	}
	if err != nil {
		return nil, err
	}

	return &PipeListener{
		addr:   PipeAddr(address),
		handle: handle,
	}, nil
}

type PipeListener struct {
	mu sync.Mutex

	addr   PipeAddr
	handle syscall.Handle
	closed bool

	acceptHandle syscall.Handle

	acceptOverlapped *syscall.Overlapped
}

func (l *PipeListener) Accept() (net.Conn, error) {
	c, err := l.AcceptPipe()
	for err == error_no_data {

		c, err = l.AcceptPipe()
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (l *PipeListener) AcceptPipe() (*PipeConn, error) {
	if l == nil {
		return nil, syscall.EINVAL
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if l.addr == "" || l.closed {
		return nil, syscall.EINVAL
	}

	handle := l.handle
	if handle == 0 {
		var err error
		handle, err = createPipe(string(l.addr), false)
		if err != nil {
			return nil, err
		}
	} else {
		l.handle = 0
	}

	overlapped, err := newOverlapped()
	if err != nil {
		return nil, err
	}
	defer syscall.CloseHandle(overlapped.HEvent)
	err = connectNamedPipe(handle, overlapped)
	if err == nil || err == error_pipe_connected {
		return &PipeConn{handle: handle, addr: l.addr}, nil
	}

	if err == error_io_incomplete || err == syscall.ERROR_IO_PENDING {
		l.acceptOverlapped = overlapped
		l.acceptHandle = handle

		l.mu.Unlock()
		defer func() {
			l.mu.Lock()
			l.acceptOverlapped = nil
			l.acceptHandle = 0

		}()
		_, err = waitForCompletion(handle, overlapped)
	}
	if err == syscall.ERROR_OPERATION_ABORTED {

		return nil, ErrClosed
	}
	if err != nil {
		return nil, err
	}
	return &PipeConn{handle: handle, addr: l.addr}, nil
}

func (l *PipeListener) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}
	l.closed = true
	if l.handle != 0 {
		err := disconnectNamedPipe(l.handle)
		if err != nil {
			return err
		}
		err = syscall.CloseHandle(l.handle)
		if err != nil {
			return err
		}
		l.handle = 0
	}
	if l.acceptOverlapped != nil && l.acceptHandle != 0 {

		if err := cancelIoEx(l.acceptHandle, l.acceptOverlapped); err != nil {
			return err
		}
		err := syscall.CloseHandle(l.acceptOverlapped.HEvent)
		if err != nil {
			return err
		}
		l.acceptOverlapped.HEvent = 0
		err = syscall.CloseHandle(l.acceptHandle)
		if err != nil {
			return err
		}
		l.acceptHandle = 0
	}
	return nil
}

func (l *PipeListener) Addr() net.Addr { return l.addr }

type PipeConn struct {
	handle syscall.Handle
	addr   PipeAddr

	readDeadline  *time.Time
	writeDeadline *time.Time
}

type iodata struct {
	n   uint32
	err error
}

func (c *PipeConn) completeRequest(data iodata, deadline *time.Time, overlapped *syscall.Overlapped) (int, error) {
	if data.err == error_io_incomplete || data.err == syscall.ERROR_IO_PENDING {
		var timer <-chan time.Time
		if deadline != nil {
			if timeDiff := deadline.Sub(time.Now()); timeDiff > 0 {
				timer = time.After(timeDiff)
			}
		}
		done := make(chan iodata)
		go func() {
			n, err := waitForCompletion(c.handle, overlapped)
			done <- iodata{n, err}
		}()
		select {
		case data = <-done:
		case <-timer:
			syscall.CancelIoEx(c.handle, overlapped)
			data = iodata{0, timeout(c.addr.String())}
		}
	}

	if data.err == syscall.ERROR_BROKEN_PIPE {
		data.err = io.EOF
	}
	return int(data.n), data.err
}

func (c *PipeConn) Read(b []byte) (int, error) {

	overlapped, err := newOverlapped()
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(overlapped.HEvent)
	var n uint32
	err = syscall.ReadFile(c.handle, b, &n, overlapped)
	return c.completeRequest(iodata{n, err}, c.readDeadline, overlapped)
}

func (c *PipeConn) Write(b []byte) (int, error) {
	overlapped, err := newOverlapped()
	if err != nil {
		return 0, err
	}
	defer syscall.CloseHandle(overlapped.HEvent)
	var n uint32
	err = syscall.WriteFile(c.handle, b, &n, overlapped)
	return c.completeRequest(iodata{n, err}, c.writeDeadline, overlapped)
}

func (c *PipeConn) Close() error {
	return syscall.CloseHandle(c.handle)
}

func (c *PipeConn) LocalAddr() net.Addr {
	return c.addr
}

func (c *PipeConn) RemoteAddr() net.Addr {

	return c.addr
}

func (c *PipeConn) SetDeadline(t time.Time) error {
	c.SetReadDeadline(t)
	c.SetWriteDeadline(t)
	return nil
}

func (c *PipeConn) SetReadDeadline(t time.Time) error {
	c.readDeadline = &t
	return nil
}

func (c *PipeConn) SetWriteDeadline(t time.Time) error {
	c.writeDeadline = &t
	return nil
}

type PipeAddr string

func (a PipeAddr) Network() string { return "pipe" }

func (a PipeAddr) String() string {
	return string(a)
}

func createPipe(address string, first bool) (syscall.Handle, error) {
	n, err := syscall.UTF16PtrFromString(address)
	if err != nil {
		return 0, err
	}
	mode := uint32(pipe_access_duplex | syscall.FILE_FLAG_OVERLAPPED)
	if first {
		mode |= file_flag_first_pipe_instance
	}
	return createNamedPipe(n,
		mode,
		pipe_type_byte,
		pipe_unlimited_instances,
		512, 512, 0, nil)
}

func badAddr(addr string) PipeError {
	return PipeError{fmt.Sprintf("Invalid pipe address '%s'.", addr), false}
}
func timeout(addr string) PipeError {
	return PipeError{fmt.Sprintf("Pipe IO timed out waiting for '%s'", addr), true}
}
