
package packet

import (
	"bytes"
	"image"
	"image/jpeg"
	"io"
	"io/ioutil"
)

const UserAttrImageSubpacket = 1

type UserAttribute struct {
	Contents []*OpaqueSubpacket
}

func NewUserAttributePhoto(photos ...image.Image) (uat *UserAttribute, err error) {
	uat = new(UserAttribute)
	for _, photo := range photos {
		var buf bytes.Buffer

		data := []byte{
			0x10, 0x00, 
			0x01,       
			0x01,       
			0, 0, 0, 0, 
			0, 0, 0, 0,
			0, 0, 0, 0}
		if _, err = buf.Write(data); err != nil {
			return
		}
		if err = jpeg.Encode(&buf, photo, nil); err != nil {
			return
		}
		uat.Contents = append(uat.Contents, &OpaqueSubpacket{
			SubType:  UserAttrImageSubpacket,
			Contents: buf.Bytes()})
	}
	return
}

func NewUserAttribute(contents ...*OpaqueSubpacket) *UserAttribute {
	return &UserAttribute{Contents: contents}
}

func (uat *UserAttribute) parse(r io.Reader) (err error) {

	b, err := ioutil.ReadAll(r)
	if err != nil {
		return
	}
	uat.Contents, err = OpaqueSubpackets(b)
	return
}

func (uat *UserAttribute) Serialize(w io.Writer) (err error) {
	var buf bytes.Buffer
	for _, sp := range uat.Contents {
		sp.Serialize(&buf)
	}
	if err = serializeHeader(w, packetTypeUserAttribute, buf.Len()); err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return
}

func (uat *UserAttribute) ImageData() (imageData [][]byte) {
	for _, sp := range uat.Contents {
		if sp.SubType == UserAttrImageSubpacket && len(sp.Contents) > 16 {
			imageData = append(imageData, sp.Contents[16:])
		}
	}
	return
}
