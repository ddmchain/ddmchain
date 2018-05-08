
package ssh

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

const debugHandshake = false

const chanSize = 16

type keyingTransport interface {
	packetConn

	prepareKeyChange(*algorithms, *kexResult) error
}

type handshakeTransport struct {
	conn   keyingTransport
	config *Config

	serverVersion []byte
	clientVersion []byte

	hostKeys []Signer

	hostKeyAlgorithms []string

	incoming  chan []byte
	readError error

	mu             sync.Mutex
	writeError     error
	sentInitPacket []byte
	sentInitMsg    *kexInitMsg
	pendingPackets [][]byte 

	requestKex chan struct{}

	startKex chan *pendingKex

	hostKeyCallback HostKeyCallback
	dialAddress     string
	remoteAddr      net.Addr

	algorithms *algorithms

	readPacketsLeft uint32
	readBytesLeft   int64

	writePacketsLeft uint32
	writeBytesLeft   int64

	sessionID []byte
}

type pendingKex struct {
	otherInit []byte
	done      chan error
}

func newHandshakeTransport(conn keyingTransport, config *Config, clientVersion, serverVersion []byte) *handshakeTransport {
	t := &handshakeTransport{
		conn:          conn,
		serverVersion: serverVersion,
		clientVersion: clientVersion,
		incoming:      make(chan []byte, chanSize),
		requestKex:    make(chan struct{}, 1),
		startKex:      make(chan *pendingKex, 1),

		config: config,
	}
	t.resetReadThresholds()
	t.resetWriteThresholds()

	t.requestKex <- struct{}{}
	return t
}

func newClientTransport(conn keyingTransport, clientVersion, serverVersion []byte, config *ClientConfig, dialAddr string, addr net.Addr) *handshakeTransport {
	t := newHandshakeTransport(conn, &config.Config, clientVersion, serverVersion)
	t.dialAddress = dialAddr
	t.remoteAddr = addr
	t.hostKeyCallback = config.HostKeyCallback
	if config.HostKeyAlgorithms != nil {
		t.hostKeyAlgorithms = config.HostKeyAlgorithms
	} else {
		t.hostKeyAlgorithms = supportedHostKeyAlgos
	}
	go t.readLoop()
	go t.kexLoop()
	return t
}

func newServerTransport(conn keyingTransport, clientVersion, serverVersion []byte, config *ServerConfig) *handshakeTransport {
	t := newHandshakeTransport(conn, &config.Config, clientVersion, serverVersion)
	t.hostKeys = config.hostKeys
	go t.readLoop()
	go t.kexLoop()
	return t
}

func (t *handshakeTransport) getSessionID() []byte {
	return t.sessionID
}

func (t *handshakeTransport) waitSession() error {
	p, err := t.readPacket()
	if err != nil {
		return err
	}
	if p[0] != msgNewKeys {
		return fmt.Errorf("ssh: first packet should be msgNewKeys")
	}

	return nil
}

func (t *handshakeTransport) id() string {
	if len(t.hostKeys) > 0 {
		return "server"
	}
	return "client"
}

func (t *handshakeTransport) printPacket(p []byte, write bool) {
	action := "got"
	if write {
		action = "sent"
	}

	if p[0] == msgChannelData || p[0] == msgChannelExtendedData {
		log.Printf("%s %s data (packet %d bytes)", t.id(), action, len(p))
	} else {
		msg, err := decode(p)
		log.Printf("%s %s %T %v (%v)", t.id(), action, msg, msg, err)
	}
}

func (t *handshakeTransport) readPacket() ([]byte, error) {
	p, ok := <-t.incoming
	if !ok {
		return nil, t.readError
	}
	return p, nil
}

func (t *handshakeTransport) readLoop() {
	first := true
	for {
		p, err := t.readOnePacket(first)
		first = false
		if err != nil {
			t.readError = err
			close(t.incoming)
			break
		}
		if p[0] == msgIgnore || p[0] == msgDebug {
			continue
		}
		t.incoming <- p
	}

	t.recordWriteError(t.readError)

	close(t.startKex)

}

func (t *handshakeTransport) pushPacket(p []byte) error {
	if debugHandshake {
		t.printPacket(p, true)
	}
	return t.conn.writePacket(p)
}

func (t *handshakeTransport) getWriteError() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.writeError
}

func (t *handshakeTransport) recordWriteError(err error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.writeError == nil && err != nil {
		t.writeError = err
	}
}

