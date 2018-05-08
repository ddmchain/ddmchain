
package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

func (c *connection) clientAuthenticate(config *ClientConfig) error {

	if err := c.transport.writePacket(Marshal(&serviceRequestMsg{serviceUserAuth})); err != nil {
		return err
	}
	packet, err := c.transport.readPacket()
	if err != nil {
		return err
	}
	var serviceAccept serviceAcceptMsg
	if err := Unmarshal(packet, &serviceAccept); err != nil {
		return err
	}

	tried := make(map[string]bool)
	var lastMethods []string

	sessionID := c.transport.getSessionID()
	for auth := AuthMethod(new(noneAuth)); auth != nil; {
		ok, methods, err := auth.auth(sessionID, config.User, c.transport, config.Rand)
		if err != nil {
			return err
		}
		if ok {

			return nil
		}
		tried[auth.method()] = true
		if methods == nil {
			methods = lastMethods
		}
		lastMethods = methods

		auth = nil

	findNext:
		for _, a := range config.Auth {
			candidateMethod := a.method()
			if tried[candidateMethod] {
				continue
			}
			for _, meth := range methods {
				if meth == candidateMethod {
					auth = a
					break findNext
				}
			}
		}
	}
	return fmt.Errorf("ssh: unable to authenticate, attempted methods %v, no supported methods remain", keys(tried))
}

func keys(m map[string]bool) []string {
	s := make([]string, 0, len(m))

	for key := range m {
		s = append(s, key)
	}
	return s
}

type AuthMethod interface {

	auth(session []byte, user string, p packetConn, rand io.Reader) (bool, []string, error)

	method() string
}

type noneAuth int

func (n *noneAuth) auth(session []byte, user string, c packetConn, rand io.Reader) (bool, []string, error) {
	if err := c.writePacket(Marshal(&userAuthRequestMsg{
		User:    user,
		Service: serviceSSH,
		Method:  "none",
	})); err != nil {
		return false, nil, err
	}

	return handleAuthResponse(c)
}

func (n *noneAuth) method() string {
	return "none"
}

type passwordCallback func() (password string, err error)

func (cb passwordCallback) auth(session []byte, user string, c packetConn, rand io.Reader) (bool, []string, error) {
	type passwordAuthMsg struct {
		User     string `sshtype:"50"`
		Service  string
		Method   string
		Reply    bool
		Password string
	}

	pw, err := cb()

	if err != nil {
		return false, nil, err
	}

	if err := c.writePacket(Marshal(&passwordAuthMsg{
		User:     user,
		Service:  serviceSSH,
		Method:   cb.method(),
		Reply:    false,
		Password: pw,
	})); err != nil {
		return false, nil, err
	}

	return handleAuthResponse(c)
}

func (cb passwordCallback) method() string {
	return "password"
}

func Password(secret string) AuthMethod {
	return passwordCallback(func() (string, error) { return secret, nil })
}

func PasswordCallback(prompt func() (secret string, err error)) AuthMethod {
	return passwordCallback(prompt)
}

type publickeyAuthMsg struct {
	User    string `sshtype:"50"`
	Service string
	Method  string

	HasSig   bool
	Algoname string
	PubKey   []byte

	Sig []byte `ssh:"rest"`
}

type publicKeyCallback func() ([]Signer, error)

func (cb publicKeyCallback) method() string {
	return "publickey"
}

func (cb publicKeyCallback) auth(session []byte, user string, c packetConn, rand io.Reader) (bool, []string, error) {

	signers, err := cb()
	if err != nil {
		return false, nil, err
	}
	var methods []string
	for _, signer := range signers {
		ok, err := validateKey(signer.PublicKey(), user, c)
		if err != nil {
			return false, nil, err
		}
		if !ok {
			continue
		}

		pub := signer.PublicKey()
		pubKey := pub.Marshal()
		sign, err := signer.Sign(rand, buildDataSignedForAuth(session, userAuthRequestMsg{
			User:    user,
			Service: serviceSSH,
			Method:  cb.method(),
		}, []byte(pub.Type()), pubKey))
		if err != nil {
			return false, nil, err
		}

		s := Marshal(sign)
		sig := make([]byte, stringLength(len(s)))
		marshalString(sig, s)
		msg := publickeyAuthMsg{
			User:     user,
			Service:  serviceSSH,
			Method:   cb.method(),
			HasSig:   true,
			Algoname: pub.Type(),
			PubKey:   pubKey,
			Sig:      sig,
		}
		p := Marshal(&msg)
		if err := c.writePacket(p); err != nil {
			return false, nil, err
		}
		var success bool
		success, methods, err = handleAuthResponse(c)
		if err != nil {
			return false, nil, err
		}

		if success || !containsMethod(methods, cb.method()) {
			return success, methods, err
		}
	}

	return false, methods, nil
}

func containsMethod(methods []string, method string) bool {
	for _, m := range methods {
		if m == method {
			return true
		}
	}

	return false
}

func validateKey(key PublicKey, user string, c packetConn) (bool, error) {
	pubKey := key.Marshal()
	msg := publickeyAuthMsg{
		User:     user,
		Service:  serviceSSH,
		Method:   "publickey",
		HasSig:   false,
		Algoname: key.Type(),
		PubKey:   pubKey,
	}
	if err := c.writePacket(Marshal(&msg)); err != nil {
		return false, err
	}

	return confirmKeyAck(key, c)
}

