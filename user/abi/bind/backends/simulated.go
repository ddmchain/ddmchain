
package backends

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/ddmchain/go-ddmchain"
	"github.com/ddmchain/go-ddmchain/user/abi/bind"
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/math"
	"github.com/ddmchain/go-ddmchain/algorithm/ddmhash"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/major/bloombits"
	"github.com/ddmchain/go-ddmchain/major/state"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/major/vm"
	"github.com/ddmchain/go-ddmchain/ddm/filters"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/signal"
	"github.com/ddmchain/go-ddmchain/part"
	"github.com/ddmchain/go-ddmchain/control"
)

var _ bind.ContractBackend = (*SimulatedBackend)(nil)

var errBlockNumberUnsupported = errors.New("SimulatedBackend cannot access blocks other than the latest block")
var errGasEstimationFailed = errors.New("gas required exceeds allowance or always failing transaction")

type SimulatedBackend struct {
	database   ddmdb.Database   
	blockchain *core.BlockChain 

	mu           sync.Mutex
	pendingBlock *types.Block   
	pendingState *state.StateDB 

	events *filters.EventSystem 

	config *params.ChainConfig
}

func NewSimulatedBackend(alloc core.GenesisAlloc) *SimulatedBackend {
	database, _ := ddmdb.NewMemDatabase()
	genesis := core.Genesis{Config: params.AllDDMhashProtocolChanges, Alloc: alloc}
	genesis.MustCommit(database)
	blockchain, _ := core.NewBlockChain(database, nil, genesis.Config, ddmhash.NewFaker(), vm.Config{})

	backend := &SimulatedBackend{
		database:   database,
		blockchain: blockchain,
		config:     genesis.Config,
		events:     filters.NewEventSystem(new(event.TypeMux), &filterBackend{database, blockchain}, false),
	}
	backend.rollback()
	return backend
}

func (b *SimulatedBackend) Commit() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if _, err := b.blockchain.InsertChain([]*types.Block{b.pendingBlock}); err != nil {
		panic(err) 
	}
	b.rollback()
}

func (b *SimulatedBackend) Rollback() {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.rollback()
}

func (b *SimulatedBackend) rollback() {
	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), ddmhash.NewFaker(), b.database, 1, func(int, *core.BlockGen) {})
	statedb, _ := b.blockchain.State()

	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), statedb.Database())
}

func (b *SimulatedBackend) CodeAt(ctx context.Context, contract common.Address, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetCode(contract), nil
}

func (b *SimulatedBackend) BalanceAt(ctx context.Context, contract common.Address, blockNumber *big.Int) (*big.Int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetBalance(contract), nil
}

func (b *SimulatedBackend) NonceAt(ctx context.Context, contract common.Address, blockNumber *big.Int) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return 0, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	return statedb.GetNonce(contract), nil
}

func (b *SimulatedBackend) StorageAt(ctx context.Context, contract common.Address, key common.Hash, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	statedb, _ := b.blockchain.State()
	val := statedb.GetState(contract, key)
	return val[:], nil
}

func (b *SimulatedBackend) TransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, _, _, _ := core.GetReceipt(b.database, txHash)
	return receipt, nil
}

func (b *SimulatedBackend) PendingCodeAt(ctx context.Context, contract common.Address) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.pendingState.GetCode(contract), nil
}

func (b *SimulatedBackend) CallContract(ctx context.Context, call ddmchain.CallMsg, blockNumber *big.Int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if blockNumber != nil && blockNumber.Cmp(b.blockchain.CurrentBlock().Number()) != 0 {
		return nil, errBlockNumberUnsupported
	}
	state, err := b.blockchain.State()
	if err != nil {
		return nil, err
	}
	rval, _, _, err := b.callContract(ctx, call, b.blockchain.CurrentBlock(), state)
	return rval, err
}

func (b *SimulatedBackend) PendingCallContract(ctx context.Context, call ddmchain.CallMsg) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	defer b.pendingState.RevertToSnapshot(b.pendingState.Snapshot())

	rval, _, _, err := b.callContract(ctx, call, b.pendingBlock, b.pendingState)
	return rval, err
}

func (b *SimulatedBackend) PendingNonceAt(ctx context.Context, account common.Address) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	return b.pendingState.GetOrNewStateObject(account).Nonce(), nil
}

func (b *SimulatedBackend) SuggestGasPrice(ctx context.Context) (*big.Int, error) {
	return big.NewInt(1), nil
}

func (b *SimulatedBackend) EstimateGas(ctx context.Context, call ddmchain.CallMsg) (uint64, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	var (
		lo  uint64 = params.TxGas - 1
		hi  uint64
		cap uint64
	)
	if call.Gas >= params.TxGas {
		hi = call.Gas
	} else {
		hi = b.pendingBlock.GasLimit()
	}
	cap = hi

	executable := func(gas uint64) bool {
		call.Gas = gas

		snapshot := b.pendingState.Snapshot()
		_, _, failed, err := b.callContract(ctx, call, b.pendingBlock, b.pendingState)
		b.pendingState.RevertToSnapshot(snapshot)

		if err != nil || failed {
			return false
		}
		return true
	}

	for lo+1 < hi {
		mid := (hi + lo) / 2
		if !executable(mid) {
			lo = mid
		} else {
			hi = mid
		}
	}

	if hi == cap {
		if !executable(hi) {
			return 0, errGasEstimationFailed
		}
	}
	return hi, nil
}

