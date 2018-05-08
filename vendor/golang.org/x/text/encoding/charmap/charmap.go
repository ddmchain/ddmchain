
//go:generate go run maketables.go

package charmap 

import (
	"unicode/utf8"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/internal"
	"golang.org/x/text/encoding/internal/identifier"
	"golang.org/x/text/transform"
)

var (

	ISO8859_6E encoding.Encoding = &iso8859_6E

	ISO8859_6I encoding.Encoding = &iso8859_6I

	ISO8859_8E encoding.Encoding = &iso8859_8E

	ISO8859_8I encoding.Encoding = &iso8859_8I

	iso8859_6E = internal.Encoding{
		ISO8859_6,
		"ISO-8859-6E",
		identifier.ISO88596E,
	}

	iso8859_6I = internal.Encoding{
		ISO8859_6,
		"ISO-8859-6I",
		identifier.ISO88596I,
	}

	iso8859_8E = internal.Encoding{
		ISO8859_8,
		"ISO-8859-8E",
		identifier.ISO88598E,
	}

	iso8859_8I = internal.Encoding{
		ISO8859_8,
		"ISO-8859-8I",
		identifier.ISO88598I,
	}
)

var All = listAll

type utf8Enc struct {
	len  uint8
	data [3]byte
}

type charmap struct {

	name string

	mib identifier.MIB

	asciiSuperset bool

	low uint8

	replacement byte

	decode [256]utf8Enc

	encode [256]uint32
}

func (m *charmap) NewDecoder() *encoding.Decoder {
	return &encoding.Decoder{Transformer: charmapDecoder{charmap: m}}
}

func (m *charmap) NewEncoder() *encoding.Encoder {
	return &encoding.Encoder{Transformer: charmapEncoder{charmap: m}}
}

func (m *charmap) String() string {
	return m.name
}

func (m *charmap) ID() (mib identifier.MIB, other string) {
	return m.mib, ""
}

type charmapDecoder struct {
	transform.NopResetter
	charmap *charmap
}

func (m charmapDecoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	for i, c := range src {
		if m.charmap.asciiSuperset && c < utf8.RuneSelf {
			if nDst >= len(dst) {
				err = transform.ErrShortDst
				break
			}
			dst[nDst] = c
			nDst++
			nSrc = i + 1
			continue
		}

		decode := &m.charmap.decode[c]
		n := int(decode.len)
		if nDst+n > len(dst) {
			err = transform.ErrShortDst
			break
		}

		for j := 0; j < n; j++ {
			dst[nDst] = decode.data[j]
			nDst++
		}
		nSrc = i + 1
	}
	return nDst, nSrc, err
}

type charmapEncoder struct {
	transform.NopResetter
	charmap *charmap
}

func (m charmapEncoder) Transform(dst, src []byte, atEOF bool) (nDst, nSrc int, err error) {
	r, size := rune(0), 0
loop:
	for nSrc < len(src) {
		if nDst >= len(dst) {
			err = transform.ErrShortDst
			break
		}
		r = rune(src[nSrc])

		if r < utf8.RuneSelf {
			if m.charmap.asciiSuperset {
				nSrc++
				dst[nDst] = uint8(r)
				nDst++
				continue
			}
			size = 1

		} else {

			r, size = utf8.DecodeRune(src[nSrc:])
			if size == 1 {

				if !atEOF && !utf8.FullRune(src[nSrc:]) {
					err = transform.ErrShortSrc
				} else {
					err = internal.RepertoireError(m.charmap.replacement)
				}
				break
			}
		}

		for low, high := int(m.charmap.low), 0x100; ; {
			if low >= high {
				err = internal.RepertoireError(m.charmap.replacement)
				break loop
			}
			mid := (low + high) / 2
			got := m.charmap.encode[mid]
			gotRune := rune(got & (1<<24 - 1))
			if gotRune < r {
				low = mid + 1
			} else if gotRune > r {
				high = mid
			} else {
				dst[nDst] = byte(got >> 24)
				nDst++
				break
			}
		}
		nSrc += size
	}
	return nDst, nSrc, err
}
