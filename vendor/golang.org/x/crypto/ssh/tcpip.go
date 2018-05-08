
package ssh

import (
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

func (c *Client) Listen(n, addr string) (net.Listener, error) {
	switch n {
	case "tcp", "tcp4", "tcp6":
		laddr, err := net.ResolveTCPAddr(n, addr)
		if err != nil {
			return nil, err
		}
		return c.ListenTCP(laddr)
	case "unix":
		return c.ListenUnix(addr)
	default:
		return nil, fmt.Errorf("ssh: unsupported protocol: %s", n)
	}
}

const openSSHPrefix = "OpenSSH_"

var portRandomizer = rand.New(rand.NewSource(time.Now().UnixNano()))

func isBrokenOpenSSHVersion(versionStr string) bool {
	i := strings.Index(versionStr, openSSHPrefix)
	if i < 0 {
		return false
	}
	i += len(openSSHPrefix)
	j := i
	for ; j < len(versionStr); j++ {
		if versionStr[j] < '0' || versionStr[j] > '9' {
			break
		}
	}
	version, _ := strconv.Atoi(versionStr[i:j])
	return version < 6
}

func (c *Client) autoPortListenWorkaround(laddr *net.TCPAddr) (net.Listener, error) {
	var sshListener net.Listener
	var err error
	const tries = 10
	for i := 0; i < tries; i++ {
		addr := *laddr
		addr.Port = 1024 + portRandomizer.Intn(60000)
		sshListener, err = c.ListenTCP(&addr)
		if err == nil {
			laddr.Port = addr.Port
			return sshListener, err
		}
	}
	return nil, fmt.Errorf("ssh: listen on random port failed after %d tries: %v", tries, err)
}

type channelForwardMsg struct {
	addr  string
	rport uint32
}

func (c *Client) ListenTCP(laddr *net.TCPAddr) (net.Listener, error) {
	if laddr.Port == 0 && isBrokenOpenSSHVersion(string(c.ServerVersion())) {
		return c.autoPortListenWorkaround(laddr)
	}

	m := channelForwardMsg{
		laddr.IP.String(),
		uint32(laddr.Port),
	}

	ok, resp, err := c.SendRequest("tcpip-forward", true, Marshal(&m))
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("ssh: tcpip-forward request denied by peer")
	}

	if laddr.Port == 0 {
		var p struct {
			Port uint32
		}
		if err := Unmarshal(resp, &p); err != nil {
			return nil, err
		}
		laddr.Port = int(p.Port)
	}

	ch := c.forwards.add(laddr)

	return &tcpListener{laddr, c, ch}, nil
}

type forwardList struct {
	sync.Mutex
	entries []forwardEntry
}

type forwardEntry struct {
	laddr net.Addr
	c     chan forward
}

type forward struct {
	newCh NewChannel 
	raddr net.Addr   
}

func (l *forwardList) add(addr net.Addr) chan forward {
	l.Lock()
	defer l.Unlock()
	f := forwardEntry{
		laddr: addr,
		c:     make(chan forward, 1),
	}
	l.entries = append(l.entries, f)
	return f.c
}

type forwardedTCPPayload struct {
	Addr       string
	Port       uint32
	OriginAddr string
	OriginPort uint32
}

func parseTCPAddr(addr string, port uint32) (*net.TCPAddr, error) {
	if port == 0 || port > 65535 {
		return nil, fmt.Errorf("ssh: port number out of range: %d", port)
	}
	ip := net.ParseIP(string(addr))
	if ip == nil {
		return nil, fmt.Errorf("ssh: cannot parse IP address %q", addr)
	}
	return &net.TCPAddr{IP: ip, Port: int(port)}, nil
}

func (l *forwardList) handleChannels(in <-chan NewChannel) {
	for ch := range in {
		var (
			laddr net.Addr
			raddr net.Addr
			err   error
		)
		switch channelType := ch.ChannelType(); channelType {
		case "forwarded-tcpip":
			var payload forwardedTCPPayload
			if err = Unmarshal(ch.ExtraData(), &payload); err != nil {
				ch.Reject(ConnectionFailed, "could not parse forwarded-tcpip payload: "+err.Error())
				continue
			}

			laddr, err = parseTCPAddr(payload.Addr, payload.Port)
			if err != nil {
				ch.Reject(ConnectionFailed, err.Error())
				continue
			}
			raddr, err = parseTCPAddr(payload.OriginAddr, payload.OriginPort)
			if err != nil {
				ch.Reject(ConnectionFailed, err.Error())
				continue
			}

		case "forwarded-streamlocal@openssh.com":
			var payload forwardedStreamLocalPayload
			if err = Unmarshal(ch.ExtraData(), &payload); err != nil {
				ch.Reject(ConnectionFailed, "could not parse forwarded-streamlocal@openssh.com payload: "+err.Error())
				continue
			}
			laddr = &net.UnixAddr{
				Name: payload.SocketPath,
				Net:  "unix",
			}
			raddr = &net.UnixAddr{
				Name: "@",
				Net:  "unix",
			}
		default:
			panic(fmt.Errorf("ssh: unknown channel type %s", channelType))
		}
		if ok := l.forward(laddr, raddr, ch); !ok {

			ch.Reject(Prohibited, "no forward for address")
			continue
		}

	}
}

