
package fs 

import (
	"encoding/binary"
	"fmt"
	"hash/fnv"
	"io"
	"log"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/context"
)

import (
	"bytes"

	"bazil.org/fuse"
	"bazil.org/fuse/fuseutil"
)

const (
	attrValidTime  = 1 * time.Minute
	entryValidTime = 1 * time.Minute
)

type FS interface {

	Root() (Node, error)
}

type FSStatfser interface {

	Statfs(ctx context.Context, req *fuse.StatfsRequest, resp *fuse.StatfsResponse) error
}

type FSDestroyer interface {

	Destroy()
}

type FSInodeGenerator interface {

	GenerateInode(parentInode uint64, name string) uint64
}

type Node interface {

	Attr(ctx context.Context, attr *fuse.Attr) error
}

type NodeGetattrer interface {

	Getattr(ctx context.Context, req *fuse.GetattrRequest, resp *fuse.GetattrResponse) error
}

type NodeSetattrer interface {

	Setattr(ctx context.Context, req *fuse.SetattrRequest, resp *fuse.SetattrResponse) error
}

type NodeSymlinker interface {

	Symlink(ctx context.Context, req *fuse.SymlinkRequest) (Node, error)
}

type NodeReadlinker interface {

	Readlink(ctx context.Context, req *fuse.ReadlinkRequest) (string, error)
}

type NodeLinker interface {

	Link(ctx context.Context, req *fuse.LinkRequest, old Node) (Node, error)
}

type NodeRemover interface {

	Remove(ctx context.Context, req *fuse.RemoveRequest) error
}

type NodeAccesser interface {

	Access(ctx context.Context, req *fuse.AccessRequest) error
}

type NodeStringLookuper interface {

	Lookup(ctx context.Context, name string) (Node, error)
}

type NodeRequestLookuper interface {

	Lookup(ctx context.Context, req *fuse.LookupRequest, resp *fuse.LookupResponse) (Node, error)
}

type NodeMkdirer interface {
	Mkdir(ctx context.Context, req *fuse.MkdirRequest) (Node, error)
}

type NodeOpener interface {

	Open(ctx context.Context, req *fuse.OpenRequest, resp *fuse.OpenResponse) (Handle, error)
}

type NodeCreater interface {

	Create(ctx context.Context, req *fuse.CreateRequest, resp *fuse.CreateResponse) (Node, Handle, error)
}

type NodeForgetter interface {

	Forget()
}

type NodeRenamer interface {
	Rename(ctx context.Context, req *fuse.RenameRequest, newDir Node) error
}

type NodeMknoder interface {
	Mknod(ctx context.Context, req *fuse.MknodRequest) (Node, error)
}

type NodeFsyncer interface {
	Fsync(ctx context.Context, req *fuse.FsyncRequest) error
}

type NodeGetxattrer interface {

	Getxattr(ctx context.Context, req *fuse.GetxattrRequest, resp *fuse.GetxattrResponse) error
}

type NodeListxattrer interface {

	Listxattr(ctx context.Context, req *fuse.ListxattrRequest, resp *fuse.ListxattrResponse) error
}

type NodeSetxattrer interface {

	Setxattr(ctx context.Context, req *fuse.SetxattrRequest) error
}

type NodeRemovexattrer interface {

	Removexattr(ctx context.Context, req *fuse.RemovexattrRequest) error
}

var startTime = time.Now()

func nodeAttr(ctx context.Context, n Node, attr *fuse.Attr) error {
	attr.Valid = attrValidTime
	attr.Nlink = 1
	attr.Atime = startTime
	attr.Mtime = startTime
	attr.Ctime = startTime
	attr.Crtime = startTime
	if err := n.Attr(ctx, attr); err != nil {
		return err
	}
	return nil
}

type Handle interface {
}

type HandleFlusher interface {

	Flush(ctx context.Context, req *fuse.FlushRequest) error
}

