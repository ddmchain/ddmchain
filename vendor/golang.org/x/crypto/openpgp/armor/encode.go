
package armor

import (
	"encoding/base64"
	"io"
)

var armorHeaderSep = []byte(": ")
var blockEnd = []byte("\n=")
var newline = []byte("\n")
var armorEndOfLineOut = []byte("-----\n")

func writeSlices(out io.Writer, slices ...[]byte) (err error) {
	for _, s := range slices {
		_, err = out.Write(s)
		if err != nil {
			return err
		}
	}
	return
}

type lineBreaker struct {
	lineLength  int
	line        []byte
	used        int
	out         io.Writer
	haveWritten bool
}

func newLineBreaker(out io.Writer, lineLength int) *lineBreaker {
	return &lineBreaker{
		lineLength: lineLength,
		line:       make([]byte, lineLength),
		used:       0,
		out:        out,
	}
}

func (l *lineBreaker) Write(b []byte) (n int, err error) {
	n = len(b)

	if n == 0 {
		return
	}

	if l.used == 0 && l.haveWritten {
		_, err = l.out.Write([]byte{'\n'})
		if err != nil {
			return
		}
	}

	if l.used+len(b) < l.lineLength {
		l.used += copy(l.line[l.used:], b)
		return
	}

	l.haveWritten = true
	_, err = l.out.Write(l.line[0:l.used])
	if err != nil {
		return
	}
	excess := l.lineLength - l.used
	l.used = 0

	_, err = l.out.Write(b[0:excess])
	if err != nil {
		return
	}

	_, err = l.Write(b[excess:])
	return
}

func (l *lineBreaker) Close() (err error) {
	if l.used > 0 {
		_, err = l.out.Write(l.line[0:l.used])
		if err != nil {
			return
		}
	}

	return
}

type encoding struct {
	out       io.Writer
	breaker   *lineBreaker
	b64       io.WriteCloser
	crc       uint32
	blockType []byte
}

func (e *encoding) Write(data []byte) (n int, err error) {
	e.crc = crc24(e.crc, data)
	return e.b64.Write(data)
}

func (e *encoding) Close() (err error) {
	err = e.b64.Close()
	if err != nil {
		return
	}
	e.breaker.Close()

	var checksumBytes [3]byte
	checksumBytes[0] = byte(e.crc >> 16)
	checksumBytes[1] = byte(e.crc >> 8)
	checksumBytes[2] = byte(e.crc)

	var b64ChecksumBytes [4]byte
	base64.StdEncoding.Encode(b64ChecksumBytes[:], checksumBytes[:])

	return writeSlices(e.out, blockEnd, b64ChecksumBytes[:], newline, armorEnd, e.blockType, armorEndOfLine)
}

func Encode(out io.Writer, blockType string, headers map[string]string) (w io.WriteCloser, err error) {
	bType := []byte(blockType)
	err = writeSlices(out, armorStart, bType, armorEndOfLineOut)
	if err != nil {
		return
	}

	for k, v := range headers {
		err = writeSlices(out, []byte(k), armorHeaderSep, []byte(v), newline)
		if err != nil {
			return
		}
	}

	_, err = out.Write(newline)
	if err != nil {
		return
	}

	e := &encoding{
		out:       out,
		breaker:   newLineBreaker(out, 64),
		crc:       crc24Init,
		blockType: bType,
	}
	e.b64 = base64.NewEncoder(base64.StdEncoding, e.breaker)
	return e, nil
}
