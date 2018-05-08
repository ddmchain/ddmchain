
package packet

import (
	"bytes"
	"crypto"
	"crypto/dsa"
	"crypto/ecdsa"
	"encoding/asn1"
	"encoding/binary"
	"hash"
	"io"
	"math/big"
	"strconv"
	"time"

	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/s2k"
)

const (

	KeyFlagCertify = 1 << iota
	KeyFlagSign
	KeyFlagEncryptCommunications
	KeyFlagEncryptStorage
)

type Signature struct {
	SigType    SignatureType
	PubKeyAlgo PublicKeyAlgorithm
	Hash       crypto.Hash

	HashSuffix []byte

	HashTag      [2]byte
	CreationTime time.Time

	RSASignature         parsedMPI
	DSASigR, DSASigS     parsedMPI
	ECDSASigR, ECDSASigS parsedMPI

	rawSubpackets []outputSubpacket

	SigLifetimeSecs, KeyLifetimeSecs                        *uint32
	PreferredSymmetric, PreferredHash, PreferredCompression []uint8
	IssuerKeyId                                             *uint64
	IsPrimaryId                                             *bool

	FlagsValid                                                           bool
	FlagCertify, FlagSign, FlagEncryptCommunications, FlagEncryptStorage bool

	RevocationReason     *uint8
	RevocationReasonText string

	MDC bool

	EmbeddedSignature *Signature

	outSubpackets []outputSubpacket
}

func (sig *Signature) parse(r io.Reader) (err error) {

	var buf [5]byte
	_, err = readFull(r, buf[:1])
	if err != nil {
		return
	}
	if buf[0] != 4 {
		err = errors.UnsupportedError("signature packet version " + strconv.Itoa(int(buf[0])))
		return
	}

	_, err = readFull(r, buf[:5])
	if err != nil {
		return
	}
	sig.SigType = SignatureType(buf[0])
	sig.PubKeyAlgo = PublicKeyAlgorithm(buf[1])
	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly, PubKeyAlgoDSA, PubKeyAlgoECDSA:
	default:
		err = errors.UnsupportedError("public key algorithm " + strconv.Itoa(int(sig.PubKeyAlgo)))
		return
	}

	var ok bool
	sig.Hash, ok = s2k.HashIdToHash(buf[2])
	if !ok {
		return errors.UnsupportedError("hash function " + strconv.Itoa(int(buf[2])))
	}

	hashedSubpacketsLength := int(buf[3])<<8 | int(buf[4])
	l := 6 + hashedSubpacketsLength
	sig.HashSuffix = make([]byte, l+6)
	sig.HashSuffix[0] = 4
	copy(sig.HashSuffix[1:], buf[:5])
	hashedSubpackets := sig.HashSuffix[6:l]
	_, err = readFull(r, hashedSubpackets)
	if err != nil {
		return
	}

	trailer := sig.HashSuffix[l:]
	trailer[0] = 4
	trailer[1] = 0xff
	trailer[2] = uint8(l >> 24)
	trailer[3] = uint8(l >> 16)
	trailer[4] = uint8(l >> 8)
	trailer[5] = uint8(l)

	err = parseSignatureSubpackets(sig, hashedSubpackets, true)
	if err != nil {
		return
	}

	_, err = readFull(r, buf[:2])
	if err != nil {
		return
	}
	unhashedSubpacketsLength := int(buf[0])<<8 | int(buf[1])
	unhashedSubpackets := make([]byte, unhashedSubpacketsLength)
	_, err = readFull(r, unhashedSubpackets)
	if err != nil {
		return
	}
	err = parseSignatureSubpackets(sig, unhashedSubpackets, false)
	if err != nil {
		return
	}

	_, err = readFull(r, sig.HashTag[:2])
	if err != nil {
		return
	}

	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		sig.RSASignature.bytes, sig.RSASignature.bitLength, err = readMPI(r)
	case PubKeyAlgoDSA:
		sig.DSASigR.bytes, sig.DSASigR.bitLength, err = readMPI(r)
		if err == nil {
			sig.DSASigS.bytes, sig.DSASigS.bitLength, err = readMPI(r)
		}
	case PubKeyAlgoECDSA:
		sig.ECDSASigR.bytes, sig.ECDSASigR.bitLength, err = readMPI(r)
		if err == nil {
			sig.ECDSASigS.bytes, sig.ECDSASigS.bitLength, err = readMPI(r)
		}
	default:
		panic("unreachable")
	}
	return
}