type HandleReadAller interface {
	ReadAll(ctx context.Context) ([]byte, error)
}

type HandleReadDirAller interface {
	ReadDirAll(ctx context.Context) ([]fuse.Dirent, error)
}

type HandleReader interface {

	Read(ctx context.Context, req *fuse.ReadRequest, resp *fuse.ReadResponse) error
}

type HandleWriter interface {

	Write(ctx context.Context, req *fuse.WriteRequest, resp *fuse.WriteResponse) error
}

type HandleReleaser interface {
	Release(ctx context.Context, req *fuse.ReleaseRequest) error
}

type Config struct {

	Debug func(msg interface{})

	WithContext func(ctx context.Context, req fuse.Request) context.Context
}

func New(conn *fuse.Conn, config *Config) *Server {
	s := &Server{
		conn:         conn,
		req:          map[fuse.RequestID]*serveRequest{},
		nodeRef:      map[Node]fuse.NodeID{},
		dynamicInode: GenerateDynamicInode,
	}
	if config != nil {
		s.debug = config.Debug
		s.context = config.WithContext
	}
	if s.debug == nil {
		s.debug = fuse.Debug
	}
	return s
}

type Server struct {

	conn    *fuse.Conn
	debug   func(msg interface{})
	context func(ctx context.Context, req fuse.Request) context.Context

	fs           FS
	dynamicInode func(parent uint64, name string) uint64

	meta       sync.Mutex
	req        map[fuse.RequestID]*serveRequest
	node       []*serveNode
	nodeRef    map[Node]fuse.NodeID
	handle     []*serveHandle
	freeNode   []fuse.NodeID
	freeHandle []fuse.HandleID
	nodeGen    uint64

	wg sync.WaitGroup
}

func (s *Server) Serve(fs FS) error {
	defer s.wg.Wait() 

	s.fs = fs
	if dyn, ok := fs.(FSInodeGenerator); ok {
		s.dynamicInode = dyn.GenerateInode
	}

	root, err := fs.Root()
	if err != nil {
		return fmt.Errorf("cannot obtain root node: %v", err)
	}

	s.nodeRef[root] = 1
	s.node = append(s.node, nil, &serveNode{
		inode:      1,
		generation: s.nodeGen,
		node:       root,
		refs:       1,
	})
	s.handle = append(s.handle, nil)

	for {
		req, err := s.conn.ReadRequest()
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			s.serve(req)
		}()
	}
	return nil
}

func Serve(c *fuse.Conn, fs FS) error {
	server := New(c, nil)
	return server.Serve(fs)
}

type nothing struct{}

type serveRequest struct {
	Request fuse.Request
	cancel  func()
}

type serveNode struct {
	inode      uint64
	generation uint64
	node       Node
	refs       uint64

	wg sync.WaitGroup
}

func (sn *serveNode) attr(ctx context.Context, attr *fuse.Attr) error {
	err := nodeAttr(ctx, sn.node, attr)
	if attr.Inode == 0 {
		attr.Inode = sn.inode
	}
	return err
}

type serveHandle struct {
	handle   Handle
	readData []byte
	nodeID   fuse.NodeID
}

type NodeRef struct{}

func (c *Server) saveNode(inode uint64, node Node) (id fuse.NodeID, gen uint64) {
	c.meta.Lock()
	defer c.meta.Unlock()

	if id, ok := c.nodeRef[node]; ok {
		sn := c.node[id]
		sn.refs++
		return id, sn.generation
	}

	sn := &serveNode{inode: inode, node: node, refs: 1}
	if n := len(c.freeNode); n > 0 {
		id = c.freeNode[n-1]
		c.freeNode = c.freeNode[:n-1]
		c.node[id] = sn
		c.nodeGen++
	} else {
		id = fuse.NodeID(len(c.node))
		c.node = append(c.node, sn)
	}
	sn.generation = c.nodeGen
	c.nodeRef[node] = id
	return id, sn.generation
}

