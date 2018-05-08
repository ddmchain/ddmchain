
package websocket 

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

const (
	ProtocolVersionHybi13    = 13
	ProtocolVersionHybi      = ProtocolVersionHybi13
	SupportedProtocolVersion = "13"

	ContinuationFrame = 0
	TextFrame         = 1
	BinaryFrame       = 2
	CloseFrame        = 8
	PingFrame         = 9
	PongFrame         = 10
	UnknownFrame      = 255

	DefaultMaxPayloadBytes = 32 << 20 
)

type ProtocolError struct {
	ErrorString string
}

func (err *ProtocolError) Error() string { return err.ErrorString }

var (
	ErrBadProtocolVersion   = &ProtocolError{"bad protocol version"}
	ErrBadScheme            = &ProtocolError{"bad scheme"}
	ErrBadStatus            = &ProtocolError{"bad status"}
	ErrBadUpgrade           = &ProtocolError{"missing or bad upgrade"}
	ErrBadWebSocketOrigin   = &ProtocolError{"missing or bad WebSocket-Origin"}
	ErrBadWebSocketLocation = &ProtocolError{"missing or bad WebSocket-Location"}
	ErrBadWebSocketProtocol = &ProtocolError{"missing or bad WebSocket-Protocol"}
	ErrBadWebSocketVersion  = &ProtocolError{"missing or bad WebSocket Version"}
	ErrChallengeResponse    = &ProtocolError{"mismatch challenge/response"}
	ErrBadFrame             = &ProtocolError{"bad frame"}
	ErrBadFrameBoundary     = &ProtocolError{"not on frame boundary"}
	ErrNotWebSocket         = &ProtocolError{"not websocket protocol"}
	ErrBadRequestMethod     = &ProtocolError{"bad method"}
	ErrNotSupported         = &ProtocolError{"not supported"}
)

var ErrFrameTooLarge = errors.New("websocket: frame payload size exceeds limit")

type Addr struct {
	*url.URL
}

func (addr *Addr) Network() string { return "websocket" }

type Config struct {

	Location *url.URL

	Origin *url.URL

	Protocol []string

	Version int

	TlsConfig *tls.Config

	Header http.Header

	Dialer *net.Dialer

	handshakeData map[string]string
}

type serverHandshaker interface {

	ReadHandshake(buf *bufio.Reader, req *http.Request) (code int, err error)

	AcceptHandshake(buf *bufio.Writer) (err error)

	NewServerConn(buf *bufio.ReadWriter, rwc io.ReadWriteCloser, request *http.Request) (conn *Conn)
}

type frameReader interface {

	io.Reader

	PayloadType() byte

	HeaderReader() io.Reader

	TrailerReader() io.Reader

	Len() int
}

type frameReaderFactory interface {
	NewFrameReader() (r frameReader, err error)
}

type frameWriter interface {

	io.WriteCloser
}

type frameWriterFactory interface {
	NewFrameWriter(payloadType byte) (w frameWriter, err error)
}

type frameHandler interface {
	HandleFrame(frame frameReader) (r frameReader, err error)
	WriteClose(status int) (err error)
}

type Conn struct {
	config  *Config
	request *http.Request

	buf *bufio.ReadWriter
	rwc io.ReadWriteCloser

	rio sync.Mutex
	frameReaderFactory
	frameReader

	wio sync.Mutex
	frameWriterFactory

	frameHandler
	PayloadType        byte
	defaultCloseStatus int

	MaxPayloadBytes int
}

func (ws *Conn) Read(msg []byte) (n int, err error) {
	ws.rio.Lock()
	defer ws.rio.Unlock()
again:
	if ws.frameReader == nil {
		frame, err := ws.frameReaderFactory.NewFrameReader()
		if err != nil {
			return 0, err
		}
		ws.frameReader, err = ws.frameHandler.HandleFrame(frame)
		if err != nil {
			return 0, err
		}
		if ws.frameReader == nil {
			goto again
		}
	}
	n, err = ws.frameReader.Read(msg)
	if err == io.EOF {
		if trailer := ws.frameReader.TrailerReader(); trailer != nil {
			io.Copy(ioutil.Discard, trailer)
		}
		ws.frameReader = nil
		goto again
	}
	return n, err
}

