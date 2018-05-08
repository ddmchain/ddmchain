package jwt

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"math/big"
)

var (

	ErrECDSAVerification = errors.New("crypto/ecdsa: verification error")
)

type SigningMethodECDSA struct {
	Name      string
	Hash      crypto.Hash
	KeySize   int
	CurveBits int
}

var (
	SigningMethodES256 *SigningMethodECDSA
	SigningMethodES384 *SigningMethodECDSA
	SigningMethodES512 *SigningMethodECDSA
)

func init() {

	SigningMethodES256 = &SigningMethodECDSA{"ES256", crypto.SHA256, 32, 256}
	RegisterSigningMethod(SigningMethodES256.Alg(), func() SigningMethod {
		return SigningMethodES256
	})

	SigningMethodES384 = &SigningMethodECDSA{"ES384", crypto.SHA384, 48, 384}
	RegisterSigningMethod(SigningMethodES384.Alg(), func() SigningMethod {
		return SigningMethodES384
	})

	SigningMethodES512 = &SigningMethodECDSA{"ES512", crypto.SHA512, 66, 521}
	RegisterSigningMethod(SigningMethodES512.Alg(), func() SigningMethod {
		return SigningMethodES512
	})
}

func (m *SigningMethodECDSA) Alg() string {
	return m.Name
}

func (m *SigningMethodECDSA) Verify(signingString, signature string, key interface{}) error {
	var err error

	var sig []byte
	if sig, err = DecodeSegment(signature); err != nil {
		return err
	}

	var ecdsaKey *ecdsa.PublicKey
	switch k := key.(type) {
	case *ecdsa.PublicKey:
		ecdsaKey = k
	default:
		return ErrInvalidKeyType
	}

	if len(sig) != 2*m.KeySize {
		return ErrECDSAVerification
	}

	r := big.NewInt(0).SetBytes(sig[:m.KeySize])
	s := big.NewInt(0).SetBytes(sig[m.KeySize:])

	if !m.Hash.Available() {
		return ErrHashUnavailable
	}
	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	if verifystatus := ecdsa.Verify(ecdsaKey, hasher.Sum(nil), r, s); verifystatus == true {
		return nil
	} else {
		return ErrECDSAVerification
	}
}

func (m *SigningMethodECDSA) Sign(signingString string, key interface{}) (string, error) {

	var ecdsaKey *ecdsa.PrivateKey
	switch k := key.(type) {
	case *ecdsa.PrivateKey:
		ecdsaKey = k
	default:
		return "", ErrInvalidKeyType
	}

	if !m.Hash.Available() {
		return "", ErrHashUnavailable
	}

	hasher := m.Hash.New()
	hasher.Write([]byte(signingString))

	if r, s, err := ecdsa.Sign(rand.Reader, ecdsaKey, hasher.Sum(nil)); err == nil {
		curveBits := ecdsaKey.Curve.Params().BitSize

		if m.CurveBits != curveBits {
			return "", ErrInvalidKey
		}

		keyBytes := curveBits / 8
		if curveBits%8 > 0 {
			keyBytes += 1
		}

		rBytes := r.Bytes()
		rBytesPadded := make([]byte, keyBytes)
		copy(rBytesPadded[keyBytes-len(rBytes):], rBytes)

		sBytes := s.Bytes()
		sBytesPadded := make([]byte, keyBytes)
		copy(sBytesPadded[keyBytes-len(sBytes):], sBytes)

		out := append(rBytesPadded, sBytesPadded...)

		return EncodeSegment(out), nil
	} else {
		return "", err
	}
}