func parseSignatureSubpackets(sig *Signature, subpackets []byte, isHashed bool) (err error) {
	for len(subpackets) > 0 {
		subpackets, err = parseSignatureSubpacket(sig, subpackets, isHashed)
		if err != nil {
			return
		}
	}

	if sig.CreationTime.IsZero() {
		err = errors.StructuralError("no creation time in signature")
	}

	return
}

type signatureSubpacketType uint8

const (
	creationTimeSubpacket        signatureSubpacketType = 2
	signatureExpirationSubpacket signatureSubpacketType = 3
	keyExpirationSubpacket       signatureSubpacketType = 9
	prefSymmetricAlgosSubpacket  signatureSubpacketType = 11
	issuerSubpacket              signatureSubpacketType = 16
	prefHashAlgosSubpacket       signatureSubpacketType = 21
	prefCompressionSubpacket     signatureSubpacketType = 22
	primaryUserIdSubpacket       signatureSubpacketType = 25
	keyFlagsSubpacket            signatureSubpacketType = 27
	reasonForRevocationSubpacket signatureSubpacketType = 29
	featuresSubpacket            signatureSubpacketType = 30
	embeddedSignatureSubpacket   signatureSubpacketType = 32
)

func parseSignatureSubpacket(sig *Signature, subpacket []byte, isHashed bool) (rest []byte, err error) {

	var (
		length     uint32
		packetType signatureSubpacketType
		isCritical bool
	)
	switch {
	case subpacket[0] < 192:
		length = uint32(subpacket[0])
		subpacket = subpacket[1:]
	case subpacket[0] < 255:
		if len(subpacket) < 2 {
			goto Truncated
		}
		length = uint32(subpacket[0]-192)<<8 + uint32(subpacket[1]) + 192
		subpacket = subpacket[2:]
	default:
		if len(subpacket) < 5 {
			goto Truncated
		}
		length = uint32(subpacket[1])<<24 |
			uint32(subpacket[2])<<16 |
			uint32(subpacket[3])<<8 |
			uint32(subpacket[4])
		subpacket = subpacket[5:]
	}
	if length > uint32(len(subpacket)) {
		goto Truncated
	}
	rest = subpacket[length:]
	subpacket = subpacket[:length]
	if len(subpacket) == 0 {
		err = errors.StructuralError("zero length signature subpacket")
		return
	}
	packetType = signatureSubpacketType(subpacket[0] & 0x7f)
	isCritical = subpacket[0]&0x80 == 0x80
	subpacket = subpacket[1:]
	sig.rawSubpackets = append(sig.rawSubpackets, outputSubpacket{isHashed, packetType, isCritical, subpacket})
	switch packetType {
	case creationTimeSubpacket:
		if !isHashed {
			err = errors.StructuralError("signature creation time in non-hashed area")
			return
		}
		if len(subpacket) != 4 {
			err = errors.StructuralError("signature creation time not four bytes")
			return
		}
		t := binary.BigEndian.Uint32(subpacket)
		sig.CreationTime = time.Unix(int64(t), 0)
	case signatureExpirationSubpacket:

		if !isHashed {
			return
		}
		if len(subpacket) != 4 {
			err = errors.StructuralError("expiration subpacket with bad length")
			return
		}
		sig.SigLifetimeSecs = new(uint32)
		*sig.SigLifetimeSecs = binary.BigEndian.Uint32(subpacket)
	case keyExpirationSubpacket:

		if !isHashed {
			return
		}
		if len(subpacket) != 4 {
			err = errors.StructuralError("key expiration subpacket with bad length")
			return
		}
		sig.KeyLifetimeSecs = new(uint32)
		*sig.KeyLifetimeSecs = binary.BigEndian.Uint32(subpacket)
	case prefSymmetricAlgosSubpacket:

		if !isHashed {
			return
		}
		sig.PreferredSymmetric = make([]byte, len(subpacket))
		copy(sig.PreferredSymmetric, subpacket)
	case issuerSubpacket:

		if len(subpacket) != 8 {
			err = errors.StructuralError("issuer subpacket with bad length")
			return
		}
		sig.IssuerKeyId = new(uint64)
		*sig.IssuerKeyId = binary.BigEndian.Uint64(subpacket)
	case prefHashAlgosSubpacket:

		if !isHashed {
			return
		}
		sig.PreferredHash = make([]byte, len(subpacket))
		copy(sig.PreferredHash, subpacket)
	case prefCompressionSubpacket:

		if !isHashed {
			return
		}
		sig.PreferredCompression = make([]byte, len(subpacket))
		copy(sig.PreferredCompression, subpacket)
	case primaryUserIdSubpacket:

		if !isHashed {
			return
		}
		if len(subpacket) != 1 {
			err = errors.StructuralError("primary user id subpacket with bad length")
			return
		}
		sig.IsPrimaryId = new(bool)
		if subpacket[0] > 0 {
			*sig.IsPrimaryId = true
		}
	case keyFlagsSubpacket:

		if !isHashed {
			return
		}
		if len(subpacket) == 0 {
			err = errors.StructuralError("empty key flags subpacket")
			return
		}
		sig.FlagsValid = true
		if subpacket[0]&KeyFlagCertify != 0 {
			sig.FlagCertify = true
		}
		if subpacket[0]&KeyFlagSign != 0 {
			sig.FlagSign = true
		}
		if subpacket[0]&KeyFlagEncryptCommunications != 0 {
			sig.FlagEncryptCommunications = true
		}
		if subpacket[0]&KeyFlagEncryptStorage != 0 {
			sig.FlagEncryptStorage = true
		}
	case reasonForRevocationSubpacket:

		if !isHashed {
			return
		}
		if len(subpacket) == 0 {
			err = errors.StructuralError("empty revocation reason subpacket")
			return
		}
		sig.RevocationReason = new(uint8)
		*sig.RevocationReason = subpacket[0]
		sig.RevocationReasonText = string(subpacket[1:])
	case featuresSubpacket:

		sig.MDC = len(subpacket) >= 1 && subpacket[0]&1 == 1
	case embeddedSignatureSubpacket:

		if sig.EmbeddedSignature != nil {
			err = errors.StructuralError("Cannot have multiple embedded signatures")
			return
		}
		sig.EmbeddedSignature = new(Signature)

		if err := sig.EmbeddedSignature.parse(bytes.NewBuffer(subpacket)); err != nil {
			return nil, err
		}
		if sigType := sig.EmbeddedSignature.SigType; sigType != SigTypePrimaryKeyBinding {
			return nil, errors.StructuralError("cross-signature has unexpected type " + strconv.Itoa(int(sigType)))
		}
	default:
		if isCritical {
			err = errors.UnsupportedError("unknown critical signature subpacket type " + strconv.Itoa(int(packetType)))
			return
		}
	}
	return

Truncated:
	err = errors.StructuralError("signature subpacket truncated")
	return
}

