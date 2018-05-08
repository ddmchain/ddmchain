
package packet

import (
	"crypto"
	"crypto/md5"
	"crypto/rsa"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math/big"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp/errors"
)

type PublicKeyV3 struct {
	CreationTime time.Time
	DaysToExpire uint16
	PubKeyAlgo   PublicKeyAlgorithm
	PublicKey    *rsa.PublicKey
	Fingerprint  [16]byte
	KeyId        uint64
	IsSubkey     bool

	n, e parsedMPI
}

func newRSAPublicKeyV3(creationTime time.Time, pub *rsa.PublicKey) *PublicKeyV3 {
	pk := &PublicKeyV3{
		CreationTime: creationTime,
		PublicKey:    pub,
		n:            fromBig(pub.N),
		e:            fromBig(big.NewInt(int64(pub.E))),
	}

	pk.setFingerPrintAndKeyId()
	return pk
}

func (pk *PublicKeyV3) parse(r io.Reader) (err error) {

	var buf [8]byte
	if _, err = readFull(r, buf[:]); err != nil {
		return
	}
	if buf[0] < 2 || buf[0] > 3 {
		return errors.UnsupportedError("public key version")
	}
	pk.CreationTime = time.Unix(int64(uint32(buf[1])<<24|uint32(buf[2])<<16|uint32(buf[3])<<8|uint32(buf[4])), 0)
	pk.DaysToExpire = binary.BigEndian.Uint16(buf[5:7])
	pk.PubKeyAlgo = PublicKeyAlgorithm(buf[7])
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		err = pk.parseRSA(r)
	default:
		err = errors.UnsupportedError("public key type: " + strconv.Itoa(int(pk.PubKeyAlgo)))
	}
	if err != nil {
		return
	}

	pk.setFingerPrintAndKeyId()
	return
}

func (pk *PublicKeyV3) setFingerPrintAndKeyId() {

	fingerPrint := md5.New()
	fingerPrint.Write(pk.n.bytes)
	fingerPrint.Write(pk.e.bytes)
	fingerPrint.Sum(pk.Fingerprint[:0])
	pk.KeyId = binary.BigEndian.Uint64(pk.n.bytes[len(pk.n.bytes)-8:])
}

func (pk *PublicKeyV3) parseRSA(r io.Reader) (err error) {
	if pk.n.bytes, pk.n.bitLength, err = readMPI(r); err != nil {
		return
	}
	if pk.e.bytes, pk.e.bitLength, err = readMPI(r); err != nil {
		return
	}

	if len(pk.n.bytes) < 8 {
		return errors.StructuralError("v3 public key modulus is too short")
	}
	if len(pk.e.bytes) > 3 {
		err = errors.UnsupportedError("large public exponent")
		return
	}
	rsa := &rsa.PublicKey{N: new(big.Int).SetBytes(pk.n.bytes)}
	for i := 0; i < len(pk.e.bytes); i++ {
		rsa.E <<= 8
		rsa.E |= int(pk.e.bytes[i])
	}
	pk.PublicKey = rsa
	return
}

func (pk *PublicKeyV3) SerializeSignaturePrefix(w io.Writer) {
	var pLength uint16
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		pLength += 2 + uint16(len(pk.n.bytes))
		pLength += 2 + uint16(len(pk.e.bytes))
	default:
		panic("unknown public key algorithm")
	}
	pLength += 6
	w.Write([]byte{0x99, byte(pLength >> 8), byte(pLength)})
	return
}

func (pk *PublicKeyV3) Serialize(w io.Writer) (err error) {
	length := 8 

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		length += 2 + len(pk.n.bytes)
		length += 2 + len(pk.e.bytes)
	default:
		panic("unknown public key algorithm")
	}

	packetType := packetTypePublicKey
	if pk.IsSubkey {
		packetType = packetTypePublicSubkey
	}
	if err = serializeHeader(w, packetType, length); err != nil {
		return
	}
	return pk.serializeWithoutHeaders(w)
}

func (pk *PublicKeyV3) serializeWithoutHeaders(w io.Writer) (err error) {
	var buf [8]byte

	buf[0] = 3

	t := uint32(pk.CreationTime.Unix())
	buf[1] = byte(t >> 24)
	buf[2] = byte(t >> 16)
	buf[3] = byte(t >> 8)
	buf[4] = byte(t)

	buf[5] = byte(pk.DaysToExpire >> 8)
	buf[6] = byte(pk.DaysToExpire)

	buf[7] = byte(pk.PubKeyAlgo)

	if _, err = w.Write(buf[:]); err != nil {
		return
	}

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		return writeMPIs(w, pk.n, pk.e)
	}
	return errors.InvalidArgumentError("bad public-key algorithm")
}

func (pk *PublicKeyV3) CanSign() bool {
	return pk.PubKeyAlgo != PubKeyAlgoRSAEncryptOnly
}

func (pk *PublicKeyV3) VerifySignatureV3(signed hash.Hash, sig *SignatureV3) (err error) {
	if !pk.CanSign() {
		return errors.InvalidArgumentError("public key cannot generate signatures")
	}

	suffix := make([]byte, 5)
	suffix[0] = byte(sig.SigType)
	binary.BigEndian.PutUint32(suffix[1:], uint32(sig.CreationTime.Unix()))
	signed.Write(suffix)
	hashBytes := signed.Sum(nil)

	if hashBytes[0] != sig.HashTag[0] || hashBytes[1] != sig.HashTag[1] {
		return errors.SignatureError("hash tag doesn't match")
	}

	if pk.PubKeyAlgo != sig.PubKeyAlgo {
		return errors.InvalidArgumentError("public key and signature use different algorithms")
	}

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		if err = rsa.VerifyPKCS1v15(pk.PublicKey, sig.Hash, hashBytes, sig.RSASignature.bytes); err != nil {
			return errors.SignatureError("RSA verification failure")
		}
		return
	default:

		panic("shouldn't happen")
	}
}

func (pk *PublicKeyV3) VerifyUserIdSignatureV3(id string, pub *PublicKeyV3, sig *SignatureV3) (err error) {
	h, err := userIdSignatureV3Hash(id, pk, sig.Hash)
	if err != nil {
		return err
	}
	return pk.VerifySignatureV3(h, sig)
}

func (pk *PublicKeyV3) VerifyKeySignatureV3(signed *PublicKeyV3, sig *SignatureV3) (err error) {
	h, err := keySignatureHash(pk, signed, sig.Hash)
	if err != nil {
		return err
	}
	return pk.VerifySignatureV3(h, sig)
}

func userIdSignatureV3Hash(id string, pk signingKey, hfn crypto.Hash) (h hash.Hash, err error) {
	if !hfn.Available() {
		return nil, errors.UnsupportedError("hash function")
	}
	h = hfn.New()

	pk.SerializeSignaturePrefix(h)
	pk.serializeWithoutHeaders(h)

	h.Write([]byte(id))

	return
}

func (pk *PublicKeyV3) KeyIdString() string {
	return fmt.Sprintf("%X", pk.KeyId)
}

func (pk *PublicKeyV3) KeyIdShortString() string {
	return fmt.Sprintf("%X", pk.KeyId&0xFFFFFFFF)
}

func (pk *PublicKeyV3) BitLength() (bitLength uint16, err error) {
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		bitLength = pk.n.bitLength
	default:
		err = errors.InvalidArgumentError("bad public-key algorithm")
	}
	return
}
