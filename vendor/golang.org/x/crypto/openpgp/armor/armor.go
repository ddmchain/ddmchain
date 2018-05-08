
package armor 

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"golang.org/x/crypto/openpgp/errors"
	"io"
)

type Block struct {
	Type    string            
	Header  map[string]string 
	Body    io.Reader         
	lReader lineReader
	oReader openpgpReader
}

var ArmorCorrupt error = errors.StructuralError("armor invalid")

const crc24Init = 0xb704ce
const crc24Poly = 0x1864cfb
const crc24Mask = 0xffffff

func crc24(crc uint32, d []byte) uint32 {
	for _, b := range d {
		crc ^= uint32(b) << 16
		for i := 0; i < 8; i++ {
			crc <<= 1
			if crc&0x1000000 != 0 {
				crc ^= crc24Poly
			}
		}
	}
	return crc
}

var armorStart = []byte("-----BEGIN ")
var armorEnd = []byte("-----END ")
var armorEndOfLine = []byte("-----")

type lineReader struct {
	in  *bufio.Reader
	buf []byte
	eof bool
	crc uint32
}

func (l *lineReader) Read(p []byte) (n int, err error) {
	if l.eof {
		return 0, io.EOF
	}

	if len(l.buf) > 0 {
		n = copy(p, l.buf)
		l.buf = l.buf[n:]
		return
	}

	line, isPrefix, err := l.in.ReadLine()
	if err != nil {
		return
	}
	if isPrefix {
		return 0, ArmorCorrupt
	}

	if len(line) == 5 && line[0] == '=' {

		var expectedBytes [3]byte
		var m int
		m, err = base64.StdEncoding.Decode(expectedBytes[0:], line[1:])
		if m != 3 || err != nil {
			return
		}
		l.crc = uint32(expectedBytes[0])<<16 |
			uint32(expectedBytes[1])<<8 |
			uint32(expectedBytes[2])

		line, _, err = l.in.ReadLine()
		if err != nil && err != io.EOF {
			return
		}
		if !bytes.HasPrefix(line, armorEnd) {
			return 0, ArmorCorrupt
		}

		l.eof = true
		return 0, io.EOF
	}

	if len(line) > 96 {
		return 0, ArmorCorrupt
	}

	n = copy(p, line)
	bytesToSave := len(line) - n
	if bytesToSave > 0 {
		if cap(l.buf) < bytesToSave {
			l.buf = make([]byte, 0, bytesToSave)
		}
		l.buf = l.buf[0:bytesToSave]
		copy(l.buf, line[n:])
	}

	return
}

type openpgpReader struct {
	lReader    *lineReader
	b64Reader  io.Reader
	currentCRC uint32
}

func (r *openpgpReader) Read(p []byte) (n int, err error) {
	n, err = r.b64Reader.Read(p)
	r.currentCRC = crc24(r.currentCRC, p[:n])

	if err == io.EOF {
		if r.lReader.crc != uint32(r.currentCRC&crc24Mask) {
			return 0, ArmorCorrupt
		}
	}

	return
}

func Decode(in io.Reader) (p *Block, err error) {
	r := bufio.NewReaderSize(in, 100)
	var line []byte
	ignoreNext := false

TryNextBlock:
	p = nil

	for {
		ignoreThis := ignoreNext
		line, ignoreNext, err = r.ReadLine()
		if err != nil {
			return
		}
		if ignoreNext || ignoreThis {
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) > len(armorStart)+len(armorEndOfLine) && bytes.HasPrefix(line, armorStart) {
			break
		}
	}

	p = new(Block)
	p.Type = string(line[len(armorStart) : len(line)-len(armorEndOfLine)])
	p.Header = make(map[string]string)
	nextIsContinuation := false
	var lastKey string

	for {
		isContinuation := nextIsContinuation
		line, nextIsContinuation, err = r.ReadLine()
		if err != nil {
			p = nil
			return
		}
		if isContinuation {
			p.Header[lastKey] += string(line)
			continue
		}
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			break
		}

		i := bytes.Index(line, []byte(": "))
		if i == -1 {
			goto TryNextBlock
		}
		lastKey = string(line[:i])
		p.Header[lastKey] = string(line[i+2:])
	}

	p.lReader.in = r
	p.oReader.currentCRC = crc24Init
	p.oReader.lReader = &p.lReader
	p.oReader.b64Reader = base64.NewDecoder(base64.StdEncoding, &p.lReader)
	p.Body = &p.oReader

	return
}
