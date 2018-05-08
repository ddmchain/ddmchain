
package whisperv5

import (
	"fmt"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/ptl"
	set "gopkg.in/fatih/set.v0"
)

type Peer struct {
	host    *Whisper
	peer    *p2p.Peer
	ws      p2p.MsgReadWriter
	trusted bool

	known *set.Set 

	quit chan struct{}
}

func newPeer(host *Whisper, remote *p2p.Peer, rw p2p.MsgReadWriter) *Peer {
	return &Peer{
		host:    host,
		peer:    remote,
		ws:      rw,
		trusted: false,
		known:   set.New(),
		quit:    make(chan struct{}),
	}
}

func (p *Peer) start() {
	go p.update()
	log.Trace("start", "peer", p.ID())
}

func (p *Peer) stop() {
	close(p.quit)
	log.Trace("stop", "peer", p.ID())
}

func (p *Peer) handshake() error {

	errc := make(chan error, 1)
	go func() {
		errc <- p2p.Send(p.ws, statusCode, ProtocolVersion)
	}()

	packet, err := p.ws.ReadMsg()
	if err != nil {
		return err
	}
	if packet.Code != statusCode {
		return fmt.Errorf("peer [%x] sent packet %x before status packet", p.ID(), packet.Code)
	}
	s := rlp.NewStream(packet.Payload, uint64(packet.Size))
	peerVersion, err := s.Uint()
	if err != nil {
		return fmt.Errorf("peer [%x] sent bad status message: %v", p.ID(), err)
	}
	if peerVersion != ProtocolVersion {
		return fmt.Errorf("peer [%x]: protocol version mismatch %d != %d", p.ID(), peerVersion, ProtocolVersion)
	}

	if err := <-errc; err != nil {
		return fmt.Errorf("peer [%x] failed to send status packet: %v", p.ID(), err)
	}
	return nil
}

func (p *Peer) update() {

	expire := time.NewTicker(expirationCycle)
	transmit := time.NewTicker(transmissionCycle)

	for {
		select {
		case <-expire.C:
			p.expire()

		case <-transmit.C:
			if err := p.broadcast(); err != nil {
				log.Trace("broadcast failed", "reason", err, "peer", p.ID())
				return
			}

		case <-p.quit:
			return
		}
	}
}

func (peer *Peer) mark(envelope *Envelope) {
	peer.known.Add(envelope.Hash())
}

func (peer *Peer) marked(envelope *Envelope) bool {
	return peer.known.Has(envelope.Hash())
}

func (peer *Peer) expire() {
	unmark := make(map[common.Hash]struct{})
	peer.known.Each(func(v interface{}) bool {
		if !peer.host.isEnvelopeCached(v.(common.Hash)) {
			unmark[v.(common.Hash)] = struct{}{}
		}
		return true
	})

	for hash := range unmark {
		peer.known.Remove(hash)
	}
}

func (p *Peer) broadcast() error {
	var cnt int
	envelopes := p.host.Envelopes()
	for _, envelope := range envelopes {
		if !p.marked(envelope) {
			err := p2p.Send(p.ws, messagesCode, envelope)
			if err != nil {
				return err
			} else {
				p.mark(envelope)
				cnt++
			}
		}
	}
	if cnt > 0 {
		log.Trace("broadcast", "num. messages", cnt)
	}
	return nil
}

func (p *Peer) ID() []byte {
	id := p.peer.ID()
	return id[:]
}
