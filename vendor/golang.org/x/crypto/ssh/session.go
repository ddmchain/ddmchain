
package ssh

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sync"
)

type Signal string

const (
	SIGABRT Signal = "ABRT"
	SIGALRM Signal = "ALRM"
	SIGFPE  Signal = "FPE"
	SIGHUP  Signal = "HUP"
	SIGILL  Signal = "ILL"
	SIGINT  Signal = "INT"
	SIGKILL Signal = "KILL"
	SIGPIPE Signal = "PIPE"
	SIGQUIT Signal = "QUIT"
	SIGSEGV Signal = "SEGV"
	SIGTERM Signal = "TERM"
	SIGUSR1 Signal = "USR1"
	SIGUSR2 Signal = "USR2"
)

var signals = map[Signal]int{
	SIGABRT: 6,
	SIGALRM: 14,
	SIGFPE:  8,
	SIGHUP:  1,
	SIGILL:  4,
	SIGINT:  2,
	SIGKILL: 9,
	SIGPIPE: 13,
	SIGQUIT: 3,
	SIGSEGV: 11,
	SIGTERM: 15,
}

type TerminalModes map[uint8]uint32

const (
	tty_OP_END    = 0
	VINTR         = 1
	VQUIT         = 2
	VERASE        = 3
	VKILL         = 4
	VEOF          = 5
	VEOL          = 6
	VEOL2         = 7
	VSTART        = 8
	VSTOP         = 9
	VSUSP         = 10
	VDSUSP        = 11
	VREPRINT      = 12
	VWERASE       = 13
	VLNEXT        = 14
	VFLUSH        = 15
	VSWTCH        = 16
	VSTATUS       = 17
	VDISCARD      = 18
	IGNPAR        = 30
	PARMRK        = 31
	INPCK         = 32
	ISTRIP        = 33
	INLCR         = 34
	IGNCR         = 35
	ICRNL         = 36
	IUCLC         = 37
	IXON          = 38
	IXANY         = 39
	IXOFF         = 40
	IMAXBEL       = 41
	ISIG          = 50
	ICANON        = 51
	XCASE         = 52
	ECHO          = 53
	ECHOE         = 54
	ECHOK         = 55
	ECHONL        = 56
	NOFLSH        = 57
	TOSTOP        = 58
	IEXTEN        = 59
	ECHOCTL       = 60
	ECHOKE        = 61
	PENDIN        = 62
	OPOST         = 70
	OLCUC         = 71
	ONLCR         = 72
	OCRNL         = 73
	ONOCR         = 74
	ONLRET        = 75
	CS7           = 90
	CS8           = 91
	PARENB        = 92
	PARODD        = 93
	TTY_OP_ISPEED = 128
	TTY_OP_OSPEED = 129
)

type Session struct {

	Stdin io.Reader

	Stdout io.Writer
	Stderr io.Writer

	ch        Channel 
	started   bool    
	copyFuncs []func() error
	errors    chan error 

	stdinpipe, stdoutpipe, stderrpipe bool

	stdinPipeWriter io.WriteCloser

	exitStatus chan error
}

func (s *Session) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	return s.ch.SendRequest(name, wantReply, payload)
}

func (s *Session) Close() error {
	return s.ch.Close()
}

type setenvRequest struct {
	Name  string
	Value string
}

func (s *Session) Setenv(name, value string) error {
	msg := setenvRequest{
		Name:  name,
		Value: value,
	}
	ok, err := s.ch.SendRequest("env", true, Marshal(&msg))
	if err == nil && !ok {
		err = errors.New("ssh: setenv failed")
	}
	return err
}

type ptyRequestMsg struct {
	Term     string
	Columns  uint32
	Rows     uint32
	Width    uint32
	Height   uint32
	Modelist string
}

func (s *Session) RequestPty(term string, h, w int, termmodes TerminalModes) error {
	var tm []byte
	for k, v := range termmodes {
		kv := struct {
			Key byte
			Val uint32
		}{k, v}

		tm = append(tm, Marshal(&kv)...)
	}
	tm = append(tm, tty_OP_END)
	req := ptyRequestMsg{
		Term:     term,
		Columns:  uint32(w),
		Rows:     uint32(h),
		Width:    uint32(w * 8),
		Height:   uint32(h * 8),
		Modelist: string(tm),
	}
	ok, err := s.ch.SendRequest("pty-req", true, Marshal(&req))
	if err == nil && !ok {
		err = errors.New("ssh: pty-req failed")
	}
	return err
}

