
// +build darwin,kqueue dragonfly freebsd netbsd openbsd solaris

package notify

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
)

type trigger interface {

	Close() error

	Stop() error

	NewWatched(string, os.FileInfo) (*watched, error)

	Record(*watched)

	Del(*watched)

	Watched(interface{}) (*watched, int64, error)

	Init() error

	Watch(os.FileInfo, *watched, int64) error

	Unwatch(*watched) error

	Wait() (interface{}, error)

	IsStop(n interface{}, err error) bool
}

type trgWatched struct {

	p string

	fi os.FileInfo

	eDir Event

	eNonDir Event
}

var encode func(Event, bool) int64

var (

	nat2not map[Event]Event

	not2nat map[Event]Event
)

type trg struct {
	sync.Mutex

	s chan struct{}

	c chan<- EventInfo

	pthLkp map[string]*watched

	t trigger
}

func newWatcher(c chan<- EventInfo) watcher {
	t := &trg{
		s:      make(chan struct{}, 1),
		pthLkp: make(map[string]*watched, 0),
		c:      c,
	}
	t.t = newTrigger(t.pthLkp)
	if err := t.t.Init(); err != nil {
		panic(err)
	}
	go t.monitor()
	return t
}

func (t *trg) Close() (err error) {
	t.Lock()
	if err = t.t.Stop(); err != nil {
		t.Unlock()
		return
	}
	<-t.s
	var e error
	for _, w := range t.pthLkp {
		if e = t.unwatch(w.p, w.fi); e != nil {
			dbgprintf("trg: unwatch %q failed: %q\n", w.p, e)
			err = nonil(err, e)
		}
	}
	if e = t.t.Close(); e != nil {
		dbgprintf("trg: closing native watch failed: %q\n", e)
		err = nonil(err, e)
	}
	if remaining := len(t.pthLkp); remaining != 0 {
		err = nonil(err, fmt.Errorf("Not all watches were removed: len(t.pthLkp) == %v", len(t.pthLkp)))
	}
	t.Unlock()
	return
}

func (t *trg) send(evn []event) {
	for i := range evn {
		t.c <- &evn[i]
	}
}

func (t *trg) singlewatch(p string, e Event, direct mode, fi os.FileInfo) (err error) {
	w, ok := t.pthLkp[p]
	if !ok {
		if w, err = t.t.NewWatched(p, fi); err != nil {
			return
		}
	}
	switch direct {
	case dir:
		w.eDir |= e
	case ndir:
		w.eNonDir |= e
	case both:
		w.eDir |= e
		w.eNonDir |= e
	}
	if err = t.t.Watch(fi, w, encode(w.eDir|w.eNonDir, fi.IsDir())); err != nil {
		return
	}
	if !ok {
		t.t.Record(w)
		return nil
	}
	return errAlreadyWatched
}

func decode(o int64, w Event) (e Event) {
	for f, n := range nat2not {
		if o&int64(f) != 0 {
			if w&f != 0 {
				e |= f
			}
			if w&n != 0 {
				e |= n
			}
		}
	}

	return
}

