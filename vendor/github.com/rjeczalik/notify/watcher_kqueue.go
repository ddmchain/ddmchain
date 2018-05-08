
// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import (
	"fmt"
	"os"
	"syscall"
)

func newTrigger(pthLkp map[string]*watched) trigger {
	return &kq{
		pthLkp: pthLkp,
		idLkp:  make(map[int]*watched),
	}
}

type kq struct {

	fd int

	pipefds [2]int

	idLkp map[int]*watched

	pthLkp map[string]*watched
}

type watched struct {
	trgWatched

	fd int
}

func (k *kq) Stop() (err error) {

	_, err = syscall.Write(k.pipefds[1], []byte{0x00})
	return
}

func (k *kq) Close() error {
	return syscall.Close(k.fd)
}

func (*kq) NewWatched(p string, fi os.FileInfo) (*watched, error) {
	fd, err := syscall.Open(p, syscall.O_NONBLOCK|syscall.O_RDONLY, 0)
	if err != nil {
		return nil, err
	}
	return &watched{
		trgWatched: trgWatched{p: p, fi: fi},
		fd:         fd,
	}, nil
}

func (k *kq) Record(w *watched) {
	k.idLkp[w.fd], k.pthLkp[w.p] = w, w
}

func (k *kq) Del(w *watched) {
	syscall.Close(w.fd)
	delete(k.idLkp, w.fd)
	delete(k.pthLkp, w.p)
}

func inter2kq(n interface{}) syscall.Kevent_t {
	kq, ok := n.(syscall.Kevent_t)
	if !ok {
		panic(fmt.Sprintf("kqueue: type should be Kevent_t, %T instead", n))
	}
	return kq
}

func (k *kq) Init() (err error) {
	if k.fd, err = syscall.Kqueue(); err != nil {
		return
	}

	if err = syscall.Pipe(k.pipefds[:]); err != nil {
		return nonil(err, k.Close())
	}
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], k.pipefds[0], syscall.EVFILT_READ, syscall.EV_ADD)
	if _, err = syscall.Kevent(k.fd, kevn[:], nil, nil); err != nil {
		return nonil(err, k.Close())
	}
	return
}

func (k *kq) Unwatch(w *watched) (err error) {
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE, syscall.EV_DELETE)

	_, err = syscall.Kevent(k.fd, kevn[:], nil, nil)
	return
}

func (k *kq) Watch(fi os.FileInfo, w *watched, e int64) (err error) {
	var kevn [1]syscall.Kevent_t
	syscall.SetKevent(&kevn[0], w.fd, syscall.EVFILT_VNODE,
		syscall.EV_ADD|syscall.EV_CLEAR)
	kevn[0].Fflags = uint32(e)

	_, err = syscall.Kevent(k.fd, kevn[:], nil, nil)
	return
}

func (k *kq) Wait() (interface{}, error) {
	var (
		kevn [1]syscall.Kevent_t
		err  error
	)
	kevn[0] = syscall.Kevent_t{}
	_, err = syscall.Kevent(k.fd, nil, kevn[:], nil)

	return kevn[0], err
}

func (k *kq) Watched(n interface{}) (*watched, int64, error) {
	kevn, ok := n.(syscall.Kevent_t)
	if !ok {
		panic(fmt.Sprintf("kq: type should be syscall.Kevent_t, %T instead", kevn))
	}
	if _, ok = k.idLkp[int(kevn.Ident)]; !ok {
		return nil, 0, errNotWatched
	}
	return k.idLkp[int(kevn.Ident)], int64(kevn.Fflags), nil
}

func (k *kq) IsStop(n interface{}, err error) bool {
	return int(inter2kq(n).Ident) == k.pipefds[0]
}

func init() {
	encode = func(e Event, dir bool) (o int64) {

		o = int64(e &^ Create)
		if (e&Create != 0 && dir) || e&Write != 0 {
			o = (o &^ int64(Write)) | int64(NoteWrite)
		}
		if e&Rename != 0 {
			o = (o &^ int64(Rename)) | int64(NoteRename)
		}
		if e&Remove != 0 {
			o = (o &^ int64(Remove)) | int64(NoteDelete)
		}
		return
	}
	nat2not = map[Event]Event{
		NoteWrite:  Write,
		NoteRename: Rename,
		NoteDelete: Remove,
		NoteExtend: Event(0),
		NoteAttrib: Event(0),
		NoteRevoke: Event(0),
		NoteLink:   Event(0),
	}
	not2nat = map[Event]Event{
		Write:  NoteWrite,
		Rename: NoteRename,
		Remove: NoteDelete,
	}
}
