
// +build windows

package notify

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

const readBufferSize = 4096

const (
	stateRewatch uint32 = 1 << (28 + iota)
	stateUnwatch
	stateCPClose
)

const (
	onlyNotifyChanges uint32 = 0x00000FFF
	onlyNGlobalEvents uint32 = 0x0FF00000
	onlyMachineStates uint32 = 0xF0000000
)

type grip struct {
	handle    syscall.Handle
	filter    uint32
	recursive bool
	pathw     []uint16
	buffer    [readBufferSize]byte
	parent    *watched
	ovlapped  *overlappedEx
}

type overlappedEx struct {
	syscall.Overlapped
	parent *grip
}

func newGrip(cph syscall.Handle, parent *watched, filter uint32) (*grip, error) {
	g := &grip{
		handle:    syscall.InvalidHandle,
		filter:    filter,
		recursive: parent.recursive,
		pathw:     parent.pathw,
		parent:    parent,
		ovlapped:  &overlappedEx{},
	}
	if err := g.register(cph); err != nil {
		return nil, err
	}
	g.ovlapped.parent = g
	return g, nil
}

func (g *grip) register(cph syscall.Handle) (err error) {
	if g.handle, err = syscall.CreateFile(
		&g.pathw[0],
		syscall.FILE_LIST_DIRECTORY,
		syscall.FILE_SHARE_READ|syscall.FILE_SHARE_WRITE|syscall.FILE_SHARE_DELETE,
		nil,
		syscall.OPEN_EXISTING,
		syscall.FILE_FLAG_BACKUP_SEMANTICS|syscall.FILE_FLAG_OVERLAPPED,
		0,
	); err != nil {
		return
	}
	if _, err = syscall.CreateIoCompletionPort(g.handle, cph, 0, 0); err != nil {
		syscall.CloseHandle(g.handle)
		return
	}
	return g.readDirChanges()
}

func (g *grip) readDirChanges() error {
	return syscall.ReadDirectoryChanges(
		g.handle,
		&g.buffer[0],
		uint32(unsafe.Sizeof(g.buffer)),
		g.recursive,
		encode(g.filter),
		nil,
		(*syscall.Overlapped)(unsafe.Pointer(g.ovlapped)),
		0,
	)
}

func encode(filter uint32) uint32 {
	e := Event(filter & (onlyNGlobalEvents | onlyNotifyChanges))
	if e&dirmarker != 0 {
		return uint32(FileNotifyChangeDirName)
	}
	if e&Create != 0 {
		e = (e ^ Create) | FileNotifyChangeFileName
	}
	if e&Remove != 0 {
		e = (e ^ Remove) | FileNotifyChangeFileName
	}
	if e&Write != 0 {
		e = (e ^ Write) | FileNotifyChangeAttributes | FileNotifyChangeSize |
			FileNotifyChangeCreation | FileNotifyChangeSecurity
	}
	if e&Rename != 0 {
		e = (e ^ Rename) | FileNotifyChangeFileName
	}
	return uint32(e)
}

type watched struct {
	filter    uint32
	recursive bool
	count     uint8
	pathw     []uint16
	digrip    [2]*grip
}

func newWatched(cph syscall.Handle, filter uint32, recursive bool,
	path string) (wd *watched, err error) {
	wd = &watched{
		filter:    filter,
		recursive: recursive,
	}
	if wd.pathw, err = syscall.UTF16FromString(path); err != nil {
		return
	}
	if err = wd.recreate(cph); err != nil {
		return
	}
	return wd, nil
}

func (wd *watched) recreate(cph syscall.Handle) (err error) {
	filefilter := wd.filter &^ uint32(FileNotifyChangeDirName)
	if err = wd.updateGrip(0, cph, filefilter == 0, filefilter); err != nil {
		return
	}
	dirfilter := wd.filter & uint32(FileNotifyChangeDirName|Create|Remove)
	if err = wd.updateGrip(1, cph, dirfilter == 0, wd.filter|uint32(dirmarker)); err != nil {
		return
	}
	wd.filter &^= onlyMachineStates
	return
}

func (wd *watched) updateGrip(idx int, cph syscall.Handle, reset bool,
	newflag uint32) (err error) {
	if reset {
		wd.digrip[idx] = nil
	} else {
		if wd.digrip[idx] == nil {
			if wd.digrip[idx], err = newGrip(cph, wd, newflag); err != nil {
				wd.closeHandle()
				return
			}
		} else {
			wd.digrip[idx].filter = newflag
			wd.digrip[idx].recursive = wd.recursive
			if err = wd.digrip[idx].register(cph); err != nil {
				wd.closeHandle()
				return
			}
		}
		wd.count++
	}
	return
}

func (wd *watched) closeHandle() (err error) {
	for _, g := range wd.digrip {
		if g != nil && g.handle != syscall.InvalidHandle {
			switch suberr := syscall.CloseHandle(g.handle); {
			case suberr == nil:
				g.handle = syscall.InvalidHandle
			case err == nil:
				err = suberr
			}
		}
	}
	return
}

