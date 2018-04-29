//
// This file is part of the go-ddmchain library.
//
// The go-ddmchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ddmchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ddmchain library. If not, see <http://www.gnu.org/licenses/>.

// Package ddm implements the DDMchain protocol.
package ddm

import (
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/ddmchain/go-ddmchain/accounts"
	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/common/hexutil"
	"github.com/ddmchain/go-ddmchain/consensus"
	"github.com/ddmchain/go-ddmchain/consensus/ddmdpos"
	"github.com/ddmchain/go-ddmchain/consensus/ddmhash"
	"github.com/ddmchain/go-ddmchain/core"
	"github.com/ddmchain/go-ddmchain/core/bloombits"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/core/vm"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/ddm/filters"
	"github.com/ddmchain/go-ddmchain/ddm/gasprice"
	"github.com/ddmchain/go-ddmchain/ddmdb"
	"github.com/ddmchain/go-ddmchain/event"
	"github.com/ddmchain/go-ddmchain/internal/ddmapi"
	"github.com/ddmchain/go-ddmchain/log"
	"github.com/ddmchain/go-ddmchain/miner"
	"github.com/ddmchain/go-ddmchain/node"
	"github.com/ddmchain/go-ddmchain/p2p"
	"github.com/ddmchain/go-ddmchain/params"
	"github.com/ddmchain/go-ddmchain/rlp"
	"github.com/ddmchain/go-ddmchain/rpc"
)

type LesServer interface {
	Start(srvr *p2p.Server)
	Stop()
	Protocols() []p2p.Protocol
	SetBloomBitsIndexer(bbIndexer *core.ChainIndexer)
}

// DDMchain implements the DDMchain full node service.
type DDMchain struct {
	config      *Config
	chainConfig *params.ChainConfig

	// Channel for shutting down the service
	shutdownChan  chan bool    // Channel for shutting down the ddmchain
	stopDbUpgrade func() error // stop chain db sequential key upgrade

	// Handlers
	txPool          *core.TxPool
	blockchain      *core.BlockChain
	protocolManager *ProtocolManager
	lesServer       LesServer

	// DB interfaces
	chainDb ddmdb.Database // Block chain database

	eventMux       *event.TypeMux
	engine         consensus.Engine
	accountManager *accounts.Manager

	bloomRequests chan chan *bloombits.Retrieval // Channel receiving bloom data retrieval requests
	bloomIndexer  *core.ChainIndexer             // Bloom indexer operating during block imports

	ApiBackend *DDMApiBackend

	miner     *miner.Miner
	gasPrice  *big.Int
	ddmxbase common.Address

	networkId     uint64
	netRPCService *ddmapi.PublicNetAPI

	lock sync.RWMutex // Protects the variadic fields (e.g. gas price and ddmxbase)
}

func (s *DDMchain) AddLesServer(ls LesServer) {
	s.lesServer = ls
	ls.SetBloomBitsIndexer(s.bloomIndexer)
}

// New creates a new DDMchain object (including the
// initialisation of the common DDMchain object)
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
	log.Info("Initialised chain configuration", "config", chainConfig)

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
	// Rewind the chain in case of an incompatible config upgrade.
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
		// create default extradata
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

// CreateDB creates the chain database.
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

// CreateConsensusEngine creates the required type of consensus engine instance for an DDMchain service
func CreateConsensusEngine(ctx *node.ServiceContext, config *ddmhash.Config, chainConfig *params.ChainConfig, db ddmdb.Database) consensus.Engine {
	if chainConfig.DPos != nil {
		return ddmdpos.New(chainConfig.DPos, db)
	}
	// Otherwise assume proof-of-work
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
		engine.SetThreads(-1) // Disable CPU mining
		return engine
	}
}

// APIs returns the collection of RPC services the ddmchain package offers.
// NOTE, some of these services probably need to be moved to somewhere else.
func (s *DDMchain) APIs() []rpc.API {
	apis := ddmapi.GetAPIs(s.ApiBackend)

	// Append any APIs exposed explicitly by the consensus engine
	apis = append(apis, s.engine.APIs(s.BlockChain())...)

	// Append all the local APIs and return
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

// set in js console via admin interface or wrapper from cli flags
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
	if dpos, ok := s.engine.(*ddmdpos.DPos); ok {
		wallet, err := s.accountManager.Find(accounts.Account{Address: eb})
		if wallet == nil || err != nil {
			log.Error("DDMXbase account unavailable locally", "err", err)
			return fmt.Errorf("signer missing: %v", err)
		}
		dpos.Authorize(eb, wallet.SignHash)
	}
	if local {
		// If local (CPU) mining is started, we can disable the transaction rejection
		// mechanism introduced to speed sync times. CPU mining on mainnet is ludicrous
		// so noone will ever hit this path, whereas marking sync done on CPU mining
		// will ensure that private networks work in single miner mode too.
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
func (s *DDMchain) IsListening() bool                  { return true } // Always listening
func (s *DDMchain) DDMVersion() int                    { return int(s.protocolManager.SubProtocols[0].Version) }
func (s *DDMchain) NetVersion() uint64                 { return s.networkId }
func (s *DDMchain) Downloader() *downloader.Downloader { return s.protocolManager.downloader }

// Protocols implements node.Service, returning all the currently configured
// network protocols to start.
func (s *DDMchain) Protocols() []p2p.Protocol {
	if s.lesServer == nil {
		return s.protocolManager.SubProtocols
	}
	return append(s.protocolManager.SubProtocols, s.lesServer.Protocols()...)
}

// Start implements node.Service, starting all internal goroutines needed by the
// DDMchain protocol implementation.
func (s *DDMchain) Start(srvr *p2p.Server) error {
	// Start the bloom bits servicing goroutines
	s.startBloomHandlers()

	// Start the RPC service
	s.netRPCService = ddmapi.NewPublicNetAPI(srvr, s.NetVersion())

	// Figure out a max peers count based on the server limits
	maxPeers := srvr.MaxPeers
	if s.config.LightServ > 0 {
		if s.config.LightPeers >= srvr.MaxPeers {
			return fmt.Errorf("invalid peer config: light peer count (%d) >= total peer count (%d)", s.config.LightPeers, srvr.MaxPeers)
		}
		maxPeers -= s.config.LightPeers
	}
	// Start the networking layer and the light server if requested
	s.protocolManager.Start(maxPeers)
	if s.lesServer != nil {
		s.lesServer.Start(srvr)
	}
	return nil
}

// Stop implements node.Service, terminating all internal goroutines used by the
// DDMchain protocol.
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
