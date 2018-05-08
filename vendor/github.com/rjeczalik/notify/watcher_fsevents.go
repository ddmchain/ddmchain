
// +build darwin,!kqueue

package notify

import (
	"errors"
	"strings"
	"sync/atomic"
)

const (
	failure = uint32(FSEventsMustScanSubDirs | FSEventsUserDropped | FSEventsKernelDropped)
	filter  = uint32(FSEventsCreated | FSEventsRemoved | FSEventsRenamed |
		FSEventsModified | FSEventsInodeMetaMod)
)

type FSEvent struct {
	Path  string 
	ID    uint64 
	Flags uint32 
}

func splitflags(set uint32) (e []uint32) {
	for i := uint32(1); set != 0; i, set = i<<1, set>>1 {
		if (set & 1) != 0 {
			e = append(e, i)
		}
	}
	return
}

type watch struct {

	prev    map[string]uint32
	c       chan<- EventInfo
	stream  *stream
	path    string
	events  uint32
	isrec   int32
	flushed bool
}

func (w *watch) strip(base string, set uint32) uint32 {
	const (
		write = FSEventsModified | FSEventsInodeMetaMod
		both  = FSEventsCreated | FSEventsRemoved
	)
	switch w.prev[base] {
	case FSEventsCreated:
		set &^= FSEventsCreated
		if set&FSEventsRemoved != 0 {
			w.prev[base] = FSEventsRemoved
			set &^= write
		}
	case FSEventsRemoved:
		set &^= FSEventsRemoved
		if set&FSEventsCreated != 0 {
			w.prev[base] = FSEventsCreated
		}
	default:
		switch set & both {
		case FSEventsCreated:
			w.prev[base] = FSEventsCreated
		case FSEventsRemoved:
			w.prev[base] = FSEventsRemoved
			set &^= write
		}
	}
	dbgprintf("split()=%v\n", Event(set))
	return set
}

func (w *watch) Dispatch(ev []FSEvent) {
	events := atomic.LoadUint32(&w.events)
	isrec := (atomic.LoadInt32(&w.isrec) == 1)
	for i := range ev {
		if ev[i].Flags&FSEventsHistoryDone != 0 {
			w.flushed = true
			continue
		}
		if !w.flushed {
			continue
		}
		dbgprintf("%v (0x%x) (%s, i=%d, ID=%d, len=%d)\n", Event(ev[i].Flags),
			ev[i].Flags, ev[i].Path, i, ev[i].ID, len(ev))
		if ev[i].Flags&failure != 0 {

			continue
		}
		if !strings.HasPrefix(ev[i].Path, w.path) {
			continue
		}
		n := len(w.path)
		base := ""
		if len(ev[i].Path) > n {
			if ev[i].Path[n] != '/' {
				continue
			}
			base = ev[i].Path[n+1:]
			if !isrec && strings.IndexByte(base, '/') != -1 {
				continue
			}
		}

		e := w.strip(string(base), ev[i].Flags) & events
		if e == 0 {
			continue
		}
		for _, e := range splitflags(e) {
			dbgprintf("%d: single event: %v", ev[i].ID, Event(e))
			w.c <- &event{
				fse:   ev[i],
				event: Event(e),
			}
		}
	}
}

func (w *watch) Stop() {
	w.stream.Stop()

	atomic.StoreUint32(&w.events, 0)
	atomic.StoreInt32(&w.isrec, 0)
}

type fsevents struct {
	watches map[string]*watch
	c       chan<- EventInfo
}

func newWatcher(c chan<- EventInfo) watcher {
	return &fsevents{
		watches: make(map[string]*watch),
		c:       c,
	}
}

func (fse *fsevents) watch(path string, event Event, isrec int32) (err error) {
	if _, ok := fse.watches[path]; ok {
		return errAlreadyWatched
	}
	w := &watch{
		prev:   make(map[string]uint32),
		c:      fse.c,
		path:   path,
		events: uint32(event),
		isrec:  isrec,
	}
	w.stream = newStream(path, w.Dispatch)
	if err = w.stream.Start(); err != nil {
		return err
	}
	fse.watches[path] = w
	return nil
}

func (fse *fsevents) unwatch(path string) (err error) {
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	w.stream.Stop()
	delete(fse.watches, path)
	return nil
}

func (fse *fsevents) Watch(path string, event Event) error {
	return fse.watch(path, event, 0)
}

func (fse *fsevents) Unwatch(path string) error {
	return fse.unwatch(path)
}

func (fse *fsevents) Rewatch(path string, oldevent, newevent Event) error {
	w, ok := fse.watches[path]
	if !ok {
		return errNotWatched
	}
	if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
		return errInvalidEventSet
	}
	atomic.StoreInt32(&w.isrec, 0)
	return nil
}

func (fse *fsevents) RecursiveWatch(path string, event Event) error {
	return fse.watch(path, event, 1)
}

func (fse *fsevents) RecursiveUnwatch(path string) error {
	return fse.unwatch(path)
}

func (fse *fsevents) RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error {
	switch [2]bool{oldpath == newpath, oldevent == newevent} {
	case [2]bool{true, true}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		atomic.StoreInt32(&w.isrec, 1)
		return nil
	case [2]bool{true, false}:
		w, ok := fse.watches[oldpath]
		if !ok {
			return errNotWatched
		}
		if !atomic.CompareAndSwapUint32(&w.events, uint32(oldevent), uint32(newevent)) {
			return errors.New("invalid event state diff")
		}
		atomic.StoreInt32(&w.isrec, 1)
		return nil
	default:

		if _, ok := fse.watches[newpath]; ok {
			return errAlreadyWatched
		}
		if err := fse.Unwatch(oldpath); err != nil {
			return err
		}

		return fse.watch(newpath, newevent, 1)
	}
}

func (fse *fsevents) Close() error {
	for _, w := range fse.watches {
		w.Stop()
	}
	fse.watches = nil
	return nil
}