type readdcw struct {
	sync.Mutex
	m     map[string]*watched
	cph   syscall.Handle
	start bool
	wg    sync.WaitGroup
	c     chan<- EventInfo
}

func newWatcher(c chan<- EventInfo) watcher {
	r := &readdcw{
		m:   make(map[string]*watched),
		cph: syscall.InvalidHandle,
		c:   c,
	}
	runtime.SetFinalizer(r, func(r *readdcw) {
		if r.cph != syscall.InvalidHandle {
			syscall.CloseHandle(r.cph)
		}
	})
	return r
}

func (r *readdcw) Watch(path string, event Event) error {
	return r.watch(path, event, false)
}

func (r *readdcw) RecursiveWatch(path string, event Event) error {
	return r.watch(path, event, true)
}

func (r *readdcw) watch(path string, event Event, recursive bool) (err error) {
	if event&^(All|fileNotifyChangeAll) != 0 {
		return errors.New("notify: unknown event")
	}
	r.Lock()
	wd, ok := r.m[path]
	r.Unlock()
	if !ok {
		if err = r.lazyinit(); err != nil {
			return
		}
		r.Lock()
		defer r.Unlock()
		if wd, ok = r.m[path]; ok {
			dbgprint("watch: exists already")
			return
		}
		if wd, err = newWatched(r.cph, uint32(event), recursive, path); err != nil {
			return
		}
		r.m[path] = wd
		dbgprint("watch: new watch added")
	} else {
		dbgprint("watch: exists already")
	}
	return nil
}

func (r *readdcw) lazyinit() (err error) {
	invalid := uintptr(syscall.InvalidHandle)
	if atomic.LoadUintptr((*uintptr)(&r.cph)) == invalid {
		r.Lock()
		defer r.Unlock()
		if atomic.LoadUintptr((*uintptr)(&r.cph)) == invalid {
			cph := syscall.InvalidHandle
			if cph, err = syscall.CreateIoCompletionPort(cph, 0, 0, 0); err != nil {
				return
			}
			r.cph, r.start = cph, true
			go r.loop()
		}
	}
	return
}

func (r *readdcw) loop() {
	var n, key uint32
	var overlapped *syscall.Overlapped
	for {
		err := syscall.GetQueuedCompletionStatus(r.cph, &n, &key, &overlapped, syscall.INFINITE)
		if key == stateCPClose {
			r.Lock()
			handle := r.cph
			r.cph = syscall.InvalidHandle
			r.Unlock()
			syscall.CloseHandle(handle)
			r.wg.Done()
			return
		}
		if overlapped == nil {

			continue
		}
		overEx := (*overlappedEx)(unsafe.Pointer(overlapped))
		if n != 0 {
			r.loopevent(n, overEx)
			if err = overEx.parent.readDirChanges(); err != nil {

			}
		}
		r.loopstate(overEx)
	}
}

func (r *readdcw) loopstate(overEx *overlappedEx) {
	r.Lock()
	defer r.Unlock()
	filter := overEx.parent.parent.filter
	if filter&onlyMachineStates == 0 {
		return
	}
	if overEx.parent.parent.count--; overEx.parent.parent.count == 0 {
		switch filter & onlyMachineStates {
		case stateRewatch:
			dbgprint("loopstate rewatch")
			overEx.parent.parent.recreate(r.cph)
		case stateUnwatch:
			dbgprint("loopstate unwatch")
			delete(r.m, syscall.UTF16ToString(overEx.parent.pathw))
		case stateCPClose:
		default:
			panic(`notify: windows loopstate logic error`)
		}
	}
}

func (r *readdcw) loopevent(n uint32, overEx *overlappedEx) {
	events := []*event{}
	var currOffset uint32
	for {
		raw := (*syscall.FileNotifyInformation)(unsafe.Pointer(&overEx.parent.buffer[currOffset]))
		name := syscall.UTF16ToString((*[syscall.MAX_LONG_PATH]uint16)(unsafe.Pointer(&raw.FileName))[:raw.FileNameLength>>1])
		events = append(events, &event{
			pathw:  overEx.parent.pathw,
			filter: overEx.parent.filter,
			action: raw.Action,
			name:   name,
		})
		if raw.NextEntryOffset == 0 {
			break
		}
		if currOffset += raw.NextEntryOffset; currOffset >= n {
			break
		}
	}
	r.send(events)
}

func (r *readdcw) send(es []*event) {
	for _, e := range es {
		var syse Event
		if e.e, syse = decode(e.filter, e.action); e.e == 0 && syse == 0 {
			continue
		}
		switch {
		case e.action == syscall.FILE_ACTION_MODIFIED:
			e.ftype = fTypeUnknown
		case e.filter&uint32(dirmarker) != 0:
			e.ftype = fTypeDirectory
		default:
			e.ftype = fTypeFile
		}
		switch {
		case e.e == 0:
			e.e = syse
		case syse != 0:
			r.c <- &event{
				pathw:  e.pathw,
				name:   e.name,
				ftype:  e.ftype,
				action: e.action,
				filter: e.filter,
				e:      syse,
			}
		}
		r.c <- e
	}
}

