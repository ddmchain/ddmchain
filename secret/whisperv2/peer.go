
package whisperv2

import (
	"fmt"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/ptl"
	"gopkg.in/fatih/set.v0"
)

type peer struct {
	host *Whisper
	peer *p2p.Peer
	ws   p2p.MsgReadWriter

	known *set.Set 

	quit chan struct{}
}

func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return &peer{
		host:  host,
		peer:  remote,
		ws:    rw,
		known: set.New(),
		quit:  make(chan struct{}),
	}
}

func (self *peer) start() {
	go self.update()
	log.Debug(fmt.Sprintf("%v: whisper started", self.peer))
}

func (self *peer) stop() {
	close(self.quit)
	log.Debug(fmt.Sprintf("%v: whisper stopped", self.peer))
}

func (self *peer) handshake() error {

	errc := make(chan error, 1)
	go func() {
		errc <- p2p.SendItems(self.ws, statusCode, protocolVersion)
	}()

	packet, err := self.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer sent %x before status packet", packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	if _, err := s.List(); err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("bad status message: %v", err)
	}
	if peerVersion != protocolVersion {
		return fmt.Errorf("protocol version mismatch %d != %d", peerVersion, protocolVersion)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("failed to send status packet: %v", err)
	}
	return nil
}

func (self *peer) update() {

	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	for {
		select {
		case <-expire.C:
			self.expire()

		case <-transmit.C:
			if err := self.broadcast(); err != nil {
				log.Info(fmt.Sprintf("%v: broadcast failed: %v", self.peer, err))
				return
			}

		case <-self.quit:
			return
		}
	}
}

func (self *peer) mark(envelope *Envelope) {
	self.known.Add(envelope.Hash())
}

func (self *peer) marked(envelope *Envelope) bool {
	return self.known.Has(envelope.Hash())
}

func (self *peer) expire() {

	available := set.NewNonTS()
	for _, envelope := range self.host.envelopes() {
		available.Add(envelope.Hash())
	}

	unmark := make(map[common.Hash]struct{})
	self.known.Each(func(v interface{}) bool {
		if !available.Has(v.(common.Hash)) {
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})

	for hash := range unmark {
		self.known.Remove(hash)
	}
}

func (self *peer) broadcast() error {

	envelopes := self.host.envelopes()
	transmit := make([]*Envelope, 0, len(envelopes))
	for _, envelope := range envelopes {
		if !self.marked(envelope) {
			transmit = append(transmit, envelope)
			self.mark(envelope)
		}
	}

	if err := p2p.Send(self.ws, messagesCode, transmit); err != nil {
		return err
	}
	log.Trace(fmt.Sprint(self.peer, "broadcasted", len(transmit), "message(s)"))
	return nil
}