func confirmKeyAck(key PublicKey, c packetConn) (bool, error) {
	pubKey := key.Marshal()
	algoname := key.Type()

	for {
		packet, err := c.readPacket()
		if err != nil {
			return false, err
		}
		switch packet[0] {
		case msgUserAuthBanner:

		case msgUserAuthPubKeyOk:
			var msg userAuthPubKeyOkMsg
			if err := Unmarshal(packet, &msg); err != nil {
				return false, err
			}
			if msg.Algo != algoname || !bytes.Equal(msg.PubKey, pubKey) {
				return false, nil
			}
			return true, nil
		case msgUserAuthFailure:
			return false, nil
		default:
			return false, unexpectedMessageError(msgUserAuthSuccess, packet[0])
		}
	}
}

func PublicKeys(signers ...Signer) AuthMethod {
	return publicKeyCallback(func() ([]Signer, error) { return signers, nil })
}

func PublicKeysCallback(getSigners func() (signers []Signer, err error)) AuthMethod {
	return publicKeyCallback(getSigners)
}

func handleAuthResponse(c packetConn) (bool, []string, error) {
	for {
		packet, err := c.readPacket()
		if err != nil {
			return false, nil, err
		}

		switch packet[0] {
		case msgUserAuthBanner:

		case msgUserAuthFailure:
			var msg userAuthFailureMsg
			if err := Unmarshal(packet, &msg); err != nil {
				return false, nil, err
			}
			return false, msg.Methods, nil
		case msgUserAuthSuccess:
			return true, nil, nil
		default:
			return false, nil, unexpectedMessageError(msgUserAuthSuccess, packet[0])
		}
	}
}

type KeyboardInteractiveChallenge func(user, instruction string, questions []string, echos []bool) (answers []string, err error)

func KeyboardInteractive(challenge KeyboardInteractiveChallenge) AuthMethod {
	return challenge
}

func (cb KeyboardInteractiveChallenge) method() string {
	return "keyboard-interactive"
}

func (cb KeyboardInteractiveChallenge) auth(session []byte, user string, c packetConn, rand io.Reader) (bool, []string, error) {
	type initiateMsg struct {
		User       string `sshtype:"50"`
		Service    string
		Method     string
		Language   string
		Submethods string
	}

	if err := c.writePacket(Marshal(&initiateMsg{
		User:    user,
		Service: serviceSSH,
		Method:  "keyboard-interactive",
	})); err != nil {
		return false, nil, err
	}

	for {
		packet, err := c.readPacket()
		if err != nil {
			return false, nil, err
		}

		switch packet[0] {
		case msgUserAuthBanner:

			continue
		case msgUserAuthInfoRequest:

		case msgUserAuthFailure:
			var msg userAuthFailureMsg
			if err := Unmarshal(packet, &msg); err != nil {
				return false, nil, err
			}
			return false, msg.Methods, nil
		case msgUserAuthSuccess:
			return true, nil, nil
		default:
			return false, nil, unexpectedMessageError(msgUserAuthInfoRequest, packet[0])
		}

		var msg userAuthInfoRequestMsg
		if err := Unmarshal(packet, &msg); err != nil {
			return false, nil, err
		}

		rest := msg.Prompts
		var prompts []string
		var echos []bool
		for i := 0; i < int(msg.NumPrompts); i++ {
			prompt, r, ok := parseString(rest)
			if !ok || len(r) == 0 {
				return false, nil, errors.New("ssh: prompt format error")
			}
			prompts = append(prompts, string(prompt))
			echos = append(echos, r[0] != 0)
			rest = r[1:]
		}

		if len(rest) != 0 {
			return false, nil, errors.New("ssh: extra data following keyboard-interactive pairs")
		}

		answers, err := cb(msg.User, msg.Instruction, prompts, echos)
		if err != nil {
			return false, nil, err
		}

		if len(answers) != len(prompts) {
			return false, nil, errors.New("ssh: not enough answers from keyboard-interactive callback")
		}
		responseLength := 1 + 4
		for _, a := range answers {
			responseLength += stringLength(len(a))
		}
		serialized := make([]byte, responseLength)
		p := serialized
		p[0] = msgUserAuthInfoResponse
		p = p[1:]
		p = marshalUint32(p, uint32(len(answers)))
		for _, a := range answers {
			p = marshalString(p, []byte(a))
		}

		if err := c.writePacket(serialized); err != nil {
			return false, nil, err
		}
	}
}

type retryableAuthMethod struct {
	authMethod AuthMethod
	maxTries   int
}

func (r *retryableAuthMethod) auth(session []byte, user string, c packetConn, rand io.Reader) (ok bool, methods []string, err error) {
	for i := 0; r.maxTries <= 0 || i < r.maxTries; i++ {
		ok, methods, err = r.authMethod.auth(session, user, c, rand)
		if ok || err != nil { 
			return ok, methods, err
		}
	}
	return ok, methods, err
}

func (r *retryableAuthMethod) method() string {
	return r.authMethod.method()
}

func RetryableAuthMethod(auth AuthMethod, maxTries int) AuthMethod {
	return &retryableAuthMethod{authMethod: auth, maxTries: maxTries}
}
