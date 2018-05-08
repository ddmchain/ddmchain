
package packet

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math/big"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp/elgamal"
	"golang.org/x/crypto/openpgp/errors"
)

var (

	oidCurveP256 []byte = []byte{0x2A, 0x86, 0x48, 0xCE, 0x3D, 0x03, 0x01, 0x07}

	oidCurveP384 []byte = []byte{0x2B, 0x81, 0x04, 0x00, 0x22}

	oidCurveP521 []byte = []byte{0x2B, 0x81, 0x04, 0x00, 0x23}
)

const maxOIDLength = 8

type ecdsaKey struct {

	oid []byte

	p parsedMPI
}

func parseOID(r io.Reader) (oid []byte, err error) {
	buf := make([]byte, maxOIDLength)
	if _, err = readFull(r, buf[:1]); err != nil {
		return
	}
	oidLen := buf[0]
	if int(oidLen) > len(buf) {
		err = errors.UnsupportedError("invalid oid length: " + strconv.Itoa(int(oidLen)))
		return
	}
	oid = buf[:oidLen]
	_, err = readFull(r, oid)
	return
}

func (f *ecdsaKey) parse(r io.Reader) (err error) {
	if f.oid, err = parseOID(r); err != nil {
		return err
	}
	f.p.bytes, f.p.bitLength, err = readMPI(r)
	return
}

func (f *ecdsaKey) serialize(w io.Writer) (err error) {
	buf := make([]byte, maxOIDLength+1)
	buf[0] = byte(len(f.oid))
	copy(buf[1:], f.oid)
	if _, err = w.Write(buf[:len(f.oid)+1]); err != nil {
		return
	}
	return writeMPIs(w, f.p)
}

func (f *ecdsaKey) newECDSA() (*ecdsa.PublicKey, error) {
	var c elliptic.Curve
	if bytes.Equal(f.oid, oidCurveP256) {
		c = elliptic.P256()
	} else if bytes.Equal(f.oid, oidCurveP384) {
		c = elliptic.P384()
	} else if bytes.Equal(f.oid, oidCurveP521) {
		c = elliptic.P521()
	} else {
		return nil, errors.UnsupportedError(fmt.Sprintf("unsupported oid: %x", f.oid))
	}
	x, y := elliptic.Unmarshal(c, f.p.bytes)
	if x == nil {
		return nil, errors.UnsupportedError("failed to parse EC point")
	}
	return &ecdsa.PublicKey{Curve: c, X: x, Y: y}, nil
}

func (f *ecdsaKey) byteLen() int {
	return 1 + len(f.oid) + 2 + len(f.p.bytes)
}

type kdfHashFunction byte
type kdfAlgorithm byte

type ecdhKdf struct {
	KdfHash kdfHashFunction
	KdfAlgo kdfAlgorithm
}

func (f *ecdhKdf) parse(r io.Reader) (err error) {
	buf := make([]byte, 1)
	if _, err = readFull(r, buf); err != nil {
		return
	}
	kdfLen := int(buf[0])
	if kdfLen < 3 {
		return errors.UnsupportedError("Unsupported ECDH KDF length: " + strconv.Itoa(kdfLen))
	}
	buf = make([]byte, kdfLen)
	if _, err = readFull(r, buf); err != nil {
		return
	}
	reserved := int(buf[0])
	f.KdfHash = kdfHashFunction(buf[1])
	f.KdfAlgo = kdfAlgorithm(buf[2])
	if reserved != 0x01 {
		return errors.UnsupportedError("Unsupported KDF reserved field: " + strconv.Itoa(reserved))
	}
	return
}

func (f *ecdhKdf) serialize(w io.Writer) (err error) {
	buf := make([]byte, 4)

	buf[0] = byte(0x03) 
	buf[1] = byte(0x01) 
	buf[2] = byte(f.KdfHash)
	buf[3] = byte(f.KdfAlgo)
	_, err = w.Write(buf[:])
	return
}

func (f *ecdhKdf) byteLen() int {
	return 4
}

type PublicKey struct {
	CreationTime time.Time
	PubKeyAlgo   PublicKeyAlgorithm
	PublicKey    interface{} 
	Fingerprint  [20]byte
	KeyId        uint64
	IsSubkey     bool

	n, e, p, q, g, y parsedMPI

	ec   *ecdsaKey
	ecdh *ecdhKdf
}

