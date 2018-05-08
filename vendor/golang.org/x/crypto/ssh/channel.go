
package ssh

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log"
	"sync"
)

const (
	minPacketLength = 9

	channelMaxPacket = 1 << 15

	channelWindowSize = 64 * channelMaxPacket
)

type NewChannel interface {

	Accept() (Channel, <-chan *Request, error)

	Reject(reason RejectionReason, message string) error

	ChannelType() string

	ExtraData() []byte
}

type Channel interface {

	Read(data []byte) (int, error)

	Write(data []byte) (int, error)

	Close() error

	CloseWrite() error

	SendRequest(name string, wantReply bool, payload []byte) (bool, error)

	Stderr() io.ReadWriter
}

type Request struct {
	Type      string
	WantReply bool
	Payload   []byte

	ch  *channel
	mux *mux
}

func (r *Request) Reply(ok bool, payload []byte) error {
	if !r.WantReply {
		return nil
	}

	if r.ch == nil {
		return r.mux.ackRequest(ok, payload)
	}

	return r.ch.ackRequest(ok)
}

type RejectionReason uint32

const (
	Prohibited RejectionReason = iota + 1
	ConnectionFailed
	UnknownChannelType
	ResourceShortage
)

func (r RejectionReason) String() string {
	switch r {
	case Prohibited:
		return "administratively prohibited"
	case ConnectionFailed:
		return "connect failed"
	case UnknownChannelType:
		return "unknown channel type"
	case ResourceShortage:
		return "resource shortage"
	}
	return fmt.Sprintf("unknown reason %d", int(r))
}

func min(a uint32, b int) uint32 {
	if a < uint32(b) {
		return a
	}
	return uint32(b)
}

type channelDirection uint8

const (
	channelInbound channelDirection = iota
	channelOutbound
)

type channel struct {

	chanType          string
	extraData         []byte
	localId, remoteId uint32

	maxIncomingPayload uint32
	maxRemotePayload   uint32

	mux *mux

	decided bool

	direction channelDirection

	msg chan interface{}

	sentRequestMu sync.Mutex

	incomingRequests chan *Request

	sentEOF bool

	remoteWin  window
	pending    *buffer
	extPending *buffer

	windowMu sync.Mutex
	myWindow uint32

	writeMu   sync.Mutex
	sentClose bool

	packetPool map[uint32][]byte
}

func (c *channel) writePacket(packet []byte) error {
	c.writeMu.Lock()
	if c.sentClose {
		c.writeMu.Unlock()
		return io.EOF
	}
	c.sentClose = (packet[0] == msgChannelClose)
	err := c.mux.conn.writePacket(packet)
	c.writeMu.Unlock()
	return err
}

func (c *channel) sendMessage(msg interface{}) error {
	if debugMux {
		log.Printf("send(%d): %#v", c.mux.chanList.offset, msg)
	}

	p := Marshal(msg)
	binary.BigEndian.PutUint32(p[1:], c.remoteId)
	return c.writePacket(p)
}

func (c *channel) WriteExtended(data []byte, extendedCode uint32) (n int, err error) {
	if c.sentEOF {
		return 0, io.EOF
	}

	opCode := byte(msgChannelData)
	headerLength := uint32(9)
	if extendedCode > 0 {
		headerLength += 4
		opCode = msgChannelExtendedData
	}

	c.writeMu.Lock()
	packet := c.packetPool[extendedCode]

	c.writeMu.Unlock()

	for len(data) > 0 {
		space := min(c.maxRemotePayload, len(data))
		if space, err = c.remoteWin.reserve(space); err != nil {
			return n, err
		}
		if want := headerLength + space; uint32(cap(packet)) < want {
			packet = make([]byte, want)
		} else {
			packet = packet[:want]
		}

		todo := data[:space]

		packet[0] = opCode
		binary.BigEndian.PutUint32(packet[1:], c.remoteId)
		if extendedCode > 0 {
			binary.BigEndian.PutUint32(packet[5:], uint32(extendedCode))
		}
		binary.BigEndian.PutUint32(packet[headerLength-4:], uint32(len(todo)))
		copy(packet[headerLength:], todo)
		if err = c.writePacket(packet); err != nil {
			return n, err
		}

		n += len(todo)
		data = data[len(todo):]
	}

	c.writeMu.Lock()
	c.packetPool[extendedCode] = packet
	c.writeMu.Unlock()

	return n, err
}

