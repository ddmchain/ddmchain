
package ddmhash

import (
	crand "crypto/rand"
	"math"
	"math/big"
	"math/rand"
	"runtime"
	"sync"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/algorithm"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/sign"
)

func (ddmhash *DDMhash) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {

	if ddmhash.config.PowMode == ModeFake || ddmhash.config.PowMode == ModeFullFake {
		header := block.Header()
		header.Nonce, header.MixDigest = types.BlockNonce{}, common.Hash{}
		return block.WithSeal(header), nil
	}

	if ddmhash.shared != nil {
		return ddmhash.shared.Seal(chain, block, stop)
	}

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
		threads = 0 
	}
	var pend sync.WaitGroup
	for i := 0; i < threads; i++ {
		pend.Add(1)
		go func(id int, nonce uint64) {
			defer pend.Done()
			ddmhash.mine(block, id, nonce, abort, found)
		}(i, uint64(ddmhash.rand.Int63()))
	}

	var result *types.Block
	select {
	case <-stop:

		close(abort)
	case result = <-found:

		close(abort)
	case <-ddmhash.update:

		close(abort)
		pend.Wait()
		return ddmhash.Seal(chain, block, stop)
	}

	pend.Wait()
	return result, nil
}

func (ddmhash *DDMhash) mine(block *types.Block, id int, seed uint64, abort chan struct{}, found chan *types.Block) {

	var (
		header  = block.Header()
		hash    = header.HashNoNonce().Bytes()
		target  = new(big.Int).Div(maxUint256, header.Difficulty)
		number  = header.Number.Uint64()
		dataset = ddmhash.dataset(number)
	)

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

			logger.Trace("DDMhash nonce search aborted", "attempts", nonce-seed)
			ddmhash.hashrate.Mark(attempts)
			break search

		default:

			attempts++
			if (attempts % (1 << 15)) == 0 {
				ddmhash.hashrate.Mark(attempts)
				attempts = 0
			}

			digest, result := hashimotoFull(dataset.dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {

				header = types.CopyHeader(header)
				header.Nonce = types.EncodeNonce(nonce)
				header.MixDigest = common.BytesToHash(digest)

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

	runtime.KeepAlive(dataset)
}
