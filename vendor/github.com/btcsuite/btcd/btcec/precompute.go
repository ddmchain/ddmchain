
package btcec

import (
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io/ioutil"
	"strings"
)

//go:generate go run -tags gensecp256k1 genprecomps.go

func loadS256BytePoints() error {

	bp := secp256k1BytePoints
	if len(bp) == 0 {
		return nil
	}

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(bp))
	r, err := zlib.NewReader(decoder)
	if err != nil {
		return err
	}
	serialized, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	offset := 0
	var bytePoints [32][256][3]fieldVal
	for byteNum := 0; byteNum < 32; byteNum++ {

		for i := 0; i < 256; i++ {
			px := &bytePoints[byteNum][i][0]
			py := &bytePoints[byteNum][i][1]
			pz := &bytePoints[byteNum][i][2]
			for i := 0; i < 10; i++ {
				px.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				py.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
			for i := 0; i < 10; i++ {
				pz.n[i] = binary.LittleEndian.Uint32(serialized[offset:])
				offset += 4
			}
		}
	}
	secp256k1.bytePoints = &bytePoints
	return nil
}
