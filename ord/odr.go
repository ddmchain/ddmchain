
package les

import (
	"context"

	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/thin"
	"github.com/ddmchain/go-ddmchain/sign"
)

type LesOdr struct {
	db                                         ddmdb.Database
	chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer
	retriever                                  *retrieveManager
	stop                                       chan struct{}
}

func NewLesOdr(db ddmdb.Database, chtIndexer, bloomTrieIndexer, bloomIndexer *core.ChainIndexer, retriever *retrieveManager) *LesOdr {
	return &LesOdr{
		db:               db,
		chtIndexer:       chtIndexer,
		bloomTrieIndexer: bloomTrieIndexer,
		bloomIndexer:     bloomIndexer,
		retriever:        retriever,
		stop:             make(chan struct{}),
	}
}

func (odr *LesOdr) Stop() {
	close(odr.stop)
}

func (odr *LesOdr) Database() ddmdb.Database {
	return odr.db
}

func (odr *LesOdr) ChtIndexer() *core.ChainIndexer {
	return odr.chtIndexer
}

func (odr *LesOdr) BloomTrieIndexer() *core.ChainIndexer {
	return odr.bloomTrieIndexer
}

func (odr *LesOdr) BloomIndexer() *core.ChainIndexer {
	return odr.bloomIndexer
}

const (
	MsgBlockBodies = iota
	MsgCode
	MsgReceipts
	MsgProofsV1
	MsgProofsV2
	MsgHeaderProofs
	MsgHelperTrieProofs
)

type Msg struct {
	MsgType int
	ReqID   uint64
	Obj     interface{}
}

func (odr *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)

	reqID := genReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			return lreq.GetCost(dp.(*peer))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*peer)
			return lreq.CanSend(p)
		},
		request: func(dp distPeer) func() {
			p := dp.(*peer)
			cost := lreq.GetCost(p)
			p.fcServer.QueueRequest(reqID, cost)
			return func() { lreq.Request(reqID, p) }
		},
	}

	if err = odr.retriever.retrieve(ctx, reqID, rq, func(p distPeer, msg *Msg) error { return lreq.Validate(odr.db, msg) }, odr.stop); err == nil {

		req.StoreResult(odr.db)
	} else {
		log.Debug("Failed to retrieve data from network", "err", err)
	}
	return
}
