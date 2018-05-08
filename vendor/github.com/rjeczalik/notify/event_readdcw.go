
// +build windows

package notify

import (
	"os"
	"path/filepath"
	"syscall"
)

const (
	osSpecificCreate Event = 1 << (20 + iota)
	osSpecificRemove
	osSpecificWrite
	osSpecificRename

	recursive

	omit

	dirmarker
)

const (
	FileNotifyChangeFileName   = Event(syscall.FILE_NOTIFY_CHANGE_FILE_NAME)
	FileNotifyChangeDirName    = Event(syscall.FILE_NOTIFY_CHANGE_DIR_NAME)
	FileNotifyChangeAttributes = Event(syscall.FILE_NOTIFY_CHANGE_ATTRIBUTES)
	FileNotifyChangeSize       = Event(syscall.FILE_NOTIFY_CHANGE_SIZE)
	FileNotifyChangeLastWrite  = Event(syscall.FILE_NOTIFY_CHANGE_LAST_WRITE)
	FileNotifyChangeLastAccess = Event(syscall.FILE_NOTIFY_CHANGE_LAST_ACCESS)
	FileNotifyChangeCreation   = Event(syscall.FILE_NOTIFY_CHANGE_CREATION)
	FileNotifyChangeSecurity   = Event(syscallFileNotifyChangeSecurity)
)

const (
	fileNotifyChangeAll      = 0x17f 
	fileNotifyChangeModified = fileNotifyChangeAll &^ (FileNotifyChangeFileName | FileNotifyChangeDirName)
)

const syscallFileNotifyChangeSecurity = 0x00000100

const (
	FileActionAdded          = Event(syscall.FILE_ACTION_ADDED) << 12
	FileActionRemoved        = Event(syscall.FILE_ACTION_REMOVED) << 12
	FileActionModified       = Event(syscall.FILE_ACTION_MODIFIED) << 14
	FileActionRenamedOldName = Event(syscall.FILE_ACTION_RENAMED_OLD_NAME) << 15
	FileActionRenamedNewName = Event(syscall.FILE_ACTION_RENAMED_NEW_NAME) << 16
)

const fileActionAll = 0x7f000 

var osestr = map[Event]string{
	FileNotifyChangeFileName:   "notify.FileNotifyChangeFileName",
	FileNotifyChangeDirName:    "notify.FileNotifyChangeDirName",
	FileNotifyChangeAttributes: "notify.FileNotifyChangeAttributes",
	FileNotifyChangeSize:       "notify.FileNotifyChangeSize",
	FileNotifyChangeLastWrite:  "notify.FileNotifyChangeLastWrite",
	FileNotifyChangeLastAccess: "notify.FileNotifyChangeLastAccess",
	FileNotifyChangeCreation:   "notify.FileNotifyChangeCreation",
	FileNotifyChangeSecurity:   "notify.FileNotifyChangeSecurity",

	FileActionAdded:          "notify.FileActionAdded",
	FileActionRemoved:        "notify.FileActionRemoved",
	FileActionModified:       "notify.FileActionModified",
	FileActionRenamedOldName: "notify.FileActionRenamedOldName",
	FileActionRenamedNewName: "notify.FileActionRenamedNewName",
}

const (
	fTypeUnknown uint8 = iota
	fTypeFile
	fTypeDirectory
)

type event struct {
	pathw  []uint16
	name   string
	ftype  uint8
	action uint32
	filter uint32
	e      Event
}

func (e *event) Event() Event     { return e.e }
func (e *event) Path() string     { return filepath.Join(syscall.UTF16ToString(e.pathw), e.name) }
func (e *event) Sys() interface{} { return e.ftype }

func (e *event) isDir() (bool, error) {
	if e.ftype != fTypeUnknown {
		return e.ftype == fTypeDirectory, nil
	}
	fi, err := os.Stat(e.Path())
	if err != nil {
		return false, err
	}
	return fi.IsDir(), nil
}