func subpacketLengthLength(length int) int {
	if length < 192 {
		return 1
	}
	if length < 16320 {
		return 2
	}
	return 5
}

func serializeSubpacketLength(to []byte, length int) int {

	if length < 192 {
		to[0] = byte(length)
		return 1
	}
	if length < 16320 {
		length -= 192
		to[0] = byte((length >> 8) + 192)
		to[1] = byte(length)
		return 2
	}
	to[0] = 255
	to[1] = byte(length >> 24)
	to[2] = byte(length >> 16)
	to[3] = byte(length >> 8)
	to[4] = byte(length)
	return 5
}

func subpacketsLength(subpackets []outputSubpacket, hashed bool) (length int) {
	for _, subpacket := range subpackets {
		if subpacket.hashed == hashed {
			length += subpacketLengthLength(len(subpacket.contents) + 1)
			length += 1 
			length += len(subpacket.contents)
		}
	}
	return
}

func serializeSubpackets(to []byte, subpackets []outputSubpacket, hashed bool) {
	for _, subpacket := range subpackets {
		if subpacket.hashed == hashed {
			n := serializeSubpacketLength(to, len(subpacket.contents)+1)
			to[n] = byte(subpacket.subpacketType)
			to = to[1+n:]
			n = copy(to, subpacket.contents)
			to = to[n:]
		}
	}
	return
}

func (sig *Signature) KeyExpired(currentTime time.Time) bool {
	if sig.KeyLifetimeSecs == nil {
		return false
	}
	expiry := sig.CreationTime.Add(time.Duration(*sig.KeyLifetimeSecs) * time.Second)
	return currentTime.After(expiry)
}

