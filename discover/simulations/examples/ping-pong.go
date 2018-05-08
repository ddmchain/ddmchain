
package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sync/atomic"
	"time"

	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/pitch"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/discover/discover"
	"github.com/ddmchain/go-ddmchain/discover/simulations"
	"github.com/ddmchain/go-ddmchain/discover/simulations/adapters"
	"github.com/ddmchain/go-ddmchain/control"
)

var adapterType = flag.String("adapter", "sim", `node adapter to use (one of "sim", "exec" or "docker")`)

func main() {
	flag.Parse()

	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(false))))

	services := map[string]adapters.ServiceFunc{
		"ping-pong": func(ctx *adapters.ServiceContext) (node.Service, error) {
			return newPingPongService(ctx.Config.ID), nil
		},
	}
	adapters.RegisterServices(services)

	var adapter adapters.NodeAdapter

	switch *adapterType {

	case "sim":
		log.Info("using sim adapter")
		adapter = adapters.NewSimAdapter(services)

	case "exec":
		tmpdir, err := ioutil.TempDir("", "p2p-example")
		if err != nil {
			log.Crit("error creating temp dir", "err", err)
		}
		defer os.RemoveAll(tmpdir)
		log.Info("using exec adapter", "tmpdir", tmpdir)
		adapter = adapters.NewExecAdapter(tmpdir)

	case "docker":
		log.Info("using docker adapter")
		var err error
		adapter, err = adapters.NewDockerAdapter()
		if err != nil {
			log.Crit("error creating docker adapter", "err", err)
		}

	default:
		log.Crit(fmt.Sprintf("unknown node adapter %q", *adapterType))
	}

	log.Info("starting simulation server on 0.0.0.0:8888...")
	network := simulations.NewNetwork(adapter, &simulations.NetworkConfig{
		DefaultService: "ping-pong",
	})
	if err := http.ListenAndServe(":8888", simulations.NewServer(network)); err != nil {
		log.Crit("error starting simulation server", "err", err)
	}
}

type pingPongService struct {
	id       discover.NodeID
	log      log.Logger
	received int64
}

func newPingPongService(id discover.NodeID) *pingPongService {
	return &pingPongService{
		id:  id,
		log: log.New("node.id", id),
	}
}

func (p *pingPongService) Protocols() []p2p.Protocol {
	return []p2p.Protocol{{
		Name:     "ping-pong",
		Version:  1,
		Length:   2,
		Run:      p.Run,
		NodeInfo: p.Info,
	}}
}

func (p *pingPongService) APIs() []rpc.API {
	return nil
}

func (p *pingPongService) Start(server *p2p.Server) error {
	p.log.Info("ping-pong service starting")
	return nil
}

func (p *pingPongService) Stop() error {
	p.log.Info("ping-pong service stopping")
	return nil
}

func (p *pingPongService) Info() interface{} {
	return struct {
		Received int64 `json:"received"`
	}{
		atomic.LoadInt64(&p.received),
	}
}

const (
	pingMsgCode = iota
	pongMsgCode
)

func (p *pingPongService) Run(peer *p2p.Peer, rw p2p.MsgReadWriter) error {
	log := p.log.New("peer.id", peer.ID())

	errC := make(chan error)
	go func() {
		for range time.Tick(10 * time.Second) {
			log.Info("sending ping")
			if err := p2p.Send(rw, pingMsgCode, "PING"); err != nil {
				errC <- err
				return
			}
		}
	}()
	go func() {
		for {
			msg, err := rw.ReadMsg()
			if err != nil {
				errC <- err
				return
			}
			payload, err := ioutil.ReadAll(msg.Payload)
			if err != nil {
				errC <- err
				return
			}
			log.Info("received message", "msg.code", msg.Code, "msg.payload", string(payload))
			atomic.AddInt64(&p.received, 1)
			if msg.Code == pingMsgCode {
				log.Info("sending pong")
				go p2p.Send(rw, pongMsgCode, "PONG")
			}
		}
	}()
	return <-errC
}