func (ws *Conn) Write(msg []byte) (n int, err error) {
	ws.wio.Lock()
	defer ws.wio.Unlock()
	w, err := ws.frameWriterFactory.NewFrameWriter(ws.PayloadType)
	if err != nil {
		return 0, err
	}
	n, err = w.Write(msg)
	w.Close()
	return n, err
}

func (ws *Conn) Close() error {
	err := ws.frameHandler.WriteClose(ws.defaultCloseStatus)
	err1 := ws.rwc.Close()
	if err != nil {
		return err
	}
	return err1
}

func (ws *Conn) IsClientConn() bool { return ws.request == nil }
func (ws *Conn) IsServerConn() bool { return ws.request != nil }

func (ws *Conn) LocalAddr() net.Addr {
	if ws.IsClientConn() {
		return &Addr{ws.config.Origin}
	}
	return &Addr{ws.config.Location}
}

func (ws *Conn) RemoteAddr() net.Addr {
	if ws.IsClientConn() {
		return &Addr{ws.config.Location}
	}
	return &Addr{ws.config.Origin}
}

var errSetDeadline = errors.New("websocket: cannot set deadline: not using a net.Conn")

func (ws *Conn) SetDeadline(t time.Time) error {
	if conn, ok := ws.rwc.(net.Conn); ok {
		return conn.SetDeadline(t)
	}
	return errSetDeadline
}

func (ws *Conn) SetReadDeadline(t time.Time) error {
	if conn, ok := ws.rwc.(net.Conn); ok {
		return conn.SetReadDeadline(t)
	}
	return errSetDeadline
}

func (ws *Conn) SetWriteDeadline(t time.Time) error {
	if conn, ok := ws.rwc.(net.Conn); ok {
		return conn.SetWriteDeadline(t)
	}
	return errSetDeadline
}

func (ws *Conn) Config() *Config { return ws.config }

func (ws *Conn) Request() *http.Request { return ws.request }

type Codec struct {
	Marshal   func(v interface{}) (data []byte, payloadType byte, err error)
	Unmarshal func(data []byte, payloadType byte, v interface{}) (err error)
}

func (cd Codec) Send(ws *Conn, v interface{}) (err error) {
	data, payloadType, err := cd.Marshal(v)
	if err != nil {
		return err
	}
	ws.wio.Lock()
	defer ws.wio.Unlock()
	w, err := ws.frameWriterFactory.NewFrameWriter(payloadType)
	if err != nil {
		return err
	}
	_, err = w.Write(data)
	w.Close()
	return err
}

func (cd Codec) Receive(ws *Conn, v interface{}) (err error) {
	ws.rio.Lock()
	defer ws.rio.Unlock()
	if ws.frameReader != nil {
		_, err = io.Copy(ioutil.Discard, ws.frameReader)
		if err != nil {
			return err
		}
		ws.frameReader = nil
	}
again:
	frame, err := ws.frameReaderFactory.NewFrameReader()
	if err != nil {
		return err
	}
	frame, err = ws.frameHandler.HandleFrame(frame)
	if err != nil {
		return err
	}
	if frame == nil {
		goto again
	}
	maxPayloadBytes := ws.MaxPayloadBytes
	if maxPayloadBytes == 0 {
		maxPayloadBytes = DefaultMaxPayloadBytes
	}
	if hf, ok := frame.(*hybiFrameReader); ok && hf.header.Length > int64(maxPayloadBytes) {

		ws.frameReader = frame
		return ErrFrameTooLarge
	}
	payloadType := frame.PayloadType()
	data, err := ioutil.ReadAll(frame)
	if err != nil {
		return err
	}
	return cd.Unmarshal(data, payloadType, v)
}

func marshal(v interface{}) (msg []byte, payloadType byte, err error) {
	switch data := v.(type) {
	case string:
		return []byte(data), TextFrame, nil
	case []byte:
		return data, BinaryFrame, nil
	}
	return nil, UnknownFrame, ErrNotSupported
}

func unmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	switch data := v.(type) {
	case *string:
		*data = string(msg)
		return nil
	case *[]byte:
		*data = msg
		return nil
	}
	return ErrNotSupported
}

var Message = Codec{marshal, unmarshal}

func jsonMarshal(v interface{}) (msg []byte, payloadType byte, err error) {
	msg, err = json.Marshal(v)
	return msg, TextFrame, err
}

func jsonUnmarshal(msg []byte, payloadType byte, v interface{}) (err error) {
	return json.Unmarshal(msg, v)
}

var JSON = Codec{jsonMarshal, jsonUnmarshal}
