
package ddm

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/rule"
	"github.com/ddmchain/go-ddmchain/rule/misc"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/ddm/fetcher"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/signal"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/discover/discover"
	"github.com/ddmchain/go-ddmchain/part"
	"github.com/ddmchain/go-ddmchain/ptl"
)

const (
	softResponseLimit = 2 * 1024 * 1024 * 4 
	estHeaderRlpSize  = 500             

	txChanSize = 4096 *10	
)

var (
	daoChallengeTimeout = 15 * time.Second 
)

var errIncompatibleConfig = errors.New("incompatible configuration")

func errResp(code errCode, format string, v ...interface{}) error {
	return fmt.Errorf("%v - %v", code, fmt.Sprintf(format, v...))
}

type ProtocolManager struct {
	networkId uint64

	fastSync  uint32 
	acceptTxs uint32 

	txpool      txPool
	blockchain  *core.BlockChain
	chainconfig *params.ChainConfig
	maxPeers    int

	downloader *downloader.Downloader
	fetcher    *fetcher.Fetcher
	peers      *peerSet

	SubProtocols []p2p.Protocol

	eventMux      *event.TypeMux
	txCh          chan core.TxPreEvent
	txSub         event.Subscription
	minedBlockSub *event.TypeMuxSubscription

	newPeerCh   chan *peer
	txsyncCh    chan *txsync
	quitSync    chan struct{}
	noMorePeers chan struct{}

	wg sync.WaitGroup

}

func NewProtocolManager(config *params.ChainConfig, mode downloader.SyncMode, networkId uint64, mux *event.TypeMux, txpool txPool, engine consensus.Engine, blockchain *core.BlockChain, chaindb ddmdb.Database) (*ProtocolManager, error) {

	manager := &ProtocolManager{
		networkId:   networkId,
		eventMux:    mux,
		txpool:      txpool,
		blockchain:  blockchain,
		chainconfig: config,
		peers:       newPeerSet(),
		newPeerCh:   make(chan *peer),
		noMorePeers: make(chan struct{}),
		txsyncCh:    make(chan *txsync),
		quitSync:    make(chan struct{}),
	}

	if mode == downloader.FastSync && blockchain.CurrentBlock().NumberU64() > 0 {
		log.Warn("Blockchain not empty, fast sync disabled")
		mode = downloader.FullSync
	}
	if mode == downloader.FastSync {
		manager.fastSync = uint32(1)
	}

	manager.SubProtocols = make([]p2p.Protocol, 0, len(ProtocolVersions))
	for i, version := range ProtocolVersions {

		if mode == downloader.FastSync && version < ddm63 {
			continue
		}

		version := version 
		manager.SubProtocols = append(manager.SubProtocols, p2p.Protocol{
			Name:    ProtocolName,
			Version: version,
			Length:  ProtocolLengths[i],
			Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
				peer := manager.newPeer(int(version), p, rw)
				select {
				case manager.newPeerCh <- peer:
					manager.wg.Add(1)
					defer manager.wg.Done()
					return manager.handle(peer)
				case <-manager.quitSync:
					return p2p.DiscQuitting
				}
			},
			NodeInfo: func() interface{} {
				return manager.NodeInfo()
			},
			PeerInfo: func(id discover.NodeID) interface{} {
				if p := manager.peers.Peer(fmt.Sprintf("%x", id[:8])); p != nil {
					return p.Info()
				}
				return nil
			},
		})
	}
	if len(manager.SubProtocols) == 0 {
		return nil, errIncompatibleConfig
	}

	manager.downloader = downloader.New(mode, chaindb, manager.eventMux, blockchain, nil, manager.removePeer)

	validator := func(header *types.Header) error {
		return engine.VerifyHeader(blockchain, header, true)
	}
	heighter := func() uint64 {
		return blockchain.CurrentBlock().NumberU64()
	}
	inserter := func(blocks types.Blocks) (int, error) {

		if atomic.LoadUint32(&manager.fastSync) == 1 {
			log.Warn("Discarded bad propagated block", "number", blocks[0].Number(), "hash", blocks[0].Hash())
			return 0, nil
		}
		atomic.StoreUint32(&manager.acceptTxs, 1) 
		return manager.blockchain.InsertChain(blocks)
	}
	manager.fetcher = fetcher.New(blockchain.GetBlockByHash, validator, manager.BroadcastBlock, heighter, inserter, manager.removePeer)

	return manager, nil
}

