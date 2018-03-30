// Copyright 2015 The go-ddmchain Authors
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

package ddm

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
	"github.com/ddmchain/go-ddmchain/params"
	"github.com/ddmchain/go-ddmchain/rpc"
)

// DDMApiBackend implements ddmapi.Backend for full nodes
type DDMApiBackend struct {
	ddm *DDMchain
	gpo *gasprice.Oracle
}

func (b *DDMApiBackend) ChainConfig() *params.ChainConfig {
	return b.ddm.chainConfig
}

func (b *DDMApiBackend) CurrentBlock() *types.Block {
	return b.ddm.blockchain.CurrentBlock()
}

func (b *DDMApiBackend) SetHead(number uint64) {
	b.ddm.protocolManager.downloader.Cancel()
	b.ddm.blockchain.SetHead(number)
}

func (b *DDMApiBackend) HeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Header, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.ddm.miner.PendingBlock()
		return block.Header(), nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ddm.blockchain.CurrentBlock().Header(), nil
	}
	return b.ddm.blockchain.GetHeaderByNumber(uint64(blockNr)), nil
}

func (b *DDMApiBackend) BlockByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*types.Block, error) {
	// Pending block is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block := b.ddm.miner.PendingBlock()
		return block, nil
	}
	// Otherwise resolve and return the block
	if blockNr == rpc.LatestBlockNumber {
		return b.ddm.blockchain.CurrentBlock(), nil
	}
	return b.ddm.blockchain.GetBlockByNumber(uint64(blockNr)), nil
}

func (b *DDMApiBackend) StateAndHeaderByNumber(ctx context.Context, blockNr rpc.BlockNumber) (*state.StateDB, *types.Header, error) {
	// Pending state is only known by the miner
	if blockNr == rpc.PendingBlockNumber {
		block, state := b.ddm.miner.Pending()
		return state, block.Header(), nil
	}
	// Otherwise resolve the block number and return its state
	header, err := b.HeaderByNumber(ctx, blockNr)
	if header == nil || err != nil {
		return nil, nil, err
	}
	stateDb, err := b.ddm.BlockChain().StateAt(header.Root)
	return stateDb, header, err
}

func (b *DDMApiBackend) GetBlock(ctx context.Context, blockHash common.Hash) (*types.Block, error) {
	return b.ddm.blockchain.GetBlockByHash(blockHash), nil
}

func (b *DDMApiBackend) GetReceipts(ctx context.Context, blockHash common.Hash) (types.Receipts, error) {
	return core.GetBlockReceipts(b.ddm.chainDb, blockHash, core.GetBlockNumber(b.ddm.chainDb, blockHash)), nil
}

func (b *DDMApiBackend) GetTd(blockHash common.Hash) *big.Int {
	return b.ddm.blockchain.GetTdByHash(blockHash)
}

func (b *DDMApiBackend) GetEVM(ctx context.Context, msg core.Message, state *state.StateDB, header *types.Header, vmCfg vm.Config) (*vm.EVM, func() error, error) {
	state.SetBalance(msg.From(), math.MaxBig256)
	vmError := func() error { return nil }

	context := core.NewEVMContext(msg, header, b.ddm.BlockChain(), nil)
	return vm.NewEVM(context, state, b.ddm.chainConfig, vmCfg), vmError, nil
}

func (b *DDMApiBackend) SubscribeRemovedLogsEvent(ch chan<- core.RemovedLogsEvent) event.Subscription {
	return b.ddm.BlockChain().SubscribeRemovedLogsEvent(ch)
}

func (b *DDMApiBackend) SubscribeChainEvent(ch chan<- core.ChainEvent) event.Subscription {
	return b.ddm.BlockChain().SubscribeChainEvent(ch)
}

func (b *DDMApiBackend) SubscribeChainHeadEvent(ch chan<- core.ChainHeadEvent) event.Subscription {
	return b.ddm.BlockChain().SubscribeChainHeadEvent(ch)
}

func (b *DDMApiBackend) SubscribeChainSideEvent(ch chan<- core.ChainSideEvent) event.Subscription {
	return b.ddm.BlockChain().SubscribeChainSideEvent(ch)
}

func (b *DDMApiBackend) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return b.ddm.BlockChain().SubscribeLogsEvent(ch)
}

func (b *DDMApiBackend) SendTx(ctx context.Context, signedTx *types.Transaction) error {
	return b.ddm.txPool.AddLocal(signedTx)
}

func (b *DDMApiBackend) GetPoolTransactions() (types.Transactions, error) {
	pending, err := b.ddm.txPool.Pending()
	if err != nil {
		return nil, err
	}
	var txs types.Transactions
	for _, batch := range pending {
		txs = append(txs, batch...)
	}
	return txs, nil
}

func (b *DDMApiBackend) GetPoolTransaction(hash common.Hash) *types.Transaction {
	return b.ddm.txPool.Get(hash)
}

func (b *DDMApiBackend) GetPoolNonce(ctx context.Context, addr common.Address) (uint64, error) {
	return b.ddm.txPool.State().GetNonce(addr), nil
}

func (b *DDMApiBackend) Stats() (pending int, queued int) {
	return b.ddm.txPool.Stats()
}

func (b *DDMApiBackend) TxPoolContent() (map[common.Address]types.Transactions, map[common.Address]types.Transactions) {
	return b.ddm.TxPool().Content()
}

func (b *DDMApiBackend) SubscribeTxPreEvent(ch chan<- core.TxPreEvent) event.Subscription {
	return b.ddm.TxPool().SubscribeTxPreEvent(ch)
}

func (b *DDMApiBackend) Downloader() *downloader.Downloader {
	return b.ddm.Downloader()
}

func (b *DDMApiBackend) ProtocolVersion() int {
	return b.ddm.DDMVersion()
}

func (b *DDMApiBackend) SuggestPrice(ctx context.Context) (*big.Int, error) {
	return b.gpo.SuggestPrice(ctx)
}

func (b *DDMApiBackend) ChainDb() ddmdb.Database {
	return b.ddm.ChainDb()
}

func (b *DDMApiBackend) EventMux() *event.TypeMux {
	return b.ddm.EventMux()
}

func (b *DDMApiBackend) AccountManager() *accounts.Manager {
	return b.ddm.AccountManager()
}

func (b *DDMApiBackend) BloomStatus() (uint64, uint64) {
	sections, _, _ := b.ddm.bloomIndexer.Sections()
	return params.BloomBitsBlocks, sections
}

func (b *DDMApiBackend) ServiceFilter(ctx context.Context, session *bloombits.MatcherSession) {
	for i := 0; i < bloomFilterThreads; i++ {
		go session.Multiplex(bloomRetrievalBatch, bloomRetrievalWait, b.ddm.bloomRequests)
	}
}