func (c *channel) handleData(packet []byte) error {
	headerLen := 9
	isExtendedData := packet[0] == msgChannelExtendedData
	if isExtendedData {
		headerLen = 13
	}
	if len(packet) < headerLen {

		return parseError(packet[0])
	}

	var extended uint32
	if isExtendedData {
		extended = binary.BigEndian.Uint32(packet[5:])
	}

	length := binary.BigEndian.Uint32(packet[headerLen-4 : headerLen])
	if length == 0 {
		return nil
	}
	if length > c.maxIncomingPayload {

		return errors.New("ssh: incoming packet exceeds maximum payload size")
	}

	data := packet[headerLen:]
	if length != uint32(len(data)) {
		return errors.New("ssh: wrong packet length")
	}

	c.windowMu.Lock()
	if c.myWindow < length {
		c.windowMu.Unlock()

		return errors.New("ssh: remote side wrote too much")
	}
	c.myWindow -= length
	c.windowMu.Unlock()

	if extended == 1 {
		c.extPending.write(data)
	} else if extended > 0 {

	} else {
		c.pending.write(data)
	}
	return nil
}

func (c *channel) adjustWindow(n uint32) error {
	c.windowMu.Lock()

	c.myWindow += uint32(n)
	c.windowMu.Unlock()
	return c.sendMessage(windowAdjustMsg{
		AdditionalBytes: uint32(n),
	})
}

func (c *channel) ReadExtended(data []byte, extended uint32) (n int, err error) {
	switch extended {
	case 1:
		n, err = c.extPending.Read(data)
	case 0:
		n, err = c.pending.Read(data)
	default:
		return 0, fmt.Errorf("ssh: extended code %d unimplemented", extended)
	}

	if n > 0 {
		err = c.adjustWindow(uint32(n))

		if n > 0 && err == io.EOF {
			err = nil
		}
	}

	return n, err
}

func (c *channel) close() {
	c.pending.eof()
	c.extPending.eof()
	close(c.msg)
	close(c.incomingRequests)
	c.writeMu.Lock()

	c.sentClose = true
	c.writeMu.Unlock()

	c.remoteWin.close()
}

func (c *channel) responseMessageReceived() error {
	if c.direction == channelInbound {
		return errors.New("ssh: channel response message received on inbound channel")
	}
	if c.decided {
		return errors.New("ssh: duplicate response received for channel")
	}
	c.decided = true
	return nil
}

func (c *channel) handlePacket(packet []byte) error {
	switch packet[0] {
	case msgChannelData, msgChannelExtendedData:
		return c.handleData(packet)
	case msgChannelClose:
		c.sendMessage(channelCloseMsg{PeersId: c.remoteId})
		c.mux.chanList.remove(c.localId)
		c.close()
		return nil
	case msgChannelEOF:

		c.extPending.eof()
		c.pending.eof()
		return nil
	}

	decoded, err := decode(packet)
	if err != nil {
		return err
	}

	switch msg := decoded.(type) {
	case *channelOpenFailureMsg:
		if err := c.responseMessageReceived(); err != nil {
			return err
		}
		c.mux.chanList.remove(msg.PeersId)
		c.msg <- msg
	case *channelOpenConfirmMsg:
		if err := c.responseMessageReceived(); err != nil {
			return err
		}
		if msg.MaxPacketSize < minPacketLength || msg.MaxPacketSize > 1<<31 {
			return fmt.Errorf("ssh: invalid MaxPacketSize %d from peer", msg.MaxPacketSize)
		}
		c.remoteId = msg.MyId
		c.maxRemotePayload = msg.MaxPacketSize
		c.remoteWin.add(msg.MyWindow)
		c.msg <- msg
	case *windowAdjustMsg:
		if !c.remoteWin.add(msg.AdditionalBytes) {
			return fmt.Errorf("ssh: invalid window update for %d bytes", msg.AdditionalBytes)
		}
	case *channelRequestMsg:
		req := Request{
			Type:      msg.Request,
			WantReply: msg.WantReply,
			Payload:   msg.RequestSpecificData,
			ch:        c,
		}

		c.incomingRequests <- &req
	default:
		c.msg <- msg
	}
	return nil
}

