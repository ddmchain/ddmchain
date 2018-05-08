
// +build solaris

package notify

import (
	"fmt"
	"os"
	"syscall"
)

func newTrigger(pthLkp map[string]*watched) trigger {
	return &fen{
		pthLkp: pthLkp,
		cf:     newCfen(),
	}
}

type fen struct {

	p int

	pthLkp map[string]*watched

	cf cfen
}

type watched struct {
	trgWatched
}

func (f *fen) Stop() error {
	return f.cf.portAlert(f.p)
}

func (f *fen) Close() (err error) {
	return syscall.Close(f.p)
}

func (*fen) NewWatched(p string, fi os.FileInfo) (*watched, error) {
	return &watched{trgWatched{p: p, fi: fi}}, nil
}

func (f *fen) Record(w *watched) {
	f.pthLkp[w.p] = w
}

func (f *fen) Del(w *watched) {
	delete(f.pthLkp, w.p)
}

func inter2pe(n interface{}) PortEvent {
	pe, ok := n.(PortEvent)
	if !ok {
		panic(fmt.Sprintf("fen: type should be PortEvent, %T instead", n))
	}
	return pe
}

func (f *fen) Watched(n interface{}) (*watched, int64, error) {
	pe := inter2pe(n)
	fo, ok := pe.PortevObject.(*FileObj)
	if !ok || fo == nil {
		panic(fmt.Sprintf("fen: type should be *FileObj, %T instead", fo))
	}
	w, ok := f.pthLkp[fo.Name]
	if !ok {
		return nil, 0, errNotWatched
	}
	return w, int64(pe.PortevEvents), nil
}

func (f *fen) Init() (err error) {
	f.p, err = f.cf.portCreate()
	return
}

func fi2fo(fi os.FileInfo, p string) FileObj {
	st, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		panic(fmt.Sprintf("fen: type should be *syscall.Stat_t, %T instead", st))
	}
	return FileObj{Name: p, Atim: st.Atim, Mtim: st.Mtim, Ctim: st.Ctim}
}

func (f *fen) Unwatch(w *watched) error {
	return f.cf.portDissociate(f.p, FileObj{Name: w.p})
}

func (f *fen) Watch(fi os.FileInfo, w *watched, e int64) error {
	return f.cf.portAssociate(f.p, fi2fo(fi, w.p), int(e))
}

func (f *fen) Wait() (interface{}, error) {
	var (
		pe  PortEvent
		err error
	)
	err = f.cf.portGet(f.p, &pe)
	return pe, err
}

func (f *fen) IsStop(n interface{}, err error) bool {
	return err == syscall.EBADF || inter2pe(n).PortevSource == srcAlert
}

func init() {
	encode = func(e Event, dir bool) (o int64) {

		o = int64(e &^ Create)
		if (e&Create != 0 && dir) || e&Write != 0 {
			o = (o &^ int64(Write)) | int64(FileModified)
		}

		o &= int64(^Rename & ^Remove &^ FileDelete &^ FileRenameTo &^
			FileRenameFrom &^ Unmounted &^ MountedOver)
		return
	}
	nat2not = map[Event]Event{
		FileModified:   Write,
		FileRenameFrom: Rename,
		FileDelete:     Remove,
		FileAccess:     Event(0),
		FileAttrib:     Event(0),
		FileRenameTo:   Event(0),
		FileTrunc:      Event(0),
		FileNoFollow:   Event(0),
		Unmounted:      Event(0),
		MountedOver:    Event(0),
	}
	not2nat = map[Event]Event{
		Write:  FileModified,
		Rename: FileRenameFrom,
		Remove: FileDelete,
	}
}
