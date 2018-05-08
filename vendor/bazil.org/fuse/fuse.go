
package fuse 

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

type Conn struct {

	Ready <-chan struct{}

	MountError error

	dev *os.File
	wio sync.RWMutex
	rio sync.RWMutex

	proto Protocol
}

type MountpointDoesNotExistError struct {
	Path string
}

var _ error = (*MountpointDoesNotExistError)(nil)

func (e *MountpointDoesNotExistError) Error() string {
	return fmt.Sprintf("mountpoint does not exist: %v", e.Path)
}

func Mount(dir string, options ...MountOption) (*Conn, error) {
	conf := mountConfig{
		options: make(map[string]string),
	}
	for _, option := range options {
		if err := option(&conf); err != nil {
			return nil, err
		}
	}

	ready := make(chan struct{}, 1)
	c := &Conn{
		Ready: ready,
	}
	f, err := mount(dir, &conf, ready, &c.MountError)
	if err != nil {
		return nil, err
	}
	c.dev = f

	if err := initMount(c, &conf); err != nil {
		c.Close()
		if err == ErrClosedWithoutInit {

			<-c.Ready
			if err := c.MountError; err != nil {
				return nil, err
			}
		}
		return nil, err
	}

	return c, nil
}

type OldVersionError struct {
	Kernel     Protocol
	LibraryMin Protocol
}

func (e *OldVersionError) Error() string {
	return fmt.Sprintf("kernel FUSE version is too old: %v < %v", e.Kernel, e.LibraryMin)
}

var (
	ErrClosedWithoutInit = errors.New("fuse connection closed without init")
)

func initMount(c *Conn, conf *mountConfig) error {
	req, err := c.ReadRequest()
	if err != nil {
		if err == io.EOF {
			return ErrClosedWithoutInit
		}
		return err
	}
	r, ok := req.(*InitRequest)
	if !ok {
		return fmt.Errorf("missing init, got: %T", req)
	}

	min := Protocol{protoVersionMinMajor, protoVersionMinMinor}
	if r.Kernel.LT(min) {
		req.RespondError(Errno(syscall.EPROTO))
		c.Close()
		return &OldVersionError{
			Kernel:     r.Kernel,
			LibraryMin: min,
		}
	}

	proto := Protocol{protoVersionMaxMajor, protoVersionMaxMinor}
	if r.Kernel.LT(proto) {

		proto = r.Kernel
	}
	c.proto = proto

	s := &InitResponse{
		Library:      proto,
		MaxReadahead: conf.maxReadahead,
		MaxWrite:     maxWrite,
		Flags:        InitBigWrites | conf.initFlags,
	}
	r.Respond(s)
	return nil
}

type Request interface {

	Hdr() *Header

	RespondError(error)

	String() string
}

type RequestID uint64

func (r RequestID) String() string {
	return fmt.Sprintf("%#x", uint64(r))
}

type NodeID uint64

func (n NodeID) String() string {
	return fmt.Sprintf("%#x", uint64(n))
}

type HandleID uint64

func (h HandleID) String() string {
	return fmt.Sprintf("%#x", uint64(h))
}

const RootID NodeID = rootID

type Header struct {
	Conn *Conn     `json:"-"` 
	ID   RequestID 
	Node NodeID    
	Uid  uint32    
	Gid  uint32    
	Pid  uint32    

	msg *message
}

func (h *Header) String() string {
	return fmt.Sprintf("ID=%v Node=%v Uid=%d Gid=%d Pid=%d", h.ID, h.Node, h.Uid, h.Gid, h.Pid)
}

func (h *Header) Hdr() *Header {
	return h
}

func (h *Header) noResponse() {
	putMessage(h.msg)
}

func (h *Header) respond(msg []byte) {
	out := (*outHeader)(unsafe.Pointer(&msg[0]))
	out.Unique = uint64(h.ID)
	h.Conn.respond(msg)
	putMessage(h.msg)
}

type ErrorNumber interface {

	Errno() Errno
}

const (

	ENOSYS = Errno(syscall.ENOSYS)

	ESTALE = Errno(syscall.ESTALE)

	ENOENT = Errno(syscall.ENOENT)
	EIO    = Errno(syscall.EIO)
	EPERM  = Errno(syscall.EPERM)

	EINTR = Errno(syscall.EINTR)

	ERANGE  = Errno(syscall.ERANGE)
	ENOTSUP = Errno(syscall.ENOTSUP)
	EEXIST  = Errno(syscall.EEXIST)
)

const DefaultErrno = EIO

var errnoNames = map[Errno]string{
	ENOSYS: "ENOSYS",
	ESTALE: "ESTALE",
	ENOENT: "ENOENT",
	EIO:    "EIO",
	EPERM:  "EPERM",
	EINTR:  "EINTR",
	EEXIST: "EEXIST",
}

type Errno syscall.Errno

var _ = ErrorNumber(Errno(0))
var _ = error(Errno(0))

func (e Errno) Errno() Errno {
	return e
}

func (e Errno) String() string {
	return syscall.Errno(e).Error()
}

func (e Errno) Error() string {
	return syscall.Errno(e).Error()
}

func (e Errno) ErrnoName() string {
	s := errnoNames[e]
	if s == "" {
		s = fmt.Sprint(e.Errno())
	}
	return s
}

func (e Errno) MarshalText() ([]byte, error) {
	s := e.ErrnoName()
	return []byte(s), nil
}

func (h *Header) RespondError(err error) {
	errno := DefaultErrno
	if ferr, ok := err.(ErrorNumber); ok {
		errno = ferr.Errno()
	}

	buf := newBuffer(0)
	hOut := (*outHeader)(unsafe.Pointer(&buf[0]))
	hOut.Error = -int32(errno)
	h.respond(buf)
}

var maxRequestSize = syscall.Getpagesize()
var bufSize = maxRequestSize + maxWrite

var reqPool = sync.Pool{
	New: allocMessage,
}

func allocMessage() interface{} {
	m := &message{buf: make([]byte, bufSize)}
	m.hdr = (*inHeader)(unsafe.Pointer(&m.buf[0]))
	return m
}

func getMessage(c *Conn) *message {
	m := reqPool.Get().(*message)
	m.conn = c
	return m
}

func putMessage(m *message) {
	m.buf = m.buf[:bufSize]
	m.conn = nil
	m.off = 0
	reqPool.Put(m)
}