func (sig *Signature) buildHashSuffix() (err error) {
	hashedSubpacketsLen := subpacketsLength(sig.outSubpackets, true)

	var ok bool
	l := 6 + hashedSubpacketsLen
	sig.HashSuffix = make([]byte, l+6)
	sig.HashSuffix[0] = 4
	sig.HashSuffix[1] = uint8(sig.SigType)
	sig.HashSuffix[2] = uint8(sig.PubKeyAlgo)
	sig.HashSuffix[3], ok = s2k.HashToHashId(sig.Hash)
	if !ok {
		sig.HashSuffix = nil
		return errors.InvalidArgumentError("hash cannot be represented in OpenPGP: " + strconv.Itoa(int(sig.Hash)))
	}
	sig.HashSuffix[4] = byte(hashedSubpacketsLen >> 8)
	sig.HashSuffix[5] = byte(hashedSubpacketsLen)
	serializeSubpackets(sig.HashSuffix[6:l], sig.outSubpackets, true)
	trailer := sig.HashSuffix[l:]
	trailer[0] = 4
	trailer[1] = 0xff
	trailer[2] = byte(l >> 24)
	trailer[3] = byte(l >> 16)
	trailer[4] = byte(l >> 8)
	trailer[5] = byte(l)
	return
}

func (sig *Signature) signPrepareHash(h hash.Hash) (digest []byte, err error) {
	err = sig.buildHashSuffix()
	if err != nil {
		return
	}

	h.Write(sig.HashSuffix)
	digest = h.Sum(nil)
	copy(sig.HashTag[:], digest)
	return
}

func (sig *Signature) Sign(h hash.Hash, priv *PrivateKey, config *Config) (err error) {
	sig.outSubpackets = sig.buildSubpackets()
	digest, err := sig.signPrepareHash(h)
	if err != nil {
		return
	}

	switch priv.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:

		sig.RSASignature.bytes, err = priv.PrivateKey.(crypto.Signer).Sign(config.Random(), digest, sig.Hash)
		sig.RSASignature.bitLength = uint16(8 * len(sig.RSASignature.bytes))
	case PubKeyAlgoDSA:
		dsaPriv := priv.PrivateKey.(*dsa.PrivateKey)

		subgroupSize := (dsaPriv.Q.BitLen() + 7) / 8
		if len(digest) > subgroupSize {
			digest = digest[:subgroupSize]
		}
		r, s, err := dsa.Sign(config.Random(), dsaPriv, digest)
		if err == nil {
			sig.DSASigR.bytes = r.Bytes()
			sig.DSASigR.bitLength = uint16(8 * len(sig.DSASigR.bytes))
			sig.DSASigS.bytes = s.Bytes()
			sig.DSASigS.bitLength = uint16(8 * len(sig.DSASigS.bytes))
		}
	case PubKeyAlgoECDSA:
		var r, s *big.Int
		if pk, ok := priv.PrivateKey.(*ecdsa.PrivateKey); ok {

			r, s, err = ecdsa.Sign(config.Random(), pk, digest)
		} else {
			var b []byte
			b, err = priv.PrivateKey.(crypto.Signer).Sign(config.Random(), digest, nil)
			if err == nil {
				r, s, err = unwrapECDSASig(b)
			}
		}
		if err == nil {
			sig.ECDSASigR = fromBig(r)
			sig.ECDSASigS = fromBig(s)
		}
	default:
		err = errors.UnsupportedError("public key algorithm: " + strconv.Itoa(int(sig.PubKeyAlgo)))
	}

	return
}

func unwrapECDSASig(b []byte) (r, s *big.Int, err error) {
	var ecsdaSig struct {
		R, S *big.Int
	}
	_, err = asn1.Unmarshal(b, &ecsdaSig)
	if err != nil {
		return
	}
	return ecsdaSig.R, ecsdaSig.S, nil
}

func (sig *Signature) SignUserId(id string, pub *PublicKey, priv *PrivateKey, config *Config) error {
	h, err := userIdSignatureHash(id, pub, sig.Hash)
	if err != nil {
		return err
	}
	return sig.Sign(h, priv, config)
}

func (sig *Signature) SignKey(pub *PublicKey, priv *PrivateKey, config *Config) error {
	h, err := keySignatureHash(&priv.PublicKey, pub, sig.Hash)
	if err != nil {
		return err
	}
	return sig.Sign(h, priv, config)
}

