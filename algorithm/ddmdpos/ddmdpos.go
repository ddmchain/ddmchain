package ddmdpos

import (
	"bytes"
	"errors"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/ddmchain/go-ddmchain/act"
	"github.com/ddmchain/go-ddmchain/public"
	"github.com/ddmchain/go-ddmchain/public/hexutil"
	"github.com/ddmchain/go-ddmchain/agreement"
	"github.com/ddmchain/go-ddmchain/agreement/misc"
	"github.com/ddmchain/go-ddmchain/kernel/state"
	"github.com/ddmchain/go-ddmchain/kernel/types"
	"github.com/ddmchain/go-ddmchain/code"
	"github.com/ddmchain/go-ddmchain/code/sha3"
	"github.com/ddmchain/go-ddmchain/data"
	"github.com/ddmchain/go-ddmchain/book"
	"github.com/ddmchain/go-ddmchain/content"
	"github.com/ddmchain/go-ddmchain/process"
	"github.com/ddmchain/go-ddmchain/remote"
	lru "github.com/hashicorp/golang-lru"
)

const (
	checkpointInterval = 1024
	inmemoryArchives  = 128
	inmemorySignatures = 4096

	wiggleTime = 500 * time.Millisecond
)

var (
	epochLength = uint64(30000)
	blockPeriod = uint64(15)

	extraVanity = 32
	extraSeal   = 65

	nonceAuthVote = hexutil.MustDecode("0xffffffffffffffff")
	nonceDropVote = hexutil.MustDecode("0x0000000000000000")

	uncleHash = types.CalcUncleHash(nil)

	diffInTurn = big.NewInt(2)
	diffNoTurn = big.NewInt(1)
)

var (
	errUnknownBlock = errors.New("unknown block")
	errInvalidCheckpointBeneficiary = errors.New("beneficiary in checkpoint block non-zero")
	errInvalidVote = errors.New("vote nonce not 0x00..0 or 0xff..f")
	errInvalidCheckpointVote = errors.New("vote nonce in checkpoint block non-zero")
	errMissingVanity = errors.New("extra-data 32 byte vanity prefix missing")
	errMissingSignature = errors.New("extra-data 65 byte suffix signature missing")
	errExtraSigners = errors.New("non-checkpoint block contains extra signer list")
	errInvalidCheckpointSigners = errors.New("invalid signer list on checkpoint block")
	errInvalidMixDigest = errors.New("non-zero mix digest")
	errInvalidUncleHash = errors.New("non empty uncle hash")
	errInvalidDifficulty = errors.New("invalid difficulty")
	ErrInvalidTimestamp = errors.New("invalid timestamp")
	errInvalidVotingChain = errors.New("invalid voting chain")
	errUnauthorized = errors.New("unauthorized")
	errWaitTransactions = errors.New("waiting for transactions")
)

type SignerFn func(accounts.Account, []byte) ([]byte, error)

func sigHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewKeccak256()

	rlp.Encode(hasher, []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra[:len(header.Extra)-65],
		header.MixDigest,
		header.Nonce,
	})
	hasher.Sum(hash[:0])
	return hash
}

func ecrecover(header *types.Header, sigcache *lru.ARCCache) (common.Address, error) {
	hash := header.Hash()
	if address, known := sigcache.Get(hash); known {
		return address.(common.Address), nil
	}
	if len(header.Extra) < extraSeal {
		return common.Address{}, errMissingSignature
	}
	signature := header.Extra[len(header.Extra)-extraSeal:]

	pubkey, err := crypto.Ecrecover(sigHash(header).Bytes(), signature)
	if err != nil {
		return common.Address{}, err
	}
	var signer common.Address
	copy(signer[:], crypto.Keccak256(pubkey[1:])[12:])

	sigcache.Add(hash, signer)
	return signer, nil
}

type DPos struct {
	config *params.DPosConfig
	db     ddmdb.Database

	recents    *lru.ARCCache
	signatures *lru.ARCCache

	proposals map[common.Address]bool

	signer common.Address
	signFn SignerFn
	lock   sync.RWMutex
}

