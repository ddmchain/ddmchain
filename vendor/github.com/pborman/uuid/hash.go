
package uuid

import (
	"crypto/md5"
	"crypto/sha1"
	"hash"
)

var (
	NameSpace_DNS  = Parse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")
	NameSpace_URL  = Parse("6ba7b811-9dad-11d1-80b4-00c04fd430c8")
	NameSpace_OID  = Parse("6ba7b812-9dad-11d1-80b4-00c04fd430c8")
	NameSpace_X500 = Parse("6ba7b814-9dad-11d1-80b4-00c04fd430c8")
	NIL            = Parse("00000000-0000-0000-0000-000000000000")
)

func NewHash(h hash.Hash, space UUID, data []byte, version int) UUID {
	h.Reset()
	h.Write(space)
	h.Write([]byte(data))
	s := h.Sum(nil)
	uuid := make([]byte, 16)
	copy(uuid, s)
	uuid[6] = (uuid[6] & 0x0f) | uint8((version&0xf)<<4)
	uuid[8] = (uuid[8] & 0x3f) | 0x80 
	return uuid
}

func NewMD5(space UUID, data []byte) UUID {
	return NewHash(md5.New(), space, data, 3)
}

func NewSHA1(space UUID, data []byte) UUID {
	return NewHash(sha1.New(), space, data, 5)
}