type subsystemRequestMsg struct {
	Subsystem string
}

func (s *Session) RequestSubsystem(subsystem string) error {
	msg := subsystemRequestMsg{
		Subsystem: subsystem,
	}
	ok, err := s.ch.SendRequest("subsystem", true, Marshal(&msg))
	if err == nil && !ok {
		err = errors.New("ssh: subsystem request failed")
	}
	return err
}

type ptyWindowChangeMsg struct {
	Columns uint32
	Rows    uint32
	Width   uint32
	Height  uint32
}

func (s *Session) WindowChange(h, w int) error {
	req := ptyWindowChangeMsg{
		Columns: uint32(w),
		Rows:    uint32(h),
		Width:   uint32(w * 8),
		Height:  uint32(h * 8),
	}
	_, err := s.ch.SendRequest("window-change", false, Marshal(&req))
	return err
}

type signalMsg struct {
	Signal string
}

func (s *Session) Signal(sig Signal) error {
	msg := signalMsg{
		Signal: string(sig),
	}

	_, err := s.ch.SendRequest("signal", false, Marshal(&msg))
	return err
}

type execMsg struct {
	Command string
}

func (s *Session) Start(cmd string) error {
	if s.started {
		return errors.New("ssh: session already started")
	}
	req := execMsg{
		Command: cmd,
	}

	ok, err := s.ch.SendRequest("exec", true, Marshal(&req))
	if err == nil && !ok {
		err = fmt.Errorf("ssh: command %v failed", cmd)
	}
	if err != nil {
		return err
	}
	return s.start()
}

func (s *Session) Run(cmd string) error {
	err := s.Start(cmd)
	if err != nil {
		return err
	}
	return s.Wait()
}

func (s *Session) Output(cmd string) ([]byte, error) {
	if s.Stdout != nil {
		return nil, errors.New("ssh: Stdout already set")
	}
	var b bytes.Buffer
	s.Stdout = &b
	err := s.Run(cmd)
	return b.Bytes(), err
}

type singleWriter struct {
	b  bytes.Buffer
	mu sync.Mutex
}

func (w *singleWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.b.Write(p)
}

func (s *Session) CombinedOutput(cmd string) ([]byte, error) {
	if s.Stdout != nil {
		return nil, errors.New("ssh: Stdout already set")
	}
	if s.Stderr != nil {
		return nil, errors.New("ssh: Stderr already set")
	}
	var b singleWriter
	s.Stdout = &b
	s.Stderr = &b
	err := s.Run(cmd)
	return b.b.Bytes(), err
}

func (s *Session) Shell() error {
	if s.started {
		return errors.New("ssh: session already started")
	}

	ok, err := s.ch.SendRequest("shell", true, nil)
	if err == nil && !ok {
		return errors.New("ssh: could not start shell")
	}
	if err != nil {
		return err
	}
	return s.start()
}

func (s *Session) start() error {
	s.started = true

	type F func(*Session)
	for _, setupFd := range []F{(*Session).stdin, (*Session).stdout, (*Session).stderr} {
		setupFd(s)
	}

	s.errors = make(chan error, len(s.copyFuncs))
	for _, fn := range s.copyFuncs {
		go func(fn func() error) {
			s.errors <- fn()
		}(fn)
	}
	return nil
}

func (s *Session) Wait() error {
	if !s.started {
		return errors.New("ssh: session not started")
	}
	waitErr := <-s.exitStatus

	if s.stdinPipeWriter != nil {
		s.stdinPipeWriter.Close()
	}
	var copyError error
	for _ = range s.copyFuncs {
		if err := <-s.errors; err != nil && copyError == nil {
			copyError = err
		}
	}
	if waitErr != nil {
		return waitErr
	}
	return copyError
}

