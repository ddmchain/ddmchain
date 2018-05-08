
package notify

const buffer = 128

type tree interface {
	Watch(string, chan<- EventInfo, ...Event) error
	Stop(chan<- EventInfo)
	Close() error
}

func newTree() tree {
	c := make(chan EventInfo, buffer)
	w := newWatcher(c)
	if rw, ok := w.(recursiveWatcher); ok {
		return newRecursiveTree(rw, c)
	}
	return newNonrecursiveTree(w, c, make(chan EventInfo, buffer))
}