type signingKey interface {
	SerializeSignaturePrefix(io.Writer)
	serializeWithoutHeaders(io.Writer) error
}

func fromBig(n *big.Int) parsedMPI {
	return parsedMPI{
		bytes:     n.Bytes(),
		bitLength: uint16(n.BitLen()),
	}
}

func NewRSAPublicKey(creationTime time.Time, pub *rsa.PublicKey) *PublicKey {
	pk := &PublicKey{
		CreationTime: creationTime,
		PubKeyAlgo:   PubKeyAlgoRSA,
		PublicKey:    pub,
		n:            fromBig(pub.N),
		e:            fromBig(big.NewInt(int64(pub.E))),
	}

	pk.setFingerPrintAndKeyId()
	return pk
}

func NewDSAPublicKey(creationTime time.Time, pub *dsa.PublicKey) *PublicKey {
	pk := &PublicKey{
		CreationTime: creationTime,
		PubKeyAlgo:   PubKeyAlgoDSA,
		PublicKey:    pub,
		p:            fromBig(pub.P),
		q:            fromBig(pub.Q),
		g:            fromBig(pub.G),
		y:            fromBig(pub.Y),
	}

	pk.setFingerPrintAndKeyId()
	return pk
}

func NewElGamalPublicKey(creationTime time.Time, pub *elgamal.PublicKey) *PublicKey {
	pk := &PublicKey{
		CreationTime: creationTime,
		PubKeyAlgo:   PubKeyAlgoElGamal,
		PublicKey:    pub,
		p:            fromBig(pub.P),
		g:            fromBig(pub.G),
		y:            fromBig(pub.Y),
	}

	pk.setFingerPrintAndKeyId()
	return pk
}

func NewECDSAPublicKey(creationTime time.Time, pub *ecdsa.PublicKey) *PublicKey {
	pk := &PublicKey{
		CreationTime: creationTime,
		PubKeyAlgo:   PubKeyAlgoECDSA,
		PublicKey:    pub,
		ec:           new(ecdsaKey),
	}

	switch pub.Curve {
	case elliptic.P256():
		pk.ec.oid = oidCurveP256
	case elliptic.P384():
		pk.ec.oid = oidCurveP384
	case elliptic.P521():
		pk.ec.oid = oidCurveP521
	default:
		panic("unknown elliptic curve")
	}

	pk.ec.p.bytes = elliptic.Marshal(pub.Curve, pub.X, pub.Y)
	pk.ec.p.bitLength = uint16(8 * len(pk.ec.p.bytes))

	pk.setFingerPrintAndKeyId()
	return pk
}

func (pk *PublicKey) parse(r io.Reader) (err error) {

	var buf [6]byte
	_, err = readFull(r, buf[:])
	if err != nil {
		return
	}
	if buf[0] != 4 {
		return errors.UnsupportedError("public key version")
	}
	pk.CreationTime = time.Unix(int64(uint32(buf[1])<<24|uint32(buf[2])<<16|uint32(buf[3])<<8|uint32(buf[4])), 0)
	pk.PubKeyAlgo = PublicKeyAlgorithm(buf[5])
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		err = pk.parseRSA(r)
	case PubKeyAlgoDSA:
		err = pk.parseDSA(r)
	case PubKeyAlgoElGamal:
		err = pk.parseElGamal(r)
	case PubKeyAlgoECDSA:
		pk.ec = new(ecdsaKey)
		if err = pk.ec.parse(r); err != nil {
			return err
		}
		pk.PublicKey, err = pk.ec.newECDSA()
	case PubKeyAlgoECDH:
		pk.ec = new(ecdsaKey)
		if err = pk.ec.parse(r); err != nil {
			return
		}
		pk.ecdh = new(ecdhKdf)
		if err = pk.ecdh.parse(r); err != nil {
			return
		}

		pk.PublicKey, err = pk.ec.newECDSA()
	default:
		err = errors.UnsupportedError("public key type: " + strconv.Itoa(int(pk.PubKeyAlgo)))
	}
	if err != nil {
		return
	}

	pk.setFingerPrintAndKeyId()
	return
}

func (pk *PublicKey) setFingerPrintAndKeyId() {

	fingerPrint := sha1.New()
	pk.SerializeSignaturePrefix(fingerPrint)
	pk.serializeWithoutHeaders(fingerPrint)
	copy(pk.Fingerprint[:], fingerPrint.Sum(nil))
	pk.KeyId = binary.BigEndian.Uint64(pk.Fingerprint[12:20])
}

