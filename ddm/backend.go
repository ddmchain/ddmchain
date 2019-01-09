
package ddm

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ddmchain/go-ddmchain/user"
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/hexutil"
	"github.com/ddmchain/go-ddmchain/rule"
	"github.com/ddmchain/go-ddmchain/rule/dpos"
	"github.com/ddmchain/go-ddmchain/rule/ddmhash"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/major/bloombits"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/major/vm"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/ddm/filters"
	"github.com/ddmchain/go-ddmchain/ddm/gasprice"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/signal"
	"github.com/ddmchain/go-ddmchain/ddmin/ddmapi"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/pack"
	"github.com/ddmchain/go-ddmchain/pitch"
	"github.com/ddmchain/go-ddmchain/discover"
	"github.com/ddmchain/go-ddmchain/part"
	"github.com/ddmchain/go-ddmchain/ptl"
	"github.com/ddmchain/go-ddmchain/control"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

type DDMchain struct {
	config      *Config
	chainConfig *params.ChainConfig

	shutdownChan  chan bool
	stopDbUpgrade func() error

	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	chainDb ddmdb.Database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval
	bloomIndexer  *core.ChainIndexer

	ApiBackend *DDMApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	ddmxbase common.Address

	networkId     uint64
	netRPCService *ddmapi.PublicNetAPI

	lock sync.RWMutex
}