func (c *Server) saveHandle(handle Handle, nodeID fuse.NodeID) (id fuse.HandleID) {
	c.meta.Lock()
	shandle := &serveHandle{handle: handle, nodeID: nodeID}
	if n := len(c.freeHandle); n > 0 {
		id = c.freeHandle[n-1]
		c.freeHandle = c.freeHandle[:n-1]
		c.handle[id] = shandle
	} else {
		id = fuse.HandleID(len(c.handle))
		c.handle = append(c.handle, shandle)
	}
	c.meta.Unlock()
	return
}

type nodeRefcountDropBug struct {
	N    uint64
	Refs uint64
	Node fuse.NodeID
}

func (n *nodeRefcountDropBug) String() string {
	return fmt.Sprintf("bug: trying to drop %d of %d references to %v", n.N, n.Refs, n.Node)
}

func (c *Server) dropNode(id fuse.NodeID, n uint64) (forget bool) {
	c.meta.Lock()
	defer c.meta.Unlock()
	snode := c.node[id]

	if snode == nil {

		c.debug(nodeRefcountDropBug{N: n, Node: id})

		return true
	}

	if n > snode.refs {
		c.debug(nodeRefcountDropBug{N: n, Refs: snode.refs, Node: id})
		n = snode.refs
	}

	snode.refs -= n
	if snode.refs == 0 {
		snode.wg.Wait()
		c.node[id] = nil
		delete(c.nodeRef, snode.node)
		c.freeNode = append(c.freeNode, id)
		return true
	}
	return false
}

func (c *Server) dropHandle(id fuse.HandleID) {
	c.meta.Lock()
	c.handle[id] = nil
	c.freeHandle = append(c.freeHandle, id)
	c.meta.Unlock()
}

type missingHandle struct {
	Handle    fuse.HandleID
	MaxHandle fuse.HandleID
}

func (m missingHandle) String() string {
	return fmt.Sprint("missing handle: ", m.Handle, m.MaxHandle)
}

func (c *Server) getHandle(id fuse.HandleID) (shandle *serveHandle) {
	c.meta.Lock()
	defer c.meta.Unlock()
	if id < fuse.HandleID(len(c.handle)) {
		shandle = c.handle[uint(id)]
	}
	if shandle == nil {
		c.debug(missingHandle{
			Handle:    id,
			MaxHandle: fuse.HandleID(len(c.handle)),
		})
	}
	return
}

type request struct {
	Op      string
	Request *fuse.Header
	In      interface{} `json:",omitempty"`
}

func (r request) String() string {
	return fmt.Sprintf("<- %s", r.In)
}

type logResponseHeader struct {
	ID fuse.RequestID
}

func (m logResponseHeader) String() string {
	return fmt.Sprintf("ID=%v", m.ID)
}

type response struct {
	Op      string
	Request logResponseHeader
	Out     interface{} `json:",omitempty"`

	Errno string `json:",omitempty"`

	Error string `json:",omitempty"`
}

func (r response) errstr() string {
	s := r.Errno
	if r.Error != "" {

		s = s + ": " + r.Error
	}
	return s
}

func (r response) String() string {
	switch {
	case r.Errno != "" && r.Out != nil:
		return fmt.Sprintf("-> [%v] %v error=%s", r.Request, r.Out, r.errstr())
	case r.Errno != "":
		return fmt.Sprintf("-> [%v] %s error=%s", r.Request, r.Op, r.errstr())
	case r.Out != nil:

		switch r.Out.(type) {
		case string:
			return fmt.Sprintf("-> [%v] %s %q", r.Request, r.Op, r.Out)
		case []byte:
			return fmt.Sprintf("-> [%v] %s [% x]", r.Request, r.Op, r.Out)
		default:
			return fmt.Sprintf("-> [%v] %v", r.Request, r.Out)
		}
	default:
		return fmt.Sprintf("-> [%v] %s", r.Request, r.Op)
	}
}

