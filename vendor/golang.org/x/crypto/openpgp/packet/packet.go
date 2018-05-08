
package packet 

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"golang.org/x/crypto/cast5"
	"golang.org/x/crypto/openpgp/errors"
	"io"
	"math/big"
)

func readFull(r io.Reader, buf []byte) (n int, err error) {
	n, err = io.ReadFull(r, buf)
	if err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func readLength(r io.Reader) (length int64, isPartial bool, err error) {
	var buf [4]byte
	_, err = readFull(r, buf[:1])
	if err != nil {
		return
	}
	switch {
	case buf[0] < 192:
		length = int64(buf[0])
	case buf[0] < 224:
		length = int64(buf[0]-192) << 8
		_, err = readFull(r, buf[0:1])
		if err != nil {
			return
		}
		length += int64(buf[0]) + 192
	case buf[0] < 255:
		length = int64(1) << (buf[0] & 0x1f)
		isPartial = true
	default:
		_, err = readFull(r, buf[0:4])
		if err != nil {
			return
		}
		length = int64(buf[0])<<24 |
			int64(buf[1])<<16 |
			int64(buf[2])<<8 |
			int64(buf[3])
	}
	return
}

type partialLengthReader struct {
	r         io.Reader
	remaining int64
	isPartial bool
}

func (r *partialLengthReader) Read(p []byte) (n int, err error) {
	for r.remaining == 0 {
		if !r.isPartial {
			return 0, io.EOF
		}
		r.remaining, r.isPartial, err = readLength(r.r)
		if err != nil {
			return 0, err
		}
	}

	toRead := int64(len(p))
	if toRead > r.remaining {
		toRead = r.remaining
	}

	n, err = r.r.Read(p[:int(toRead)])
	r.remaining -= int64(n)
	if n < int(toRead) && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

type partialLengthWriter struct {
	w          io.WriteCloser
	lengthByte [1]byte
}

func (w *partialLengthWriter) Write(p []byte) (n int, err error) {
	for len(p) > 0 {
		for power := uint(14); power < 32; power-- {
			l := 1 << power
			if len(p) >= l {
				w.lengthByte[0] = 224 + uint8(power)
				_, err = w.w.Write(w.lengthByte[:])
				if err != nil {
					return
				}
				var m int
				m, err = w.w.Write(p[:l])
				n += m
				if err != nil {
					return
				}
				p = p[l:]
				break
			}
		}
	}
	return
}

func (w *partialLengthWriter) Close() error {
	w.lengthByte[0] = 0
	_, err := w.w.Write(w.lengthByte[:])
	if err != nil {
		return err
	}
	return w.w.Close()
}

type spanReader struct {
	r io.Reader
	n int64
}

func (l *spanReader) Read(p []byte) (n int, err error) {
	if l.n <= 0 {
		return 0, io.EOF
	}
	if int64(len(p)) > l.n {
		p = p[0:l.n]
	}
	n, err = l.r.Read(p)
	l.n -= int64(n)
	if l.n > 0 && err == io.EOF {
		err = io.ErrUnexpectedEOF
	}
	return
}

func readHeader(r io.Reader) (tag packetType, length int64, contents io.Reader, err error) {
	var buf [4]byte
	_, err = io.ReadFull(r, buf[:1])
	if err != nil {
		return
	}
	if buf[0]&0x80 == 0 {
		err = errors.StructuralError("tag byte does not have MSB set")
		return
	}
	if buf[0]&0x40 == 0 {

		tag = packetType((buf[0] & 0x3f) >> 2)
		lengthType := buf[0] & 3
		if lengthType == 3 {
			length = -1
			contents = r
			return
		}
		lengthBytes := 1 << lengthType
		_, err = readFull(r, buf[0:lengthBytes])
		if err != nil {
			return
		}
		for i := 0; i < lengthBytes; i++ {
			length <<= 8
			length |= int64(buf[i])
		}
		contents = &spanReader{r, length}
		return
	}

	tag = packetType(buf[0] & 0x3f)
	length, isPartial, err := readLength(r)
	if err != nil {
		return
	}
	if isPartial {
		contents = &partialLengthReader{
			remaining: length,
			isPartial: true,
			r:         r,
		}
		length = -1
	} else {
		contents = &spanReader{r, length}
	}
	return
}

func serializeHeader(w io.Writer, ptype packetType, length int) (err error) {
	var buf [6]byte
	var n int

	buf[0] = 0x80 | 0x40 | byte(ptype)
	if length < 192 {
		buf[1] = byte(length)
		n = 2
	} else if length < 8384 {
		length -= 192
		buf[1] = 192 + byte(length>>8)
		buf[2] = byte(length)
		n = 3
	} else {
		buf[1] = 255
		buf[2] = byte(length >> 24)
		buf[3] = byte(length >> 16)
		buf[4] = byte(length >> 8)
		buf[5] = byte(length)
		n = 6
	}

	_, err = w.Write(buf[:n])
	return
}

func serializeStreamHeader(w io.WriteCloser, ptype packetType) (out io.WriteCloser, err error) {
	var buf [1]byte
	buf[0] = 0x80 | 0x40 | byte(ptype)
	_, err = w.Write(buf[:])
	if err != nil {
		return
	}
	out = &partialLengthWriter{w: w}
	return
}

type Packet interface {
	parse(io.Reader) error
}

func consumeAll(r io.Reader) (n int64, err error) {
	var m int
	var buf [1024]byte

	for {
		m, err = r.Read(buf[:])
		n += int64(m)
		if err == io.EOF {
			err = nil
			return
		}
		if err != nil {
			return
		}
	}
}

type packetType uint8

const (
	packetTypeEncryptedKey              packetType = 1
	packetTypeSignature                 packetType = 2
	packetTypeSymmetricKeyEncrypted     packetType = 3
	packetTypeOnePassSignature          packetType = 4
	packetTypePrivateKey                packetType = 5
	packetTypePublicKey                 packetType = 6
	packetTypePrivateSubkey             packetType = 7
	packetTypeCompressed                packetType = 8
	packetTypeSymmetricallyEncrypted    packetType = 9
	packetTypeLiteralData               packetType = 11
	packetTypeUserId                    packetType = 13
	packetTypePublicSubkey              packetType = 14
	packetTypeUserAttribute             packetType = 17
	packetTypeSymmetricallyEncryptedMDC packetType = 18
)

func peekVersion(r io.Reader) (bufr *bufio.Reader, ver byte, err error) {
	bufr = bufio.NewReader(r)
	var verBuf []byte
	if verBuf, err = bufr.Peek(1); err != nil {
		return
	}
	ver = verBuf[0]
	return
}

func Read(r io.Reader) (p Packet, err error) {
	tag, _, contents, err := readHeader(r)
	if err != nil {
		return
	}

	switch tag {
	case packetTypeEncryptedKey:
		p = new(EncryptedKey)
	case packetTypeSignature:
		var version byte

		if contents, version, err = peekVersion(contents); err != nil {
			return
		}
		if version < 4 {
			p = new(SignatureV3)
		} else {
			p = new(Signature)
		}
	case packetTypeSymmetricKeyEncrypted:
		p = new(SymmetricKeyEncrypted)
	case packetTypeOnePassSignature:
		p = new(OnePassSignature)
	case packetTypePrivateKey, packetTypePrivateSubkey:
		pk := new(PrivateKey)
		if tag == packetTypePrivateSubkey {
			pk.IsSubkey = true
		}
		p = pk
	case packetTypePublicKey, packetTypePublicSubkey:
		var version byte
		if contents, version, err = peekVersion(contents); err != nil {
			return
		}
		isSubkey := tag == packetTypePublicSubkey
		if version < 4 {
			p = &PublicKeyV3{IsSubkey: isSubkey}
		} else {
			p = &PublicKey{IsSubkey: isSubkey}
		}
	case packetTypeCompressed:
		p = new(Compressed)
	case packetTypeSymmetricallyEncrypted:
		p = new(SymmetricallyEncrypted)
	case packetTypeLiteralData:
		p = new(LiteralData)
	case packetTypeUserId:
		p = new(UserId)
	case packetTypeUserAttribute:
		p = new(UserAttribute)
	case packetTypeSymmetricallyEncryptedMDC:
		se := new(SymmetricallyEncrypted)
		se.MDC = true
		p = se
	default:
		err = errors.UnknownPacketTypeError(tag)
	}
	if p != nil {
		err = p.parse(contents)
	}
	if err != nil {
		consumeAll(contents)
	}
	return
}

type SignatureType uint8

const (
	SigTypeBinary            SignatureType = 0
	SigTypeText                            = 1
	SigTypeGenericCert                     = 0x10
	SigTypePersonaCert                     = 0x11
	SigTypeCasualCert                      = 0x12
	SigTypePositiveCert                    = 0x13
	SigTypeSubkeyBinding                   = 0x18
	SigTypePrimaryKeyBinding               = 0x19
	SigTypeDirectSignature                 = 0x1F
	SigTypeKeyRevocation                   = 0x20
	SigTypeSubkeyRevocation                = 0x28
)

type PublicKeyAlgorithm uint8

const (
	PubKeyAlgoRSA            PublicKeyAlgorithm = 1
	PubKeyAlgoRSAEncryptOnly PublicKeyAlgorithm = 2
	PubKeyAlgoRSASignOnly    PublicKeyAlgorithm = 3
	PubKeyAlgoElGamal        PublicKeyAlgorithm = 16
	PubKeyAlgoDSA            PublicKeyAlgorithm = 17

	PubKeyAlgoECDH  PublicKeyAlgorithm = 18
	PubKeyAlgoECDSA PublicKeyAlgorithm = 19
)

func (pka PublicKeyAlgorithm) CanEncrypt() bool {
	switch pka {
	case PubKeyAlgoRSA, PubKeyAlgoRSAEncryptOnly, PubKeyAlgoElGamal:
		return true
	}
	return false
}

func (pka PublicKeyAlgorithm) CanSign() bool {
	switch pka {
	case PubKeyAlgoRSA, PubKeyAlgoRSASignOnly, PubKeyAlgoDSA, PubKeyAlgoECDSA:
		return true
	}
	return false
}

type CipherFunction uint8

const (
	Cipher3DES   CipherFunction = 2
	CipherCAST5  CipherFunction = 3
	CipherAES128 CipherFunction = 7
	CipherAES192 CipherFunction = 8
	CipherAES256 CipherFunction = 9
)

func (cipher CipherFunction) KeySize() int {
	switch cipher {
	case Cipher3DES:
		return 24
	case CipherCAST5:
		return cast5.KeySize
	case CipherAES128:
		return 16
	case CipherAES192:
		return 24
	case CipherAES256:
		return 32
	}
	return 0
}

func (cipher CipherFunction) blockSize() int {
	switch cipher {
	case Cipher3DES:
		return des.BlockSize
	case CipherCAST5:
		return 8
	case CipherAES128, CipherAES192, CipherAES256:
		return 16
	}
	return 0
}

func (cipher CipherFunction) new(key []byte) (block cipher.Block) {
	switch cipher {
	case Cipher3DES:
		block, _ = des.NewTripleDESCipher(key)
	case CipherCAST5:
		block, _ = cast5.NewCipher(key)
	case CipherAES128, CipherAES192, CipherAES256:
		block, _ = aes.NewCipher(key)
	}
	return
}

func readMPI(r io.Reader) (mpi []byte, bitLength uint16, err error) {
	var buf [2]byte
	_, err = readFull(r, buf[0:])
	if err != nil {
		return
	}
	bitLength = uint16(buf[0])<<8 | uint16(buf[1])
	numBytes := (int(bitLength) + 7) / 8
	mpi = make([]byte, numBytes)
	_, err = readFull(r, mpi)
	return
}

func mpiLength(n *big.Int) (mpiLengthInBytes int) {
	mpiLengthInBytes = 2 
	mpiLengthInBytes += (n.BitLen() + 7) / 8
	return
}

func writeMPI(w io.Writer, bitLength uint16, mpiBytes []byte) (err error) {
	_, err = w.Write([]byte{byte(bitLength >> 8), byte(bitLength)})
	if err == nil {
		_, err = w.Write(mpiBytes)
	}
	return
}

func writeBig(w io.Writer, i *big.Int) error {
	return writeMPI(w, uint16(i.BitLen()), i.Bytes())
}

type CompressionAlgo uint8

const (
	CompressionNone CompressionAlgo = 0
	CompressionZIP  CompressionAlgo = 1
	CompressionZLIB CompressionAlgo = 2
)
