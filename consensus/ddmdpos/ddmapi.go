package ddmdpos

import (
	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/consensus"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/rpc"
)

type DAPI struct {
	chain  consensus.ChainReader
	dpos *DPos
}

func (d *DAPI) GetArchive(number *rpc.BlockNumber) (*Archive, error) {
	var header *types.Header
	if number == nil || *number == rpc.LatestBlockNumber {
		header = d.chain.CurrentHeader()
	} else {
		header = d.chain.GetHeaderByNumber(uint64(number.Int64()))
	}
	if header == nil {
		return nil, errUnknownBlock
	}
	return d.dpos.archive(d.chain, header.Number.Uint64(), header.Hash(), nil)
}

func (d *DAPI) GetArchiveAtHash(hash common.Hash) (*Archive, error) {
	header := d.chain.GetHeaderByHash(hash)
	if header == nil {
		return nil, errUnknownBlock
	}
	return d.dpos.archive(d.chain, header.Number.Uint64(), header.Hash(), nil)
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
	archive, err := d.dpos.archive(d.chain, header.Number.Uint64(), header.Hash(), nil)
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
	archive, err := d.dpos.archive(d.chain, header.Number.Uint64(), header.Hash(), nil)
	if err != nil {
		return nil, err
	}
	return archive.signers(), nil
}

func (d *DAPI) Proposals() map[common.Address]bool {
	d.dpos.lock.RLock()
	defer d.dpos.lock.RUnlock()

	proposals := make(map[common.Address]bool)
	for address, auth := range d.dpos.proposals {
		proposals[address] = auth
	}
	return proposals
}

func (d *DAPI) Propose(address common.Address, auth bool) {
	d.dpos.lock.Lock()
	defer d.dpos.lock.Unlock()

	d.dpos.proposals[address] = auth
}

func (d *DAPI) Discard(address common.Address) {
	d.dpos.lock.Lock()
	defer d.dpos.lock.Unlock()

	delete(d.dpos.proposals, address)
}