func (s *Session) wait(reqs <-chan *Request) error {
	wm := Waitmsg{status: -1}

	for msg := range reqs {
		switch msg.Type {
		case "exit-status":
			wm.status = int(binary.BigEndian.Uint32(msg.Payload))
		case "exit-signal":
			var sigval struct {
				Signal     string
				CoreDumped bool
				Error      string
				Lang       string
			}
			if err := Unmarshal(msg.Payload, &sigval); err != nil {
				return err
			}

			wm.signal = sigval.Signal
			wm.msg = sigval.Error
			wm.lang = sigval.Lang
		default:

			if msg.WantReply {
				msg.Reply(false, nil)
			}
		}
	}
	if wm.status == 0 {
		return nil
	}
	if wm.status == -1 {

		if wm.signal == "" {

			return &ExitMissingError{}
		}
		wm.status = 128
		if _, ok := signals[Signal(wm.signal)]; ok {
			wm.status += signals[Signal(wm.signal)]
		}
	}

	return &ExitError{wm}
}

type ExitMissingError struct{}

func (e *ExitMissingError) Error() string {
	return "wait: remote command exited without exit status or exit signal"
}

func (s *Session) stdin() {
	if s.stdinpipe {
		return
	}
	var stdin io.Reader
	if s.Stdin == nil {
		stdin = new(bytes.Buffer)
	} else {
		r, w := io.Pipe()
		go func() {
			_, err := io.Copy(w, s.Stdin)
			w.CloseWithError(err)
		}()
		stdin, s.stdinPipeWriter = r, w
	}
	s.copyFuncs = append(s.copyFuncs, func() error {
		_, err := io.Copy(s.ch, stdin)
		if err1 := s.ch.CloseWrite(); err == nil && err1 != io.EOF {
			err = err1
		}
		return err
	})
}

func (s *Session) stdout() {
	if s.stdoutpipe {
		return
	}
	if s.Stdout == nil {
		s.Stdout = ioutil.Discard
	}
	s.copyFuncs = append(s.copyFuncs, func() error {
		_, err := io.Copy(s.Stdout, s.ch)
		return err
	})
}

func (s *Session) stderr() {
	if s.stderrpipe {
		return
	}
	if s.Stderr == nil {
		s.Stderr = ioutil.Discard
	}
	s.copyFuncs = append(s.copyFuncs, func() error {
		_, err := io.Copy(s.Stderr, s.ch.Stderr())
		return err
	})
}

type sessionStdin struct {
	io.Writer
	ch Channel
}

func (s *sessionStdin) Close() error {
	return s.ch.CloseWrite()
}

func (s *Session) StdinPipe() (io.WriteCloser, error) {
	if s.Stdin != nil {
		return nil, errors.New("ssh: Stdin already set")
	}
	if s.started {
		return nil, errors.New("ssh: StdinPipe after process started")
	}
	s.stdinpipe = true
	return &sessionStdin{s.ch, s.ch}, nil
}

func (s *Session) StdoutPipe() (io.Reader, error) {
	if s.Stdout != nil {
		return nil, errors.New("ssh: Stdout already set")
	}
	if s.started {
		return nil, errors.New("ssh: StdoutPipe after process started")
	}
	s.stdoutpipe = true
	return s.ch, nil
}

func (s *Session) StderrPipe() (io.Reader, error) {
	if s.Stderr != nil {
		return nil, errors.New("ssh: Stderr already set")
	}
	if s.started {
		return nil, errors.New("ssh: StderrPipe after process started")
	}
	s.stderrpipe = true
	return s.ch.Stderr(), nil
}

func newSession(ch Channel, reqs <-chan *Request) (*Session, error) {
	s := &Session{
		ch: ch,
	}
	s.exitStatus = make(chan error, 1)
	go func() {
		s.exitStatus <- s.wait(reqs)
	}()

	return s, nil
}

type ExitError struct {
	Waitmsg
}

func (e *ExitError) Error() string {
	return e.Waitmsg.String()
}

type Waitmsg struct {
	status int
	signal string
	msg    string
	lang   string
}

func (w Waitmsg) ExitStatus() int {
	return w.status
}

func (w Waitmsg) Signal() string {
	return w.signal
}

func (w Waitmsg) Msg() string {
	return w.msg
}

func (w Waitmsg) Lang() string {
	return w.lang
}

func (w Waitmsg) String() string {
	str := fmt.Sprintf("Process exited with status %v", w.status)
	if w.signal != "" {
		str += fmt.Sprintf(" from signal %v", w.signal)
	}
	if w.msg != "" {
		str += fmt.Sprintf(". Reason was: %v", w.msg)
	}
	return str
}
