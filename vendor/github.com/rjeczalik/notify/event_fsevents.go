
// +build darwin,!kqueue

package notify

const (
	osSpecificCreate = Event(FSEventsCreated)
	osSpecificRemove = Event(FSEventsRemoved)
	osSpecificWrite  = Event(FSEventsModified)
	osSpecificRename = Event(FSEventsRenamed)

	recursive = Event(0x200000)

	omit = Event(0x400000)
)

const (
	FSEventsMustScanSubDirs Event = 0x00001
	FSEventsUserDropped           = 0x00002
	FSEventsKernelDropped         = 0x00004
	FSEventsEventIdsWrapped       = 0x00008
	FSEventsHistoryDone           = 0x00010
	FSEventsRootChanged           = 0x00020
	FSEventsMount                 = 0x00040
	FSEventsUnmount               = 0x00080
	FSEventsCreated               = 0x00100
	FSEventsRemoved               = 0x00200
	FSEventsInodeMetaMod          = 0x00400
	FSEventsRenamed               = 0x00800
	FSEventsModified              = 0x01000
	FSEventsFinderInfoMod         = 0x02000
	FSEventsChangeOwner           = 0x04000
	FSEventsXattrMod              = 0x08000
	FSEventsIsFile                = 0x10000
	FSEventsIsDir                 = 0x20000
	FSEventsIsSymlink             = 0x40000
)

var osestr = map[Event]string{
	FSEventsMustScanSubDirs: "notify.FSEventsMustScanSubDirs",
	FSEventsUserDropped:     "notify.FSEventsUserDropped",
	FSEventsKernelDropped:   "notify.FSEventsKernelDropped",
	FSEventsEventIdsWrapped: "notify.FSEventsEventIdsWrapped",
	FSEventsHistoryDone:     "notify.FSEventsHistoryDone",
	FSEventsRootChanged:     "notify.FSEventsRootChanged",
	FSEventsMount:           "notify.FSEventsMount",
	FSEventsUnmount:         "notify.FSEventsUnmount",
	FSEventsInodeMetaMod:    "notify.FSEventsInodeMetaMod",
	FSEventsFinderInfoMod:   "notify.FSEventsFinderInfoMod",
	FSEventsChangeOwner:     "notify.FSEventsChangeOwner",
	FSEventsXattrMod:        "notify.FSEventsXattrMod",
	FSEventsIsFile:          "notify.FSEventsIsFile",
	FSEventsIsDir:           "notify.FSEventsIsDir",
	FSEventsIsSymlink:       "notify.FSEventsIsSymlink",
}

type event struct {
	fse   FSEvent
	event Event
}

func (ei *event) Event() Event         { return ei.event }
func (ei *event) Path() string         { return ei.fse.Path }
func (ei *event) Sys() interface{}     { return &ei.fse }
func (ei *event) isDir() (bool, error) { return ei.fse.Flags&FSEventsIsDir != 0, nil }