func (r *readdcw) Rewatch(path string, oldevent, newevent Event) error {
	return r.rewatch(path, uint32(oldevent), uint32(newevent), false)
}

func (r *readdcw) RecursiveRewatch(oldpath, newpath string, oldevent,
	newevent Event) error {
	if oldpath != newpath {
		if err := r.unwatch(oldpath); err != nil {
			return err
		}
		return r.watch(newpath, newevent, true)
	}
	return r.rewatch(newpath, uint32(oldevent), uint32(newevent), true)
}

func (r *readdcw) rewatch(path string, oldevent, newevent uint32, recursive bool) (err error) {
	if Event(newevent)&^(All|fileNotifyChangeAll) != 0 {
		return errors.New("notify: unknown event")
	}
	var wd *watched
	r.Lock()
	defer r.Unlock()
	if wd, err = r.nonStateWatchedLocked(path); err != nil {
		return
	}
	if wd.filter&(onlyNotifyChanges|onlyNGlobalEvents) != oldevent {
		panic(`notify: windows re-watcher logic error`)
	}
	wd.filter = stateRewatch | newevent
	wd.recursive, recursive = recursive, wd.recursive
	if err = wd.closeHandle(); err != nil {
		wd.filter = oldevent
		wd.recursive = recursive
		return
	}
	return
}

func (r *readdcw) nonStateWatchedLocked(path string) (wd *watched, err error) {
	wd, ok := r.m[path]
	if !ok || wd == nil {
		err = errors.New(`notify: ` + path + ` path is unwatched`)
		return
	}
	if wd.filter&onlyMachineStates != 0 {
		err = errors.New(`notify: another re/unwatching operation in progress`)
		return
	}
	return
}

func (r *readdcw) Unwatch(path string) error {
	return r.unwatch(path)
}

func (r *readdcw) RecursiveUnwatch(path string) error {
	return r.unwatch(path)
}

func (r *readdcw) unwatch(path string) (err error) {
	var wd *watched
	r.Lock()
	defer r.Unlock()
	if wd, err = r.nonStateWatchedLocked(path); err != nil {
		return
	}
	wd.filter |= stateUnwatch
	if err = wd.closeHandle(); err != nil {
		wd.filter &^= stateUnwatch
		return
	}
	if _, attrErr := syscall.GetFileAttributes(&wd.pathw[0]); attrErr != nil {
		for _, g := range wd.digrip {
			if g != nil {
				dbgprint("unwatch: posting")
				if err = syscall.PostQueuedCompletionStatus(r.cph, 0, 0, (*syscall.Overlapped)(unsafe.Pointer(g.ovlapped))); err != nil {
					wd.filter &^= stateUnwatch
					return
				}
			}
		}
	}
	return
}

func (r *readdcw) Close() (err error) {
	r.Lock()
	if !r.start {
		r.Unlock()
		return nil
	}
	for _, wd := range r.m {
		wd.filter &^= onlyMachineStates
		wd.filter |= stateCPClose
		if e := wd.closeHandle(); e != nil && err == nil {
			err = e
		}
	}
	r.start = false
	r.Unlock()
	r.wg.Add(1)
	if e := syscall.PostQueuedCompletionStatus(r.cph, 0, stateCPClose, nil); e != nil && err == nil {
		return e
	}
	r.wg.Wait()
	return
}

func decode(filter, action uint32) (Event, Event) {
	switch action {
	case syscall.FILE_ACTION_ADDED:
		return gensys(filter, Create, FileActionAdded)
	case syscall.FILE_ACTION_REMOVED:
		return gensys(filter, Remove, FileActionRemoved)
	case syscall.FILE_ACTION_MODIFIED:
		return gensys(filter, Write, FileActionModified)
	case syscall.FILE_ACTION_RENAMED_OLD_NAME:
		return gensys(filter, Rename, FileActionRenamedOldName)
	case syscall.FILE_ACTION_RENAMED_NEW_NAME:
		return gensys(filter, Rename, FileActionRenamedNewName)
	}
	panic(`notify: cannot decode internal mask`)
}

func gensys(filter uint32, ge, se Event) (gene, syse Event) {
	isdir := filter&uint32(dirmarker) != 0
	if isdir && filter&uint32(FileNotifyChangeDirName) != 0 ||
		!isdir && filter&uint32(FileNotifyChangeFileName) != 0 ||
		filter&uint32(fileNotifyChangeModified) != 0 {
		syse = se
	}
	if filter&uint32(ge) != 0 {
		gene = ge
	}
	return
}