func (sig *Signature) Serialize(w io.Writer) (err error) {
	if len(sig.outSubpackets) == 0 {
		sig.outSubpackets = sig.rawSubpackets
	}
	if sig.RSASignature.bytes == nil && sig.DSASigR.bytes == nil && sig.ECDSASigR.bytes == nil {
		return errors.InvalidArgumentError("Signature: need to call Sign, SignUserId or SignKey before Serialize")
	}

	sigLength := 0
	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		sigLength = 2 + len(sig.RSASignature.bytes)
	case PubKeyAlgoDSA:
		sigLength = 2 + len(sig.DSASigR.bytes)
		sigLength += 2 + len(sig.DSASigS.bytes)
	case PubKeyAlgoECDSA:
		sigLength = 2 + len(sig.ECDSASigR.bytes)
		sigLength += 2 + len(sig.ECDSASigS.bytes)
	default:
		panic("impossible")
	}

	unhashedSubpacketsLen := subpacketsLength(sig.outSubpackets, false)
	length := len(sig.HashSuffix) - 6  +
		2  + unhashedSubpacketsLen +
		2  + sigLength
	err = serializeHeader(w, packetTypeSignature, length)
	if err != nil {
		return
	}

	_, err = w.Write(sig.HashSuffix[:len(sig.HashSuffix)-6])
	if err != nil {
		return
	}

	unhashedSubpackets := make([]byte, 2+unhashedSubpacketsLen)
	unhashedSubpackets[0] = byte(unhashedSubpacketsLen >> 8)
	unhashedSubpackets[1] = byte(unhashedSubpacketsLen)
	serializeSubpackets(unhashedSubpackets[2:], sig.outSubpackets, false)

	_, err = w.Write(unhashedSubpackets)
	if err != nil {
		return
	}
	_, err = w.Write(sig.HashTag[:])
	if err != nil {
		return
	}

	switch sig.PubKeyAlgo {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly:
		err = writeMPIs(w, sig.RSASignature)
	case PubKeyAlgoDSA:
		err = writeMPIs(w, sig.DSASigR, sig.DSASigS)
	case PubKeyAlgoECDSA:
		err = writeMPIs(w, sig.ECDSASigR, sig.ECDSASigS)
	default:
		panic("impossible")
	}
	return
}

type outputSubpacket struct {
	hashed        bool 
	subpacketType signatureSubpacketType
	isCritical    bool
	contents      []byte
}

func (sig *Signature) buildSubpackets() (subpackets []outputSubpacket) {
	creationTime := make([]byte, 4)
	binary.BigEndian.PutUint32(creationTime, uint32(sig.CreationTime.Unix()))
	subpackets = append(subpackets, outputSubpacket{true, creationTimeSubpacket, false, creationTime})

	if sig.IssuerKeyId != nil {
		keyId := make([]byte, 8)
		binary.BigEndian.PutUint64(keyId, *sig.IssuerKeyId)
		subpackets = append(subpackets, outputSubpacket{true, issuerSubpacket, false, keyId})
	}

	if sig.SigLifetimeSecs != nil && *sig.SigLifetimeSecs != 0 {
		sigLifetime := make([]byte, 4)
		binary.BigEndian.PutUint32(sigLifetime, *sig.SigLifetimeSecs)
		subpackets = append(subpackets, outputSubpacket{true, signatureExpirationSubpacket, true, sigLifetime})
	}

	if sig.FlagsValid {
		var flags byte
		if sig.FlagCertify {
			flags |= KeyFlagCertify
		}
		if sig.FlagSign {
			flags |= KeyFlagSign
		}
		if sig.FlagEncryptCommunications {
			flags |= KeyFlagEncryptCommunications
		}
		if sig.FlagEncryptStorage {
			flags |= KeyFlagEncryptStorage
		}
		subpackets = append(subpackets, outputSubpacket{true, keyFlagsSubpacket, false, []byte{flags}})
	}

	if sig.KeyLifetimeSecs != nil && *sig.KeyLifetimeSecs != 0 {
		keyLifetime := make([]byte, 4)
		binary.BigEndian.PutUint32(keyLifetime, *sig.KeyLifetimeSecs)
		subpackets = append(subpackets, outputSubpacket{true, keyExpirationSubpacket, true, keyLifetime})
	}

	if sig.IsPrimaryId != nil && *sig.IsPrimaryId {
		subpackets = append(subpackets, outputSubpacket{true, primaryUserIdSubpacket, false, []byte{1}})
	}

	if len(sig.PreferredSymmetric) > 0 {
		subpackets = append(subpackets, outputSubpacket{true, prefSymmetricAlgosSubpacket, false, sig.PreferredSymmetric})
	}

	if len(sig.PreferredHash) > 0 {
		subpackets = append(subpackets, outputSubpacket{true, prefHashAlgosSubpacket, false, sig.PreferredHash})
	}

	if len(sig.PreferredCompression) > 0 {
		subpackets = append(subpackets, outputSubpacket{true, prefCompressionSubpacket, false, sig.PreferredCompression})
	}

	return
}
