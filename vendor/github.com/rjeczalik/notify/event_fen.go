
// +build solaris

package notify

const (
	osSpecificCreate Event = 0x00000100 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename

	recursive

	omit
)

const (

	FileAccess = fileAccess

	FileModified = fileModified

	FileAttrib = fileAttrib

	FileDelete = fileDelete

	FileRenameTo = fileRenameTo

	FileRenameFrom = fileRenameFrom

	FileTrunc = fileTrunc

	FileNoFollow = fileNoFollow

	Unmounted = unmounted

	MountedOver = mountedOver
)

var osestr = map[Event]string{
	FileAccess:     "notify.FileAccess",
	FileModified:   "notify.FileModified",
	FileAttrib:     "notify.FileAttrib",
	FileDelete:     "notify.FileDelete",
	FileRenameTo:   "notify.FileRenameTo",
	FileRenameFrom: "notify.FileRenameFrom",
	FileTrunc:      "notify.FileTrunc",
	FileNoFollow:   "notify.FileNoFollow",
	Unmounted:      "notify.Unmounted",
	MountedOver:    "notify.MountedOver",
}