func (t *trg) watch(p string, e Event, fi os.FileInfo) error {
	if err := t.singlewatch(p, e, dir, fi); err != nil {
		if err != errAlreadyWatched {
			return err
		}
	}
	if fi.IsDir() {
		err := t.walk(p, func(fi os.FileInfo) (err error) {
			if err = t.singlewatch(filepath.Join(p, fi.Name()), e, ndir,
				fi); err != nil {
				if err != errAlreadyWatched {
					return
				}
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (t *trg) walk(p string, fn func(os.FileInfo) error) error {
	fp, err := os.Open(p)
	if err != nil {
		return err
	}
	ls, err := fp.Readdir(0)
	fp.Close()
	if err != nil {
		return err
	}
	for i := range ls {
		if err := fn(ls[i]); err != nil {
			return err
		}
	}
	return nil
}

func (t *trg) unwatch(p string, fi os.FileInfo) error {
	if fi.IsDir() {
		err := t.walk(p, func(fi os.FileInfo) error {
			err := t.singleunwatch(filepath.Join(p, fi.Name()), ndir)
			if err != errNotWatched {
				return err
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	return t.singleunwatch(p, dir)
}

func (t *trg) Watch(p string, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	err = t.watch(p, e, fi)
	t.Unlock()
	return err
}

func (t *trg) Unwatch(p string) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	err = t.unwatch(p, fi)
	t.Unlock()
	return err
}

func (t *trg) Rewatch(p string, _, e Event) error {
	fi, err := os.Stat(p)
	if err != nil {
		return err
	}
	t.Lock()
	if err = t.unwatch(p, fi); err == nil {

		err = t.watch(p, e, fi)
	}
	t.Unlock()
	return nil
}

func (*trg) file(w *watched, n interface{}, e Event) (evn []event) {
	evn = append(evn, event{w.p, e, w.fi.IsDir(), n})
	return
}

func (t *trg) dir(w *watched, n interface{}, e, ge Event) (evn []event) {

	if (ge & (not2nat[Rename] | not2nat[Remove])) != 0 {

		evn = append(evn, event{w.p, e & ^Write & ^not2nat[Write], true, n})
		if ge&not2nat[Rename] != 0 {
			for p := range t.pthLkp {
				if strings.HasPrefix(p, w.p+string(os.PathSeparator)) {
					if err := t.singleunwatch(p, both); err != nil && err != errNotWatched &&
						!os.IsNotExist(err) {
						dbgprintf("trg: failed stop watching moved file (%q): %q\n",
							p, err)
					}
					if (w.eDir|w.eNonDir)&(not2nat[Rename]|Rename) != 0 {
						evn = append(evn, event{
							p, (w.eDir | w.eNonDir) & e &^ Write &^ not2nat[Write],
							w.fi.IsDir(), nil,
						})
					}
				}
			}
		}
		t.t.Del(w)
		return
	}
	if (ge & not2nat[Write]) != 0 {
		switch err := t.walk(w.p, func(fi os.FileInfo) error {
			p := filepath.Join(w.p, fi.Name())
			switch err := t.singlewatch(p, w.eDir, ndir, fi); {
			case os.IsNotExist(err) && ((w.eDir & Remove) != 0):
				evn = append(evn, event{p, Remove, fi.IsDir(), n})
			case err == errAlreadyWatched:
			case err != nil:
				dbgprintf("trg: watching %q failed: %q", p, err)
			case (w.eDir & Create) != 0:
				evn = append(evn, event{p, Create, fi.IsDir(), n})
			default:
			}
			return nil
		}); {
		case os.IsNotExist(err):
			return
		case err != nil:
			dbgprintf("trg: dir processing failed: %q", err)
		default:
		}
	}
	return
}

type mode uint

const (
	dir mode = iota
	ndir
	both
)

func (t *trg) singleunwatch(p string, direct mode) error {
	w, ok := t.pthLkp[p]
	if !ok {
		return errNotWatched
	}
	switch direct {
	case dir:
		w.eDir = 0
	case ndir:
		w.eNonDir = 0
	case both:
		w.eDir, w.eNonDir = 0, 0
	}
	if err := t.t.Unwatch(w); err != nil {
		return err
	}
	if w.eNonDir|w.eDir != 0 {
		mod := dir
		if w.eNonDir != 0 {
			mod = ndir
		}
		if err := t.singlewatch(p, w.eNonDir|w.eDir, mod,
			w.fi); err != nil && err != errAlreadyWatched {
			return err
		}
	} else {
		t.t.Del(w)
	}
	return nil
}

func (t *trg) monitor() {
	var (
		n   interface{}
		err error
	)
	for {
		switch n, err = t.t.Wait(); {
		case err == syscall.EINTR:
		case t.t.IsStop(n, err):
			t.s <- struct{}{}
			return
		case err != nil:
			dbgprintf("trg: failed to read events: %q\n", err)
		default:
			t.send(t.process(n))
		}
	}
}

func (t *trg) process(n interface{}) (evn []event) {
	t.Lock()
	w, ge, err := t.t.Watched(n)
	if err != nil {
		t.Unlock()
		dbgprintf("trg: %v event lookup failed: %q", Event(ge), err)
		return
	}

	e := decode(ge, w.eDir|w.eNonDir)
	if ge&int64(not2nat[Remove]|not2nat[Rename]) == 0 {
		switch fi, err := os.Stat(w.p); {
		case err != nil:
		default:
			if err = t.t.Watch(fi, w, encode(w.eDir|w.eNonDir, fi.IsDir())); err != nil {
				dbgprintf("trg: %q is no longer watched: %q", w.p, err)
				t.t.Del(w)
			}
		}
	}
	if e == Event(0) && (!w.fi.IsDir() || (ge&int64(not2nat[Write])) == 0) {
		t.Unlock()
		return
	}

	if w.fi.IsDir() {
		evn = append(evn, t.dir(w, n, e, Event(ge))...)
	} else {
		evn = append(evn, t.file(w, n, e)...)
	}
	if Event(ge)&(not2nat[Remove]|not2nat[Rename]) != 0 {
		t.t.Del(w)
	}
	t.Unlock()
	return
}