func (pk *PublicKey) parseRSA(r io.Reader) (err error) {
	pk.n.bytes, pk.n.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.e.bytes, pk.e.bitLength, err = readMPI(r)
	if err != nil {
		return
	}

	if len(pk.e.bytes) > 3 {
		err = errors.UnsupportedError("large public exponent")
		return
	}
	rsa := &rsa.PublicKey{
		N: new(big.Int).SetBytes(pk.n.bytes),
		E: 0,
	}
	for i := 0; i < len(pk.e.bytes); i++ {
		rsa.E <<= 8
		rsa.E |= int(pk.e.bytes[i])
	}
	pk.PublicKey = rsa
	return
}

func (pk *PublicKey) parseDSA(r io.Reader) (err error) {
	pk.p.bytes, pk.p.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.q.bytes, pk.q.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.g.bytes, pk.g.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.y.bytes, pk.y.bitLength, err = readMPI(r)
	if err != nil {
		return
	}

	dsa := new(dsa.PublicKey)
	dsa.P = new(big.Int).SetBytes(pk.p.bytes)
	dsa.Q = new(big.Int).SetBytes(pk.q.bytes)
	dsa.G = new(big.Int).SetBytes(pk.g.bytes)
	dsa.Y = new(big.Int).SetBytes(pk.y.bytes)
	pk.PublicKey = dsa
	return
}

func (pk *PublicKey) parseElGamal(r io.Reader) (err error) {
	pk.p.bytes, pk.p.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.g.bytes, pk.g.bitLength, err = readMPI(r)
	if err != nil {
		return
	}
	pk.y.bytes, pk.y.bitLength, err = readMPI(r)
	if err != nil {
		return
	}

	elgamal := new(elgamal.PublicKey)
	elgamal.P = new(big.Int).SetBytes(pk.p.bytes)
	elgamal.G = new(big.Int).SetBytes(pk.g.bytes)
	elgamal.Y = new(big.Int).SetBytes(pk.y.bytes)
	pk.PublicKey = elgamal
	return
}

func (pk *PublicKey) SerializeSignaturePrefix(h io.Writer) {
	var pLength uint16
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		pLength += 2 + uint16(len(pk.n.bytes))
		pLength += 2 + uint16(len(pk.e.bytes))
	case PubKeyAlgoDSA:
		pLength += 2 + uint16(len(pk.p.bytes))
		pLength += 2 + uint16(len(pk.q.bytes))
		pLength += 2 + uint16(len(pk.g.bytes))
		pLength += 2 + uint16(len(pk.y.bytes))
	case PubKeyAlgoElGamal:
		pLength += 2 + uint16(len(pk.p.bytes))
		pLength += 2 + uint16(len(pk.g.bytes))
		pLength += 2 + uint16(len(pk.y.bytes))
	case PubKeyAlgoECDSA:
		pLength += uint16(pk.ec.byteLen())
	case PubKeyAlgoECDH:
		pLength += uint16(pk.ec.byteLen())
		pLength += uint16(pk.ecdh.byteLen())
	default:
		panic("unknown public key algorithm")
	}
	pLength += 6
	h.Write([]byte{0x99, byte(pLength >> 8), byte(pLength)})
	return
}

func (pk *PublicKey) Serialize(w io.Writer) (err error) {
	length := 6 

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		length += 2 + len(pk.n.bytes)
		length += 2 + len(pk.e.bytes)
	case PubKeyAlgoDSA:
		length += 2 + len(pk.p.bytes)
		length += 2 + len(pk.q.bytes)
		length += 2 + len(pk.g.bytes)
		length += 2 + len(pk.y.bytes)
	case PubKeyAlgoElGamal:
		length += 2 + len(pk.p.bytes)
		length += 2 + len(pk.g.bytes)
		length += 2 + len(pk.y.bytes)
	case PubKeyAlgoECDSA:
		length += pk.ec.byteLen()
	case PubKeyAlgoECDH:
		length += pk.ec.byteLen()
		length += pk.ecdh.byteLen()
	default:
		panic("unknown public key algorithm")
	}

	packetType := packetTypePublicKey
	if pk.IsSubkey {
		packetType = packetTypePublicSubkey
	}
	err = serializeHeader(w, packetType, length)
	if err != nil {
		return
	}
	return pk.serializeWithoutHeaders(w)
}