func (l *forwardList) remove(addr net.Addr) {
	l.Lock()
	defer l.Unlock()
	for i, f := range l.entries {
		if addr.Network() == f.laddr.Network() && addr.String() == f.laddr.String() {
			l.entries = append(l.entries[:i], l.entries[i+1:]...)
			close(f.c)
			return
		}
	}
}

func (l *forwardList) closeAll() {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		close(f.c)
	}
	l.entries = nil
}

func (l *forwardList) forward(laddr, raddr net.Addr, ch NewChannel) bool {
	l.Lock()
	defer l.Unlock()
	for _, f := range l.entries {
		if laddr.Network() == f.laddr.Network() && laddr.String() == f.laddr.String() {
			f.c <- forward{newCh: ch, raddr: raddr}
			return true
		}
	}
	return false
}

type tcpListener struct {
	laddr *net.TCPAddr

	conn *Client
	in   <-chan forward
}

func (l *tcpListener) Accept() (net.Conn, error) {
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
		laddr:   l.laddr,
		raddr:   s.raddr,
	}, nil
}

func (l *tcpListener) Close() error {
	m := channelForwardMsg{
		l.laddr.IP.String(),
		uint32(l.laddr.Port),
	}

	l.conn.forwards.remove(l.laddr)
	ok, _, err := l.conn.SendRequest("cancel-tcpip-forward", true, Marshal(&m))
	if err == nil && !ok {
		err = errors.New("ssh: cancel-tcpip-forward failed")
	}
	return err
}

func (l *tcpListener) Addr() net.Addr {
	return l.laddr
}

func (c *Client) Dial(n, addr string) (net.Conn, error) {
	var ch Channel
	switch n {
	case "tcp", "tcp4", "tcp6":

		host, portString, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		port, err := strconv.ParseUint(portString, 10, 16)
		if err != nil {
			return nil, err
		}
		ch, err = c.dial(net.IPv4zero.String(), 0, host, int(port))
		if err != nil {
			return nil, err
		}

		zeroAddr := &net.TCPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		}
		return &chanConn{
			Channel: ch,
			laddr:   zeroAddr,
			raddr:   zeroAddr,
		}, nil
	case "unix":
		var err error
		ch, err = c.dialStreamLocal(addr)
		if err != nil {
			return nil, err
		}
		return &chanConn{
			Channel: ch,
			laddr: &net.UnixAddr{
				Name: "@",
				Net:  "unix",
			},
			raddr: &net.UnixAddr{
				Name: addr,
				Net:  "unix",
			},
		}, nil
	default:
		return nil, fmt.Errorf("ssh: unsupported protocol: %s", n)
	}
}

func (c *Client) DialTCP(n string, laddr, raddr *net.TCPAddr) (net.Conn, error) {
	if laddr == nil {
		laddr = &net.TCPAddr{
			IP:   net.IPv4zero,
			Port: 0,
		}
	}
	ch, err := c.dial(laddr.IP.String(), laddr.Port, raddr.IP.String(), raddr.Port)
	if err != nil {
		return nil, err
	}
	return &chanConn{
		Channel: ch,
		laddr:   laddr,
		raddr:   raddr,
	}, nil
}

type channelOpenDirectMsg struct {
	raddr string
	rport uint32
	laddr string
	lport uint32
}

func (c *Client) dial(laddr string, lport int, raddr string, rport int) (Channel, error) {
	msg := channelOpenDirectMsg{
		raddr: raddr,
		rport: uint32(rport),
		laddr: laddr,
		lport: uint32(lport),
	}
	ch, in, err := c.OpenChannel("direct-tcpip", Marshal(&msg))
	if err != nil {
		return nil, err
	}
	go DiscardRequests(in)
	return ch, err
}

type tcpChan struct {
	Channel 
}

type chanConn struct {
	Channel
	laddr, raddr net.Addr
}

func (t *chanConn) LocalAddr() net.Addr {
	return t.laddr
}

func (t *chanConn) RemoteAddr() net.Addr {
	return t.raddr
}

func (t *chanConn) SetDeadline(deadline time.Time) error {
	if err := t.SetReadDeadline(deadline); err != nil {
		return err
	}
	return t.SetWriteDeadline(deadline)
}

func (t *chanConn) SetReadDeadline(deadline time.Time) error {

	return errors.New("ssh: tcpChan: deadline not supported")
}

func (t *chanConn) SetWriteDeadline(deadline time.Time) error {
	return errors.New("ssh: tcpChan: deadline not supported")
}