type notification struct {
	Op   string
	Node fuse.NodeID
	Out  interface{} `json:",omitempty"`
	Err  string      `json:",omitempty"`
}

func (n notification) String() string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "=> %s %v", n.Op, n.Node)
	if n.Out != nil {

		switch n.Out.(type) {
		case string:
			fmt.Fprintf(&buf, " %q", n.Out)
		case []byte:
			fmt.Fprintf(&buf, " [% x]", n.Out)
		default:
			fmt.Fprintf(&buf, " %s", n.Out)
		}
	}
	if n.Err != "" {
		fmt.Fprintf(&buf, " Err:%v", n.Err)
	}
	return buf.String()
}

type logMissingNode struct {
	MaxNode fuse.NodeID
}

func opName(req fuse.Request) string {
	t := reflect.Indirect(reflect.ValueOf(req)).Type()
	s := t.Name()
	s = strings.TrimSuffix(s, "Request")
	return s
}

type logLinkRequestOldNodeNotFound struct {
	Request *fuse.Header
	In      *fuse.LinkRequest
}

func (m *logLinkRequestOldNodeNotFound) String() string {
	return fmt.Sprintf("In LinkRequest (request %v), node %d not found", m.Request.Hdr().ID, m.In.OldNode)
}

type renameNewDirNodeNotFound struct {
	Request *fuse.Header
	In      *fuse.RenameRequest
}

func (m *renameNewDirNodeNotFound) String() string {
	return fmt.Sprintf("In RenameRequest (request %v), node %d not found", m.Request.Hdr().ID, m.In.NewDir)
}

type handlerPanickedError struct {
	Request interface{}
	Err     interface{}
}

var _ error = handlerPanickedError{}

func (h handlerPanickedError) Error() string {
	return fmt.Sprintf("handler panicked: %v", h.Err)
}

var _ fuse.ErrorNumber = handlerPanickedError{}

func (h handlerPanickedError) Errno() fuse.Errno {
	if err, ok := h.Err.(fuse.ErrorNumber); ok {
		return err.Errno()
	}
	return fuse.DefaultErrno
}

type handlerTerminatedError struct {
	Request interface{}
}

var _ error = handlerTerminatedError{}

func (h handlerTerminatedError) Error() string {
	return fmt.Sprintf("handler terminated (called runtime.Goexit)")
}

var _ fuse.ErrorNumber = handlerTerminatedError{}

func (h handlerTerminatedError) Errno() fuse.Errno {
	return fuse.DefaultErrno
}

type handleNotReaderError struct {
	handle Handle
}

var _ error = handleNotReaderError{}

func (e handleNotReaderError) Error() string {
	return fmt.Sprintf("handle has no Read: %T", e.handle)
}

var _ fuse.ErrorNumber = handleNotReaderError{}

func (e handleNotReaderError) Errno() fuse.Errno {
	return fuse.ENOTSUP
}

func initLookupResponse(s *fuse.LookupResponse) {
	s.EntryValid = entryValidTime
}

