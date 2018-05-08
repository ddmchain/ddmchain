
// +build !darwin,!linux,!freebsd,!dragonfly,!netbsd,!openbsd,!windows
// +build !kqueue,!solaris

package notify

import "errors"

type stub struct{ error }

func newWatcher(chan<- EventInfo) watcher {
	return stub{errors.New("notify: not implemented")}
}

func (s stub) Watch(string, Event) error          { return s }
func (s stub) Rewatch(string, Event, Event) error { return s }
func (s stub) Unwatch(string) (err error)         { return s }
func (s stub) Close() error                       { return s }
