
package consensus

import (
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/major/state"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/part"
	"github.com/ddmchain/go-ddmchain/control"
	"math/big"
)

type ChainReader interface {

	Config() *params.ChainConfig

	CurrentHeader() *types.Header

	GetHeader(hash common.Hash, number uint64) *types.Header

	GetHeaderByNumber(number uint64) *types.Header

	GetHeaderByHash(hash common.Hash) *types.Header

	GetBlock(hash common.Hash, number uint64) *types.Block
}

type Engine interface {

	Author(header *types.Header) (common.Address, error)

	VerifyHeader(chain ChainReader, header *types.Header, seal bool) error

	VerifyHeaders(chain ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error)

	VerifyUncles(chain ChainReader, block *types.Block) error

	VerifySeal(chain ChainReader, header *types.Header) error

	Prepare(chain ChainReader, header *types.Header) error

	Finalize(chain ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
		uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error)

	Seal(chain ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error)

	CalcDifficulty(chain ChainReader, time uint64, parent *types.Header) *big.Int

	APIs(chain ChainReader) []rpc.API
}

type PoW interface {
	Engine

	Hashrate() float64
}
