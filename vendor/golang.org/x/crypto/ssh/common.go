
package ssh

import (
	"crypto"
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"sync"

	_ "crypto/sha1"
	_ "crypto/sha256"
	_ "crypto/sha512"
)

const (
	compressionNone = "none"
	serviceUserAuth = "ssh-userauth"
	serviceSSH      = "ssh-connection"
)

var supportedCiphers = []string{
	"aes128-ctr", "aes192-ctr", "aes256-ctr",
	"aes128-gcm@openssh.com",
	"arcfour256", "arcfour128",
}

var supportedKexAlgos = []string{
	kexAlgoCurve25519SHA256,

	kexAlgoECDH256, kexAlgoECDH384, kexAlgoECDH521,
	kexAlgoDH14SHA1, kexAlgoDH1SHA1,
}

var supportedHostKeyAlgos = []string{
	CertAlgoRSAv01, CertAlgoDSAv01, CertAlgoECDSA256v01,
	CertAlgoECDSA384v01, CertAlgoECDSA521v01, CertAlgoED25519v01,

	KeyAlgoECDSA256, KeyAlgoECDSA384, KeyAlgoECDSA521,
	KeyAlgoRSA, KeyAlgoDSA,

	KeyAlgoED25519,
}

var supportedMACs = []string{
	"hmac-sha2-256-etm@openssh.com", "hmac-sha2-256", "hmac-sha1", "hmac-sha1-96",
}

var supportedCompressions = []string{compressionNone}

var hashFuncs = map[string]crypto.Hash{
	KeyAlgoRSA:          crypto.SHA1,
	KeyAlgoDSA:          crypto.SHA1,
	KeyAlgoECDSA256:     crypto.SHA256,
	KeyAlgoECDSA384:     crypto.SHA384,
	KeyAlgoECDSA521:     crypto.SHA512,
	CertAlgoRSAv01:      crypto.SHA1,
	CertAlgoDSAv01:      crypto.SHA1,
	CertAlgoECDSA256v01: crypto.SHA256,
	CertAlgoECDSA384v01: crypto.SHA384,
	CertAlgoECDSA521v01: crypto.SHA512,
}

func unexpectedMessageError(expected, got uint8) error {
	return fmt.Errorf("ssh: unexpected message type %d (expected %d)", got, expected)
}

func parseError(tag uint8) error {
	return fmt.Errorf("ssh: parse error in message type %d", tag)
}

func findCommon(what string, client []string, server []string) (common string, err error) {
	for _, c := range client {
		for _, s := range server {
			if c == s {
				return c, nil
			}
		}
	}
	return "", fmt.Errorf("ssh: no common algorithm for %s; client offered: %v, server offered: %v", what, client, server)
}

type directionAlgorithms struct {
	Cipher      string
	MAC         string
	Compression string
}

func (a *directionAlgorithms) rekeyBytes() int64 {

	switch a.Cipher {
	case "aes128-ctr", "aes192-ctr", "aes256-ctr", gcmCipherID, aes128cbcID:
		return 16 * (1 << 32)

	}

	return 1 << 30
}

type algorithms struct {
	kex     string
	hostKey string
	w       directionAlgorithms
	r       directionAlgorithms
}

func findAgreedAlgorithms(clientKexInit, serverKexInit *kexInitMsg) (algs *algorithms, err error) {
	result := &algorithms{}

	result.kex, err = findCommon("key exchange", clientKexInit.KexAlgos, serverKexInit.KexAlgos)
	if err != nil {
		return
	}

	result.hostKey, err = findCommon("host key", clientKexInit.ServerHostKeyAlgos, serverKexInit.ServerHostKeyAlgos)
	if err != nil {
		return
	}

	result.w.Cipher, err = findCommon("client to server cipher", clientKexInit.CiphersClientServer, serverKexInit.CiphersClientServer)
	if err != nil {
		return
	}

	result.r.Cipher, err = findCommon("server to client cipher", clientKexInit.CiphersServerClient, serverKexInit.CiphersServerClient)
	if err != nil {
		return
	}

	result.w.MAC, err = findCommon("client to server MAC", clientKexInit.MACsClientServer, serverKexInit.MACsClientServer)
	if err != nil {
		return
	}

	result.r.MAC, err = findCommon("server to client MAC", clientKexInit.MACsServerClient, serverKexInit.MACsServerClient)
	if err != nil {
		return
	}

	result.w.Compression, err = findCommon("client to server compression", clientKexInit.CompressionClientServer, serverKexInit.CompressionClientServer)
	if err != nil {
		return
	}

	result.r.Compression, err = findCommon("server to client compression", clientKexInit.CompressionServerClient, serverKexInit.CompressionServerClient)
	if err != nil {
		return
	}

	return result, nil
}

