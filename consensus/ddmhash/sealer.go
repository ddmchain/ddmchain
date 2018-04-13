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

package ddmhash

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"

	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/consensus"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/log"
)

// Seal implements consensus.Engine, attempting to find a nonce that satisfies
// the block's difficulty requirements.
func (ddmhash *DDMhash) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	// If we're running a fake PoW, simply return a 0 nonce immediately
	if ddmhash.config.PowMode == ModeFake || ddmhash.config.PowMode == ModeFullFake {
		header := block.Header()
		header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
		return block.WithSeal(header), nil
	}
	// If we're running a shared PoW, delegate sealing to it
	if ddmhash.shared != nil {
		return ddmhash.shared.Seal(chain, block, stop)
	}
	// Create a runner and the multiple search threads it directs
	abort := make(chan struct{})
	found := make(chan *types.Block)

	ddmhash.lock.Lock()
	threads := ddmhash.threads
	if ddmhash.rand == nil {
		seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
		if err != nil {
			ddmhash.lock.Unlock()
			return nil, err
		}
		ddmhash.rand = rand.New(rand.NewSource(seed.Int64()))
	}
	ddmhash.lock.Unlock()
	if threads == 0 {
		threads = runtime.NumCPU()
	}
	if threads < 0 {
		threads = 0 // Allows disabling local mining without extra logic around local/remote
	}
	var pend sync.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			ddmhash.mine(block, id, nonce, abort, found)
		}(i, uint64(ddmhash.rand.Int63()))
	}
	// Wait until sealing is terminated or a nonce is found
	var result *types.Block
	select {
	case <-stop:
		// Outside abort, stop all miner threads
		close(abort)
	case result = <-found:
		// One of the threads found a block, abort all others
		close(abort)
	case <-ddmhash.update:
		// Thread count was changed on user request, restart
		close(abort)
		pend.Wait()
		return ddmhash.Seal(chain, block, stop)
	}
	// Wait for all miners to terminate and return the block
	pend.Wait()
	return result, nil
}

// mine is the actual proof-of-work miner that searches for a nonce starting from
// seed that results in correct final block difficulty.
func (ddmhash *DDMhash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {
	// Extract some data from the header
	var (
		header  = block.Header()
		hash    = header.HashNoNonce().Bytes()
		target  = new(big.Int).Div(maxUint256, header.Difficulty)
		number  = header.Number.Uint64()
		dataset = ddmhash.dataset(number)
	)
	// Start generating random nonces until we abort or find a good one
	var (
		attempts = int64(0)
		nonce    = seed
	)
	logger := log.New("miner", id)
	logger.Trace("Started ddmhash search for new nonces", "seed", seed)
search:
	for {
		select {
		case <-abort:
			// Mining terminated, update stats and abort
			logger.Trace("DDMhash nonce search aborted", "attempts", nonce-seed)
			ddmhash.hashrate.Mark(attempts)
			break search

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				ddmhash.hashrate.Mark(attempts)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			digest, result := hashimotoFull(dataset.dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				// Correct nonce found, create a new header with it
				header = types.CopyHeader(header)
				header.Nonce = types.EncodeNonce(nonce)
				header.MixDigest = common.BytesToHash(digest)

				// Seal and return a block (if still needed)
				select {
				case found <- block.WithSeal(header):
					logger.Trace("DDMhash nonce found and reported", "attempts", nonce-seed, "nonce", nonce)
				case <-abort:
					logger.Trace("DDMhash nonce found but discarded", "attempts", nonce-seed, "nonce", nonce)
				}
				break search
			}
			nonce++
		}
	}
	// Datasets are unmapped in a finalizer. Ensure that the dataset stays live
	// during sealing so it's not unmapped while being read.
	runtime.KeepAlive(dataset)
}
