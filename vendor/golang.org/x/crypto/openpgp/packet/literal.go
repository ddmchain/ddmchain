
package packet

import (
	"encoding/binary"
	"io"
)

type LiteralData struct {
	IsBinary bool
	FileName string
	Time     uint32 
	Body     io.Reader
}

func (l *LiteralData) ForEyesOnly() bool {
	return l.FileName == "_CONSOLE"
}

func (l *LiteralData) parse(r io.Reader) (err error) {
	var buf [256]byte

	_, err = readFull(r, buf[:2])
	if err != nil {
		return
	}

	l.IsBinary = buf[0] == 'b'
	fileNameLen := int(buf[1])

	_, err = readFull(r, buf[:fileNameLen])
	if err != nil {
		return
	}

	l.FileName = string(buf[:fileNameLen])

	_, err = readFull(r, buf[:4])
	if err != nil {
		return
	}

	l.Time = binary.BigEndian.Uint32(buf[:4])
	l.Body = r
	return
}

func SerializeLiteral(w io.WriteCloser, isBinary bool, fileName string, time uint32) (plaintext io.WriteCloser, err error) {
	var buf [4]byte
	buf[0] = 't'
	if isBinary {
		buf[0] = 'b'
	}
	if len(fileName) > 255 {
		fileName = fileName[:255]
	}
	buf[1] = byte(len(fileName))

	inner, err := serializeStreamHeader(w, packetTypeLiteralData)
	if err != nil {
		return
	}

	_, err = inner.Write(buf[:2])
	if err != nil {
		return
	}
	_, err = inner.Write([]byte(fileName))
	if err != nil {
		return
	}
	binary.BigEndian.PutUint32(buf[:], time)
	_, err = inner.Write(buf[:])
	if err != nil {
		return
	}

	plaintext = inner
	return
}
