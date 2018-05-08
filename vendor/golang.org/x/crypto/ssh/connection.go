
package ssh

import (
	"fmt"
	"net"
)

type OpenChannelError struct {
	Reason  RejectionReason
	Message string
}

func (e *OpenChannelError) Error() string {
	return fmt.Sprintf("ssh: rejected: %s (%s)", e.Reason, e.Message)
}

type ConnMetadata interface {

	User() string

	SessionID() []byte

	ClientVersion() []byte

	ServerVersion() []byte

	RemoteAddr() net.Addr

	LocalAddr() net.Addr
}

type Conn interface {
	ConnMetadata

	SendRequest(name string, wantReply bool, payload []byte) (bool, []byte, error)

	OpenChannel(name string, data []byte) (Channel, <-chan *Request, error)

	Close() error

	Wait() error

}

func DiscardRequests(in <-chan *Request) {
	for req := range in {
		if req.WantReply {
			req.Reply(false, nil)
		}
	}
}

type connection struct {
	transport *handshakeTransport
	sshConn

	*mux
}

func (c *connection) Close() error {
	return c.sshConn.conn.Close()
}

type sshConn struct {
	conn net.Conn

	user          string
	sessionID     []byte
	clientVersion []byte
	serverVersion []byte
}

func dup(src []byte) []byte {
	dst := make([]byte, len(src))
	copy(dst, src)
	return dst
}

func (c *sshConn) User() string {
	return c.user
}

func (c *sshConn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

func (c *sshConn) Close() error {
	return c.conn.Close()
}

func (c *sshConn) LocalAddr() net.Addr {
	return c.conn.LocalAddr()
}

func (c *sshConn) SessionID() []byte {
	return dup(c.sessionID)
}

func (c *sshConn) ClientVersion() []byte {
	return dup(c.clientVersion)
}

func (c *sshConn) ServerVersion() []byte {
	return dup(c.serverVersion)
}
