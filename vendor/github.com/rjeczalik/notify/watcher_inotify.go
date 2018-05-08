
// +build linux

package notify

import (
	"bytes"
	"errors"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"unsafe"

	"golang.org/x/sys/unix"
)

const eventBufferSize = 64 * (unix.SizeofInotifyEvent + unix.PathMax + 1)

const consumersCount = 2

const invalidDescriptor = -1

type watched struct {
	path string
	mask uint32
}

type inotify struct {
	sync.RWMutex                       
	m            map[int32]*watched    
	fd           int32                 
	pipefd       []int                 
	epfd         int                   
	epes         []unix.EpollEvent     
	buffer       [eventBufferSize]byte 
	wg           sync.WaitGroup        
	c            chan<- EventInfo      
}

func newWatcher(c chan<- EventInfo) watcher {
	i := &inotify{
		m:      make(map[int32]*watched),
		fd:     invalidDescriptor,
		pipefd: []int{invalidDescriptor, invalidDescriptor},
		epfd:   invalidDescriptor,
		epes:   make([]unix.EpollEvent, 0),
		c:      c,
	}
	runtime.SetFinalizer(i, func(i *inotify) {
		i.epollclose()
		if i.fd != invalidDescriptor {
			unix.Close(int(i.fd))
		}
	})
	return i
}

func (i *inotify) Watch(path string, e Event) error {
	return i.watch(path, e)
}

func (i *inotify) Rewatch(path string, _, newevent Event) error {
	return i.watch(path, newevent)
}

func (i *inotify) watch(path string, e Event) (err error) {
	if e&^(All|Event(unix.IN_ALL_EVENTS)) != 0 {
		return errors.New("notify: unknown event")
	}
	if err = i.lazyinit(); err != nil {
		return
	}
	iwd, err := unix.InotifyAddWatch(int(i.fd), path, encode(e))
	if err != nil {
		return
	}
	i.RLock()
	wd := i.m[int32(iwd)]
	i.RUnlock()
	if wd == nil {
		i.Lock()
		if i.m[int32(iwd)] == nil {
			i.m[int32(iwd)] = &watched{path: path, mask: uint32(e)}
		}
		i.Unlock()
	} else {
		i.Lock()
		wd.mask = uint32(e)
		i.Unlock()
	}
	return nil
}

func (i *inotify) lazyinit() error {
	if atomic.LoadInt32(&i.fd) == invalidDescriptor {
		i.Lock()
		defer i.Unlock()
		if atomic.LoadInt32(&i.fd) == invalidDescriptor {
			fd, err := unix.InotifyInit1(unix.IN_CLOEXEC)
			if err != nil {
				return err
			}
			i.fd = int32(fd)
			if err = i.epollinit(); err != nil {
				_, _ = i.epollclose(), unix.Close(int(fd)) 
				i.fd = invalidDescriptor
				return err
			}
			esch := make(chan []*event)
			go i.loop(esch)
			i.wg.Add(consumersCount)
			for n := 0; n < consumersCount; n++ {
				go i.send(esch)
			}
		}
	}
	return nil
}

func (i *inotify) epollinit() (err error) {
	if i.epfd, err = unix.EpollCreate1(0); err != nil {
		return
	}
	if err = unix.Pipe(i.pipefd); err != nil {
		return
	}
	i.epes = []unix.EpollEvent{
		{Events: unix.EPOLLIN, Fd: i.fd},
		{Events: unix.EPOLLIN, Fd: int32(i.pipefd[0])},
	}
	if err = unix.EpollCtl(i.epfd, unix.EPOLL_CTL_ADD, int(i.fd), &i.epes[0]); err != nil {
		return
	}
	return unix.EpollCtl(i.epfd, unix.EPOLL_CTL_ADD, i.pipefd[0], &i.epes[1])
}

func (i *inotify) epollclose() (err error) {
	if i.epfd != invalidDescriptor {
		if err = unix.Close(i.epfd); err == nil {
			i.epfd = invalidDescriptor
		}
	}
	for n, fd := range i.pipefd {
		if fd != invalidDescriptor {
			switch e := unix.Close(fd); {
			case e != nil && err == nil:
				err = e
			case e == nil:
				i.pipefd[n] = invalidDescriptor
			}
		}
	}
	return
}

func (i *inotify) loop(esch chan<- []*event) {
	epes := make([]unix.EpollEvent, 1)
	fd := atomic.LoadInt32(&i.fd)
	for {
		switch _, err := unix.EpollWait(i.epfd, epes, -1); err {
		case nil:
			switch epes[0].Fd {
			case fd:
				esch <- i.read()
				epes[0].Fd = 0
			case int32(i.pipefd[0]):
				i.Lock()
				defer i.Unlock()
				if err = unix.Close(int(fd)); err != nil && err != unix.EINTR {
					panic("notify: close(2) error " + err.Error())
				}
				atomic.StoreInt32(&i.fd, invalidDescriptor)
				if err = i.epollclose(); err != nil && err != unix.EINTR {
					panic("notify: epollclose error " + err.Error())
				}
				close(esch)
				return
			}
		case unix.EINTR:
			continue
		default: 
			panic("notify: epoll_wait(2) error " + err.Error())
		}
	}
}

