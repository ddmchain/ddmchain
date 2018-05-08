
package packet

import (
	"bytes"
	"crypto/cipher"
	"io"
	"strconv"

	"golang.org/x/crypto/openpgp/errors"
	"golang.org/x/crypto/openpgp/s2k"
)

const maxSessionKeySizeInBytes = 64

type SymmetricKeyEncrypted struct {
	CipherFunc   CipherFunction
	s2k          func(out, in []byte)
	encryptedKey []byte
}

const symmetricKeyEncryptedVersion = 4

func (ske *SymmetricKeyEncrypted) parse(r io.Reader) error {

	var buf [2]byte
	if _, err := readFull(r, buf[:]); err != nil {
		return err
	}
	if buf[0] != symmetricKeyEncryptedVersion {
		return errors.UnsupportedError("SymmetricKeyEncrypted version")
	}
	ske.CipherFunc = CipherFunction(buf[1])

	if ske.CipherFunc.KeySize() == 0 {
		return errors.UnsupportedError("unknown cipher: " + strconv.Itoa(int(buf[1])))
	}

	var err error
	ske.s2k, err = s2k.Parse(r)
	if err != nil {
		return err
	}

	encryptedKey := make([]byte, maxSessionKeySizeInBytes)

	n, err := readFull(r, encryptedKey)
	if err != nil && err != io.ErrUnexpectedEOF {
		return err
	}

	if n != 0 {
		if n == maxSessionKeySizeInBytes {
			return errors.UnsupportedError("oversized encrypted session key")
		}
		ske.encryptedKey = encryptedKey[:n]
	}

	return nil
}

func (ske *SymmetricKeyEncrypted) Decrypt(passphrase []byte) ([]byte, CipherFunction, error) {
	key := make([]byte, ske.CipherFunc.KeySize())
	ske.s2k(key, passphrase)

	if len(ske.encryptedKey) == 0 {
		return key, ske.CipherFunc, nil
	}

	iv := make([]byte, ske.CipherFunc.blockSize())
	c := cipher.NewCFBDecrypter(ske.CipherFunc.new(key), iv)
	plaintextKey := make([]byte, len(ske.encryptedKey))
	c.XORKeyStream(plaintextKey, ske.encryptedKey)
	cipherFunc := CipherFunction(plaintextKey[0])
	if cipherFunc.blockSize() == 0 {
		return nil, ske.CipherFunc, errors.UnsupportedError("unknown cipher: " + strconv.Itoa(int(cipherFunc)))
	}
	plaintextKey = plaintextKey[1:]
	if l, cipherKeySize := len(plaintextKey), cipherFunc.KeySize(); l != cipherFunc.KeySize() {
		return nil, cipherFunc, errors.StructuralError("length of decrypted key (" + strconv.Itoa(l) + ") " +
			"not equal to cipher keysize (" + strconv.Itoa(cipherKeySize) + ")")
	}
	return plaintextKey, cipherFunc, nil
}

func SerializeSymmetricKeyEncrypted(w io.Writer, passphrase []byte, config *Config) (key []byte, err error) {
	cipherFunc := config.Cipher()
	keySize := cipherFunc.KeySize()
	if keySize == 0 {
		return nil, errors.UnsupportedError("unknown cipher: " + strconv.Itoa(int(cipherFunc)))
	}

	s2kBuf := new(bytes.Buffer)
	keyEncryptingKey := make([]byte, keySize)

	err = s2k.Serialize(s2kBuf, keyEncryptingKey, config.Random(), passphrase, &s2k.Config{Hash: config.Hash(), S2KCount: config.PasswordHashIterations()})
	if err != nil {
		return
	}
	s2kBytes := s2kBuf.Bytes()

	packetLength := 2  + len(s2kBytes) + 1  + keySize
	err = serializeHeader(w, packetTypeSymmetricKeyEncrypted, packetLength)
	if err != nil {
		return
	}

	var buf [2]byte
	buf[0] = symmetricKeyEncryptedVersion
	buf[1] = byte(cipherFunc)
	_, err = w.Write(buf[:])
	if err != nil {
		return
	}
	_, err = w.Write(s2kBytes)
	if err != nil {
		return
	}

	sessionKey := make([]byte, keySize)
	_, err = io.ReadFull(config.Random(), sessionKey)
	if err != nil {
		return
	}
	iv := make([]byte, cipherFunc.blockSize())
	c := cipher.NewCFBEncrypter(cipherFunc.new(keyEncryptingKey), iv)
	encryptedCipherAndKey := make([]byte, keySize+1)
	c.XORKeyStream(encryptedCipherAndKey, buf[1:])
	c.XORKeyStream(encryptedCipherAndKey[1:], sessionKey)
	_, err = w.Write(encryptedCipherAndKey)
	if err != nil {
		return
	}

	key = sessionKey
	return
}