type message struct {
	conn *Conn
	buf  []byte    
	hdr  *inHeader 
	off  int       
}

func (m *message) len() uintptr {
	return uintptr(len(m.buf) - m.off)
}

func (m *message) data() unsafe.Pointer {
	var p unsafe.Pointer
	if m.off < len(m.buf) {
		p = unsafe.Pointer(&m.buf[m.off])
	}
	return p
}

func (m *message) bytes() []byte {
	return m.buf[m.off:]
}

func (m *message) Header() Header {
	h := m.hdr
	return Header{
		Conn: m.conn,
		ID:   RequestID(h.Unique),
		Node: NodeID(h.Nodeid),
		Uid:  h.Uid,
		Gid:  h.Gid,
		Pid:  h.Pid,

		msg: m,
	}
}

func fileMode(unixMode uint32) os.FileMode {
	mode := os.FileMode(unixMode & 0777)
	switch unixMode & syscall.S_IFMT {
	case syscall.S_IFREG:

	case syscall.S_IFDIR:
		mode |= os.ModeDir
	case syscall.S_IFCHR:
		mode |= os.ModeCharDevice | os.ModeDevice
	case syscall.S_IFBLK:
		mode |= os.ModeDevice
	case syscall.S_IFIFO:
		mode |= os.ModeNamedPipe
	case syscall.S_IFLNK:
		mode |= os.ModeSymlink
	case syscall.S_IFSOCK:
		mode |= os.ModeSocket
	default:

		mode |= os.ModeDevice
	}
	if unixMode&syscall.S_ISUID != 0 {
		mode |= os.ModeSetuid
	}
	if unixMode&syscall.S_ISGID != 0 {
		mode |= os.ModeSetgid
	}
	return mode
}

type noOpcode struct {
	Opcode uint32
}

func (m noOpcode) String() string {
	return fmt.Sprintf("No opcode %v", m.Opcode)
}

type malformedMessage struct {
}

func (malformedMessage) String() string {
	return "malformed message"
}

func (c *Conn) Close() error {
	c.wio.Lock()
	defer c.wio.Unlock()
	c.rio.Lock()
	defer c.rio.Unlock()
	return c.dev.Close()
}

func (c *Conn) fd() int {
	return int(c.dev.Fd())
}

func (c *Conn) Protocol() Protocol {
	return c.proto
}

