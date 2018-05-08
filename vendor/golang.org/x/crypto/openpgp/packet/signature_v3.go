
package packet

import (
	"crypto"
	"encoding/binary"
	"fmt"
	"io"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/s2k"
)

type SignatureV3 struct {
	SigType      SignatureType
	CreationTime time.Time
	IssuerKeyId  uint64
	PubKeyAlgo   PublicKeyAlgorithm
	Hash         crypto.Hash
	HashTag      [2]byte

	RSASignature     parsedMPI
	DSASigR, DSASigS parsedMPI
}

func (sig *SignatureV3) parse(r io.Reader) (err error) {

	var buf [8]byte
	if _, err = readFull(r, buf[:1]); err != nil {
		return
	}
	if buf[0] < 2 || buf[0] > 3 {
		err = errors.UnsupportedError("signature packet version " + strconv.Itoa(int(buf[0])))
		return
	}
	if _, err = readFull(r, buf[:1]); err != nil {
		return
	}
	if buf[0] != 5 {
		err = errors.UnsupportedError(
			"invalid hashed material length " + strconv.Itoa(int(buf[0])))
		return
	}

	if _, err = readFull(r, buf[:5]); err != nil {
		return
	}
	sig.SigType = SignatureType(buf[0])
	t := binary.BigEndian.Uint32(buf[1:5])
	sig.CreationTime = time.Unix(int64(t), 0)

	if _, err = readFull(r, buf[:8]); err != nil {
		return
	}
	sig.IssuerKeyId = binary.BigEndian.Uint64(buf[:])

	if _, err = readFull(r, buf[:2]); err != nil {
		return
	}
	sig.PubKeyAlgo = PublicKeyAlgorithm(buf[0])
	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly, PubKeyAlgoDSA:
	default:
		err = errors.UnsupportedError("public key algorithm " + strconv.Itoa(int(sig.PubKeyAlgo)))
		return
	}
	var ok bool
	if sig.Hash, ok = s2k.HashIdToHash(buf[1]); !ok {
		return errors.UnsupportedError("hash function " + strconv.Itoa(int(buf[2])))
	}

	if _, err = readFull(r, sig.HashTag[:2]); err != nil {
		return
	}

	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		sig.RSASignature.bytes, sig.RSASignature.bitLength, err = readMPI(r)
	case PubKeyAlgoDSA:
		if sig.DSASigR.bytes, sig.DSASigR.bitLength, err = readMPI(r); err != nil {
			return
		}
		sig.DSASigS.bytes, sig.DSASigS.bitLength, err = readMPI(r)
	default:
		panic("unreachable")
	}
	return
}

func (sig *SignatureV3) Serialize(w io.Writer) (err error) {
	buf := make([]byte, 8)

	buf[0] = byte(sig.SigType)
	binary.BigEndian.PutUint32(buf[1:5], uint32(sig.CreationTime.Unix()))
	if _, err = w.Write(buf[:5]); err != nil {
		return
	}

	binary.BigEndian.PutUint64(buf[:8], sig.IssuerKeyId)
	if _, err = w.Write(buf[:8]); err != nil {
		return
	}

	buf[0] = byte(sig.PubKeyAlgo)
	hashId, ok := s2k.HashToHashId(sig.Hash)
	if !ok {
		return errors.UnsupportedError(fmt.Sprintf("hash function %v", sig.Hash))
	}
	buf[1] = hashId
	copy(buf[2:4], sig.HashTag[:])
	if _, err = w.Write(buf[:4]); err != nil {
		return
	}

	if sig.RSASignature.bytes == nil && sig.DSASigR.bytes == nil {
		return errors.InvalidArgumentError("Signature: need to call Sign, SignUserId or SignKey before Serialize")
	}

	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		err = writeMPIs(w, sig.RSASignature)
	case PubKeyAlgoDSA:
		err = writeMPIs(w, sig.DSASigR, sig.DSASigS)
	default:
		panic("impossible")
	}
	return
}
