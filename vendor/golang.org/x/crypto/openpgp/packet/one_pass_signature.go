
package packet

import (
	"crypto"
	"encoding/binary"
	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/s2k"
	"io"
	"strconv"
)

type OnePassSignature struct {
	SigType    SignatureType
	Hash       crypto.Hash
	PubKeyAlgo PublicKeyAlgorithm
	KeyId      uint64
	IsLast     bool
}

const onePassSignatureVersion = 3

func (ops *OnePassSignature) parse(r io.Reader) (err error) {
	var buf [13]byte

	_, err = readFull(r, buf[:])
	if err != nil {
		return
	}
	if buf[0] != onePassSignatureVersion {
		err = errors.UnsupportedError("one-pass-signature packet version " + strconv.Itoa(int(buf[0])))
	}

	var ok bool
	ops.Hash, ok = s2k.HashIdToHash(buf[2])
	if !ok {
		return errors.UnsupportedError("hash function: " + strconv.Itoa(int(buf[2])))
	}

	ops.SigType = SignatureType(buf[1])
	ops.PubKeyAlgo = PublicKeyAlgorithm(buf[3])
	ops.KeyId = binary.BigEndian.Uint64(buf[4:12])
	ops.IsLast = buf[12] != 0
	return
}

func (ops *OnePassSignature) Serialize(w io.Writer) error {
	var buf [13]byte
	buf[0] = onePassSignatureVersion
	buf[1] = uint8(ops.SigType)
	var ok bool
	buf[2], ok = s2k.HashToHashId(ops.Hash)
	if !ok {
		return errors.UnsupportedError("hash type: " + strconv.Itoa(int(ops.Hash)))
	}
	buf[3] = uint8(ops.PubKeyAlgo)
	binary.BigEndian.PutUint64(buf[4:12], ops.KeyId)
	if ops.IsLast {
		buf[12] = 1
	}

	if err := serializeHeader(w, packetTypeOnePassSignature, len(buf)); err != nil {
		return err
	}
	_, err := w.Write(buf[:])
	return err
}
