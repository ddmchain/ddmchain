
package les

import (
	"fmt"
	"sync"
	"time"

	"github.com/ddmchain/go-ddmchain/user"
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/hexutil"
	"github.com/ddmchain/go-ddmchain/algorithm"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/major/bloombits"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/ddm"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/ddm/filters"
	"github.com/ddmchain/go-ddmchain/ddm/gasprice"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/signal"
	"github.com/ddmchain/go-ddmchain/ddmin/ddmapi"
	"github.com/ddmchain/go-ddmchain/thin"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/pitch"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/discover/discv5"
	"github.com/ddmchain/go-ddmchain/part"
	rpc "github.com/ddmchain/go-ddmchain/control"
)

type LightDDMchain struct {
	config *ddm.Config

	odr         *LesOdr
	relay       *LesTxRelay
	chainConfig *params.ChainConfig

	shutdownChan chan bool

	peers           *peerSet
	txPool          *light.TxPool
	blockchain      *light.LightChain
	protocolManager *ProtocolManager
	serverPool      *serverPool
	reqDist         *requestDistributor
	retriever       *retrieveManager

	chainDb ddmdb.Database 

	bloomRequests                              chan chan *bloombits.Retrieval 
	bloomIndexer, chtIndexer, bloomTrieIndexer *core.ChainIndexer

	ApiBackend *LesApiBackend

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	networkId     uint64
	netRPCService *ddmapi.PublicNetAPI

	wg sync.WaitGroup
}

func New(ctx *node.ServiceContext, config *ddm.Config) (*LightDDMchain, error) {
	chainDb, err := ddm.CreateDB(ctx, config, "lightchaindata")
	if err != nil {
		return nil, err
	}
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, isCompat := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !isCompat {
		return nil, genesisErr
	}
	log.Info("Initialised chain configuration", "config", chainConfig)

	peers := newPeerSet()
	quitSync := make(chan struct{})

	lddm := &LightDDMchain{
		config:           config,
		chainConfig:      chainConfig,
		chainDb:          chainDb,
		eventMux:         ctx.EventMux,
		peers:            peers,
		reqDist:          newRequestDistributor(peers, quitSync),
		accountManager:   ctx.AccountManager,
		engine:           ddm.CreateConsensusEngine(ctx, &config.DDMhash, chainConfig, chainDb),
		shutdownChan:     make(chan bool),
		networkId:        config.NetworkId,
		bloomRequests:    make(chan chan *bloombits.Retrieval),
		bloomIndexer:     ddm.NewBloomIndexer(chainDb, light.BloomTrieFrequency),
		chtIndexer:       light.NewChtIndexer(chainDb, true),
		bloomTrieIndexer: light.NewBloomTrieIndexer(chainDb, true),
	}

	lddm.relay = NewLesTxRelay(peers, lddm.reqDist)
	lddm.serverPool = newServerPool(chainDb, quitSync, &lddm.wg)
	lddm.retriever = newRetrieveManager(peers, lddm.reqDist, lddm.serverPool)
	lddm.odr = NewLesOdr(chainDb, lddm.chtIndexer, lddm.bloomTrieIndexer, lddm.bloomIndexer, lddm.retriever)
	if lddm.blockchain, err = light.NewLightChain(lddm.odr, lddm.chainConfig, lddm.engine); err != nil {
		return nil, err
	}
	lddm.bloomIndexer.Start(lddm.blockchain)

	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		lddm.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}

	lddm.txPool = light.NewTxPool(lddm.chainConfig, lddm.blockchain, lddm.relay)
	if lddm.protocolManager, err = NewProtocolManager(lddm.chainConfig, true, ClientProtocolVersions, config.NetworkId, lddm.eventMux, lddm.engine, lddm.peers, lddm.blockchain, nil, chainDb, lddm.odr, lddm.relay, quitSync, &lddm.wg); err != nil {
		return nil, err
	}
	lddm.ApiBackend = &LesApiBackend{lddm, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	lddm.ApiBackend.gpo = gasprice.NewOracle(lddm.ApiBackend, gpoParams)
	return lddm, nil
}

func lesTopic(genesisHash common.Hash, protocolVersion uint) discv5.Topic {
	var name string
	switch protocolVersion {
	case lpv1:
		name = "LES"
	case lpv2:
		name = "LES2"
	default:
		panic(nil)
	}
	return discv5.Topic(name + "@" + common.Bytes2Hex(genesisHash.Bytes()[0:8]))
}

type LightDummyAPI struct{}

func (s *LightDummyAPI) DDMXbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

func (s *LightDummyAPI) Coinbase() (common.Address, error) {
	return common.Address{}, fmt.Errorf("not supported")
}

func (s *LightDummyAPI) Hashrate() hexutil.Uint {
	return 0
}

func (s *LightDummyAPI) Mining() bool {
	return false
}

func (s *LightDDMchain) APIs() []rpc.API {
	return append(ddmapi.GetAPIs(s.ApiBackend), []rpc.API{
		{
			Namespace: "ddm",
			Version:   "1.0",
			Service:   &LightDummyAPI{},
			Public:    true,
		}, {
			Namespace: "ddm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "ddm",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, true),
			Public:    true,
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *LightDDMchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *LightDDMchain) BlockChain() *light.LightChain      { return s.blockchain }
func (s *LightDDMchain) TxPool() *light.TxPool              { return s.txPool }
func (s *LightDDMchain) Engine() consensus.Engine           { return s.engine }
func (s *LightDDMchain) LesVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *LightDDMchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }
func (s *LightDDMchain) EventMux() *event.TypeMux           { return s.eventMux }

func (s *LightDDMchain) Protocols() []p2p.Protocol {
	return s.protocolManager.SubProtocols
}

func (s *LightDDMchain) Start(srvr *p2p.Server) error {
	s.startBloomHandlers()
	log.Warn("Light client mode is an experimental feature")
	s.netRPCService = ddmapi.NewPublicNetAPI(srvr, s.networkId)

	protocolVersion := AdvertiseProtocolVersions[0]
	s.serverPool.start(srvr, lesTopic(s.blockchain.Genesis().Hash(), protocolVersion))
	s.protocolManager.Start(s.config.LightPeers)
	return nil
}

func (s *LightDDMchain) Stop() error {
	s.odr.Stop()
	if s.bloomIndexer != nil {
		s.bloomIndexer.Close()
	}
	if s.chtIndexer != nil {
		s.chtIndexer.Close()
	}
	if s.bloomTrieIndexer != nil {
		s.bloomTrieIndexer.Close()
	}
	s.blockchain.Stop()
	s.protocolManager.Stop()
	s.txPool.Stop()

	s.eventMux.Stop()

	time.Sleep(time.Millisecond * 200)
	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
