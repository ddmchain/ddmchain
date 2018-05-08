package httpu

import (
	"bufio"
	"bytes"
	"log"
	"net"
	"net/http"
	"regexp"
)

const (
	DefaultMaxMessageBytes = 2048
)

var (
	trailingWhitespaceRx = regexp.MustCompile(" +\r\n")
	crlf                 = []byte("\r\n")
)

type Handler interface {

	ServeMessage(r *http.Request)
}

type HandlerFunc func(r *http.Request)

func (f HandlerFunc) ServeMessage(r *http.Request) {
	f(r)
}

type Server struct {
	Addr            string         
	Multicast       bool           
	Interface       *net.Interface 
	Handler         Handler        
	MaxMessageBytes int            
}

func (srv *Server) ListenAndServe() error {
	var err error

	var addr *net.UDPAddr
	if addr, err = net.ResolveUDPAddr("udp", srv.Addr); err != nil {
		log.Fatal(err)
	}

	var conn net.PacketConn
	if srv.Multicast {
		if conn, err = net.ListenMulticastUDP("udp", srv.Interface, addr); err != nil {
			return err
		}
	} else {
		if conn, err = net.ListenUDP("udp", addr); err != nil {
			return err
		}
	}

	return srv.Serve(conn)
}

func (srv *Server) Serve(l net.PacketConn) error {
	maxMessageBytes := DefaultMaxMessageBytes
	if srv.MaxMessageBytes != 0 {
		maxMessageBytes = srv.MaxMessageBytes
	}
	for {
		buf := make([]byte, maxMessageBytes)
		n, peerAddr, err := l.ReadFrom(buf)
		if err != nil {
			return err
		}
		buf = buf[:n]

		go func(buf []byte, peerAddr net.Addr) {

			buf = trailingWhitespaceRx.ReplaceAllLiteral(buf, crlf)

			req, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(buf)))
			if err != nil {
				log.Printf("httpu: Failed to parse request: %v", err)
				return
			}
			req.RemoteAddr = peerAddr.String()
			srv.Handler.ServeMessage(req)

		}(buf, peerAddr)
	}
}

func Serve(l net.PacketConn, handler Handler) error {
	srv := Server{
		Handler:         handler,
		MaxMessageBytes: DefaultMaxMessageBytes,
	}
	return srv.Serve(l)
}
