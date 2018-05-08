package ssh

import (
	"errors"
	"io"
	"net"
)

type streamLocalChannelOpenDirectMsg struct {
	socketPath string
	reserved0  string
	reserved1  uint32
}

type forwardedStreamLocalPayload struct {
	SocketPath string
	Reserved0  string
}

type streamLocalChannelForwardMsg struct {
	socketPath string
}

func (c *Client) ListenUnix(socketPath string) (net.Listener, error) {
	m := streamLocalChannelForwardMsg{
		socketPath,
	}

	ok, _, err := c.SendRequest("streamlocal-forward@openssh.com", true, Marshal(&m))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("ssh: streamlocal-forward@openssh.com request denied by peer")
	}
	ch := c.forwards.add(&net.UnixAddr{Name: socketPath, Net: "unix"})

	return &unixListener{socketPath, c, ch}, nil
}

func (c *Client) dialStreamLocal(socketPath string) (Channel, error) {
	msg := streamLocalChannelOpenDirectMsg{
		socketPath: socketPath,
	}
	ch, in, err := c.OpenChannel("direct-streamlocal@openssh.com", Marshal(&msg))
	if err != nil {
		return nil, err
	}
	go DiscardRequests(in)
	return ch, err
}

type unixListener struct {
	socketPath string

	conn *Client
	in   <-chan forward
}

func (l *unixListener) Accept() (net.Conn, error) {
	s, ok := <-l.in
	if !ok {
		return nil, io.EOF
	}
	ch, incoming, err := s.newCh.Accept()
	if err != nil {
		return nil, err
	}
	go DiscardRequests(incoming)

	return &chanConn{
		Channel: ch,
		laddr: &net.UnixAddr{
			Name: l.socketPath,
			Net:  "unix",
		},
		raddr: &net.UnixAddr{
			Name: "@",
			Net:  "unix",
		},
	}, nil
}

func (l *unixListener) Close() error {

	l.conn.forwards.remove(&net.UnixAddr{Name: l.socketPath, Net: "unix"})
	m := streamLocalChannelForwardMsg{
		l.socketPath,
	}
	ok, _, err := l.conn.SendRequest("cancel-streamlocal-forward@openssh.com", true, Marshal(&m))
	if err == nil && !ok {
		err = errors.New("ssh: cancel-streamlocal-forward@openssh.com failed")
	}
	return err
}

func (l *unixListener) Addr() net.Addr {
	return &net.UnixAddr{
		Name: l.socketPath,
		Net:  "unix",
	}
}