func (b *SimulatedBackend) callContract(ctx context.Context, call ddmchain.CallMsg, block *types.Block, statedb *state.StateDB) ([]byte, uint64, bool, error) {

	if call.GasPrice == nil {
		call.GasPrice = big.NewInt(1)
	}
	if call.Gas == 0 {
		call.Gas = 50000000
	}
	if call.Value == nil {
		call.Value = new(big.Int)
	}

	from := statedb.GetOrNewStateObject(call.From)
	from.SetBalance(math.MaxBig256)

	msg := callmsg{call}

	evmContext := core.NewEVMContext(msg, block.Header(), b.blockchain, nil)

	vmenv := vm.NewEVM(evmContext, statedb, b.config, vm.Config{})
	gaspool := new(core.GasPool).AddGas(math.MaxUint64)

	return core.NewStateTransition(vmenv, msg, gaspool).TransitionDb()
}

func (b *SimulatedBackend) SendTransaction(ctx context.Context, tx *types.Transaction) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	sender, err := types.Sender(types.HomesteadSigner{}, tx)
	if err != nil {
		panic(fmt.Errorf("invalid transaction: %v", err))
	}
	nonce := b.pendingState.GetNonce(sender)
	if tx.Nonce() != nonce {
		panic(fmt.Errorf("invalid transaction nonce: got %d, want %d", tx.Nonce(), nonce))
	}

	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), ddmhash.NewFaker(), b.database, 1, func(number int, block *core.BlockGen) {
		for _, tx := range b.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.AddTx(tx)
	})
	statedb, _ := b.blockchain.State()

	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), statedb.Database())
	return nil
}

func (b *SimulatedBackend) FilterLogs(ctx context.Context, query ddmchain.FilterQuery) ([]types.Log, error) {

	from := int64(0)
	if query.FromBlock != nil {
		from = query.FromBlock.Int64()
	}
	to := int64(-1)
	if query.ToBlock != nil {
		to = query.ToBlock.Int64()
	}

	filter := filters.New(&filterBackend{b.database, b.blockchain}, from, to, query.Addresses, query.Topics)

	logs, err := filter.Logs(ctx)
	if err != nil {
		return nil, err
	}
	res := make([]types.Log, len(logs))
	for i, log := range logs {
		res[i] = *log
	}
	return res, nil
}

func (b *SimulatedBackend) SubscribeFilterLogs(ctx context.Context, query ddmchain.FilterQuery, ch chan<- types.Log) (ddmchain.Subscription, error) {

	sink := make(chan []*types.Log)

	sub, err := b.events.SubscribeLogs(query, sink)
	if err != nil {
		return nil, err
	}

	return event.NewSubscription(func(quit <-chan struct{}) error {
		defer sub.Unsubscribe()
		for {
			select {
			case logs := <-sink:
				for _, log := range logs {
					select {
					case ch <- *log:
					case err := <-sub.Err():
						return err
					case <-quit:
						return nil
					}
				}
			case err := <-sub.Err():
				return err
			case <-quit:
				return nil
			}
		}
	}), nil
}

func (b *SimulatedBackend) AdjustTime(adjustment time.Duration) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	blocks, _ := core.GenerateChain(b.config, b.blockchain.CurrentBlock(), ddmhash.NewFaker(), b.database, 1, func(number int, block *core.BlockGen) {
		for _, tx := range b.pendingBlock.Transactions() {
			block.AddTx(tx)
		}
		block.OffsetTime(int64(adjustment.Seconds()))
	})
	statedb, _ := b.blockchain.State()

	b.pendingBlock = blocks[0]
	b.pendingState, _ = state.New(b.pendingBlock.Root(), statedb.Database())

	return nil
}

type callmsg struct {
	ddmchain.CallMsg
}

func (m callmsg) From() common.Address { return m.CallMsg.From }
func (m callmsg) Nonce() uint64        { return 0 }
func (m callmsg) CheckNonce() bool     { return false }
func (m callmsg) To() *common.Address  { return m.CallMsg.To }
func (m callmsg) GasPrice() *big.Int   { return m.CallMsg.GasPrice }
func (m callmsg) Gas() uint64          { return m.CallMsg.Gas }
func (m callmsg) Value() *big.Int      { return m.CallMsg.Value }
func (m callmsg) Data() []byte         { return m.CallMsg.Data }

type filterBackend struct {
	db ddmdb.Database
	bc *core.BlockChain
}

func (fb *filterBackend) ChainDb() ddmdb.Database  { return fb.db }
func (fb *filterBackend) EventMux() *event.TypeMux { panic("not supported") }

func (fb *filterBackend) HeaderByNumber(ctx context.Context, block rpc.BlockNumber) (*types.Header, error) {
	if block == rpc.LatestBlockNumber {
		return fb.bc.CurrentHeader(), nil
	}
	return fb.bc.GetHeaderByNumber(uint64(block.Int64())), nil
}
func (fb *filterBackend) GetReceipts(ctx context.Context, hash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(fb.db, hash, core.GetBlockNumber(fb.db, hash)), nil
}

func (fb *filterBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return event.NewSubscription(func(quit <-chan struct{}) error {
		<-quit
		return nil
	})
}
func (fb *filterBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return fb.bc.SubscribeChainEvent(ch)
}
func (fb *filterBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return fb.bc.SubscribeRemovedLogsEvent(ch)
}
func (fb *filterBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return fb.bc.SubscribeLogsEvent(ch)
}

func (fb *filterBackend) BloomStatus() (uint64, uint64) { return 4096, 0 }
func (fb *filterBackend) ServiceFilter(ctx context.Context, ms *bloombits.MatcherSession) {
	panic("not supported")
}