func (c *Conn) ReadRequest() (Request, error) {
	m := getMessage(c)
loop:
	c.rio.RLock()
	n, err := syscall.Read(c.fd(), m.buf)
	c.rio.RUnlock()
	if err == syscall.EINTR {

		goto loop
	}
	if err != nil && err != syscall.ENODEV {
		putMessage(m)
		return nil, err
	}
	if n <= 0 {
		putMessage(m)
		return nil, io.EOF
	}
	m.buf = m.buf[:n]

	if n < inHeaderSize {
		putMessage(m)
		return nil, errors.New("fuse: message too short")
	}

	if n == inHeaderSize+initInSize && m.hdr.Opcode == opInit && m.hdr.Len < uint32(n) {
		m.hdr.Len = uint32(n)
	}

	if m.hdr.Len < uint32(n) && m.hdr.Len >= uint32(unsafe.Sizeof(writeIn{})) && m.hdr.Opcode == opWrite {
		m.hdr.Len = uint32(n)
	}

	if m.hdr.Len != uint32(n) {

		err := fmt.Errorf("fuse: read %d opcode %d but expected %d", n, m.hdr.Opcode, m.hdr.Len)
		putMessage(m)
		return nil, err
	}

	m.off = inHeaderSize

	var req Request
	switch m.hdr.Opcode {
	default:
		Debug(noOpcode{Opcode: m.hdr.Opcode})
		goto unrecognized

	case opLookup:
		buf := m.bytes()
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			goto corrupt
		}
		req = &LookupRequest{
			Header: m.Header(),
			Name:   string(buf[:n-1]),
		}

	case opForget:
		in := (*forgetIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &ForgetRequest{
			Header: m.Header(),
			N:      in.Nlookup,
		}

	case opGetattr:
		switch {
		case c.proto.LT(Protocol{7, 9}):
			req = &GetattrRequest{
				Header: m.Header(),
			}

		default:
			in := (*getattrIn)(m.data())
			if m.len() < unsafe.Sizeof(*in) {
				goto corrupt
			}
			req = &GetattrRequest{
				Header: m.Header(),
				Flags:  GetattrFlags(in.GetattrFlags),
				Handle: HandleID(in.Fh),
			}
		}

	case opSetattr:
		in := (*setattrIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &SetattrRequest{
			Header:   m.Header(),
			Valid:    SetattrValid(in.Valid),
			Handle:   HandleID(in.Fh),
			Size:     in.Size,
			Atime:    time.Unix(int64(in.Atime), int64(in.AtimeNsec)),
			Mtime:    time.Unix(int64(in.Mtime), int64(in.MtimeNsec)),
			Mode:     fileMode(in.Mode),
			Uid:      in.Uid,
			Gid:      in.Gid,
			Bkuptime: in.BkupTime(),
			Chgtime:  in.Chgtime(),
			Flags:    in.Flags(),
		}

	case opReadlink:
		if len(m.bytes()) > 0 {
			goto corrupt
		}
		req = &ReadlinkRequest{
			Header: m.Header(),
		}

	case opSymlink:

		names := m.bytes()
		if len(names) == 0 || names[len(names)-1] != 0 {
			goto corrupt
		}
		i := bytes.IndexByte(names, '\x00')
		if i < 0 {
			goto corrupt
		}
		newName, target := names[0:i], names[i+1:len(names)-1]
		req = &SymlinkRequest{
			Header:  m.Header(),
			NewName: string(newName),
			Target:  string(target),
		}

	case opLink:
		in := (*linkIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		newName := m.bytes()[unsafe.Sizeof(*in):]
		if len(newName) < 2 || newName[len(newName)-1] != 0 {
			goto corrupt
		}
		newName = newName[:len(newName)-1]
		req = &LinkRequest{
			Header:  m.Header(),
			OldNode: NodeID(in.Oldnodeid),
			NewName: string(newName),
		}

	case opMknod:
		size := mknodInSize(c.proto)
		if m.len() < size {
			goto corrupt
		}
		in := (*mknodIn)(m.data())
		name := m.bytes()[size:]
		if len(name) < 2 || name[len(name)-1] != '\x00' {
			goto corrupt
		}
		name = name[:len(name)-1]
		r := &MknodRequest{
			Header: m.Header(),
			Mode:   fileMode(in.Mode),
			Rdev:   in.Rdev,
			Name:   string(name),
		}
		if c.proto.GE(Protocol{7, 12}) {
			r.Umask = fileMode(in.Umask) & os.ModePerm
		}
		req = r

	case opMkdir:
		size := mkdirInSize(c.proto)
		if m.len() < size {
			goto corrupt
		}
		in := (*mkdirIn)(m.data())
		name := m.bytes()[size:]
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			goto corrupt
		}
		r := &MkdirRequest{
			Header: m.Header(),
			Name:   string(name[:i]),

			Mode: fileMode((in.Mode &^ syscall.S_IFMT) | syscall.S_IFDIR),
		}
		if c.proto.GE(Protocol{7, 12}) {
			r.Umask = fileMode(in.Umask) & os.ModePerm
		}
		req = r

	case opUnlink, opRmdir:
		buf := m.bytes()
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			goto corrupt
		}
		req = &RemoveRequest{
			Header: m.Header(),
			Name:   string(buf[:n-1]),
			Dir:    m.hdr.Opcode == opRmdir,
		}

	case opRename:
		in := (*renameIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		newDirNodeID := NodeID(in.Newdir)
		oldNew := m.bytes()[unsafe.Sizeof(*in):]

		if len(oldNew) < 4 {
			goto corrupt
		}
		if oldNew[len(oldNew)-1] != '\x00' {
			goto corrupt
		}
		i := bytes.IndexByte(oldNew, '\x00')
		if i < 0 {
			goto corrupt
		}
		oldName, newName := string(oldNew[:i]), string(oldNew[i+1:len(oldNew)-1])
		req = &RenameRequest{
			Header:  m.Header(),
			NewDir:  newDirNodeID,
			OldName: oldName,
			NewName: newName,
		}

	case opOpendir, opOpen:
		in := (*openIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &OpenRequest{
			Header: m.Header(),
			Dir:    m.hdr.Opcode == opOpendir,
			Flags:  openFlags(in.Flags),
		}

	case opRead, opReaddir:
		in := (*readIn)(m.data())
		if m.len() < readInSize(c.proto) {
			goto corrupt
		}
		r := &ReadRequest{
			Header: m.Header(),
			Dir:    m.hdr.Opcode == opReaddir,
			Handle: HandleID(in.Fh),
			Offset: int64(in.Offset),
			Size:   int(in.Size),
		}
		if c.proto.GE(Protocol{7, 9}) {
			r.Flags = ReadFlags(in.ReadFlags)
			r.LockOwner = in.LockOwner
			r.FileFlags = openFlags(in.Flags)
		}
		req = r

	case opWrite:
		in := (*writeIn)(m.data())
		if m.len() < writeInSize(c.proto) {
			goto corrupt
		}
		r := &WriteRequest{
			Header: m.Header(),
			Handle: HandleID(in.Fh),
			Offset: int64(in.Offset),
			Flags:  WriteFlags(in.WriteFlags),
		}
		if c.proto.GE(Protocol{7, 9}) {
			r.LockOwner = in.LockOwner
			r.FileFlags = openFlags(in.Flags)
		}
		buf := m.bytes()[writeInSize(c.proto):]
		if uint32(len(buf)) < in.Size {
			goto corrupt
		}
		r.Data = buf
		req = r

	case opStatfs:
		req = &StatfsRequest{
			Header: m.Header(),
		}

	case opRelease, opReleasedir:
		in := (*releaseIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &ReleaseRequest{
			Header:       m.Header(),
			Dir:          m.hdr.Opcode == opReleasedir,
			Handle:       HandleID(in.Fh),
			Flags:        openFlags(in.Flags),
			ReleaseFlags: ReleaseFlags(in.ReleaseFlags),
			LockOwner:    in.LockOwner,
		}

	case opFsync, opFsyncdir:
		in := (*fsyncIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &FsyncRequest{
			Dir:    m.hdr.Opcode == opFsyncdir,
			Header: m.Header(),
			Handle: HandleID(in.Fh),
			Flags:  in.FsyncFlags,
		}

	case opSetxattr:
		in := (*setxattrIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		m.off += int(unsafe.Sizeof(*in))
		name := m.bytes()
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			goto corrupt
		}
		xattr := name[i+1:]
		if uint32(len(xattr)) < in.Size {
			goto corrupt
		}
		xattr = xattr[:in.Size]
		req = &SetxattrRequest{
			Header:   m.Header(),
			Flags:    in.Flags,
			Position: in.position(),
			Name:     string(name[:i]),
			Xattr:    xattr,
		}

	case opGetxattr:
		in := (*getxattrIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		name := m.bytes()[unsafe.Sizeof(*in):]
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			goto corrupt
		}
		req = &GetxattrRequest{
			Header:   m.Header(),
			Name:     string(name[:i]),
			Size:     in.Size,
			Position: in.position(),
		}

	case opListxattr:
		in := (*getxattrIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &ListxattrRequest{
			Header:   m.Header(),
			Size:     in.Size,
			Position: in.position(),
		}

	case opRemovexattr:
		buf := m.bytes()
		n := len(buf)
		if n == 0 || buf[n-1] != '\x00' {
			goto corrupt
		}
		req = &RemovexattrRequest{
			Header: m.Header(),
			Name:   string(buf[:n-1]),
		}

	case opFlush:
		in := (*flushIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &FlushRequest{
			Header:    m.Header(),
			Handle:    HandleID(in.Fh),
			Flags:     in.FlushFlags,
			LockOwner: in.LockOwner,
		}

	case opInit:
		in := (*initIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &InitRequest{
			Header:       m.Header(),
			Kernel:       Protocol{in.Major, in.Minor},
			MaxReadahead: in.MaxReadahead,
			Flags:        InitFlags(in.Flags),
		}

	case opGetlk:
		panic("opGetlk")
	case opSetlk:
		panic("opSetlk")
	case opSetlkw:
		panic("opSetlkw")

	case opAccess:
		in := (*accessIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &AccessRequest{
			Header: m.Header(),
			Mask:   in.Mask,
		}

	case opCreate:
		size := createInSize(c.proto)
		if m.len() < size {
			goto corrupt
		}
		in := (*createIn)(m.data())
		name := m.bytes()[size:]
		i := bytes.IndexByte(name, '\x00')
		if i < 0 {
			goto corrupt
		}
		r := &CreateRequest{
			Header: m.Header(),
			Flags:  openFlags(in.Flags),
			Mode:   fileMode(in.Mode),
			Name:   string(name[:i]),
		}
		if c.proto.GE(Protocol{7, 12}) {
			r.Umask = fileMode(in.Umask) & os.ModePerm
		}
		req = r

	case opInterrupt:
		in := (*interruptIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		req = &InterruptRequest{
			Header: m.Header(),
			IntrID: RequestID(in.Unique),
		}

	case opBmap:
		panic("opBmap")

	case opDestroy:
		req = &DestroyRequest{
			Header: m.Header(),
		}

	case opSetvolname:
		panic("opSetvolname")
	case opGetxtimes:
		panic("opGetxtimes")
	case opExchange:
		in := (*exchangeIn)(m.data())
		if m.len() < unsafe.Sizeof(*in) {
			goto corrupt
		}
		oldDirNodeID := NodeID(in.Olddir)
		newDirNodeID := NodeID(in.Newdir)
		oldNew := m.bytes()[unsafe.Sizeof(*in):]

		if len(oldNew) < 4 {
			goto corrupt
		}
		if oldNew[len(oldNew)-1] != '\x00' {
			goto corrupt
		}
		i := bytes.IndexByte(oldNew, '\x00')
		if i < 0 {
			goto corrupt
		}
		oldName, newName := string(oldNew[:i]), string(oldNew[i+1:len(oldNew)-1])
		req = &ExchangeDataRequest{
			Header:  m.Header(),
			OldDir:  oldDirNodeID,
			NewDir:  newDirNodeID,
			OldName: oldName,
			NewName: newName,

		}
	}

	return req, nil

corrupt:
	Debug(malformedMessage{})
	putMessage(m)
	return nil, fmt.Errorf("fuse: malformed message")

unrecognized:

	h := m.Header()
	return &h, nil
}

type bugShortKernelWrite struct {
	Written int64
	Length  int64
	Error   string
	Stack   string
}

func (b bugShortKernelWrite) String() string {
	return fmt.Sprintf("short kernel write: written=%d/%d error=%q stack=\n%s", b.Written, b.Length, b.Error, b.Stack)
}

type bugKernelWriteError struct {
	Error string
	Stack string
}

func (b bugKernelWriteError) String() string {
	return fmt.Sprintf("kernel write error: error=%q stack=\n%s", b.Error, b.Stack)
}

func errorString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (c *Conn) writeToKernel(msg []byte) error {
	out := (*outHeader)(unsafe.Pointer(&msg[0]))
	out.Len = uint32(len(msg))

	c.wio.RLock()
	defer c.wio.RUnlock()
	nn, err := syscall.Write(c.fd(), msg)
	if err == nil && nn != len(msg) {
		Debug(bugShortKernelWrite{
			Written: int64(nn),
			Length:  int64(len(msg)),
			Error:   errorString(err),
			Stack:   stack(),
		})
	}
	return err
}

func (c *Conn) respond(msg []byte) {
	if err := c.writeToKernel(msg); err != nil {
		Debug(bugKernelWriteError{
			Error: errorString(err),
			Stack: stack(),
		})
	}
}

type notCachedError struct{}

func (notCachedError) Error() string {
	return "node not cached"
}

var _ ErrorNumber = notCachedError{}

func (notCachedError) Errno() Errno {

	return ENOENT
}

var (
	ErrNotCached = notCachedError{}
)

func (c *Conn) sendInvalidate(msg []byte) error {
	switch err := c.writeToKernel(msg); err {
	case syscall.ENOENT:
		return ErrNotCached
	default:
		return err
	}
}

func (c *Conn) InvalidateNode(nodeID NodeID, off int64, size int64) error {
	buf := newBuffer(unsafe.Sizeof(notifyInvalInodeOut{}))
	h := (*outHeader)(unsafe.Pointer(&buf[0]))

	h.Error = notifyCodeInvalInode
	out := (*notifyInvalInodeOut)(buf.alloc(unsafe.Sizeof(notifyInvalInodeOut{})))
	out.Ino = uint64(nodeID)
	out.Off = off
	out.Len = size
	return c.sendInvalidate(buf)
}

func (c *Conn) InvalidateEntry(parent NodeID, name string) error {
	const maxUint32 = ^uint32(0)
	if uint64(len(name)) > uint64(maxUint32) {

		return syscall.ENAMETOOLONG
	}
	buf := newBuffer(unsafe.Sizeof(notifyInvalEntryOut{}) + uintptr(len(name)) + 1)
	h := (*outHeader)(unsafe.Pointer(&buf[0]))

	h.Error = notifyCodeInvalEntry
	out := (*notifyInvalEntryOut)(buf.alloc(unsafe.Sizeof(notifyInvalEntryOut{})))
	out.Parent = uint64(parent)
	out.Namelen = uint32(len(name))
	buf = append(buf, name...)
	buf = append(buf, '\x00')
	return c.sendInvalidate(buf)
}

type InitRequest struct {
	Header `json:"-"`
	Kernel Protocol

	MaxReadahead uint32
	Flags        InitFlags
}

var _ = Request(&InitRequest{})

func (r *InitRequest) String() string {
	return fmt.Sprintf("Init [%v] %v ra=%d fl=%v", &r.Header, r.Kernel, r.MaxReadahead, r.Flags)
}

type InitResponse struct {
	Library Protocol

	MaxReadahead uint32
	Flags        InitFlags

	MaxWrite uint32
}

func (r *InitResponse) String() string {
	return fmt.Sprintf("Init %v ra=%d fl=%v w=%d", r.Library, r.MaxReadahead, r.Flags, r.MaxWrite)
}

func (r *InitRequest) Respond(resp *InitResponse) {
	buf := newBuffer(unsafe.Sizeof(initOut{}))
	out := (*initOut)(buf.alloc(unsafe.Sizeof(initOut{})))
	out.Major = resp.Library.Major
	out.Minor = resp.Library.Minor
	out.MaxReadahead = resp.MaxReadahead
	out.Flags = uint32(resp.Flags)
	out.MaxWrite = resp.MaxWrite

	if out.MaxWrite > maxWrite {
		out.MaxWrite = maxWrite
	}
	r.respond(buf)
}

type StatfsRequest struct {
	Header `json:"-"`
}

var _ = Request(&StatfsRequest{})

func (r *StatfsRequest) String() string {
	return fmt.Sprintf("Statfs [%s]", &r.Header)
}

func (r *StatfsRequest) Respond(resp *StatfsResponse) {
	buf := newBuffer(unsafe.Sizeof(statfsOut{}))
	out := (*statfsOut)(buf.alloc(unsafe.Sizeof(statfsOut{})))
	out.St = kstatfs{
		Blocks:  resp.Blocks,
		Bfree:   resp.Bfree,
		Bavail:  resp.Bavail,
		Files:   resp.Files,
		Bsize:   resp.Bsize,
		Namelen: resp.Namelen,
		Frsize:  resp.Frsize,
	}
	r.respond(buf)
}

type StatfsResponse struct {
	Blocks  uint64 
	Bfree   uint64 
	Bavail  uint64 
	Files   uint64 
	Ffree   uint64 
	Bsize   uint32 
	Namelen uint32 
	Frsize  uint32 
}

func (r *StatfsResponse) String() string {
	return fmt.Sprintf("Statfs blocks=%d/%d/%d files=%d/%d bsize=%d frsize=%d namelen=%d",
		r.Bavail, r.Bfree, r.Blocks,
		r.Ffree, r.Files,
		r.Bsize,
		r.Frsize,
		r.Namelen,
	)
}

type AccessRequest struct {
	Header `json:"-"`
	Mask   uint32
}

var _ = Request(&AccessRequest{})

func (r *AccessRequest) String() string {
	return fmt.Sprintf("Access [%s] mask=%#x", &r.Header, r.Mask)
}

func (r *AccessRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type Attr struct {
	Valid time.Duration 

	Inode     uint64      
	Size      uint64      
	Blocks    uint64      
	Atime     time.Time   
	Mtime     time.Time   
	Ctime     time.Time   
	Crtime    time.Time   
	Mode      os.FileMode 
	Nlink     uint32      
	Uid       uint32      
	Gid       uint32      
	Rdev      uint32      
	Flags     uint32      
	BlockSize uint32      
}

func (a Attr) String() string {
	return fmt.Sprintf("valid=%v ino=%v size=%d mode=%v", a.Valid, a.Inode, a.Size, a.Mode)
}

func unix(t time.Time) (sec uint64, nsec uint32) {
	nano := t.UnixNano()
	sec = uint64(nano / 1e9)
	nsec = uint32(nano % 1e9)
	return
}

func (a *Attr) attr(out *attr, proto Protocol) {
	out.Ino = a.Inode
	out.Size = a.Size
	out.Blocks = a.Blocks
	out.Atime, out.AtimeNsec = unix(a.Atime)
	out.Mtime, out.MtimeNsec = unix(a.Mtime)
	out.Ctime, out.CtimeNsec = unix(a.Ctime)
	out.SetCrtime(unix(a.Crtime))
	out.Mode = uint32(a.Mode) & 0777
	switch {
	default:
		out.Mode |= syscall.S_IFREG
	case a.Mode&os.ModeDir != 0:
		out.Mode |= syscall.S_IFDIR
	case a.Mode&os.ModeDevice != 0:
		if a.Mode&os.ModeCharDevice != 0 {
			out.Mode |= syscall.S_IFCHR
		} else {
			out.Mode |= syscall.S_IFBLK
		}
	case a.Mode&os.ModeNamedPipe != 0:
		out.Mode |= syscall.S_IFIFO
	case a.Mode&os.ModeSymlink != 0:
		out.Mode |= syscall.S_IFLNK
	case a.Mode&os.ModeSocket != 0:
		out.Mode |= syscall.S_IFSOCK
	}
	if a.Mode&os.ModeSetuid != 0 {
		out.Mode |= syscall.S_ISUID
	}
	if a.Mode&os.ModeSetgid != 0 {
		out.Mode |= syscall.S_ISGID
	}
	out.Nlink = a.Nlink
	out.Uid = a.Uid
	out.Gid = a.Gid
	out.Rdev = a.Rdev
	out.SetFlags(a.Flags)
	if proto.GE(Protocol{7, 9}) {
		out.Blksize = a.BlockSize
	}

	return
}

type GetattrRequest struct {
	Header `json:"-"`
	Flags  GetattrFlags
	Handle HandleID
}

var _ = Request(&GetattrRequest{})

func (r *GetattrRequest) String() string {
	return fmt.Sprintf("Getattr [%s] %v fl=%v", &r.Header, r.Handle, r.Flags)
}

func (r *GetattrRequest) Respond(resp *GetattrResponse) {
	size := attrOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*attrOut)(buf.alloc(size))
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type GetattrResponse struct {
	Attr Attr 
}

func (r *GetattrResponse) String() string {
	return fmt.Sprintf("Getattr %v", r.Attr)
}

type GetxattrRequest struct {
	Header `json:"-"`

	Size uint32

	Name string

	Position uint32
}

var _ = Request(&GetxattrRequest{})

func (r *GetxattrRequest) String() string {
	return fmt.Sprintf("Getxattr [%s] %q %d @%d", &r.Header, r.Name, r.Size, r.Position)
}

func (r *GetxattrRequest) Respond(resp *GetxattrResponse) {
	if r.Size == 0 {
		buf := newBuffer(unsafe.Sizeof(getxattrOut{}))
		out := (*getxattrOut)(buf.alloc(unsafe.Sizeof(getxattrOut{})))
		out.Size = uint32(len(resp.Xattr))
		r.respond(buf)
	} else {
		buf := newBuffer(uintptr(len(resp.Xattr)))
		buf = append(buf, resp.Xattr...)
		r.respond(buf)
	}
}

type GetxattrResponse struct {
	Xattr []byte
}

func (r *GetxattrResponse) String() string {
	return fmt.Sprintf("Getxattr %x", r.Xattr)
}

type ListxattrRequest struct {
	Header   `json:"-"`
	Size     uint32 
	Position uint32 
}

var _ = Request(&ListxattrRequest{})

func (r *ListxattrRequest) String() string {
	return fmt.Sprintf("Listxattr [%s] %d @%d", &r.Header, r.Size, r.Position)
}

func (r *ListxattrRequest) Respond(resp *ListxattrResponse) {
	if r.Size == 0 {
		buf := newBuffer(unsafe.Sizeof(getxattrOut{}))
		out := (*getxattrOut)(buf.alloc(unsafe.Sizeof(getxattrOut{})))
		out.Size = uint32(len(resp.Xattr))
		r.respond(buf)
	} else {
		buf := newBuffer(uintptr(len(resp.Xattr)))
		buf = append(buf, resp.Xattr...)
		r.respond(buf)
	}
}

type ListxattrResponse struct {
	Xattr []byte
}

func (r *ListxattrResponse) String() string {
	return fmt.Sprintf("Listxattr %x", r.Xattr)
}

func (r *ListxattrResponse) Append(names ...string) {
	for _, name := range names {
		r.Xattr = append(r.Xattr, name...)
		r.Xattr = append(r.Xattr, '\x00')
	}
}

type RemovexattrRequest struct {
	Header `json:"-"`
	Name   string 
}

var _ = Request(&RemovexattrRequest{})

func (r *RemovexattrRequest) String() string {
	return fmt.Sprintf("Removexattr [%s] %q", &r.Header, r.Name)
}

func (r *RemovexattrRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type SetxattrRequest struct {
	Header `json:"-"`

	Flags uint32

	Position uint32

	Name  string
	Xattr []byte
}

var _ = Request(&SetxattrRequest{})

func trunc(b []byte, max int) ([]byte, string) {
	if len(b) > max {
		return b[:max], "..."
	}
	return b, ""
}

func (r *SetxattrRequest) String() string {
	xattr, tail := trunc(r.Xattr, 16)
	return fmt.Sprintf("Setxattr [%s] %q %x%s fl=%v @%#x", &r.Header, r.Name, xattr, tail, r.Flags, r.Position)
}

func (r *SetxattrRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type LookupRequest struct {
	Header `json:"-"`
	Name   string
}

var _ = Request(&LookupRequest{})

func (r *LookupRequest) String() string {
	return fmt.Sprintf("Lookup [%s] %q", &r.Header, r.Name)
}

func (r *LookupRequest) Respond(resp *LookupResponse) {
	size := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*entryOut)(buf.alloc(size))
	out.Nodeid = uint64(resp.Node)
	out.Generation = resp.Generation
	out.EntryValid = uint64(resp.EntryValid / time.Second)
	out.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type LookupResponse struct {
	Node       NodeID
	Generation uint64
	EntryValid time.Duration
	Attr       Attr
}

func (r *LookupResponse) string() string {
	return fmt.Sprintf("%v gen=%d valid=%v attr={%v}", r.Node, r.Generation, r.EntryValid, r.Attr)
}

func (r *LookupResponse) String() string {
	return fmt.Sprintf("Lookup %s", r.string())
}

type OpenRequest struct {
	Header `json:"-"`
	Dir    bool 
	Flags  OpenFlags
}

var _ = Request(&OpenRequest{})

func (r *OpenRequest) String() string {
	return fmt.Sprintf("Open [%s] dir=%v fl=%v", &r.Header, r.Dir, r.Flags)
}

func (r *OpenRequest) Respond(resp *OpenResponse) {
	buf := newBuffer(unsafe.Sizeof(openOut{}))
	out := (*openOut)(buf.alloc(unsafe.Sizeof(openOut{})))
	out.Fh = uint64(resp.Handle)
	out.OpenFlags = uint32(resp.Flags)
	r.respond(buf)
}

type OpenResponse struct {
	Handle HandleID
	Flags  OpenResponseFlags
}

func (r *OpenResponse) string() string {
	return fmt.Sprintf("%v fl=%v", r.Handle, r.Flags)
}

func (r *OpenResponse) String() string {
	return fmt.Sprintf("Open %s", r.string())
}

type CreateRequest struct {
	Header `json:"-"`
	Name   string
	Flags  OpenFlags
	Mode   os.FileMode

	Umask os.FileMode
}

var _ = Request(&CreateRequest{})

func (r *CreateRequest) String() string {
	return fmt.Sprintf("Create [%s] %q fl=%v mode=%v umask=%v", &r.Header, r.Name, r.Flags, r.Mode, r.Umask)
}

func (r *CreateRequest) Respond(resp *CreateResponse) {
	eSize := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(eSize + unsafe.Sizeof(openOut{}))

	e := (*entryOut)(buf.alloc(eSize))
	e.Nodeid = uint64(resp.Node)
	e.Generation = resp.Generation
	e.EntryValid = uint64(resp.EntryValid / time.Second)
	e.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	e.AttrValid = uint64(resp.Attr.Valid / time.Second)
	e.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&e.Attr, r.Header.Conn.proto)

	o := (*openOut)(buf.alloc(unsafe.Sizeof(openOut{})))
	o.Fh = uint64(resp.Handle)
	o.OpenFlags = uint32(resp.Flags)

	r.respond(buf)
}

type CreateResponse struct {
	LookupResponse
	OpenResponse
}

func (r *CreateResponse) String() string {
	return fmt.Sprintf("Create {%s} {%s}", r.LookupResponse.string(), r.OpenResponse.string())
}

type MkdirRequest struct {
	Header `json:"-"`
	Name   string
	Mode   os.FileMode

	Umask os.FileMode
}

var _ = Request(&MkdirRequest{})

func (r *MkdirRequest) String() string {
	return fmt.Sprintf("Mkdir [%s] %q mode=%v umask=%v", &r.Header, r.Name, r.Mode, r.Umask)
}

func (r *MkdirRequest) Respond(resp *MkdirResponse) {
	size := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*entryOut)(buf.alloc(size))
	out.Nodeid = uint64(resp.Node)
	out.Generation = resp.Generation
	out.EntryValid = uint64(resp.EntryValid / time.Second)
	out.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type MkdirResponse struct {
	LookupResponse
}

func (r *MkdirResponse) String() string {
	return fmt.Sprintf("Mkdir %v", r.LookupResponse.string())
}

type ReadRequest struct {
	Header    `json:"-"`
	Dir       bool 
	Handle    HandleID
	Offset    int64
	Size      int
	Flags     ReadFlags
	LockOwner uint64
	FileFlags OpenFlags
}

var _ = Request(&ReadRequest{})

func (r *ReadRequest) String() string {
	return fmt.Sprintf("Read [%s] %v %d @%#x dir=%v fl=%v lock=%d ffl=%v", &r.Header, r.Handle, r.Size, r.Offset, r.Dir, r.Flags, r.LockOwner, r.FileFlags)
}

func (r *ReadRequest) Respond(resp *ReadResponse) {
	buf := newBuffer(uintptr(len(resp.Data)))
	buf = append(buf, resp.Data...)
	r.respond(buf)
}

type ReadResponse struct {
	Data []byte
}

func (r *ReadResponse) String() string {
	return fmt.Sprintf("Read %d", len(r.Data))
}

type jsonReadResponse struct {
	Len uint64
}

func (r *ReadResponse) MarshalJSON() ([]byte, error) {
	j := jsonReadResponse{
		Len: uint64(len(r.Data)),
	}
	return json.Marshal(j)
}

type ReleaseRequest struct {
	Header       `json:"-"`
	Dir          bool 
	Handle       HandleID
	Flags        OpenFlags 
	ReleaseFlags ReleaseFlags
	LockOwner    uint32
}

var _ = Request(&ReleaseRequest{})

func (r *ReleaseRequest) String() string {
	return fmt.Sprintf("Release [%s] %v fl=%v rfl=%v owner=%#x", &r.Header, r.Handle, r.Flags, r.ReleaseFlags, r.LockOwner)
}

func (r *ReleaseRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type DestroyRequest struct {
	Header `json:"-"`
}

var _ = Request(&DestroyRequest{})

func (r *DestroyRequest) String() string {
	return fmt.Sprintf("Destroy [%s]", &r.Header)
}

func (r *DestroyRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type ForgetRequest struct {
	Header `json:"-"`
	N      uint64
}

var _ = Request(&ForgetRequest{})

func (r *ForgetRequest) String() string {
	return fmt.Sprintf("Forget [%s] %d", &r.Header, r.N)
}

func (r *ForgetRequest) Respond() {

	r.noResponse()
}

type Dirent struct {

	Inode uint64

	Type DirentType

	Name string
}

type DirentType uint32

const (

	DT_Unknown DirentType = 0
	DT_Socket  DirentType = syscall.S_IFSOCK >> 12
	DT_Link    DirentType = syscall.S_IFLNK >> 12
	DT_File    DirentType = syscall.S_IFREG >> 12
	DT_Block   DirentType = syscall.S_IFBLK >> 12
	DT_Dir     DirentType = syscall.S_IFDIR >> 12
	DT_Char    DirentType = syscall.S_IFCHR >> 12
	DT_FIFO    DirentType = syscall.S_IFIFO >> 12
)

func (t DirentType) String() string {
	switch t {
	case DT_Unknown:
		return "unknown"
	case DT_Socket:
		return "socket"
	case DT_Link:
		return "link"
	case DT_File:
		return "file"
	case DT_Block:
		return "block"
	case DT_Dir:
		return "dir"
	case DT_Char:
		return "char"
	case DT_FIFO:
		return "fifo"
	}
	return "invalid"
}

func AppendDirent(data []byte, dir Dirent) []byte {
	de := dirent{
		Ino:     dir.Inode,
		Namelen: uint32(len(dir.Name)),
		Type:    uint32(dir.Type),
	}
	de.Off = uint64(len(data) + direntSize + (len(dir.Name)+7)&^7)
	data = append(data, (*[direntSize]byte)(unsafe.Pointer(&de))[:]...)
	data = append(data, dir.Name...)
	n := direntSize + uintptr(len(dir.Name))
	if n%8 != 0 {
		var pad [8]byte
		data = append(data, pad[:8-n%8]...)
	}
	return data
}

type WriteRequest struct {
	Header
	Handle    HandleID
	Offset    int64
	Data      []byte
	Flags     WriteFlags
	LockOwner uint64
	FileFlags OpenFlags
}

var _ = Request(&WriteRequest{})

func (r *WriteRequest) String() string {
	return fmt.Sprintf("Write [%s] %v %d @%d fl=%v lock=%d ffl=%v", &r.Header, r.Handle, len(r.Data), r.Offset, r.Flags, r.LockOwner, r.FileFlags)
}

type jsonWriteRequest struct {
	Handle HandleID
	Offset int64
	Len    uint64
	Flags  WriteFlags
}

func (r *WriteRequest) MarshalJSON() ([]byte, error) {
	j := jsonWriteRequest{
		Handle: r.Handle,
		Offset: r.Offset,
		Len:    uint64(len(r.Data)),
		Flags:  r.Flags,
	}
	return json.Marshal(j)
}

func (r *WriteRequest) Respond(resp *WriteResponse) {
	buf := newBuffer(unsafe.Sizeof(writeOut{}))
	out := (*writeOut)(buf.alloc(unsafe.Sizeof(writeOut{})))
	out.Size = uint32(resp.Size)
	r.respond(buf)
}

type WriteResponse struct {
	Size int
}

func (r *WriteResponse) String() string {
	return fmt.Sprintf("Write %d", r.Size)
}

type SetattrRequest struct {
	Header `json:"-"`
	Valid  SetattrValid
	Handle HandleID
	Size   uint64
	Atime  time.Time
	Mtime  time.Time
	Mode   os.FileMode
	Uid    uint32
	Gid    uint32

	Bkuptime time.Time
	Chgtime  time.Time
	Crtime   time.Time
	Flags    uint32 
}

var _ = Request(&SetattrRequest{})

func (r *SetattrRequest) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "Setattr [%s]", &r.Header)
	if r.Valid.Mode() {
		fmt.Fprintf(&buf, " mode=%v", r.Mode)
	}
	if r.Valid.Uid() {
		fmt.Fprintf(&buf, " uid=%d", r.Uid)
	}
	if r.Valid.Gid() {
		fmt.Fprintf(&buf, " gid=%d", r.Gid)
	}
	if r.Valid.Size() {
		fmt.Fprintf(&buf, " size=%d", r.Size)
	}
	if r.Valid.Atime() {
		fmt.Fprintf(&buf, " atime=%v", r.Atime)
	}
	if r.Valid.AtimeNow() {
		fmt.Fprintf(&buf, " atime=now")
	}
	if r.Valid.Mtime() {
		fmt.Fprintf(&buf, " mtime=%v", r.Mtime)
	}
	if r.Valid.MtimeNow() {
		fmt.Fprintf(&buf, " mtime=now")
	}
	if r.Valid.Handle() {
		fmt.Fprintf(&buf, " handle=%v", r.Handle)
	} else {
		fmt.Fprintf(&buf, " handle=INVALID-%v", r.Handle)
	}
	if r.Valid.LockOwner() {
		fmt.Fprintf(&buf, " lockowner")
	}
	if r.Valid.Crtime() {
		fmt.Fprintf(&buf, " crtime=%v", r.Crtime)
	}
	if r.Valid.Chgtime() {
		fmt.Fprintf(&buf, " chgtime=%v", r.Chgtime)
	}
	if r.Valid.Bkuptime() {
		fmt.Fprintf(&buf, " bkuptime=%v", r.Bkuptime)
	}
	if r.Valid.Flags() {
		fmt.Fprintf(&buf, " flags=%v", r.Flags)
	}
	return buf.String()
}

func (r *SetattrRequest) Respond(resp *SetattrResponse) {
	size := attrOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*attrOut)(buf.alloc(size))
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type SetattrResponse struct {
	Attr Attr 
}

func (r *SetattrResponse) String() string {
	return fmt.Sprintf("Setattr %v", r.Attr)
}

type FlushRequest struct {
	Header    `json:"-"`
	Handle    HandleID
	Flags     uint32
	LockOwner uint64
}

var _ = Request(&FlushRequest{})

func (r *FlushRequest) String() string {
	return fmt.Sprintf("Flush [%s] %v fl=%#x lk=%#x", &r.Header, r.Handle, r.Flags, r.LockOwner)
}

func (r *FlushRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type RemoveRequest struct {
	Header `json:"-"`
	Name   string 
	Dir    bool   
}

var _ = Request(&RemoveRequest{})

func (r *RemoveRequest) String() string {
	return fmt.Sprintf("Remove [%s] %q dir=%v", &r.Header, r.Name, r.Dir)
}

func (r *RemoveRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type SymlinkRequest struct {
	Header          `json:"-"`
	NewName, Target string
}

var _ = Request(&SymlinkRequest{})

func (r *SymlinkRequest) String() string {
	return fmt.Sprintf("Symlink [%s] from %q to target %q", &r.Header, r.NewName, r.Target)
}

func (r *SymlinkRequest) Respond(resp *SymlinkResponse) {
	size := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*entryOut)(buf.alloc(size))
	out.Nodeid = uint64(resp.Node)
	out.Generation = resp.Generation
	out.EntryValid = uint64(resp.EntryValid / time.Second)
	out.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type SymlinkResponse struct {
	LookupResponse
}

func (r *SymlinkResponse) String() string {
	return fmt.Sprintf("Symlink %v", r.LookupResponse.string())
}

type ReadlinkRequest struct {
	Header `json:"-"`
}

var _ = Request(&ReadlinkRequest{})

func (r *ReadlinkRequest) String() string {
	return fmt.Sprintf("Readlink [%s]", &r.Header)
}

func (r *ReadlinkRequest) Respond(target string) {
	buf := newBuffer(uintptr(len(target)))
	buf = append(buf, target...)
	r.respond(buf)
}

type LinkRequest struct {
	Header  `json:"-"`
	OldNode NodeID
	NewName string
}

var _ = Request(&LinkRequest{})

func (r *LinkRequest) String() string {
	return fmt.Sprintf("Link [%s] node %d to %q", &r.Header, r.OldNode, r.NewName)
}

func (r *LinkRequest) Respond(resp *LookupResponse) {
	size := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*entryOut)(buf.alloc(size))
	out.Nodeid = uint64(resp.Node)
	out.Generation = resp.Generation
	out.EntryValid = uint64(resp.EntryValid / time.Second)
	out.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type RenameRequest struct {
	Header           `json:"-"`
	NewDir           NodeID
	OldName, NewName string
}

var _ = Request(&RenameRequest{})

func (r *RenameRequest) String() string {
	return fmt.Sprintf("Rename [%s] from %q to dirnode %v %q", &r.Header, r.OldName, r.NewDir, r.NewName)
}

func (r *RenameRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type MknodRequest struct {
	Header `json:"-"`
	Name   string
	Mode   os.FileMode
	Rdev   uint32

	Umask os.FileMode
}

var _ = Request(&MknodRequest{})

func (r *MknodRequest) String() string {
	return fmt.Sprintf("Mknod [%s] Name %q mode=%v umask=%v rdev=%d", &r.Header, r.Name, r.Mode, r.Umask, r.Rdev)
}

func (r *MknodRequest) Respond(resp *LookupResponse) {
	size := entryOutSize(r.Header.Conn.proto)
	buf := newBuffer(size)
	out := (*entryOut)(buf.alloc(size))
	out.Nodeid = uint64(resp.Node)
	out.Generation = resp.Generation
	out.EntryValid = uint64(resp.EntryValid / time.Second)
	out.EntryValidNsec = uint32(resp.EntryValid % time.Second / time.Nanosecond)
	out.AttrValid = uint64(resp.Attr.Valid / time.Second)
	out.AttrValidNsec = uint32(resp.Attr.Valid % time.Second / time.Nanosecond)
	resp.Attr.attr(&out.Attr, r.Header.Conn.proto)
	r.respond(buf)
}

type FsyncRequest struct {
	Header `json:"-"`
	Handle HandleID

	Flags uint32
	Dir   bool
}

var _ = Request(&FsyncRequest{})

func (r *FsyncRequest) String() string {
	return fmt.Sprintf("Fsync [%s] Handle %v Flags %v", &r.Header, r.Handle, r.Flags)
}

func (r *FsyncRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}

type InterruptRequest struct {
	Header `json:"-"`
	IntrID RequestID 
}

var _ = Request(&InterruptRequest{})

func (r *InterruptRequest) Respond() {

	r.noResponse()
}

func (r *InterruptRequest) String() string {
	return fmt.Sprintf("Interrupt [%s] ID %v", &r.Header, r.IntrID)
}

type ExchangeDataRequest struct {
	Header           `json:"-"`
	OldDir, NewDir   NodeID
	OldName, NewName string

}

var _ = Request(&ExchangeDataRequest{})

func (r *ExchangeDataRequest) String() string {

	return fmt.Sprintf("ExchangeData [%s] %v %q and %v %q", &r.Header, r.OldDir, r.OldName, r.NewDir, r.NewName)
}

func (r *ExchangeDataRequest) Respond() {
	buf := newBuffer(0)
	r.respond(buf)
}
