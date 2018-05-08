
// +build linux

package notify

import "golang.org/x/sys/unix"

const (
	osSpecificCreate Event = 0x100000 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename

	recursive

	omit
)

const (
	InAccess       = Event(unix.IN_ACCESS)        
	InModify       = Event(unix.IN_MODIFY)        
	InAttrib       = Event(unix.IN_ATTRIB)        
	InCloseWrite   = Event(unix.IN_CLOSE_WRITE)   
	InCloseNowrite = Event(unix.IN_CLOSE_NOWRITE) 
	InOpen         = Event(unix.IN_OPEN)          
	InMovedFrom    = Event(unix.IN_MOVED_FROM)    
	InMovedTo      = Event(unix.IN_MOVED_TO)      
	InCreate       = Event(unix.IN_CREATE)        
	InDelete       = Event(unix.IN_DELETE)        
	InDeleteSelf   = Event(unix.IN_DELETE_SELF)   
	InMoveSelf     = Event(unix.IN_MOVE_SELF)     
)

var osestr = map[Event]string{
	InAccess:       "notify.InAccess",
	InModify:       "notify.InModify",
	InAttrib:       "notify.InAttrib",
	InCloseWrite:   "notify.InCloseWrite",
	InCloseNowrite: "notify.InCloseNowrite",
	InOpen:         "notify.InOpen",
	InMovedFrom:    "notify.InMovedFrom",
	InMovedTo:      "notify.InMovedTo",
	InCreate:       "notify.InCreate",
	InDelete:       "notify.InDelete",
	InDeleteSelf:   "notify.InDeleteSelf",
	InMoveSelf:     "notify.InMoveSelf",
}

const (
	inDontFollow = Event(unix.IN_DONT_FOLLOW)
	inExclUnlink = Event(unix.IN_EXCL_UNLINK)
	inMaskAdd    = Event(unix.IN_MASK_ADD)
	inOneshot    = Event(unix.IN_ONESHOT)
	inOnlydir    = Event(unix.IN_ONLYDIR)
)

type event struct {
	sys   unix.InotifyEvent
	path  string
	event Event
}

func (e *event) Event() Event         { return e.event }
func (e *event) Path() string         { return e.path }
func (e *event) Sys() interface{}     { return &e.sys }
func (e *event) isDir() (bool, error) { return e.sys.Mask&unix.IN_ISDIR != 0, nil }
