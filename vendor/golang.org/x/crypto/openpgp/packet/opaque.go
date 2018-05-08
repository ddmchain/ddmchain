
package packet

import (
	"bytes"
	"io"
	"io/ioutil"

	"golang.org/x/crypto/openpgp/errors"
)

type OpaquePacket struct {

	Tag uint8

	Reason error

	Contents []byte
}

func (op *OpaquePacket) parse(r io.Reader) (err error) {
	op.Contents, err = ioutil.ReadAll(r)
	return
}

func (op *OpaquePacket) Serialize(w io.Writer) (err error) {
	err = serializeHeader(w, packetType(op.Tag), len(op.Contents))
	if err == nil {
		_, err = w.Write(op.Contents)
	}
	return
}

func (op *OpaquePacket) Parse() (p Packet, err error) {
	hdr := bytes.NewBuffer(nil)
	err = serializeHeader(hdr, packetType(op.Tag), len(op.Contents))
	if err != nil {
		op.Reason = err
		return op, err
	}
	p, err = Read(io.MultiReader(hdr, bytes.NewBuffer(op.Contents)))
	if err != nil {
		op.Reason = err
		p = op
	}
	return
}

type OpaqueReader struct {
	r io.Reader
}

func NewOpaqueReader(r io.Reader) *OpaqueReader {
	return &OpaqueReader{r: r}
}

func (or *OpaqueReader) Next() (op *OpaquePacket, err error) {
	tag, _, contents, err := readHeader(or.r)
	if err != nil {
		return
	}
	op = &OpaquePacket{Tag: uint8(tag), Reason: err}
	err = op.parse(contents)
	if err != nil {
		consumeAll(contents)
	}
	return
}

type OpaqueSubpacket struct {
	SubType  uint8
	Contents []byte
}

func OpaqueSubpackets(contents []byte) (result []*OpaqueSubpacket, err error) {
	var (
		subHeaderLen int
		subPacket    *OpaqueSubpacket
	)
	for len(contents) > 0 {
		subHeaderLen, subPacket, err = nextSubpacket(contents)
		if err != nil {
			break
		}
		result = append(result, subPacket)
		contents = contents[subHeaderLen+len(subPacket.Contents):]
	}
	return
}

func nextSubpacket(contents []byte) (subHeaderLen int, subPacket *OpaqueSubpacket, err error) {

	var subLen uint32
	if len(contents) < 1 {
		goto Truncated
	}
	subPacket = &OpaqueSubpacket{}
	switch {
	case contents[0] < 192:
		subHeaderLen = 2 
		if len(contents) < subHeaderLen {
			goto Truncated
		}
		subLen = uint32(contents[0])
		contents = contents[1:]
	case contents[0] < 255:
		subHeaderLen = 3 
		if len(contents) < subHeaderLen {
			goto Truncated
		}
		subLen = uint32(contents[0]-192)<<8 + uint32(contents[1]) + 192
		contents = contents[2:]
	default:
		subHeaderLen = 6 
		if len(contents) < subHeaderLen {
			goto Truncated
		}
		subLen = uint32(contents[1])<<24 |
			uint32(contents[2])<<16 |
			uint32(contents[3])<<8 |
			uint32(contents[4])
		contents = contents[5:]
	}
	if subLen > uint32(len(contents)) || subLen == 0 {
		goto Truncated
	}
	subPacket.SubType = contents[0]
	subPacket.Contents = contents[1:subLen]
	return
Truncated:
	err = errors.StructuralError("subpacket truncated")
	return
}

func (osp *OpaqueSubpacket) Serialize(w io.Writer) (err error) {
	buf := make([]byte, 6)
	n := serializeSubpacketLength(buf, len(osp.Contents)+1)
	buf[n] = osp.SubType
	if _, err = w.Write(buf[:n+1]); err != nil {
		return
	}
	_, err = w.Write(osp.Contents)
	return
}
