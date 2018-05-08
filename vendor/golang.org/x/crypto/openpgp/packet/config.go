
package packet

import (
	"crypto"
	"crypto/rand"
	"io"
	"time"
)

type Config struct {

	Rand io.Reader

	DefaultHash crypto.Hash

	DefaultCipher CipherFunction

	Time func() time.Time

	DefaultCompressionAlgo CompressionAlgo

	CompressionConfig *CompressionConfig

	S2KCount int

	RSABits int
}

func (c *Config) Random() io.Reader {
	if c == nil || c.Rand == nil {
		return rand.Reader
	}
	return c.Rand
}

func (c *Config) Hash() crypto.Hash {
	if c == nil || uint(c.DefaultHash) == 0 {
		return crypto.SHA256
	}
	return c.DefaultHash
}

func (c *Config) Cipher() CipherFunction {
	if c == nil || uint8(c.DefaultCipher) == 0 {
		return CipherAES128
	}
	return c.DefaultCipher
}

func (c *Config) Now() time.Time {
	if c == nil || c.Time == nil {
		return time.Now()
	}
	return c.Time()
}

func (c *Config) Compression() CompressionAlgo {
	if c == nil {
		return CompressionNone
	}
	return c.DefaultCompressionAlgo
}

func (c *Config) PasswordHashIterations() int {
	if c == nil || c.S2KCount == 0 {
		return 0
	}
	return c.S2KCount
}
