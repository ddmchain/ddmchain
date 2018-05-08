
package btcec

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"errors"
	"io"
)

var (

	ErrInvalidMAC = errors.New("invalid mac hash")

	errInputTooShort = errors.New("ciphertext too short")

	errUnsupportedCurve = errors.New("unsupported curve")

	errInvalidXLength = errors.New("invalid X length, must be 32")
	errInvalidYLength = errors.New("invalid Y length, must be 32")
	errInvalidPadding = errors.New("invalid PKCS#7 padding")

	ciphCurveBytes = [2]byte{0x02, 0xCA}

	ciphCoordLength = [2]byte{0x00, 0x20}
)

func GenerateSharedSecret(privkey *PrivateKey, pubkey *PublicKey) []byte {
	x, _ := pubkey.Curve.ScalarMult(pubkey.X, pubkey.Y, privkey.D.Bytes())
	return x.Bytes()
}

func Encrypt(pubkey *PublicKey, in []byte) ([]byte, error) {
	ephemeral, err := NewPrivateKey(S256())
	if err != nil {
		return nil, err
	}
	ecdhKey := GenerateSharedSecret(ephemeral, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	paddedIn := addPKCSPadding(in)

	out := make([]byte, aes.BlockSize+70+len(paddedIn)+sha256.Size)
	iv := out[:aes.BlockSize]
	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	pb := ephemeral.PubKey().SerializeUncompressed()
	offset := aes.BlockSize

	copy(out[offset:offset+4], append(ciphCurveBytes[:], ciphCoordLength[:]...))
	offset += 4

	copy(out[offset:offset+32], pb[1:33])
	offset += 32

	copy(out[offset:offset+2], ciphCoordLength[:])
	offset += 2

	copy(out[offset:offset+32], pb[33:])
	offset += 32

	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(out[offset:len(out)-sha256.Size], paddedIn)

	hm := hmac.New(sha256.New, keyM)
	hm.Write(out[:len(out)-sha256.Size])          
	copy(out[len(out)-sha256.Size:], hm.Sum(nil)) 

	return out, nil
}

func Decrypt(priv *PrivateKey, in []byte) ([]byte, error) {

	if len(in) < aes.BlockSize+70+aes.BlockSize+sha256.Size {
		return nil, errInputTooShort
	}

	iv := in[:aes.BlockSize]
	offset := aes.BlockSize

	if !bytes.Equal(in[offset:offset+2], ciphCurveBytes[:]) {
		return nil, errUnsupportedCurve
	}
	offset += 2

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidXLength
	}
	offset += 2

	xBytes := in[offset : offset+32]
	offset += 32

	if !bytes.Equal(in[offset:offset+2], ciphCoordLength[:]) {
		return nil, errInvalidYLength
	}
	offset += 2

	yBytes := in[offset : offset+32]
	offset += 32

	pb := make([]byte, 65)
	pb[0] = byte(0x04) 
	copy(pb[1:33], xBytes)
	copy(pb[33:], yBytes)

	pubkey, err := ParsePubKey(pb, S256())
	if err != nil {
		return nil, err
	}

	if (len(in)-aes.BlockSize-offset-sha256.Size)%aes.BlockSize != 0 {
		return nil, errInvalidPadding 
	}

	messageMAC := in[len(in)-sha256.Size:]

	ecdhKey := GenerateSharedSecret(priv, pubkey)
	derivedKey := sha512.Sum512(ecdhKey)
	keyE := derivedKey[:32]
	keyM := derivedKey[32:]

	hm := hmac.New(sha256.New, keyM)
	hm.Write(in[:len(in)-sha256.Size]) 
	expectedMAC := hm.Sum(nil)
	if !hmac.Equal(messageMAC, expectedMAC) {
		return nil, ErrInvalidMAC
	}

	block, err := aes.NewCipher(keyE)
	if err != nil {
		return nil, err
	}
	mode := cipher.NewCBCDecrypter(block, iv)

	plaintext := make([]byte, len(in)-offset-sha256.Size)
	mode.CryptBlocks(plaintext, in[offset:len(in)-sha256.Size])

	return removePKCSPadding(plaintext)
}

func addPKCSPadding(src []byte) []byte {
	padding := aes.BlockSize - len(src)%aes.BlockSize
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padtext...)
}

func removePKCSPadding(src []byte) ([]byte, error) {
	length := len(src)
	padLength := int(src[length-1])
	if padLength > aes.BlockSize || length < aes.BlockSize {
		return nil, errInvalidPadding
	}

	return src[:length-padLength], nil
}
