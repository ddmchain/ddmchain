
package packet

import (
	"crypto/cipher"
	"crypto/sha1"
	"crypto/subtle"
	"golang.org/x/crypto/openpgp/errors"
	"hash"
	"io"
	"strconv"
)

type SymmetricallyEncrypted struct {
	MDC      bool 
	contents io.Reader
	prefix   []byte
}

const symmetricallyEncryptedVersion = 1

func (se *SymmetricallyEncrypted) parse(r io.Reader) error {
	if se.MDC {

		var buf [1]byte
		_, err := readFull(r, buf[:])
		if err != nil {
			return err
		}
		if buf[0] != symmetricallyEncryptedVersion {
			return errors.UnsupportedError("unknown SymmetricallyEncrypted version")
		}
	}
	se.contents = r
	return nil
}

func (se *SymmetricallyEncrypted) Decrypt(c CipherFunction, key []byte) (io.ReadCloser, error) {
	keySize := c.KeySize()
	if keySize == 0 {
		return nil, errors.UnsupportedError("unknown cipher: " + strconv.Itoa(int(c)))
	}
	if len(key) != keySize {
		return nil, errors.InvalidArgumentError("SymmetricallyEncrypted: incorrect key length")
	}

	if se.prefix == nil {
		se.prefix = make([]byte, c.blockSize()+2)
		_, err := readFull(se.contents, se.prefix)
		if err != nil {
			return nil, err
		}
	} else if len(se.prefix) != c.blockSize()+2 {
		return nil, errors.InvalidArgumentError("can't try ciphers with different block lengths")
	}

	ocfbResync := OCFBResync
	if se.MDC {

		ocfbResync = OCFBNoResync
	}

	s := NewOCFBDecrypter(c.new(key), se.prefix, ocfbResync)
	if s == nil {
		return nil, errors.ErrKeyIncorrect
	}

	plaintext := cipher.StreamReader{S: s, R: se.contents}

	if se.MDC {

		h := sha1.New()
		h.Write(se.prefix)
		return &seMDCReader{in: plaintext, h: h}, nil
	}

	return seReader{plaintext}, nil
}

type seReader struct {
	in io.Reader
}

func (ser seReader) Read(buf []byte) (int, error) {
	return ser.in.Read(buf)
}

func (ser seReader) Close() error {
	return nil
}

const mdcTrailerSize = 1  + 1  + sha1.Size

type seMDCReader struct {
	in          io.Reader
	h           hash.Hash
	trailer     [mdcTrailerSize]byte
	scratch     [mdcTrailerSize]byte
	trailerUsed int
	error       bool
	eof         bool
}

func (ser *seMDCReader) Read(buf []byte) (n int, err error) {
	if ser.error {
		err = io.ErrUnexpectedEOF
		return
	}
	if ser.eof {
		err = io.EOF
		return
	}

	for ser.trailerUsed < mdcTrailerSize {
		n, err = ser.in.Read(ser.trailer[ser.trailerUsed:])
		ser.trailerUsed += n
		if err == io.EOF {
			if ser.trailerUsed != mdcTrailerSize {
				n = 0
				err = io.ErrUnexpectedEOF
				ser.error = true
				return
			}
			ser.eof = true
			n = 0
			return
		}

		if err != nil {
			n = 0
			return
		}
	}

	if len(buf) <= mdcTrailerSize {
		n, err = readFull(ser.in, ser.scratch[:len(buf)])
		copy(buf, ser.trailer[:n])
		ser.h.Write(buf[:n])
		copy(ser.trailer[:], ser.trailer[n:])
		copy(ser.trailer[mdcTrailerSize-n:], ser.scratch[:])
		if n < len(buf) {
			ser.eof = true
			err = io.EOF
		}
		return
	}

	n, err = ser.in.Read(buf[mdcTrailerSize:])
	copy(buf, ser.trailer[:])
	ser.h.Write(buf[:n])
	copy(ser.trailer[:], buf[n:])

	if err == io.EOF {
		ser.eof = true
	}
	return
}

const mdcPacketTagByte = byte(0x80) | 0x40 | 19

func (ser *seMDCReader) Close() error {
	if ser.error {
		return errors.SignatureError("error during reading")
	}

	for !ser.eof {

		var buf [1024]byte
		_, err := ser.Read(buf[:])
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.SignatureError("error during reading")
		}
	}

	if ser.trailer[0] != mdcPacketTagByte || ser.trailer[1] != sha1.Size {
		return errors.SignatureError("MDC packet not found")
	}
	ser.h.Write(ser.trailer[:2])

	final := ser.h.Sum(nil)
	if subtle.ConstantTimeCompare(final, ser.trailer[2:]) != 1 {
		return errors.SignatureError("hash mismatch")
	}
	return nil
}

type seMDCWriter struct {
	w io.WriteCloser
	h hash.Hash
}

func (w *seMDCWriter) Write(buf []byte) (n int, err error) {
	w.h.Write(buf)
	return w.w.Write(buf)
}

func (w *seMDCWriter) Close() (err error) {
	var buf [mdcTrailerSize]byte

	buf[0] = mdcPacketTagByte
	buf[1] = sha1.Size
	w.h.Write(buf[:2])
	digest := w.h.Sum(nil)
	copy(buf[2:], digest)

	_, err = w.w.Write(buf[:])
	if err != nil {
		return
	}
	return w.w.Close()
}

type noOpCloser struct {
	w io.Writer
}

func (c noOpCloser) Write(data []byte) (n int, err error) {
	return c.w.Write(data)
}

func (c noOpCloser) Close() error {
	return nil
}

func SerializeSymmetricallyEncrypted(w io.Writer, c CipherFunction, key []byte, config *Config) (contents io.WriteCloser, err error) {
	if c.KeySize() != len(key) {
		return nil, errors.InvalidArgumentError("SymmetricallyEncrypted.Serialize: bad key length")
	}
	writeCloser := noOpCloser{w}
	ciphertext, err := serializeStreamHeader(writeCloser, packetTypeSymmetricallyEncryptedMDC)
	if err != nil {
		return
	}

	_, err = ciphertext.Write([]byte{symmetricallyEncryptedVersion})
	if err != nil {
		return
	}

	block := c.new(key)
	blockSize := block.BlockSize()
	iv := make([]byte, blockSize)
	_, err = config.Random().Read(iv)
	if err != nil {
		return
	}
	s, prefix := NewOCFBEncrypter(block, iv, OCFBNoResync)
	_, err = ciphertext.Write(prefix)
	if err != nil {
		return
	}
	plaintext := cipher.StreamWriter{S: s, W: ciphertext}

	h := sha1.New()
	h.Write(iv)
	h.Write(iv[blockSize-2:])
	contents = &seMDCWriter{w: plaintext, h: h}
	return
}
