
package ssh

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"
)

type Client struct {
	Conn

	forwards        forwardList 
	mu              sync.Mutex
	channelHandlers map[string]chan NewChannel
}

func (c *Client) HandleChannelOpen(channelType string) <-chan NewChannel {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.channelHandlers == nil {

		c := make(chan NewChannel)
		close(c)
		return c
	}

	ch := c.channelHandlers[channelType]
	if ch != nil {
		return nil
	}

	ch = make(chan NewChannel, chanSize)
	c.channelHandlers[channelType] = ch
	return ch
}

func NewClient(c Conn, chans <-chan NewChannel, reqs <-chan *Request) *Client {
	conn := &Client{
		Conn:            c,
		channelHandlers: make(map[string]chan NewChannel, 1),
	}

	go conn.handleGlobalRequests(reqs)
	go conn.handleChannelOpens(chans)
	go func() {
		conn.Wait()
		conn.forwards.closeAll()
	}()
	go conn.forwards.handleChannels(conn.HandleChannelOpen("forwarded-tcpip"))
	go conn.forwards.handleChannels(conn.HandleChannelOpen("forwarded-streamlocal@openssh.com"))
	return conn
}

func NewClientConn(c net.Conn, addr string, config *ClientConfig) (Conn, <-chan NewChannel, <-chan *Request, error) {
	fullConf := *config
	fullConf.SetDefaults()
	if fullConf.HostKeyCallback == nil {
		c.Close()
		return nil, nil, nil, errors.New("ssh: must specify HostKeyCallback")
	}

	conn := &connection{
		sshConn: sshConn{conn: c},
	}

	if err := conn.clientHandshake(addr, &fullConf); err != nil {
		c.Close()
		return nil, nil, nil, fmt.Errorf("ssh: handshake failed: %v", err)
	}
	conn.mux = newMux(conn.transport)
	return conn, conn.mux.incomingChannels, conn.mux.incomingRequests, nil
}

func (c *connection) clientHandshake(dialAddress string, config *ClientConfig) error {
	if config.ClientVersion != "" {
		c.clientVersion = []byte(config.ClientVersion)
	} else {
		c.clientVersion = []byte(packageVersion)
	}
	var err error
	c.serverVersion, err = exchangeVersions(c.sshConn.conn, c.clientVersion)
	if err != nil {
		return err
	}

	c.transport = newClientTransport(
		newTransport(c.sshConn.conn, config.Rand, true ),
		c.clientVersion, c.serverVersion, config, dialAddress, c.sshConn.RemoteAddr())
	if err := c.transport.waitSession(); err != nil {
		return err
	}

	c.sessionID = c.transport.getSessionID()
	return c.clientAuthenticate(config)
}

func verifyHostKeySignature(hostKey PublicKey, result *kexResult) error {
	sig, rest, ok := parseSignatureBody(result.Signature)
	if len(rest) > 0 || !ok {
		return errors.New("ssh: signature parse error")
	}

	return hostKey.Verify(result.H, sig)
}

func (c *Client) NewSession() (*Session, error) {
	ch, in, err := c.OpenChannel("session", nil)
	if err != nil {
		return nil, err
	}
	return newSession(ch, in)
}

func (c *Client) handleGlobalRequests(incoming <-chan *Request) {
	for r := range incoming {

		r.Reply(false, nil)
	}
}

func (c *Client) handleChannelOpens(in <-chan NewChannel) {
	for ch := range in {
		c.mu.Lock()
		handler := c.channelHandlers[ch.ChannelType()]
		c.mu.Unlock()

		if handler != nil {
			handler <- ch
		} else {
			ch.Reject(UnknownChannelType, fmt.Sprintf("unknown channel type: %v", ch.ChannelType()))
		}
	}

	c.mu.Lock()
	for _, ch := range c.channelHandlers {
		close(ch)
	}
	c.channelHandlers = nil
	c.mu.Unlock()
}

func Dial(network, addr string, config *ClientConfig) (*Client, error) {
	conn, err := net.DialTimeout(network, addr, config.Timeout)
	if err != nil {
		return nil, err
	}
	c, chans, reqs, err := NewClientConn(conn, addr, config)
	if err != nil {
		return nil, err
	}
	return NewClient(c, chans, reqs), nil
}

type HostKeyCallback func(hostname string, remote net.Addr, key PublicKey) error

type ClientConfig struct {

	Config

	User string

	Auth []AuthMethod

	HostKeyCallback HostKeyCallback

	ClientVersion string

	HostKeyAlgorithms []string

	Timeout time.Duration
}

func InsecureIgnoreHostKey() HostKeyCallback {
	return func(hostname string, remote net.Addr, key PublicKey) error {
		return nil
	}
}

type fixedHostKey struct {
	key PublicKey
}

func (f *fixedHostKey) check(hostname string, remote net.Addr, key PublicKey) error {
	if f.key == nil {
		return fmt.Errorf("ssh: required host key was nil")
	}
	if !bytes.Equal(key.Marshal(), f.key.Marshal()) {
		return fmt.Errorf("ssh: host key mismatch")
	}
	return nil
}

func FixedHostKey(key PublicKey) HostKeyCallback {
	hk := &fixedHostKey{key}
	return hk.check
}