func (pk *PublicKey) serializeWithoutHeaders(w io.Writer) (err error) {
	var buf [6]byte
	buf[0] = 4
	t := uint32(pk.CreationTime.Unix())
	buf[1] = byte(t >> 24)
	buf[2] = byte(t >> 16)
	buf[3] = byte(t >> 8)
	buf[4] = byte(t)
	buf[5] = byte(pk.PubKeyAlgo)

	_, err = w.Write(buf[:])
	if err != nil {
		return
	}

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		return writeMPIs(w, pk.n, pk.e)
	case PubKeyAlgoDSA:
		return writeMPIs(w, pk.p, pk.q, pk.g, pk.y)
	case PubKeyAlgoElGamal:
		return writeMPIs(w, pk.p, pk.g, pk.y)
	case PubKeyAlgoECDSA:
		return pk.ec.serialize(w)
	case PubKeyAlgoECDH:
		if err = pk.ec.serialize(w); err != nil {
			return
		}
		return pk.ecdh.serialize(w)
	}
	return errors.InvalidArgumentError("bad public-key algorithm")
}

func (pk *PublicKey) CanSign() bool {
	return pk.PubKeyAlgo != PubKeyAlgoRSAEncryptOnly && pk.PubKeyAlgo != PubKeyAlgoElGamal
}

func (pk *PublicKey) VerifySignature(signed hash.Hash, sig *Signature) (err error) {
	if !pk.CanSign() {
		return errors.InvalidArgumentError("public key cannot generate signatures")
	}

	signed.Write(sig.HashSuffix)
	hashBytes := signed.Sum(nil)

	if hashBytes[0] != sig.HashTag[0] || hashBytes[1] != sig.HashTag[1] {
		return errors.SignatureError("hash tag doesn't match")
	}

	if pk.PubKeyAlgo != sig.PubKeyAlgo {
		return errors.InvalidArgumentError("public key and signature use different algorithms")
	}

	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		rsaPublicKey, _ := pk.PublicKey.(*rsa.PublicKey)
		err = rsa.VerifyPKCS1v15(rsaPublicKey, sig.Hash, hashBytes, sig.RSASignature.bytes)
		if err != nil {
			return errors.SignatureError("RSA verification failure")
		}
		return nil
	case PubKeyAlgoDSA:
		dsaPublicKey, _ := pk.PublicKey.(*dsa.PublicKey)

		subgroupSize := (dsaPublicKey.Q.BitLen() + 7) / 8
		if len(hashBytes) > subgroupSize {
			hashBytes = hashBytes[:subgroupSize]
		}
		if !dsa.Verify(dsaPublicKey, hashBytes, new(big.Int).SetBytes(sig.DSASigR.bytes), new(big.Int).SetBytes(sig.DSASigS.bytes)) {
			return errors.SignatureError("DSA verification failure")
		}
		return nil
	case PubKeyAlgoECDSA:
		ecdsaPublicKey := pk.PublicKey.(*ecdsa.PublicKey)
		if !ecdsa.Verify(ecdsaPublicKey, hashBytes, new(big.Int).SetBytes(sig.ECDSASigR.bytes), new(big.Int).SetBytes(sig.ECDSASigS.bytes)) {
			return errors.SignatureError("ECDSA verification failure")
		}
		return nil
	default:
		return errors.SignatureError("Unsupported public key algorithm used in signature")
	}
}

func (pk *PublicKey) VerifySignatureV3(signed hash.Hash, sig *SignatureV3) (err error) {
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
		rsaPublicKey := pk.PublicKey.(*rsa.PublicKey)
		if err = rsa.VerifyPKCS1v15(rsaPublicKey, sig.Hash, hashBytes, sig.RSASignature.bytes); err != nil {
			return errors.SignatureError("RSA verification failure")
		}
		return
	case PubKeyAlgoDSA:
		dsaPublicKey := pk.PublicKey.(*dsa.PublicKey)

		subgroupSize := (dsaPublicKey.Q.BitLen() + 7) / 8
		if len(hashBytes) > subgroupSize {
			hashBytes = hashBytes[:subgroupSize]
		}
		if !dsa.Verify(dsaPublicKey, hashBytes, new(big.Int).SetBytes(sig.DSASigR.bytes), new(big.Int).SetBytes(sig.DSASigS.bytes)) {
			return errors.SignatureError("DSA verification failure")
		}
		return nil
	default:
		panic("shouldn't happen")
	}
}