func New(config *params.DPosConfig, db ddmdb.Database) *DPos {
	conf := *config
	if conf.Epoch == 0 {
		conf.Epoch = epochLength
	}
	recents, _ := lru.NewARC(inmemoryArchives)
	signatures, _ := lru.NewARC(inmemorySignatures)

	return &DPos{
		config:     &conf,
		db:         db,
		recents:    recents,
		signatures: signatures,
		proposals:  make(map[common.Address]bool),
	}
}

func (c *DPos) Author(header *types.Header) (common.Address, error) {
	return ecrecover(header, c.signatures)
}

func (c *DPos) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	return c.verifyHeader(chain, header, nil)
}

func (c *DPos) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	abort := make(chan struct{})
	results := make(chan error, len(headers))

	go func() {
		for i, header := range headers {
			err := c.verifyHeader(chain, header, headers[:i])

			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

func (c *DPos) verifyHeader(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	if header.Number == nil {
		return errUnknownBlock
	}
	number := header.Number.Uint64()

	if header.Time.Cmp(big.NewInt(time.Now().Unix())) > 0 {
		return consensus.ErrFutureBlock
	}
	checkpoint := (number % c.config.Epoch) == 0
	if checkpoint && header.Coinbase != (common.Address{}) {
		return errInvalidCheckpointBeneficiary
	}
	if !bytes.Equal(header.Nonce[:], nonceAuthVote) && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidVote
	}
	if checkpoint && !bytes.Equal(header.Nonce[:], nonceDropVote) {
		return errInvalidCheckpointVote
	}
	if len(header.Extra) < extraVanity {
		return errMissingVanity
	}
	if len(header.Extra) < extraVanity+extraSeal {
		return errMissingSignature
	}
	signersBytes := len(header.Extra) - extraVanity - extraSeal
	if !checkpoint && signersBytes != 0 {
		return errExtraSigners
	}
	if checkpoint && signersBytes%common.AddressLength != 0 {
		return errInvalidCheckpointSigners
	}
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	if header.UncleHash != uncleHash {
		return errInvalidUncleHash
	}
	if number > 0 {
		if header.Difficulty == nil || (header.Difficulty.Cmp(diffInTurn) != 0 && header.Difficulty.Cmp(diffNoTurn) != 0) {
			return errInvalidDifficulty
		}
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, false); err != nil {
		return err
	}
	return c.verifyCascadingFields(chain, header, parents)
}

func (c *DPos) verifyCascadingFields(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	if number == 0 {
		return nil
	}
	var parent *types.Header
	if len(parents) > 0 {
		parent = parents[len(parents)-1]
	} else {
		parent = chain.GetHeader(header.ParentHash, number-1)
	}
	if parent == nil || parent.Number.Uint64() != number-1 || parent.Hash() != header.ParentHash {
		return consensus.ErrUnknownAncestor
	}
	if parent.Time.Uint64()+c.config.Period > header.Time.Uint64() {
		return ErrInvalidTimestamp
	}
	arch, err := c.archive(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}
	if number%c.config.Epoch == 0 {
		signers := make([]byte, len(arch.Signers)*common.AddressLength)
		for i, signer := range arch.signers() {
			copy(signers[i*common.AddressLength:], signer[:])
		}
		extraSuffix := len(header.Extra) - extraSeal
		if !bytes.Equal(header.Extra[extraVanity:extraSuffix], signers) {
			return errInvalidCheckpointSigners
		}
	}
	return c.verifySeal(chain, header, parents)
}

func (c *DPos) archive(chain consensus.ChainReader, number uint64, hash common.Hash, parents []*types.Header) (*Archive, error) {
	var (
		headers []*types.Header
		arch    *Archive
	)
	for arch == nil {
		if s, ok := c.recents.Get(hash); ok {
			arch = s.(*Archive)
			break
		}
		if number%checkpointInterval == 0 {
			if s, err := loadArchive(c.config, c.signatures, c.db, hash); err == nil {
				log.Trace("Loaded voting archive form disk", "number", number, "hash", hash)
				arch = s
				break
			}
		}
		if number == 0 {
			genesis := chain.GetHeaderByNumber(0)
			if err := c.VerifyHeader(chain, genesis, false); err != nil {
				return nil, err
			}
			signers := make([]common.Address, (len(genesis.Extra)-extraVanity-extraSeal)/common.AddressLength)
			for i := 0; i < len(signers); i++ {
				copy(signers[i][:], genesis.Extra[extraVanity+i*common.AddressLength:])
			}
			arch = newArchive(c.config, c.signatures, 0, genesis.Hash(), signers)
			if err := arch.store(c.db); err != nil {
				return nil, err
			}
			log.Trace("Stored genesis voting archive to disk")
			break
		}
		var header *types.Header
		if len(parents) > 0 {
			header = parents[len(parents)-1]
			if header.Hash() != hash || header.Number.Uint64() != number {
				return nil, consensus.ErrUnknownAncestor
			}
			parents = parents[:len(parents)-1]
		} else {
			header = chain.GetHeader(hash, number)
			if header == nil {
				return nil, consensus.ErrUnknownAncestor
			}
		}
		headers = append(headers, header)
		number, hash = number-1, header.ParentHash
	}
	for i := 0; i < len(headers)/2; i++ {
		headers[i], headers[len(headers)-1-i] = headers[len(headers)-1-i], headers[i]
	}
	arch, err := arch.apply(headers)
	if err != nil {
		return nil, err
	}
	c.recents.Add(arch.Hash, arch)

	if arch.Number%checkpointInterval == 0 && len(headers) > 0 {
		if err = arch.store(c.db); err != nil {
			return nil, err
		}
		log.Trace("Stored voting archive to disk", "number", arch.Number, "hash", arch.Hash)
	}
	return arch, err
}

func (c *DPos) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if len(block.Uncles()) > 0 {
		return errors.New("uncles not allowed")
	}
	return nil
}

