
// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue,!solaris

package notify

const (
	osSpecificCreate Event = 1 << iota
	osSpecificRemove
	osSpecificWrite
	osSpecificRename

	recursive

	omit
)

var osestr = map[Event]string{}

type event struct{}

func (e *event) Event() (_ Event)         { return }
func (e *event) Path() (_ string)         { return }
func (e *event) Sys() (_ interface{})     { return }
func (e *event) isDir() (_ bool, _ error) { return }