func (i *inotify) read() (es []*event) {
	n, err := unix.Read(int(i.fd), i.buffer[:])
	if err != nil || n < unix.SizeofInotifyEvent {
		return
	}
	var sys *unix.InotifyEvent
	nmin := n - unix.SizeofInotifyEvent
	for pos, path := 0, ""; pos <= nmin; {
		sys = (*unix.InotifyEvent)(unsafe.Pointer(&i.buffer[pos]))
		pos += unix.SizeofInotifyEvent
		if path = ""; sys.Len > 0 {
			endpos := pos + int(sys.Len)
			path = string(bytes.TrimRight(i.buffer[pos:endpos], "\x00"))
			pos = endpos
		}
		es = append(es, &event{
			sys: unix.InotifyEvent{
				Wd:     sys.Wd,
				Mask:   sys.Mask,
				Cookie: sys.Cookie,
			},
			path: path,
		})
	}
	return
}

func (i *inotify) send(esch <-chan []*event) {
	for es := range esch {
		for _, e := range i.transform(es) {
			if e != nil {
				i.c <- e
			}
		}
	}
	i.wg.Done()
}

func (i *inotify) transform(es []*event) []*event {
	var multi []*event
	i.RLock()
	for idx, e := range es {
		if e.sys.Mask&(unix.IN_IGNORED|unix.IN_Q_OVERFLOW) != 0 {
			es[idx] = nil
			continue
		}
		wd, ok := i.m[e.sys.Wd]
		if !ok || e.sys.Mask&encode(Event(wd.mask)) == 0 {
			es[idx] = nil
			continue
		}
		if e.path == "" {
			e.path = wd.path
		} else {
			e.path = filepath.Join(wd.path, e.path)
		}
		multi = append(multi, decode(Event(wd.mask), e))
		if e.event == 0 {
			es[idx] = nil
		}
	}
	i.RUnlock()
	es = append(es, multi...)
	return es
}

func encode(e Event) uint32 {
	if e&Create != 0 {
		e = (e ^ Create) | InCreate | InMovedTo
	}
	if e&Remove != 0 {
		e = (e ^ Remove) | InDelete | InDeleteSelf
	}
	if e&Write != 0 {
		e = (e ^ Write) | InModify
	}
	if e&Rename != 0 {
		e = (e ^ Rename) | InMovedFrom | InMoveSelf
	}
	return uint32(e)
}

func decode(mask Event, e *event) (syse *event) {
	if sysmask := uint32(mask) & e.sys.Mask; sysmask != 0 {
		syse = &event{sys: unix.InotifyEvent{
			Wd:     e.sys.Wd,
			Mask:   e.sys.Mask,
			Cookie: e.sys.Cookie,
		}, event: Event(sysmask), path: e.path}
	}
	imask := encode(mask)
	switch {
	case mask&Create != 0 && imask&uint32(InCreate|InMovedTo)&e.sys.Mask != 0:
		e.event = Create
	case mask&Remove != 0 && imask&uint32(InDelete|InDeleteSelf)&e.sys.Mask != 0:
		e.event = Remove
	case mask&Write != 0 && imask&uint32(InModify)&e.sys.Mask != 0:
		e.event = Write
	case mask&Rename != 0 && imask&uint32(InMovedFrom|InMoveSelf)&e.sys.Mask != 0:
		e.event = Rename
	default:
		e.event = 0
	}
	return
}

func (i *inotify) Unwatch(path string) (err error) {
	iwd := int32(invalidDescriptor)
	i.RLock()
	for iwdkey, wd := range i.m {
		if wd.path == path {
			iwd = iwdkey
			break
		}
	}
	i.RUnlock()
	if iwd == invalidDescriptor {
		return errors.New("notify: path " + path + " is already watched")
	}
	fd := atomic.LoadInt32(&i.fd)
	if err = removeInotifyWatch(fd, iwd); err != nil {
		return
	}
	i.Lock()
	delete(i.m, iwd)
	i.Unlock()
	return nil
}

func (i *inotify) Close() (err error) {
	i.Lock()
	if fd := atomic.LoadInt32(&i.fd); fd == invalidDescriptor {
		i.Unlock()
		return nil
	}
	for iwd := range i.m {
		if e := removeInotifyWatch(i.fd, iwd); e != nil && err == nil {
			err = e
		}
		delete(i.m, iwd)
	}
	switch _, errwrite := unix.Write(i.pipefd[1], []byte{0x00}); {
	case errwrite != nil && err == nil:
		err = errwrite
		fallthrough
	case errwrite != nil:
		i.Unlock()
	default:
		i.Unlock()
		i.wg.Wait()
	}
	return
}

func removeInotifyWatch(fd int32, iwd int32) (err error) {
	if _, err = unix.InotifyRmWatch(int(fd), uint32(iwd)); err != nil && err != unix.EINVAL {
		return
	}
	return nil
}