func (c *Server) serve(r fuse.Request) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	parentCtx := ctx
	if c.context != nil {
		ctx = c.context(ctx, r)
	}

	req := &serveRequest{Request: r, cancel: cancel}

	c.debug(request{
		Op:      opName(r),
		Request: r.Hdr(),
		In:      r,
	})
	var node Node
	var snode *serveNode
	c.meta.Lock()
	hdr := r.Hdr()
	if id := hdr.Node; id != 0 {
		if id < fuse.NodeID(len(c.node)) {
			snode = c.node[uint(id)]
		}
		if snode == nil {
			c.meta.Unlock()
			c.debug(response{
				Op:      opName(r),
				Request: logResponseHeader{ID: hdr.ID},
				Error:   fuse.ESTALE.ErrnoName(),

				Out: logMissingNode{
					MaxNode: fuse.NodeID(len(c.node)),
				},
			})
			r.RespondError(fuse.ESTALE)
			return
		}
		node = snode.node
	}
	if c.req[hdr.ID] != nil {

	} else {
		c.req[hdr.ID] = req
	}
	c.meta.Unlock()

	done := func(resp interface{}) {
		msg := response{
			Op:      opName(r),
			Request: logResponseHeader{ID: hdr.ID},
		}
		if err, ok := resp.(error); ok {
			msg.Error = err.Error()
			if ferr, ok := err.(fuse.ErrorNumber); ok {
				errno := ferr.Errno()
				msg.Errno = errno.ErrnoName()
				if errno == err {

					msg.Error = ""
				}
			} else {
				msg.Errno = fuse.DefaultErrno.ErrnoName()
			}
		} else {
			msg.Out = resp
		}
		c.debug(msg)

		c.meta.Lock()
		delete(c.req, hdr.ID)
		c.meta.Unlock()
	}

	var responded bool
	defer func() {
		if rec := recover(); rec != nil {
			const size = 1 << 16
			buf := make([]byte, size)
			n := runtime.Stack(buf, false)
			buf = buf[:n]
			log.Printf("fuse: panic in handler for %v: %v\n%s", r, rec, buf)
			err := handlerPanickedError{
				Request: r,
				Err:     rec,
			}
			done(err)
			r.RespondError(err)
			return
		}

		if !responded {
			err := handlerTerminatedError{
				Request: r,
			}
			done(err)
			r.RespondError(err)
		}
	}()

	if err := c.handleRequest(ctx, node, snode, r, done); err != nil {
		if err == context.Canceled {
			select {
			case <-parentCtx.Done():

				err = fuse.EINTR
			default:

			}
		}
		done(err)
		r.RespondError(err)
	}

	responded = true
}