func (pm *ProtocolManager) removePeer(id string) {

	peer := pm.peers.Peer(id)
	if peer == nil {
		return
	}
	log.Debug("Removing DDMchain peer", "peer", id)

	pm.downloader.UnregisterPeer(id)
	if err := pm.peers.Unregister(id); err != nil {
		log.Error("Peer removal failed", "peer", id, "err", err)
	}

	if peer != nil {
		peer.Peer.Disconnect(p2p.DiscUselessPeer)
	}
}

func (pm *ProtocolManager) Start(maxPeers int) {
	pm.maxPeers = maxPeers

	pm.txCh = make(chan core.TxPreEvent, txChanSize)
	pm.txSub = pm.txpool.SubscribeTxPreEvent(pm.txCh)
	go pm.txBroadcastLoop()

	pm.minedBlockSub = pm.eventMux.Subscribe(core.NewMinedBlockEvent{})
	go pm.minedBroadcastLoop()

	go pm.syncer()
	go pm.txsyncLoop()
}

func (pm *ProtocolManager) Stop() {
	log.Info("Stopping DDMchain protocol")

	pm.txSub.Unsubscribe()         
	pm.minedBlockSub.Unsubscribe() 

	pm.noMorePeers <- struct{}{}

	close(pm.quitSync)

	pm.peers.Close()

	pm.wg.Wait()

	log.Info("DDMchain protocol stopped")
}

func (pm *ProtocolManager) newPeer(pv int, p *p2p.Peer, rw p2p.MsgReadWriter) *peer {
	return newPeer(pv, p, newMeteredMsgWriter(rw))
}

func (pm *ProtocolManager) handle(p *peer) error {
	if pm.peers.Len() >= pm.maxPeers {
		return p2p.DiscTooManyPeers
	}
	p.Log().Debug("DDMchain peer connected", "name", p.Name())

	var (
		genesis = pm.blockchain.Genesis()
		head    = pm.blockchain.CurrentHeader()
		hash    = head.Hash()
		number  = head.Number.Uint64()
		td      = pm.blockchain.GetTd(hash, number)
	)
	if err := p.Handshake(pm.networkId, td, hash, genesis.Hash()); err != nil {
		p.Log().Debug("DDMchain handshake failed", "err", err)
		return err
	}
	if !p.SuperNode() {
		lenLow, lenNormal := pm.peers.LenPeersWithConn()
		if p.superNode {
			if lenNormal >= pm.maxPeers / 3 {
				return p2p.DiscTooManyPeers
			}
		} else {
			if lenLow >= pm.maxPeers-(pm.maxPeers/3) {
				return p2p.DiscTooManyPeers
			}
		}
	}
	if rw, ok := p.rw.(*meteredMsgReadWriter); ok {
		rw.Init(p.version)
	}

	if err := pm.peers.Register(p); err != nil {
		p.Log().Error("DDMchain peer registration failed", "err", err)
		return err
	}
	defer pm.removePeer(p.id)

	if err := pm.downloader.RegisterPeer(p.id, p.version, p); err != nil {
		return err
	}

	pm.syncTransactions(p)

	if daoBlock := pm.chainconfig.DAOForkBlock; daoBlock != nil {

		if err := p.RequestHeadersByNumber(daoBlock.Uint64(), 1, 0, false); err != nil {
			return err
		}

		p.forkDrop = time.AfterFunc(daoChallengeTimeout, func() {
			p.Log().Debug("Timed out DAO fork-check, dropping")
			pm.removePeer(p.id)
		})

		defer func() {
			if p.forkDrop != nil {
				p.forkDrop.Stop()
				p.forkDrop = nil
			}
		}()
	}

	for {

		if err := pm.handleMsg(p); err != nil {
			p.Log().Debug("DDMchain message handling failed", "err", err)

			return err
		}

	}
}