func keySignatureHash(pk, signed signingKey, hashFunc crypto.Hash) (h hash.Hash, err error) {
	if !hashFunc.Available() {
		return nil, errors.UnsupportedError("hash function")
	}
	h = hashFunc.New()

	pk.SerializeSignaturePrefix(h)
	pk.serializeWithoutHeaders(h)
	signed.SerializeSignaturePrefix(h)
	signed.serializeWithoutHeaders(h)
	return
}

func (pk *PublicKey) VerifyKeySignature(signed *PublicKey, sig *Signature) error {
	h, err := keySignatureHash(pk, signed, sig.Hash)
	if err != nil {
		return err
	}
	if err = pk.VerifySignature(h, sig); err != nil {
		return err
	}

	if sig.FlagSign {

		if sig.EmbeddedSignature == nil {
			return errors.StructuralError("signing subkey is missing cross-signature")
		}

		if h, err = keySignatureHash(pk, signed, sig.EmbeddedSignature.Hash); err != nil {
			return errors.StructuralError("error while hashing for cross-signature: " + err.Error())
		}
		if err := signed.VerifySignature(h, sig.EmbeddedSignature); err != nil {
			return errors.StructuralError("error while verifying cross-signature: " + err.Error())
		}
	}

	return nil
}

func keyRevocationHash(pk signingKey, hashFunc crypto.Hash) (h hash.Hash, err error) {
	if !hashFunc.Available() {
		return nil, errors.UnsupportedError("hash function")
	}
	h = hashFunc.New()

	pk.SerializeSignaturePrefix(h)
	pk.serializeWithoutHeaders(h)

	return
}

func (pk *PublicKey) VerifyRevocationSignature(sig *Signature) (err error) {
	h, err := keyRevocationHash(pk, sig.Hash)
	if err != nil {
		return err
	}
	return pk.VerifySignature(h, sig)
}

func userIdSignatureHash(id string, pk *PublicKey, hashFunc crypto.Hash) (h hash.Hash, err error) {
	if !hashFunc.Available() {
		return nil, errors.UnsupportedError("hash function")
	}
	h = hashFunc.New()

	pk.SerializeSignaturePrefix(h)
	pk.serializeWithoutHeaders(h)

	var buf [5]byte
	buf[0] = 0xb4
	buf[1] = byte(len(id) >> 24)
	buf[2] = byte(len(id) >> 16)
	buf[3] = byte(len(id) >> 8)
	buf[4] = byte(len(id))
	h.Write(buf[:])
	h.Write([]byte(id))

	return
}

func (pk *PublicKey) VerifyUserIdSignature(id string, pub *PublicKey, sig *Signature) (err error) {
	h, err := userIdSignatureHash(id, pub, sig.Hash)
	if err != nil {
		return err
	}
	return pk.VerifySignature(h, sig)
}

func (pk *PublicKey) VerifyUserIdSignatureV3(id string, pub *PublicKey, sig *SignatureV3) (err error) {
	h, err := userIdSignatureV3Hash(id, pub, sig.Hash)
	if err != nil {
		return err
	}
	return pk.VerifySignatureV3(h, sig)
}

func (pk *PublicKey) KeyIdString() string {
	return fmt.Sprintf("%X", pk.Fingerprint[12:20])
}

func (pk *PublicKey) KeyIdShortString() string {
	return fmt.Sprintf("%X", pk.Fingerprint[16:20])
}

type parsedMPI struct {
	bytes     []byte
	bitLength uint16
}

func writeMPIs(w io.Writer, mpis ...parsedMPI) (err error) {
	for _, mpi := range mpis {
		err = writeMPI(w, mpi.bitLength, mpi.bytes)
		if err != nil {
			return
		}
	}
	return
}

func (pk *PublicKey) BitLength() (bitLength uint16, err error) {
	switch pk.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoRSASignOnly:
		bitLength = pk.n.bitLength
	case PubKeyAlgoDSA:
		bitLength = pk.p.bitLength
	case PubKeyAlgoElGamal:
		bitLength = pk.p.bitLength
	default:
		err = errors.InvalidArgumentError("bad public-key algorithm")
	}
	return
}
