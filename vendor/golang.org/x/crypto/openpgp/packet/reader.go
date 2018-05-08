
package packet

import (
	"golang.org/x/crypto/openpgp/errors"
	"io"
)

type Reader struct {
	q       []Packet
	readers []io.Reader
}

const maxReaders = 32

func (r *Reader) Next() (p Packet, err error) {
	if len(r.q) > 0 {
		p = r.q[len(r.q)-1]
		r.q = r.q[:len(r.q)-1]
		return
	}

	for len(r.readers) > 0 {
		p, err = Read(r.readers[len(r.readers)-1])
		if err == nil {
			return
		}
		if err == io.EOF {
			r.readers = r.readers[:len(r.readers)-1]
			continue
		}
		if _, ok := err.(errors.UnknownPacketTypeError); !ok {
			return nil, err
		}
	}

	return nil, io.EOF
}

func (r *Reader) Push(reader io.Reader) (err error) {
	if len(r.readers) >= maxReaders {
		return errors.StructuralError("too many layers of packets")
	}
	r.readers = append(r.readers, reader)
	return nil
}

func (r *Reader) Unread(p Packet) {
	r.q = append(r.q, p)
}

func NewReader(r io.Reader) *Reader {
	return &Reader{
		q:       nil,
		readers: []io.Reader{r},
	}
}
