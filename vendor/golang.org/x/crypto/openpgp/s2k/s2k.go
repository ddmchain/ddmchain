
package s2k 

import (
	"crypto"
	"hash"
	"io"
	"strconv"

	"golang.org/x/crypto/openpgp/errors"
)

type Config struct {

	Hash crypto.Hash

	S2KCount int
}

func (c *Config) hash() crypto.Hash {
	if c == nil || uint(c.Hash) == 0 {

		return crypto.SHA1
	}

	return c.Hash
}

func (c *Config) encodedCount() uint8 {
	if c == nil || c.S2KCount == 0 {
		return 96 
	}

	i := c.S2KCount
	switch {

	case i < 1024:
		i = 1024
	case i > 65011712:
		i = 65011712
	}

	return encodeCount(i)
}

func encodeCount(i int) uint8 {
	if i < 1024 || i > 65011712 {
		panic("count arg i outside the required range")
	}

	for encoded := 0; encoded < 256; encoded++ {
		count := decodeCount(uint8(encoded))
		if count >= i {
			return uint8(encoded)
		}
	}

	return 255
}

func decodeCount(c uint8) int {
	return (16 + int(c&15)) << (uint32(c>>4) + 6)
}

func Simple(out []byte, h hash.Hash, in []byte) {
	Salted(out, h, in, nil)
}

var zero [1]byte

func Salted(out []byte, h hash.Hash, in []byte, salt []byte) {
	done := 0
	var digest []byte

	for i := 0; done < len(out); i++ {
		h.Reset()
		for j := 0; j < i; j++ {
			h.Write(zero[:])
		}
		h.Write(salt)
		h.Write(in)
		digest = h.Sum(digest[:0])
		n := copy(out[done:], digest)
		done += n
	}
}

func Iterated(out []byte, h hash.Hash, in []byte, salt []byte, count int) {
	combined := make([]byte, len(in)+len(salt))
	copy(combined, salt)
	copy(combined[len(salt):], in)

	if count < len(combined) {
		count = len(combined)
	}

	done := 0
	var digest []byte
	for i := 0; done < len(out); i++ {
		h.Reset()
		for j := 0; j < i; j++ {
			h.Write(zero[:])
		}
		written := 0
		for written < count {
			if written+len(combined) > count {
				todo := count - written
				h.Write(combined[:todo])
				written = count
			} else {
				h.Write(combined)
				written += len(combined)
			}
		}
		digest = h.Sum(digest[:0])
		n := copy(out[done:], digest)
		done += n
	}
}

func Parse(r io.Reader) (f func(out, in []byte), err error) {
	var buf [9]byte

	_, err = io.ReadFull(r, buf[:2])
	if err != nil {
		return
	}

	hash, ok := HashIdToHash(buf[1])
	if !ok {
		return nil, errors.UnsupportedError("hash for S2K function: " + strconv.Itoa(int(buf[1])))
	}
	if !hash.Available() {
		return nil, errors.UnsupportedError("hash not available: " + strconv.Itoa(int(hash)))
	}
	h := hash.New()

	switch buf[0] {
	case 0:
		f := func(out, in []byte) {
			Simple(out, h, in)
		}
		return f, nil
	case 1:
		_, err = io.ReadFull(r, buf[:8])
		if err != nil {
			return
		}
		f := func(out, in []byte) {
			Salted(out, h, in, buf[:8])
		}
		return f, nil
	case 3:
		_, err = io.ReadFull(r, buf[:9])
		if err != nil {
			return
		}
		count := decodeCount(buf[8])
		f := func(out, in []byte) {
			Iterated(out, h, in, buf[:8], count)
		}
		return f, nil
	}

	return nil, errors.UnsupportedError("S2K function")
}

func Serialize(w io.Writer, key []byte, rand io.Reader, passphrase []byte, c *Config) error {
	var buf [11]byte
	buf[0] = 3 
	buf[1], _ = HashToHashId(c.hash())
	salt := buf[2:10]
	if _, err := io.ReadFull(rand, salt); err != nil {
		return err
	}
	encodedCount := c.encodedCount()
	count := decodeCount(encodedCount)
	buf[10] = encodedCount
	if _, err := w.Write(buf[:]); err != nil {
		return err
	}

	Iterated(key, c.hash().New(), passphrase, salt, count)
	return nil
}

var hashToHashIdMapping = []struct {
	id   byte
	hash crypto.Hash
	name string
}{
	{1, crypto.MD5, "MD5"},
	{2, crypto.SHA1, "SHA1"},
	{3, crypto.RIPEMD160, "RIPEMD160"},
	{8, crypto.SHA256, "SHA256"},
	{9, crypto.SHA384, "SHA384"},
	{10, crypto.SHA512, "SHA512"},
	{11, crypto.SHA224, "SHA224"},
}

func HashIdToHash(id byte) (h crypto.Hash, ok bool) {
	for _, m := range hashToHashIdMapping {
		if m.id == id {
			return m.hash, true
		}
	}
	return 0, false
}

func HashIdToString(id byte) (name string, ok bool) {
	for _, m := range hashToHashIdMapping {
		if m.id == id {
			return m.name, true
		}
	}

	return "", false
}

func HashToHashId(h crypto.Hash) (id byte, ok bool) {
	for _, m := range hashToHashIdMapping {
		if m.hash == h {
			return m.id, true
		}
	}
	return 0, false
}