func (t *handshakeTransport) requestKeyExchange() {
	select {
	case t.requestKex <- struct{}{}:
	default:

	}
}

func (t *handshakeTransport) resetWriteThresholds() {
	t.writePacketsLeft = packetRekeyThreshold
	if t.config.RekeyThreshold > 0 {
		t.writeBytesLeft = int64(t.config.RekeyThreshold)
	} else if t.algorithms != nil {
		t.writeBytesLeft = t.algorithms.w.rekeyBytes()
	} else {
		t.writeBytesLeft = 1 << 30
	}
}

func (t *handshakeTransport) kexLoop() {

write:
	for t.getWriteError() == nil {
		var request *pendingKex
		var sent bool

		for request == nil || !sent {
			var ok bool
			select {
			case request, ok = <-t.startKex:
				if !ok {
					break write
				}
			case <-t.requestKex:
				break
			}

			if !sent {
				if err := t.sendKexInit(); err != nil {
					t.recordWriteError(err)
					break
				}
				sent = true
			}
		}

		if err := t.getWriteError(); err != nil {
			if request != nil {
				request.done <- err
			}
			break
		}

		err := t.enterKeyExchange(request.otherInit)

		t.mu.Lock()
		t.writeError = err
		t.sentInitPacket = nil
		t.sentInitMsg = nil

		t.resetWriteThresholds()

	clear:
		for {
			select {
			case <-t.requestKex:

			default:
				break clear
			}
		}

		request.done <- t.writeError

		for _, p := range t.pendingPackets {
			t.writeError = t.pushPacket(p)
			if t.writeError != nil {
				break
			}
		}
		t.pendingPackets = t.pendingPackets[:0]
		t.mu.Unlock()
	}

	go func() {
		for init := range t.startKex {
			init.done <- t.writeError
		}
	}()

	t.conn.Close()
}

const packetRekeyThreshold = (1 << 31)

func (t *handshakeTransport) resetReadThresholds() {
	t.readPacketsLeft = packetRekeyThreshold
	if t.config.RekeyThreshold > 0 {
		t.readBytesLeft = int64(t.config.RekeyThreshold)
	} else if t.algorithms != nil {
		t.readBytesLeft = t.algorithms.r.rekeyBytes()
	} else {
		t.readBytesLeft = 1 << 30
	}
}

func (t *handshakeTransport) readOnePacket(first bool) ([]byte, error) {
	p, err := t.conn.readPacket()
	if err != nil {
		return nil, err
	}

	if t.readPacketsLeft > 0 {
		t.readPacketsLeft--
	} else {
		t.requestKeyExchange()
	}

	if t.readBytesLeft > 0 {
		t.readBytesLeft -= int64(len(p))
	} else {
		t.requestKeyExchange()
	}

	if debugHandshake {
		t.printPacket(p, false)
	}

	if first && p[0] != msgKexInit {
		return nil, fmt.Errorf("ssh: first packet should be msgKexInit")
	}

	if p[0] != msgKexInit {
		return p, nil
	}

	firstKex := t.sessionID == nil

	kex := pendingKex{
		done:      make(chan error, 1),
		otherInit: p,
	}
	t.startKex <- &kex
	err = <-kex.done

	if debugHandshake {
		log.Printf("%s exited key exchange (first %v), err %v", t.id(), firstKex, err)
	}

	if err != nil {
		return nil, err
	}

	t.resetReadThresholds()

	successPacket := []byte{msgIgnore}
	if firstKex {

		successPacket = []byte{msgNewKeys}
	}

	return successPacket, nil
}

func (t *handshakeTransport) sendKexInit() error {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.sentInitMsg != nil {

		return nil
	}

	msg := &kexInitMsg{
		KexAlgos:                t.config.KeyExchanges,
		CiphersClientServer:     t.config.Ciphers,
		CiphersServerClient:     t.config.Ciphers,
		MACsClientServer:        t.config.MACs,
		MACsServerClient:        t.config.MACs,
		CompressionClientServer: supportedCompressions,
		CompressionServerClient: supportedCompressions,
	}
	io.ReadFull(rand.Reader, msg.Cookie[:])

	if len(t.hostKeys) > 0 {
		for _, k := range t.hostKeys {
			msg.ServerHostKeyAlgos = append(
				msg.ServerHostKeyAlgos, k.PublicKey().Type())
		}
	} else {
		msg.ServerHostKeyAlgos = t.hostKeyAlgorithms
	}
	packet := Marshal(msg)

	packetCopy := make([]byte, len(packet))
	copy(packetCopy, packet)

	if err := t.pushPacket(packetCopy); err != nil {
		return err
	}

	t.sentInitMsg = msg
	t.sentInitPacket = packet

	return nil
}

