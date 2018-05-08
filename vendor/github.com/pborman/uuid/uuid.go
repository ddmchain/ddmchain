
package uuid

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
)

type Array [16]byte

func (uuid Array) UUID() UUID {
	return uuid[:]
}

func (uuid Array) String() string {
	return uuid.UUID().String()
}

type UUID []byte

type Version byte

type Variant byte

const (
	Invalid   = Variant(iota) 
	RFC4122                   
	Reserved                  
	Microsoft                 
	Future                    
)

var rander = rand.Reader 

func New() string {
	return NewRandom().String()
}

func Parse(s string) UUID {
	if len(s) == 36+9 {
		if strings.ToLower(s[:9]) != "urn:uuid:" {
			return nil
		}
		s = s[9:]
	} else if len(s) != 36 {
		return nil
	}
	if s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return nil
	}
	var uuid [16]byte
	for i, x := range [16]int{
		0, 2, 4, 6,
		9, 11,
		14, 16,
		19, 21,
		24, 26, 28, 30, 32, 34} {
		if v, ok := xtob(s[x:]); !ok {
			return nil
		} else {
			uuid[i] = v
		}
	}
	return uuid[:]
}

func Equal(uuid1, uuid2 UUID) bool {
	return bytes.Equal(uuid1, uuid2)
}

func (uuid UUID) Array() Array {
	if len(uuid) != 16 {
		panic("invalid uuid")
	}
	var a Array
	copy(a[:], uuid)
	return a
}

func (uuid UUID) String() string {
	if len(uuid) != 16 {
		return ""
	}
	var buf [36]byte
	encodeHex(buf[:], uuid)
	return string(buf[:])
}

func (uuid UUID) URN() string {
	if len(uuid) != 16 {
		return ""
	}
	var buf [36 + 9]byte
	copy(buf[:], "urn:uuid:")
	encodeHex(buf[9:], uuid)
	return string(buf[:])
}

func encodeHex(dst []byte, uuid UUID) {
	hex.Encode(dst[:], uuid[:4])
	dst[8] = '-'
	hex.Encode(dst[9:13], uuid[4:6])
	dst[13] = '-'
	hex.Encode(dst[14:18], uuid[6:8])
	dst[18] = '-'
	hex.Encode(dst[19:23], uuid[8:10])
	dst[23] = '-'
	hex.Encode(dst[24:], uuid[10:])
}

func (uuid UUID) Variant() Variant {
	if len(uuid) != 16 {
		return Invalid
	}
	switch {
	case (uuid[8] & 0xc0) == 0x80:
		return RFC4122
	case (uuid[8] & 0xe0) == 0xc0:
		return Microsoft
	case (uuid[8] & 0xe0) == 0xe0:
		return Future
	default:
		return Reserved
	}
}

func (uuid UUID) Version() (Version, bool) {
	if len(uuid) != 16 {
		return 0, false
	}
	return Version(uuid[6] >> 4), true
}

func (v Version) String() string {
	if v > 15 {
		return fmt.Sprintf("BAD_VERSION_%d", v)
	}
	return fmt.Sprintf("VERSION_%d", v)
}

func (v Variant) String() string {
	switch v {
	case RFC4122:
		return "RFC4122"
	case Reserved:
		return "Reserved"
	case Microsoft:
		return "Microsoft"
	case Future:
		return "Future"
	case Invalid:
		return "Invalid"
	}
	return fmt.Sprintf("BadVariant%d", int(v))
}

func SetRand(r io.Reader) {
	if r == nil {
		rander = rand.Reader
		return
	}
	rander = r
}
