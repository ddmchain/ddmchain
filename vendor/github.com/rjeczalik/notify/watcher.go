
package notify

import "errors"

var (
	errAlreadyWatched  = errors.New("path is already watched")
	errNotWatched      = errors.New("path is not being watched")
	errInvalidEventSet = errors.New("invalid event set provided")
)

type watcher interface {

	Watch(path string, event Event) error

	Unwatch(path string) error

	Rewatch(path string, old, new Event) error

	Close() error
}

type recursiveWatcher interface {
	RecursiveWatch(path string, event Event) error

	RecursiveUnwatch(path string) error

	RecursiveRewatch(oldpath, newpath string, oldevent, newevent Event) error
}
