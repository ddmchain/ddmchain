
// +build darwin,kqueue dragonfly freebsd netbsd openbsd

package notify

import "syscall"

const (
	osSpecificCreate Event = 0x0100 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename

	recursive

	omit
)

const (

	NoteDelete = Event(syscall.NOTE_DELETE)

	NoteWrite = Event(syscall.NOTE_WRITE)

	NoteExtend = Event(syscall.NOTE_EXTEND)

	NoteAttrib = Event(syscall.NOTE_ATTRIB)

	NoteLink = Event(syscall.NOTE_LINK)

	NoteRename = Event(syscall.NOTE_RENAME)

	NoteRevoke = Event(syscall.NOTE_REVOKE)
)

var osestr = map[Event]string{
	NoteDelete: "notify.NoteDelete",
	NoteWrite:  "notify.NoteWrite",
	NoteExtend: "notify.NoteExtend",
	NoteAttrib: "notify.NoteAttrib",
	NoteLink:   "notify.NoteLink",
	NoteRename: "notify.NoteRename",
	NoteRevoke: "notify.NoteRevoke",
}