func (pm *ProtocolManager) handleMsg(p *peer) error {

	msg, err := p.rw.ReadMsg()

	if err != nil {

		return err
	}
	if msg.Size > ProtocolMaxMsgSize {

		return errResp(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	defer msg.Discard()

	switch {
	case msg.Code == StatusMsg:

		return errResp(ErrExtraStatusMsg, "uncontrolled status message")

	case msg.Code == GetBlockHeadersMsg:

		var query getBlockHeadersData
		if err := msg.Decode(&query); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		hashMode := query.Origin.Hash != (common.Hash{})

		var (
			bytes   common.StorageSize
			headers []*types.Header
			unknown bool
		)
		for !unknown && len(headers) < int(query.Amount) && bytes < softResponseLimit && len(headers) < downloader.MaxHeaderFetch {

			var origin *types.Header
			if hashMode {
				origin = pm.blockchain.GetHeaderByHash(query.Origin.Hash)
			} else {
				origin = pm.blockchain.GetHeaderByNumber(query.Origin.Number)
			}
			if origin == nil {
				break
			}
			number := origin.Number.Uint64()
			headers = append(headers, origin)
			bytes += estHeaderRlpSize

			switch {
			case query.Origin.Hash != (common.Hash{}) && query.Reverse:

				for i := 0; i < int(query.Skip)+1; i++ {
					if header := pm.blockchain.GetHeader(query.Origin.Hash, number); header != nil {
						query.Origin.Hash = header.ParentHash
						number--
					} else {
						unknown = true
						break
					}
				}
			case query.Origin.Hash != (common.Hash{}) && !query.Reverse:

				var (
					current = origin.Number.Uint64()
					next    = current + query.Skip + 1
				)
				if next <= current {
					infos, _ := json.MarshalIndent(p.Peer.Info(), "", "  ")
					p.Log().Warn("GetBlockHeaders skip overflow attack", "current", current, "skip", query.Skip, "next", next, "attacker", infos)
					unknown = true
				} else {
					if header := pm.blockchain.GetHeaderByNumber(next); header != nil {
						if pm.blockchain.GetBlockHashesFromHash(header.Hash(), query.Skip+1)[query.Skip] == query.Origin.Hash {
							query.Origin.Hash = header.Hash()
						} else {
							unknown = true
						}
					} else {
						unknown = true
					}
				}
			case query.Reverse:

				if query.Origin.Number >= query.Skip+1 {
					query.Origin.Number -= query.Skip + 1
				} else {
					unknown = true
				}

			case !query.Reverse:

				query.Origin.Number += query.Skip + 1
			}
		}
		return p.SendBlockHeaders(headers)

	case msg.Code == BlockHeadersMsg:

		var headers []*types.Header
		if err := msg.Decode(&headers); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		if len(headers) == 0 && p.forkDrop != nil {

			verifyDAO := true

			if daoHeader := pm.blockchain.GetHeaderByNumber(pm.chainconfig.DAOForkBlock.Uint64()); daoHeader != nil {
				if _, td := p.Head(); td.Cmp(pm.blockchain.GetTd(daoHeader.Hash(), daoHeader.Number.Uint64())) >= 0 {
					verifyDAO = false
				}
			}

			if verifyDAO {
				p.Log().Debug("Seems to be on the same side of the DAO fork")
				p.forkDrop.Stop()
				p.forkDrop = nil
				return nil
			}
		}

		filter := len(headers) == 1
		if filter {

			if p.forkDrop != nil && pm.chainconfig.DAOForkBlock.Cmp(headers[0].Number) == 0 {

				p.forkDrop.Stop()
				p.forkDrop = nil

				if err := misc.VerifyDAOHeaderExtraData(pm.chainconfig, headers[0]); err != nil {
					p.Log().Debug("Verified to be on the other side of the DAO fork, dropping")
					return err
				}
				p.Log().Debug("Verified to be on the same side of the DAO fork")
				return nil
			}

			headers = pm.fetcher.FilterHeaders(p.id, headers, time.Now())
		}
		if len(headers) > 0 || !filter {
			err := pm.downloader.DeliverHeaders(p.id, headers)
			if err != nil {
				log.Debug("Failed to deliver headers", "err", err)
			}
		}

	case msg.Code == GetBlockBodiesMsg:

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}

		var (
			hash   common.Hash
			bytes  int
			bodies []rlp.RawValue
		)
		for bytes < softResponseLimit && len(bodies) < downloader.MaxBlockFetch {

			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}

			if data := pm.blockchain.GetBodyRLP(hash); len(data) != 0 {
				bodies = append(bodies, data)
				bytes += len(data)

			}
		}
		return p.SendBlockBodiesRLP(bodies)

	case msg.Code == BlockBodiesMsg:

		var request blockBodiesData
		if err := msg.Decode(&request); err != nil {

			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		trasactions := make([][]*types.Transaction, len(request))
		uncles := make([][]*types.Header, len(request))

		for i, body := range request {
			trasactions[i] = body.Transactions
			uncles[i] = body.Uncles
		}

		filter := len(trasactions) > 0 || len(uncles) > 0

		if filter {
			trasactions, uncles = pm.fetcher.FilterBodies(p.id, trasactions, uncles, time.Now())
		}

		if len(trasactions) > 0 || len(uncles) > 0 || !filter {
			err := pm.downloader.DeliverBodies(p.id, trasactions, uncles)
			if err != nil {
				log.Debug("Failed to deliver bodies", "err", err, "peer.id", p.id)
			}
		}

	case p.version >= ddm63 && msg.Code == GetNodeDataMsg:

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}

		var (
			hash  common.Hash
			bytes int
			data  [][]byte
		)
		for bytes < softResponseLimit && len(data) < downloader.MaxStateFetch {

			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}

			if entry, err := pm.blockchain.TrieNode(hash); err == nil {
				data = append(data, entry)
				bytes += len(entry)

			}
		}
		return p.SendNodeData(data)

	case p.version >= ddm63 && msg.Code == NodeDataMsg:

		var data [][]byte
		if err := msg.Decode(&data); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		if err := pm.downloader.DeliverNodeData(p.id, data); err != nil {
			log.Debug("Failed to deliver node state data", "err", err)
		}

	case p.version >= ddm63 && msg.Code == GetReceiptsMsg:

		msgStream := rlp.NewStream(msg.Payload, uint64(msg.Size))
		if _, err := msgStream.List(); err != nil {
			return err
		}

		var (
			hash     common.Hash
			bytes    int
			receipts []rlp.RawValue
		)
		for bytes < softResponseLimit && len(receipts) < downloader.MaxReceiptFetch {

			if err := msgStream.Decode(&hash); err == rlp.EOL {
				break
			} else if err != nil {
				return errResp(ErrDecode, "msg %v: %v", msg, err)
			}

			results := pm.blockchain.GetReceiptsByHash(hash)
			if results == nil {
				if header := pm.blockchain.GetHeaderByHash(hash); header == nil || header.ReceiptHash != types.EmptyRootHash {
					continue
				}
			}

			if encoded, err := rlp.EncodeToBytes(results); err != nil {
				log.Error("Failed to encode receipt", "err", err)
			} else {
				receipts = append(receipts, encoded)
				bytes += len(encoded)

			}
		}
		return p.SendReceiptsRLP(receipts)

	case p.version >= ddm63 && msg.Code == ReceiptsMsg:

		var receipts [][]*types.Receipt
		if err := msg.Decode(&receipts); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}

		if err := pm.downloader.DeliverReceipts(p.id, receipts); err != nil {
			log.Debug("Failed to deliver receipts", "err", err)
		}

	case msg.Code == NewBlockHashesMsg:
		var announces newBlockHashesData
		if err := msg.Decode(&announces); err != nil {
			return errResp(ErrDecode, "%v: %v", msg, err)
		}

		for _, block := range announces {
			p.MarkBlock(block.Hash)
		}

		unknown := make(newBlockHashesData, 0, len(announces))
		for _, block := range announces {
			if !pm.blockchain.HasBlock(block.Hash, block.Number) {
				unknown = append(unknown, block)
			}
		}
		for _, block := range unknown {
			pm.fetcher.Notify(p.id, block.Hash, block.Number, time.Now(), p.RequestOneHeader, p.RequestBodies)
		}

	case msg.Code == NewBlockMsg:

		var request newBlockData
		if err := msg.Decode(&request); err != nil {

			return errResp(ErrDecode, "%v: %v", msg, err)
		}
		request.Block.ReceivedAt = msg.ReceivedAt
		request.Block.ReceivedFrom = p

		p.MarkBlock(request.Block.Hash())
		pm.fetcher.Enqueue(p.id, request.Block)

		var (
			trueHead = request.Block.ParentHash()
			trueTD   = new(big.Int).Sub(request.TD, request.Block.Difficulty())
		)

		if _, td := p.Head(); trueTD.Cmp(td) > 0 {
			p.SetHead(trueHead, trueTD)

			currentBlock := pm.blockchain.CurrentBlock()
			if trueTD.Cmp(pm.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64())) > 0 {

				go pm.synchronise(p)
			}
		}

	case msg.Code == TxMsg:

		if atomic.LoadUint32(&pm.acceptTxs) == 0 {
			break
		}

		var txs []*types.Transaction
		if err := msg.Decode(&txs); err != nil {
			return errResp(ErrDecode, "msg %v: %v", msg, err)
		}
		for i, tx := range txs {

			if tx == nil {
				return errResp(ErrDecode, "transaction %d is nil", i)
			}
			p.MarkTransaction(tx.Hash())
		}
		pm.txpool.AddRemotes(txs)

	default:
		return errResp(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (pm *ProtocolManager) BroadcastBlock(block *types.Block, propagate bool) {
	hash := block.Hash()
	peers := pm.peers.PeersWithoutBlock(hash)

	if propagate {

		var td *big.Int
		if parent := pm.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1); parent != nil {
			td = new(big.Int).Add(block.Difficulty(), pm.blockchain.GetTd(block.ParentHash(), block.NumberU64()-1))
		} else {
			log.Error("Propagating dangling block", "number", block.Number(), "hash", hash)
			return
		}

		transfer := peers[:int(math.Sqrt(float64(len(peers))))]
		for _, peer := range transfer {

			peer.SendNewBlock(block, td)

		}
		log.Trace("Propagated block", "hash", hash, "recipients", len(transfer), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
		return
	}

	if pm.blockchain.HasBlock(hash, block.NumberU64()) {
		for _, peer := range peers {
			peer.SendNewBlockHashes([]common.Hash{hash}, []uint64{block.NumberU64()})
		}
		log.Trace("Announced block", "hash", hash, "recipients", len(peers), "duration", common.PrettyDuration(time.Since(block.ReceivedAt)))
	}
}

func (pm *ProtocolManager) BroadcastTx(hash common.Hash, tx *types.Transaction) {

	peers := pm.peers.PeersWithoutTx(hash)

	for _, peer := range peers {

		peer.SendTransactions(types.Transactions{tx})

	}
	log.Trace("Broadcast transaction", "hash", hash, "recipients", len(peers))
}

func (self *ProtocolManager) minedBroadcastLoop() {

	for obj := range self.minedBlockSub.Chan() {
		switch ev := obj.Data.(type) {
		case core.NewMinedBlockEvent:
			self.BroadcastBlock(ev.Block, true)  
			self.BroadcastBlock(ev.Block, false) 
		}
	}
}

func (self *ProtocolManager) txBroadcastLoop() {
	for {
		select {
		case event := <-self.txCh:
			self.BroadcastTx(event.Tx.Hash(), event.Tx)

		case <-self.txSub.Err():
			return
		}
	}
}

type NodeInfo struct {
	Network    uint64              `json:"network"`    
	Difficulty *big.Int            `json:"difficulty"` 
	Genesis    common.Hash         `json:"genesis"`    
	Config     *params.ChainConfig `json:"config"`     
	Head       common.Hash         `json:"head"`       
}

func (self *ProtocolManager) NodeInfo() *NodeInfo {
	currentBlock := self.blockchain.CurrentBlock()
	return &NodeInfo{
		Network:    self.networkId,
		Difficulty: self.blockchain.GetTd(currentBlock.Hash(), currentBlock.NumberU64()),
		Genesis:    self.blockchain.Genesis().Hash(),
		Config:     self.blockchain.Config(),
		Head:       currentBlock.Hash(),
	}
}
