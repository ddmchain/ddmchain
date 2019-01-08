
package dpos

import (
	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/rule"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/control"
)

type API struct {
	chain  consensus.ChainReader
	dpos *Dpos
}

func (api *API) GetSnapshot(number *rpc.BlockNumber) (*Snapshot, error) {

	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}

	if header == nil {
		return nil, errUnknownBlock
	}
	return api.dpos.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

func (api *API) GetSnapshotAtHash(hash common.Hash) (*Snapshot, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return api.dpos.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
}

func (api *API) GetSigners(number *rpc.BlockNumber) ([]common.Address, error) {

	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = api.chain.CurrentHeader()
	} else {
		header = api.chain.GetHeaderByNumber(uint64(number.Int64()))
	}

	if header == nil {
		return nil, errUnknownBlock
	}
	snap, err := api.dpos.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}

func (api *API) GetSignersAtHash(hash common.Hash) ([]common.Address, error) {
	header := api.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	snap, err := api.dpos.snapshot(api.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return snap.signers(), nil
}

func (api *API) Proposals() map[common.Address]bool {
	api.dpos.lock.RLock()
	defer api.dpos.lock.RUnlock()

	proposals := make(map[common.Address]bool)
	for address, auth := range api.dpos.proposals {
		proposals[address] = auth
	}
	return proposals
}

func (api *API) Propose(address common.Address, auth bool) {
	api.dpos.lock.Lock()
	defer api.dpos.lock.Unlock()

	api.dpos.proposals[address] = auth
}

func (api *API) Discard(address common.Address) {
	api.dpos.lock.Lock()
	defer api.dpos.lock.Unlock()

	delete(api.dpos.proposals, address)
}