func (m *mux) newChannel(chanType string, direction channelDirection, extraData []byte) *channel {
	ch := &channel{
		remoteWin:        window{Cond: newCond()},
		myWindow:         channelWindowSize,
		pending:          newBuffer(),
		extPending:       newBuffer(),
		direction:        direction,
		incomingRequests: make(chan *Request, chanSize),
		msg:              make(chan interface{}, chanSize),
		chanType:         chanType,
		extraData:        extraData,
		mux:              m,
		packetPool:       make(map[uint32][]byte),
	}
	ch.localId = m.chanList.add(ch)
	return ch
}

var errUndecided = errors.New("ssh: must Accept or Reject channel")
var errDecidedAlready = errors.New("ssh: can call Accept or Reject only once")

type extChannel struct {
	code uint32
	ch   *channel
}

func (e *extChannel) Write(data []byte) (n int, err error) {
	return e.ch.WriteExtended(data, e.code)
}

func (e *extChannel) Read(data []byte) (n int, err error) {
	return e.ch.ReadExtended(data, e.code)
}

func (c *channel) Accept() (Channel, <-chan *Request, error) {
	if c.decided {
		return nil, nil, errDecidedAlready
	}
	c.maxIncomingPayload = channelMaxPacket
	confirm := channelOpenConfirmMsg{
		PeersId:       c.remoteId,
		MyId:          c.localId,
		MyWindow:      c.myWindow,
		MaxPacketSize: c.maxIncomingPayload,
	}
	c.decided = true
	if err := c.sendMessage(confirm); err != nil {
		return nil, nil, err
	}

	return c, c.incomingRequests, nil
}

func (ch *channel) Reject(reason RejectionReason, message string) error {
	if ch.decided {
		return errDecidedAlready
	}
	reject := channelOpenFailureMsg{
		PeersId:  ch.remoteId,
		Reason:   reason,
		Message:  message,
		Language: "en",
	}
	ch.decided = true
	return ch.sendMessage(reject)
}

func (ch *channel) Read(data []byte) (int, error) {
	if !ch.decided {
		return 0, errUndecided
	}
	return ch.ReadExtended(data, 0)
}

func (ch *channel) Write(data []byte) (int, error) {
	if !ch.decided {
		return 0, errUndecided
	}
	return ch.WriteExtended(data, 0)
}

func (ch *channel) CloseWrite() error {
	if !ch.decided {
		return errUndecided
	}
	ch.sentEOF = true
	return ch.sendMessage(channelEOFMsg{
		PeersId: ch.remoteId})
}

func (ch *channel) Close() error {
	if !ch.decided {
		return errUndecided
	}

	return ch.sendMessage(channelCloseMsg{
		PeersId: ch.remoteId})
}

func (ch *channel) Extended(code uint32) io.ReadWriter {
	if !ch.decided {
		return nil
	}
	return &extChannel{code, ch}
}

func (ch *channel) Stderr() io.ReadWriter {
	return ch.Extended(1)
}

func (ch *channel) SendRequest(name string, wantReply bool, payload []byte) (bool, error) {
	if !ch.decided {
		return false, errUndecided
	}

	if wantReply {
		ch.sentRequestMu.Lock()
		defer ch.sentRequestMu.Unlock()
	}

	msg := channelRequestMsg{
		PeersId:             ch.remoteId,
		Request:             name,
		WantReply:           wantReply,
		RequestSpecificData: payload,
	}

	if err := ch.sendMessage(msg); err != nil {
		return false, err
	}

	if wantReply {
		m, ok := (<-ch.msg)
		if !ok {
			return false, io.EOF
		}
		switch m.(type) {
		case *channelRequestFailureMsg:
			return false, nil
		case *channelRequestSuccessMsg:
			return true, nil
		default:
			return false, fmt.Errorf("ssh: unexpected response to channel request: %#v", m)
		}
	}

	return false, nil
}

func (ch *channel) ackRequest(ok bool) error {
	if !ch.decided {
		return errUndecided
	}

	var msg interface{}
	if !ok {
		msg = channelRequestFailureMsg{
			PeersId: ch.remoteId,
		}
	} else {
		msg = channelRequestSuccessMsg{
			PeersId: ch.remoteId,
		}
	}
	return ch.sendMessage(msg)
}

func (ch *channel) ChannelType() string {
	return ch.chanType
}

func (ch *channel) ExtraData() []byte {
	return ch.extraData
}