const minRekeyThreshold uint64 = 256

type Config struct {

	Rand io.Reader

	RekeyThreshold uint64

	KeyExchanges []string

	Ciphers []string

	MACs []string
}

func (c *Config) SetDefaults() {
	if c.Rand == nil {
		c.Rand = rand.Reader
	}
	if c.Ciphers == nil {
		c.Ciphers = supportedCiphers
	}
	var ciphers []string
	for _, c := range c.Ciphers {
		if cipherModes[c] != nil {

			ciphers = append(ciphers, c)
		}
	}
	c.Ciphers = ciphers

	if c.KeyExchanges == nil {
		c.KeyExchanges = supportedKexAlgos
	}

	if c.MACs == nil {
		c.MACs = supportedMACs
	}

	if c.RekeyThreshold == 0 {

	} else if c.RekeyThreshold < minRekeyThreshold {
		c.RekeyThreshold = minRekeyThreshold
	} else if c.RekeyThreshold >= math.MaxInt64 {

		c.RekeyThreshold = math.MaxInt64
	}
}

func buildDataSignedForAuth(sessionId []byte, req userAuthRequestMsg, algo, pubKey []byte) []byte {
	data := struct {
		Session []byte
		Type    byte
		User    string
		Service string
		Method  string
		Sign    bool
		Algo    []byte
		PubKey  []byte
	}{
		sessionId,
		msgUserAuthRequest,
		req.User,
		req.Service,
		req.Method,
		true,
		algo,
		pubKey,
	}
	return Marshal(data)
}

func appendU16(buf []byte, n uint16) []byte {
	return append(buf, byte(n>>8), byte(n))
}

func appendU32(buf []byte, n uint32) []byte {
	return append(buf, byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func appendU64(buf []byte, n uint64) []byte {
	return append(buf,
		byte(n>>56), byte(n>>48), byte(n>>40), byte(n>>32),
		byte(n>>24), byte(n>>16), byte(n>>8), byte(n))
}

func appendInt(buf []byte, n int) []byte {
	return appendU32(buf, uint32(n))
}

func appendString(buf []byte, s string) []byte {
	buf = appendU32(buf, uint32(len(s)))
	buf = append(buf, s...)
	return buf
}

func appendBool(buf []byte, b bool) []byte {
	if b {
		return append(buf, 1)
	}
	return append(buf, 0)
}

func newCond() *sync.Cond { return sync.NewCond(new(sync.Mutex)) }

type window struct {
	*sync.Cond
	win          uint32 
	writeWaiters int
	closed       bool
}

func (w *window) add(win uint32) bool {

	if win == 0 {
		return true
	}
	w.L.Lock()
	if w.win+win < win {
		w.L.Unlock()
		return false
	}
	w.win += win

	w.Broadcast()
	w.L.Unlock()
	return true
}

func (w *window) close() {
	w.L.Lock()
	w.closed = true
	w.Broadcast()
	w.L.Unlock()
}

func (w *window) reserve(win uint32) (uint32, error) {
	var err error
	w.L.Lock()
	w.writeWaiters++
	w.Broadcast()
	for w.win == 0 && !w.closed {
		w.Wait()
	}
	w.writeWaiters--
	if w.win < win {
		win = w.win
	}
	w.win -= win
	if w.closed {
		err = io.EOF
	}
	w.L.Unlock()
	return win, err
}

func (w *window) waitWriterBlocked() {
	w.Cond.L.Lock()
	for w.writeWaiters == 0 {
		w.Cond.Wait()
	}
	w.Cond.L.Unlock()
}