func (c *DPos) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	return c.verifySeal(chain, header, nil)
}

func (c *DPos) verifySeal(chain consensus.ChainReader, header *types.Header, parents []*types.Header) error {
	number := header.Number.Uint64()
	if number == 0 {
		return errUnknownBlock
	}
	arch, err := c.archive(chain, number-1, header.ParentHash, parents)
	if err != nil {
		return err
	}

	signer, err := ecrecover(header, c.signatures)
	if err != nil {
		return err
	}
	if _, ok := arch.Signers[signer]; !ok {
		return errUnauthorized
	}
	for seen, recent := range arch.Recents {
		if recent == signer {
			if limit := uint64(len(arch.Signers)/2 + 1); seen > number-limit {
				return errUnauthorized
			}
		}
	}
	inturn := arch.inturn(header.Number.Uint64(), signer, header.ParentHash)
	if inturn && header.Difficulty.Cmp(diffInTurn) != 0 {
		return errInvalidDifficulty
	}
	if !inturn && header.Difficulty.Cmp(diffNoTurn) != 0 {
		return errInvalidDifficulty
	}
	return nil
}

func (c *DPos) Prepare(chain consensus.ChainReader, header *types.Header) error {
	header.Coinbase = common.Address{}
	header.Nonce = types.BlockNonce{}

	number := header.Number.Uint64()
	arch, err := c.archive(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return err
	}
	if number%c.config.Epoch != 0 {
		c.lock.RLock()

		addresses := make([]common.Address, 0, len(c.proposals))
		for address, authorize := range c.proposals {
			if arch.validVote(address, authorize) {
				addresses = append(addresses, address)
			}
		}
		if len(addresses) > 0 {
			header.Coinbase = addresses[rand.Intn(len(addresses))]
			if c.proposals[header.Coinbase] {
				copy(header.Nonce[:], nonceAuthVote)
			} else {
				copy(header.Nonce[:], nonceDropVote)
			}
		}
		c.lock.RUnlock()
	}
	header.Difficulty = calcDifficulty(arch, c.signer)

	if len(header.Extra) < extraVanity {
		header.Extra = append(header.Extra, bytes.Repeat([]byte{0x00}, extraVanity-len(header.Extra))...)
	}
	header.Extra = header.Extra[:extraVanity]

	if number%c.config.Epoch == 0 {
		for _, signer := range arch.signers() {
			header.Extra = append(header.Extra, signer[:]...)
		}
	}
	header.Extra = append(header.Extra, make([]byte, extraSeal)...)

	header.MixDigest = common.Hash{}

	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Time = new(big.Int).Add(parent.Time, new(big.Int).SetUint64(c.config.Period))
	if header.Time.Int64() < time.Now().Unix() {
		header.Time = big.NewInt(time.Now().Unix())
	}
	return nil
}

