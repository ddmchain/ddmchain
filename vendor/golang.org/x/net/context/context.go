
package context 

import "time"

type Context interface {

	Deadline() (deadline time.Time, ok bool)

	Done() <-chan struct{}

	Err() error

	Value(key interface{}) interface{}
}

func Background() Context {
	return background
}

func TODO() Context {
	return todo
}

type CancelFunc func()
