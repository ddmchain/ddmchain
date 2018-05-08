
package ssh

import (
	"io"
	"sync"
)

type buffer struct {

	*sync.Cond

	head *element 
	tail *element 

	closed bool
}

type element struct {
	buf  []byte
	next *element
}

func newBuffer() *buffer {
	e := new(element)
	b := &buffer{
		Cond: newCond(),
		head: e,
		tail: e,
	}
	return b
}

func (b *buffer) write(buf []byte) {
	b.Cond.L.Lock()
	e := &element{buf: buf}
	b.tail.next = e
	b.tail = e
	b.Cond.Signal()
	b.Cond.L.Unlock()
}

func (b *buffer) eof() {
	b.Cond.L.Lock()
	b.closed = true
	b.Cond.Signal()
	b.Cond.L.Unlock()
}

func (b *buffer) Read(buf []byte) (n int, err error) {
	b.Cond.L.Lock()
	defer b.Cond.L.Unlock()

	for len(buf) > 0 {

		if len(b.head.buf) > 0 {
			r := copy(buf, b.head.buf)
			buf, b.head.buf = buf[r:], b.head.buf[r:]
			n += r
			continue
		}

		if len(b.head.buf) == 0 && b.head != b.tail {
			b.head = b.head.next
			continue
		}

		if n > 0 {
			break
		}

		if b.closed {
			err = io.EOF
			break
		}

		b.Cond.Wait()
	}
	return
}
