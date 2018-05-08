
package notify

var defaultTree = newTree()

func Watch(path string, c chan<- EventInfo, events ...Event) error {
	return defaultTree.Watch(path, c, events...)
}

func Stop(c chan<- EventInfo) {
	defaultTree.Stop(c)
}