func (c *Server) handleRequest(ctx context.Context, node Node, snode *serveNode, r fuse.Request, done func(resp interface{})) error {
	switch r := r.(type) {
	default:

		return fuse.ENOSYS

	case *fuse.StatfsRequest:
		s := &fuse.StatfsResponse{}
		if fs, ok := c.fs.(FSStatfser); ok {
			if err := fs.Statfs(ctx, r, s); err != nil {
				return err
			}
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.GetattrRequest:
		s := &fuse.GetattrResponse{}
		if n, ok := node.(NodeGetattrer); ok {
			if err := n.Getattr(ctx, r, s); err != nil {
				return err
			}
		} else {
			if err := snode.attr(ctx, &s.Attr); err != nil {
				return err
			}
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.SetattrRequest:
		s := &fuse.SetattrResponse{}
		if n, ok := node.(NodeSetattrer); ok {
			if err := n.Setattr(ctx, r, s); err != nil {
				return err
			}
		}

		if err := snode.attr(ctx, &s.Attr); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.SymlinkRequest:
		s := &fuse.SymlinkResponse{}
		initLookupResponse(&s.LookupResponse)
		n, ok := node.(NodeSymlinker)
		if !ok {
			return fuse.EIO 
		}
		n2, err := n.Symlink(ctx, r)
		if err != nil {
			return err
		}
		if err := c.saveLookup(ctx, &s.LookupResponse, snode, r.NewName, n2); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.ReadlinkRequest:
		n, ok := node.(NodeReadlinker)
		if !ok {
			return fuse.EIO 
		}
		target, err := n.Readlink(ctx, r)
		if err != nil {
			return err
		}
		done(target)
		r.Respond(target)
		return nil

	case *fuse.LinkRequest:
		n, ok := node.(NodeLinker)
		if !ok {
			return fuse.EIO 
		}
		c.meta.Lock()
		var oldNode *serveNode
		if int(r.OldNode) < len(c.node) {
			oldNode = c.node[r.OldNode]
		}
		c.meta.Unlock()
		if oldNode == nil {
			c.debug(logLinkRequestOldNodeNotFound{
				Request: r.Hdr(),
				In:      r,
			})
			return fuse.EIO
		}
		n2, err := n.Link(ctx, r, oldNode.node)
		if err != nil {
			return err
		}
		s := &fuse.LookupResponse{}
		initLookupResponse(s)
		if err := c.saveLookup(ctx, s, snode, r.NewName, n2); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.RemoveRequest:
		n, ok := node.(NodeRemover)
		if !ok {
			return fuse.EIO 
		}
		err := n.Remove(ctx, r)
		if err != nil {
			return err
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.AccessRequest:
		if n, ok := node.(NodeAccesser); ok {
			if err := n.Access(ctx, r); err != nil {
				return err
			}
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.LookupRequest:
		var n2 Node
		var err error
		s := &fuse.LookupResponse{}
		initLookupResponse(s)
		if n, ok := node.(NodeStringLookuper); ok {
			n2, err = n.Lookup(ctx, r.Name)
		} else if n, ok := node.(NodeRequestLookuper); ok {
			n2, err = n.Lookup(ctx, r, s)
		} else {
			return fuse.ENOENT
		}
		if err != nil {
			return err
		}
		if err := c.saveLookup(ctx, s, snode, r.Name, n2); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.MkdirRequest:
		s := &fuse.MkdirResponse{}
		initLookupResponse(&s.LookupResponse)
		n, ok := node.(NodeMkdirer)
		if !ok {
			return fuse.EPERM
		}
		n2, err := n.Mkdir(ctx, r)
		if err != nil {
			return err
		}
		if err := c.saveLookup(ctx, &s.LookupResponse, snode, r.Name, n2); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.OpenRequest:
		s := &fuse.OpenResponse{}
		var h2 Handle
		if n, ok := node.(NodeOpener); ok {
			hh, err := n.Open(ctx, r, s)
			if err != nil {
				return err
			}
			h2 = hh
		} else {
			h2 = node
		}
		s.Handle = c.saveHandle(h2, r.Hdr().Node)
		done(s)
		r.Respond(s)
		return nil

	case *fuse.CreateRequest:
		n, ok := node.(NodeCreater)
		if !ok {

			return fuse.EPERM
		}
		s := &fuse.CreateResponse{OpenResponse: fuse.OpenResponse{}}
		initLookupResponse(&s.LookupResponse)
		n2, h2, err := n.Create(ctx, r, s)
		if err != nil {
			return err
		}
		if err := c.saveLookup(ctx, &s.LookupResponse, snode, r.Name, n2); err != nil {
			return err
		}
		s.Handle = c.saveHandle(h2, r.Hdr().Node)
		done(s)
		r.Respond(s)
		return nil

	case *fuse.GetxattrRequest:
		n, ok := node.(NodeGetxattrer)
		if !ok {
			return fuse.ENOTSUP
		}
		s := &fuse.GetxattrResponse{}
		err := n.Getxattr(ctx, r, s)
		if err != nil {
			return err
		}
		if r.Size != 0 && uint64(len(s.Xattr)) > uint64(r.Size) {
			return fuse.ERANGE
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.ListxattrRequest:
		n, ok := node.(NodeListxattrer)
		if !ok {
			return fuse.ENOTSUP
		}
		s := &fuse.ListxattrResponse{}
		err := n.Listxattr(ctx, r, s)
		if err != nil {
			return err
		}
		if r.Size != 0 && uint64(len(s.Xattr)) > uint64(r.Size) {
			return fuse.ERANGE
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.SetxattrRequest:
		n, ok := node.(NodeSetxattrer)
		if !ok {
			return fuse.ENOTSUP
		}
		err := n.Setxattr(ctx, r)
		if err != nil {
			return err
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.RemovexattrRequest:
		n, ok := node.(NodeRemovexattrer)
		if !ok {
			return fuse.ENOTSUP
		}
		err := n.Removexattr(ctx, r)
		if err != nil {
			return err
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.ForgetRequest:
		forget := c.dropNode(r.Hdr().Node, r.N)
		if forget {
			n, ok := node.(NodeForgetter)
			if ok {
				n.Forget()
			}
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.ReadRequest:
		shandle := c.getHandle(r.Handle)
		if shandle == nil {
			return fuse.ESTALE
		}
		handle := shandle.handle

		s := &fuse.ReadResponse{Data: make([]byte, 0, r.Size)}
		if r.Dir {
			if h, ok := handle.(HandleReadDirAller); ok {

				if r.Offset == 0 {
					shandle.readData = nil
				}

				if shandle.readData == nil {
					dirs, err := h.ReadDirAll(ctx)
					if err != nil {
						return err
					}
					var data []byte
					for _, dir := range dirs {
						if dir.Inode == 0 {
							dir.Inode = c.dynamicInode(snode.inode, dir.Name)
						}
						data = fuse.AppendDirent(data, dir)
					}
					shandle.readData = data
				}
				fuseutil.HandleRead(r, s, shandle.readData)
				done(s)
				r.Respond(s)
				return nil
			}
		} else {
			if h, ok := handle.(HandleReadAller); ok {
				if shandle.readData == nil {
					data, err := h.ReadAll(ctx)
					if err != nil {
						return err
					}
					if data == nil {
						data = []byte{}
					}
					shandle.readData = data
				}
				fuseutil.HandleRead(r, s, shandle.readData)
				done(s)
				r.Respond(s)
				return nil
			}
			h, ok := handle.(HandleReader)
			if !ok {
				err := handleNotReaderError{handle: handle}
				return err
			}
			if err := h.Read(ctx, r, s); err != nil {
				return err
			}
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.WriteRequest:
		shandle := c.getHandle(r.Handle)
		if shandle == nil {
			return fuse.ESTALE
		}

		s := &fuse.WriteResponse{}
		if h, ok := shandle.handle.(HandleWriter); ok {
			if err := h.Write(ctx, r, s); err != nil {
				return err
			}
			done(s)
			r.Respond(s)
			return nil
		}
		return fuse.EIO

	case *fuse.FlushRequest:
		shandle := c.getHandle(r.Handle)
		if shandle == nil {
			return fuse.ESTALE
		}
		handle := shandle.handle

		if h, ok := handle.(HandleFlusher); ok {
			if err := h.Flush(ctx, r); err != nil {
				return err
			}
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.ReleaseRequest:
		shandle := c.getHandle(r.Handle)
		if shandle == nil {
			return fuse.ESTALE
		}
		handle := shandle.handle

		c.dropHandle(r.Handle)

		if h, ok := handle.(HandleReleaser); ok {
			if err := h.Release(ctx, r); err != nil {
				return err
			}
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.DestroyRequest:
		if fs, ok := c.fs.(FSDestroyer); ok {
			fs.Destroy()
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.RenameRequest:
		c.meta.Lock()
		var newDirNode *serveNode
		if int(r.NewDir) < len(c.node) {
			newDirNode = c.node[r.NewDir]
		}
		c.meta.Unlock()
		if newDirNode == nil {
			c.debug(renameNewDirNodeNotFound{
				Request: r.Hdr(),
				In:      r,
			})
			return fuse.EIO
		}
		n, ok := node.(NodeRenamer)
		if !ok {
			return fuse.EIO 
		}
		err := n.Rename(ctx, r, newDirNode.node)
		if err != nil {
			return err
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.MknodRequest:
		n, ok := node.(NodeMknoder)
		if !ok {
			return fuse.EIO
		}
		n2, err := n.Mknod(ctx, r)
		if err != nil {
			return err
		}
		s := &fuse.LookupResponse{}
		initLookupResponse(s)
		if err := c.saveLookup(ctx, s, snode, r.Name, n2); err != nil {
			return err
		}
		done(s)
		r.Respond(s)
		return nil

	case *fuse.FsyncRequest:
		n, ok := node.(NodeFsyncer)
		if !ok {
			return fuse.EIO
		}
		err := n.Fsync(ctx, r)
		if err != nil {
			return err
		}
		done(nil)
		r.Respond()
		return nil

	case *fuse.InterruptRequest:
		c.meta.Lock()
		ireq := c.req[r.IntrID]
		if ireq != nil && ireq.cancel != nil {
			ireq.cancel()
			ireq.cancel = nil
		}
		c.meta.Unlock()
		done(nil)
		r.Respond()
		return nil

	}

	panic("not reached")
}

func (c *Server) saveLookup(ctx context.Context, s *fuse.LookupResponse, snode *serveNode, elem string, n2 Node) error {
	if err := nodeAttr(ctx, n2, &s.Attr); err != nil {
		return err
	}
	if s.Attr.Inode == 0 {
		s.Attr.Inode = c.dynamicInode(snode.inode, elem)
	}

	s.Node, s.Generation = c.saveNode(s.Attr.Inode, n2)
	return nil
}

type invalidateNodeDetail struct {
	Off  int64
	Size int64
}

func (i invalidateNodeDetail) String() string {
	return fmt.Sprintf("Off:%d Size:%d", i.Off, i.Size)
}

func errstr(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func (s *Server) invalidateNode(node Node, off int64, size int64) error {
	s.meta.Lock()
	id, ok := s.nodeRef[node]
	if ok {
		snode := s.node[id]
		snode.wg.Add(1)
		defer snode.wg.Done()
	}
	s.meta.Unlock()
	if !ok {

		return fuse.ErrNotCached
	}

	err := s.conn.InvalidateNode(id, off, size)
	s.debug(notification{
		Op:   "InvalidateNode",
		Node: id,
		Out: invalidateNodeDetail{
			Off:  off,
			Size: size,
		},
		Err: errstr(err),
	})
	return err
}

func (s *Server) InvalidateNodeAttr(node Node) error {
	return s.invalidateNode(node, 0, 0)
}

func (s *Server) InvalidateNodeData(node Node) error {
	return s.invalidateNode(node, 0, -1)
}

func (s *Server) InvalidateNodeDataRange(node Node, off int64, size int64) error {
	return s.invalidateNode(node, off, size)
}

type invalidateEntryDetail struct {
	Name string
}

func (i invalidateEntryDetail) String() string {
	return fmt.Sprintf("%q", i.Name)
}

func (s *Server) InvalidateEntry(parent Node, name string) error {
	s.meta.Lock()
	id, ok := s.nodeRef[parent]
	if ok {
		snode := s.node[id]
		snode.wg.Add(1)
		defer snode.wg.Done()
	}
	s.meta.Unlock()
	if !ok {

		return fuse.ErrNotCached
	}
	err := s.conn.InvalidateEntry(id, name)
	s.debug(notification{
		Op:   "InvalidateEntry",
		Node: id,
		Out: invalidateEntryDetail{
			Name: name,
		},
		Err: errstr(err),
	})
	return err
}

func DataHandle(data []byte) Handle {
	return &dataHandle{data}
}

type dataHandle struct {
	data []byte
}

func (d *dataHandle) ReadAll(ctx context.Context) ([]byte, error) {
	return d.data, nil
}

func GenerateDynamicInode(parent uint64, name string) uint64 {
	h := fnv.New64a()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], parent)
	_, _ = h.Write(buf[:])
	_, _ = h.Write([]byte(name))
	var inode uint64
	for {
		inode = h.Sum64()
		if inode != 0 {
			break
		}

		_, _ = h.Write([]byte{'x'})
	}
	return inode
}