func (s *DDMchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

func New(ctx *node.ServiceContext, config *Config) (*DDMchain, error) {
	if config.SyncMode == downloader.LightSync {
		return nil, errors.New("can't run ddm.DDMchain in light sync mode, use les.LightDDMchain")
	}
	if !config.SyncMode.IsValid() {
		return nil, fmt.Errorf("invalid sync mode %d", config.SyncMode)
	}
	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err != nil {
		return nil, err
	}
	stopDbUpgrade := upgradeDeduplicateData(chainDb)
	chainConfig, genesisHash, genesisErr := core.SetupGenesisBlock(chainDb, config.Genesis)
	if _, ok := genesisErr.(*params.ConfigCompatError); genesisErr != nil && !ok {
		return nil, genesisErr
	}

	ddm := &DDMchain{
		config:         config,
		chainDb:        chainDb,
		chainConfig:    chainConfig,
		eventMux:       ctx.EventMux,
		accountManager: ctx.AccountManager,
		engine:         CreateConsensusEngine(ctx, &config.DDMhash, chainConfig, chainDb),
		shutdownChan:   make(chan bool),
		stopDbUpgrade:  stopDbUpgrade,
		networkId:      config.NetworkId,
		gasPrice:       config.GasPrice,
		ddmxbase:      config.DDMXbase,
		bloomRequests:  make(chan chan *bloombits.Retrieval),
		bloomIndexer:   NewBloomIndexer(chainDb, params.BloomBitsBlocks),
	}

	log.Info("Initialising DDMchain protocol", "versions", ProtocolVersions, "network", config.NetworkId)

	if !config.SkipBcVersionCheck {
		bcVersion := core.GetBlockChainVersion(chainDb)
		if bcVersion != core.BlockChainVersion && bcVersion != 0 {
			return nil, fmt.Errorf("Blockchain DB version mismatch (%d / %d). Run gddm upgradedb.\n", bcVersion, core.BlockChainVersion)
		}
		core.WriteBlockChainVersion(chainDb, core.BlockChainVersion)
	}
	var (
		vmConfig    = vm.Config{EnablePreimageRecording: config.EnablePreimageRecording}
		cacheConfig = &core.CacheConfig{Disabled: config.NoPruning, TrieNodeLimit: config.TrieCache, TrieTimeLimit: config.TrieTimeout}
	)
	ddm.blockchain, err = core.NewBlockChain(chainDb, cacheConfig, ddm.chainConfig, ddm.engine, vmConfig)
	if err != nil {
		return nil, err
	}

	if compat, ok := genesisErr.(*params.ConfigCompatError); ok {
		log.Warn("Rewinding chain to upgrade configuration", "err", compat)
		ddm.blockchain.SetHead(compat.RewindTo)
		core.WriteChainConfig(chainDb, genesisHash, chainConfig)
	}
	ddm.bloomIndexer.Start(ddm.blockchain)

	if config.TxPool.Journal != "" {
		config.TxPool.Journal = ctx.ResolvePath(config.TxPool.Journal)
	}
	ddm.txPool = core.NewTxPool(config.TxPool, ddm.chainConfig, ddm.blockchain)

	if ddm.protocolManager, err = NewProtocolManager(ddm.chainConfig, config.SyncMode, config.NetworkId, ddm.eventMux, ddm.txPool, ddm.engine, ddm.blockchain, chainDb); err != nil {
		return nil, err
	}
	ddm.miner = miner.New(ddm, ddm.chainConfig, ddm.EventMux(), ddm.engine)
	ddm.miner.SetExtra(makeExtraData(config.ExtraData))

	ddm.ApiBackend = &DDMApiBackend{ddm, nil}
	gpoParams := config.GPO
	if gpoParams.Default == nil {
		gpoParams.Default = config.GasPrice
	}
	ddm.ApiBackend.gpo = gasprice.NewOracle(ddm.ApiBackend, gpoParams)

	return ddm, nil
}

func makeExtraData(extra []byte) []byte {
	if len(extra) == 0 {

		extra, _ = rlp.EncodeToBytes([]interface{}{
			uint(params.VersionMajor<<16 | params.VersionMinor<<8 | params.VersionPatch),
			"gddm",
			runtime.Version(),
			runtime.GOOS,
		})
	}
	if uint64(len(extra)) > params.MaximumExtraDataSize {
		log.Warn("Miner extra data exceed limit", "extra", hexutil.Bytes(extra), "limit", params.MaximumExtraDataSize)
		extra = nil
	}
	return extra
}

func CreateDB(ctx *node.ServiceContext, config *Config, name string) (ddmdb.Database, error) {
	db, err := ctx.OpenDatabase(name, config.DatabaseCache, config.DatabaseHandles)
	if err != nil {
		return nil, err
	}
	if db, ok := db.(*ddmdb.LDBDatabase); ok {
		db.Meter("ddm/db/chaindata/")
	}
	return db, nil
}

func CreateConsensusEngine(ctx *node.ServiceContext, config *ddmhash.Config, chainConfig *params.ChainConfig, db ddmdb.Database) consensus.Engine {

	if chainConfig.Dpos != nil {
		return dpos.New(chainConfig.Dpos, db)
	}

	switch {
	case config.PowMode == ddmhash.ModeFake:
		log.Warn("DDMhash used in fake mode")
		return ddmhash.NewFaker()
	case config.PowMode == ddmhash.ModeTest:
		log.Warn("DDMhash used in test mode")
		return ddmhash.NewTester()
	case config.PowMode == ddmhash.ModeShared:
		log.Warn("DDMhash used in shared mode")
		return ddmhash.NewShared()
	default:
		engine := ddmhash.New(ddmhash.Config{
			CacheDir:       ctx.ResolvePath(config.CacheDir),
			CachesInMem:    config.CachesInMem,
			CachesOnDisk:   config.CachesOnDisk,
			DatasetDir:     config.DatasetDir,
			DatasetsInMem:  config.DatasetsInMem,
			DatasetsOnDisk: config.DatasetsOnDisk,
		})
		engine.SetThreads(-1)
		return engine
	}
}

func (s *DDMchain) APIs() []rpc.API {
	apis := ddmapi.GetAPIs(s.ApiBackend, s.engine)

	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	return append(apis, []rpc.API{
		{
			Namespace: "ddm",
			Version:   "1.0",
			Service:   NewPublicDDMchainAPI(s),
			Public:    true,
		}, {
			Namespace: "ddm",
			Version:   "1.0",
			Service:   NewPublicMinerAPI(s),
			Public:    true,
		}, {
			Namespace: "ddm",
			Version:   "1.0",
			Service:   downloader.NewPublicDownloaderAPI(s.protocolManager.downloader, s.eventMux),
			Public:    true,
		}, {
			Namespace: "miner",
			Version:   "1.0",
			Service:   NewPrivateMinerAPI(s),
			Public:    false,
		}, {
			Namespace: "ddm",
			Version:   "1.0",
			Service:   filters.NewPublicFilterAPI(s.ApiBackend, false),
			Public:    true,
		}, {
			Namespace: "admin",
			Version:   "1.0",
			Service:   NewPrivateAdminAPI(s),
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPublicDebugAPI(s),
			Public:    true,
		}, {
			Namespace: "debug",
			Version:   "1.0",
			Service:   NewPrivateDebugAPI(s.chainConfig, s),
		}, {
			Namespace: "net",
			Version:   "1.0",
			Service:   s.netRPCService,
			Public:    true,
		},
	}...)
}

func (s *DDMchain) ResetWithGenesisBlock(gb *types.Block) {
	s.blockchain.ResetWithGenesisBlock(gb)
}

func (s *DDMchain) DDMXbase() (eb common.Address, err error) {
	s.lock.RLock()
	ddmxbase := s.ddmxbase
	s.lock.RUnlock()

	if ddmxbase != (common.Address{}) {
		return ddmxbase, nil
	}
	if wallets := s.AccountManager().Wallets(); len(wallets) > 0 {
		if accounts := wallets[0].Accounts(); len(accounts) > 0 {
			ddmxbase := accounts[0].Address

			s.lock.Lock()
			s.ddmxbase = ddmxbase
			s.lock.Unlock()

			log.Info("DDMXbase automatically configured", "address", ddmxbase)
			return ddmxbase, nil
		}
	}
	return common.Address{}, fmt.Errorf("ddmxbase must be explicitly specified")
}

func (self *DDMchain) SetDDMXbase(ddmxbase common.Address) {
	self.lock.Lock()
	self.ddmxbase = ddmxbase
	self.lock.Unlock()

	self.miner.SetDDMXbase(ddmxbase)
}

func (s *DDMchain) StartMining(local bool) error {
	eb, err := s.DDMXbase()
	if err != nil {
		log.Error("Cannot start mining without ddmxbase", "err", err)
		return fmt.Errorf("ddmxbase missing: %v", err)
	}
	if dpos, ok := s.engine.(*dpos.Dpos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("DDMXbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		dpos.Authorize(eb, wallet.SignHash)
	}
	if local {

		atomic.StoreUint32(&s.protocolManager.acceptTxs, 1)
	}
	go s.miner.Start(eb)
	return nil
}

func (s *DDMchain) StopMining()         { s.miner.Stop() }
func (s *DDMchain) IsMining() bool      { return s.miner.Mining() }
func (s *DDMchain) Miner() *miner.Miner { return s.miner }

func (s *DDMchain) AccountManager() *accounts.Manager  { return s.accountManager }
func (s *DDMchain) BlockChain() *core.BlockChain       { return s.blockchain }
func (s *DDMchain) TxPool() *core.TxPool               { return s.txPool }
func (s *DDMchain) EventMux() *event.TypeMux           { return s.eventMux }
func (s *DDMchain) Engine() consensus.Engine           { return s.engine }
func (s *DDMchain) ChainDb() ddmdb.Database            { return s.chainDb }
func (s *DDMchain) IsListening() bool                  { return true }
func (s *DDMchain) DDMVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *DDMchain) NetVersion() uint64                 { return s.networkId }
func (s *DDMchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

func (s *DDMchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

func (s *DDMchain) Start(srvr *p2p.Server) error {

	s.startBloomHandlers()

	s.netRPCService = ddmapi.NewPublicNetAPI(srvr, s.NetVersion())

	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}

	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

func (s *DDMchain) Stop() error {
	if s.stopDbUpgrade != nil {
		s.stopDbUpgrade()
	}
	s.bloomIndexer.Close()
	s.blockchain.Stop()
	s.protocolManager.Stop()
	if s.lesServer != nil {
		s.lesServer.Stop()
	}
	s.txPool.Stop()
	s.miner.Stop()
	s.eventMux.Stop()

	s.chainDb.Close()
	close(s.shutdownChan)

	return nil
}
