package ddmdpos

import (
	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/consensus"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/rpc"
)

type DAPI struct {
	chain  consensus.ChainReader
	ddmdpos *DDMDPos
}

func (d *DAPI) GetSnapshot(number *rpc.BlockNumber) (*Archive, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = d.chain.CurrentHeader()
	} else {
		header = d.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}
	return d.ddmdpos.snapshot(d.chain, header.Number.Uint64(), header.Hash(), nil)
}

func (d *DAPI) GetSnapshotAtHash(hash common.Hash) (*Archive, error) {
	header := d.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return d.ddmdpos.snapshot(d.chain, header.Number.Uint64(), header.Hash(), nil)
}

func (d *DAPI) GetSigners(number *rpc.BlockNumber) ([]common.Address, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = d.chain.CurrentHeader()
	} else {
		header = d.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}
	archive, err := d.ddmdpos.snapshot(d.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return archive.signers(), nil
}

func (d *DAPI) GetSignersAtHash(hash common.Hash) ([]common.Address, error) {
	header := d.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	archive, err := d.ddmdpos.snapshot(d.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return archive.signers(), nil
}

func (d *DAPI) Proposals() map[common.Address]bool {
	d.ddmdpos.lock.RLock()
	defer d.ddmdpos.lock.RUnlock()

	proposals := make(map[common.Address]bool)
	for address, auth := range d.ddmdpos.proposals {
		proposals[address] = auth
	}
	return proposals
}

func (d *DAPI) Propose(address common.Address, auth bool) {
	d.ddmdpos.lock.Lock()
	defer d.ddmdpos.lock.Unlock()

	d.ddmdpos.proposals[address] = auth
}

func (d *DAPI) Discard(address common.Address) {
	d.ddmdpos.lock.Lock()
	defer d.ddmdpos.lock.Unlock()

	delete(d.ddmdpos.proposals, address)
}