func (t *handshakeTransport) writePacket(p []byte) error {
	switch p[0] {
	case msgKexInit:
		return errors.New("ssh: only handshakeTransport can send kexInit")
	case msgNewKeys:
		return errors.New("ssh: only handshakeTransport can send newKeys")
	}

	t.mu.Lock()
	defer t.mu.Unlock()
	if t.writeError != nil {
		return t.writeError
	}

	if t.sentInitMsg != nil {

		cp := make([]byte, len(p))
		copy(cp, p)
		t.pendingPackets = append(t.pendingPackets, cp)
		return nil
	}

	if t.writeBytesLeft > 0 {
		t.writeBytesLeft -= int64(len(p))
	} else {
		t.requestKeyExchange()
	}

	if t.writePacketsLeft > 0 {
		t.writePacketsLeft--
	} else {
		t.requestKeyExchange()
	}

	if err := t.pushPacket(p); err != nil {
		t.writeError = err
	}

	return nil
}

func (t *handshakeTransport) Close() error {
	return t.conn.Close()
}

func (t *handshakeTransport) enterKeyExchange(otherInitPacket []byte) error {
	if debugHandshake {
		log.Printf("%s entered key exchange", t.id())
	}

	otherInit := &kexInitMsg{}
	if err := Unmarshal(otherInitPacket, otherInit); err != nil {
		return err
	}

	magics := handshakeMagics{
		clientVersion: t.clientVersion,
		serverVersion: t.serverVersion,
		clientKexInit: otherInitPacket,
		serverKexInit: t.sentInitPacket,
	}

	clientInit := otherInit
	serverInit := t.sentInitMsg
	if len(t.hostKeys) == 0 {
		clientInit, serverInit = serverInit, clientInit

		magics.clientKexInit = t.sentInitPacket
		magics.serverKexInit = otherInitPacket
	}

	var err error
	t.algorithms, err = findAgreedAlgorithms(clientInit, serverInit)
	if err != nil {
		return err
	}

	if otherInit.FirstKexFollows && (clientInit.KexAlgos[0] != serverInit.KexAlgos[0] || clientInit.ServerHostKeyAlgos[0] != serverInit.ServerHostKeyAlgos[0]) {

		if _, err := t.conn.readPacket(); err != nil {
			return err
		}
	}

	kex, ok := kexAlgoMap[t.algorithms.kex]
	if !ok {
		return fmt.Errorf("ssh: unexpected key exchange algorithm %v", t.algorithms.kex)
	}

	var result *kexResult
	if len(t.hostKeys) > 0 {
		result, err = t.server(kex, t.algorithms, &magics)
	} else {
		result, err = t.client(kex, t.algorithms, &magics)
	}

	if err != nil {
		return err
	}

	if t.sessionID == nil {
		t.sessionID = result.H
	}
	result.SessionID = t.sessionID

	if err := t.conn.prepareKeyChange(t.algorithms, result); err != nil {
		return err
	}
	if err = t.conn.writePacket([]byte{msgNewKeys}); err != nil {
		return err
	}
	if packet, err := t.conn.readPacket(); err != nil {
		return err
	} else if packet[0] != msgNewKeys {
		return unexpectedMessageError(msgNewKeys, packet[0])
	}

	return nil
}

func (t *handshakeTransport) server(kex kexAlgorithm, algs *algorithms, magics *handshakeMagics) (*kexResult, error) {
	var hostKey Signer
	for _, k := range t.hostKeys {
		if algs.hostKey == k.PublicKey().Type() {
			hostKey = k
		}
	}

	r, err := kex.Server(t.conn, t.config.Rand, magics, hostKey)
	return r, err
}

func (t *handshakeTransport) client(kex kexAlgorithm, algs *algorithms, magics *handshakeMagics) (*kexResult, error) {
	result, err := kex.Client(t.conn, t.config.Rand, magics)
	if err != nil {
		return nil, err
	}

	hostKey, err := ParsePublicKey(result.HostKey)
	if err != nil {
		return nil, err
	}

	if err := verifyHostKeySignature(hostKey, result); err != nil {
		return nil, err
	}

	err = t.hostKeyCallback(t.dialAddress, t.remoteAddr, hostKey)
	if err != nil {
		return nil, err
	}

	return result, nil
}