func (c *DPos) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
	header.UncleHash = types.CalcUncleHash(nil)

	return types.NewBlock(header, txs, nil, receipts), nil
}

func (c *DPos) Authorize(signer common.Address, signFn SignerFn) {
	c.lock.Lock()
	defer c.lock.Unlock()

	c.signer = signer
	c.signFn = signFn
}

func (c *DPos) Seal(chain consensus.ChainReader, block *types.Block, stop <-chan struct{}) (*types.Block, error) {
	header := block.Header()

	number := header.Number.Uint64()
	if number == 0 {
		return nil, errUnknownBlock
	}
	if c.config.Period == 0 && len(block.Transactions()) == 0 {
		return nil, errWaitTransactions
	}
	c.lock.RLock()
	signer, signFn := c.signer, c.signFn
	c.lock.RUnlock()

	arch, err := c.archive(chain, number-1, header.ParentHash, nil)
	if err != nil {
		return nil, err
	}
	if _, authorized := arch.Signers[signer]; !authorized {
		return nil, errUnauthorized
	}
	for seen, recent := range arch.Recents {
		if recent == signer {
			if limit := uint64(len(arch.Signers)/2 + 1); number < limit || seen > number-limit {
				log.Info("Signed recently, must wait for others")
				<-stop
				return nil, nil
			}
		}
	}
	delay := time.Unix(header.Time.Int64(), 0).Sub(time.Now()) // nolint: gosimple
	if header.Difficulty.Cmp(diffNoTurn) == 0 {
		wiggle := time.Duration(len(arch.Signers)/2+1) * wiggleTime
		delay += time.Duration(rand.Int63n(int64(wiggle)))

		log.Trace("Out-of-turn signing requested", "wiggle", common.PrettyDuration(wiggle))
	}
	log.Trace("Waiting for slot to sign and propagate", "delay", common.PrettyDuration(delay))

	select {
	case <-stop:
		return nil, nil
	case <-time.After(delay):
	}
	sighash, err := signFn(accounts.Account{Address: signer}, sigHash(header).Bytes())
	if err != nil {
		return nil, err
	}
	copy(header.Extra[len(header.Extra)-extraSeal:], sighash)

	return block.WithSeal(header), nil
}

func (c *DPos) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	arch, err := c.archive(chain, parent.Number.Uint64(), parent.Hash(), nil)
	if err != nil {
		return nil
	}
	return calcDifficulty(arch, c.signer)
}

func calcDifficulty(arch *Archive, signer common.Address) *big.Int {
	if arch.inturn(arch.Number+1, signer, arch.Hash) {
		return new(big.Int).Set(diffInTurn)
	}
	return new(big.Int).Set(diffNoTurn)
}

func (c *DPos) APIs(chain consensus.ChainReader) []rpc.API {
	return []rpc.API{{
		Namespace: "dpos",
		Version:   "1.0",
		Service:   &EAPI{chain: chain, dpos: c},
		Public:    false,
	}}
}
