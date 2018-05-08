
package openpgp

import "hash"

func NewCanonicalTextHash(h hash.Hash) hash.Hash {
	return &canonicalTextHash{h, 0}
}

type canonicalTextHash struct {
	h hash.Hash
	s int
}

var newline = []byte{'\r', '\n'}

func (cth *canonicalTextHash) Write(buf []byte) (int, error) {
	start := 0

	for i, c := range buf {
		switch cth.s {
		case 0:
			if c == '\r' {
				cth.s = 1
			} else if c == '\n' {
				cth.h.Write(buf[start:i])
				cth.h.Write(newline)
				start = i + 1
			}
		case 1:
			cth.s = 0
		}
	}

	cth.h.Write(buf[start:])
	return len(buf), nil
}

func (cth *canonicalTextHash) Sum(in []byte) []byte {
	return cth.h.Sum(in)
}

func (cth *canonicalTextHash) Reset() {
	cth.h.Reset()
	cth.s = 0
}

func (cth *canonicalTextHash) Size() int {
	return cth.h.Size()
}

func (cth *canonicalTextHash) BlockSize() int {
	return cth.h.BlockSize()
}
