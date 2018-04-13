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

package les

import (
	"context"
	"math/big"

	"github.com/ddmchain/go-ddmchain/accounts"
	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/common/math"
	"github.com/ddmchain/go-ddmchain/core"
	"github.com/ddmchain/go-ddmchain/core/bloombits"
	"github.com/ddmchain/go-ddmchain/core/state"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/core/vm"
	"github.com/ddmchain/go-ddmchain/ddm/downloader"
	"github.com/ddmchain/go-ddmchain/ddm/gasprice"
	"github.com/ddmchain/go-ddmchain/ddmdb"
	"github.com/ddmchain/go-ddmchain/event"
	"github.com/ddmchain/go-ddmchain/light"
	"github.com/ddmchain/go-ddmchain/params"
	"github.com/ddmchain/go-ddmchain/rpc"
)

type LesApiBackend struct {
	ddm *LightDDMchain
	gpo *gasprice.Oracle
}

func (b *LesApiBackend) ChainConfig() *params.ChainConfig {
	return b.ddm.chainConfig
}

func (b *LesApiBackend) CurrentBlock() *types.Block {
	return types.NewBlockWithHeader(b.ddm.BlockChain().CurrentHeader())
}

func (b *LesApiBackend) SetHead(number uint64) {
	b.ddm.protocolManager.downloader.Cancel()
	b.ddm.blockchain.SetHead(number)
}

func (b *LesApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	if blockNr == rpc.LatestBlockNumber || blockNr == rpc.PendingBlockNumber {
		return b.ddm.blockchain.CurrentHeader(), nil
	}

	return b.ddm.blockchain.GetHeaderByNumberOdr(ctx, uint64(blockNr))
}

func (b *LesApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, err
	}
	return b.GetBlock(ctx, header.Hash())
}

func (b *LesApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	return light.NewState(ctx, header, b.ddm.odr), header, nil
}

func (b *LesApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.ddm.blockchain.GetBlockByHash(ctx, blockHash)
}

func (b *LesApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return light.GetBlockReceipts(ctx, b.ddm.odr, blockHash, core.GetBlockNumber(b.ddm.chainDb, blockHash))
}

func (b *LesApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.ddm.blockchain.GetTdByHash(blockHash)
}

func (b *LesApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	context := core.NewEVMContext(msg, header, b.ddm.blockchain, nil)
	return vm.NewEVM(context, state, b.ddm.chainConfig, vmCfg), state.Error, nil
}

func (b *LesApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.ddm.txPool.Add(ctx, signedTx)
}

func (b *LesApiBackend) RemoveTx(txHash common.Hash) {
	b.ddm.txPool.RemoveTx(txHash)
}

func (b *LesApiBackend) GetPoolTransactions() (types.Transactions, error) {
	return b.ddm.txPool.GetTransactions()
}

func (b *LesApiBackend) GetPoolTransaction(txHash common.Hash) *types.Transaction {
	return b.ddm.txPool.GetTransaction(txHash)
}

func (b *LesApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.ddm.txPool.GetNonce(ctx, addr)
}

func (b *LesApiBackend) Stats() (pending int, queued int) {
	return b.ddm.txPool.Stats(), 0
}

func (b *LesApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.ddm.txPool.Content()
}

func (b *LesApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.ddm.txPool.SubscribeTxPreEvent(ch)
}

func (b *LesApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.ddm.blockchain.SubscribeChainEvent(ch)
}

func (b *LesApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.ddm.blockchain.SubscribeChainHeadEvent(ch)
}

func (b *LesApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.ddm.blockchain.SubscribeChainSideEvent(ch)
}

func (b *LesApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.ddm.blockchain.SubscribeLogsEvent(ch)
}

func (b *LesApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.ddm.blockchain.SubscribeRemovedLogsEvent(ch)
}

func (b *LesApiBackend) Downloader() *downloader.Downloader {
	return b.ddm.Downloader()
}

func (b *LesApiBackend) ProtocolVersion() int {
	return b.ddm.LesVersion() + 10000
}

func (b *LesApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *LesApiBackend) ChainDb() ddmdb.Database {
	return b.ddm.chainDb
}

func (b *LesApiBackend) EventMux() *event.TypeMux {
	return b.ddm.eventMux
}

func (b *LesApiBackend) AccountManager() *accounts.Manager {
	return b.ddm.accountManager
}

func (b *LesApiBackend) BloomStatus() (uint64, uint64) {
	if b.ddm.bloomIndexer == nil {
		return 0, 0
	}
	sections, _, _ := b.ddm.bloomIndexer.Sections()
	return light.BloomTrieFrequency, sections
}

func (b *LesApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.ddm.bloomRequests)
	}
}
