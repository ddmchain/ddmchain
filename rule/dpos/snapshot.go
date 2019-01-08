
package dpos

import (
	"bytes"
	"encoding/json"
	"encoding/binary"
	"math/rand"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/part"
	lru "github.com/hashicorp/golang-lru"
)

type Vote struct {
	Signer    common.Address `json:"signer"`    
	Block     uint64         `json:"block"`     
	Address   common.Address `json:"address"`   
	Authorize bool           `json:"authorize"` 
}

type Tally struct {
	Authorize bool `json:"authorize"` 
	Votes     int  `json:"votes"`     
}

type Snapshot struct {
	config   *params.DposConfig 
	sigcache *lru.ARCCache        

	Number  uint64                      `json:"number"`  
	Hash    common.Hash                 `json:"hash"`    
	Signers map[common.Address]struct{} `json:"signers"` 
	Recents map[uint64]common.Address   `json:"recents"` 
	Votes   []*Vote                     `json:"votes"`   
	Tally   map[common.Address]Tally    `json:"tally"`   
}

func newSnapshot(config *params.DposConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address) *Snapshot {
	snap := &Snapshot{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
		Tally:    make(map[common.Address]Tally),
	}
	for _, signer := range signers {
		snap.Signers[signer] = struct{}{}
	}
	return snap
}

func loadSnapshot(config *params.DposConfig, sigcache *lru.ARCCache, db ddmdb.Database, hash common.Hash) (*Snapshot, error) {
	blob, err := db.Get(append([]byte("dpos-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	snap := new(Snapshot)
	if err := json.Unmarshal(blob, snap); err != nil {
		return nil, err
	}
	snap.config = config
	snap.sigcache = sigcache

	return snap, nil
}

func (s *Snapshot) store(db ddmdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("dpos-"), s.Hash[:]...), blob)
}

func (s *Snapshot) copy() *Snapshot {
	cpy := &Snapshot{
		config:   s.config,
		sigcache: s.sigcache,
		Number:   s.Number,
		Hash:     s.Hash,
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
		Votes:    make([]*Vote, len(s.Votes)),
		Tally:    make(map[common.Address]Tally),
	}
	for signer := range s.Signers {
		cpy.Signers[signer] = struct{}{}
	}
	for block, signer := range s.Recents {
		cpy.Recents[block] = signer
	}
	for address, tally := range s.Tally {
		cpy.Tally[address] = tally
	}
	copy(cpy.Votes, s.Votes)

	return cpy
}

func (s *Snapshot) validVote(address common.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}

func (s *Snapshot) cast(address common.Address, authorize bool) bool {

	if !s.validVote(address, authorize) {
		return false
	}

	if old, ok := s.Tally[address]; ok {
		old.Votes++
		s.Tally[address] = old
	} else {
		s.Tally[address] = Tally{Authorize: authorize, Votes: 1}
	}
	return true
}

func (s *Snapshot) uncast(address common.Address, authorize bool) bool {

	tally, ok := s.Tally[address]
	if !ok {
		return false
	}

	if tally.Authorize != authorize {
		return false
	}

	if tally.Votes > 1 {
		tally.Votes--
		s.Tally[address] = tally
	} else {
		delete(s.Tally, address)
	}
	return true
}

func (s *Snapshot) apply(headers []*types.Header) (*Snapshot, error) {

	if len(headers) == 0 {
		return s, nil
	}

	for i := 0; i < len(headers)-1; i++ {
		if headers[i+1].Number.Uint64() != headers[i].Number.Uint64()+1 {
			return nil, errInvalidVotingChain
		}
	}
	if headers[0].Number.Uint64() != s.Number+1 {
		return nil, errInvalidVotingChain
	}

	snap := s.copy()

	for _, header := range headers {

		number := header.Number.Uint64()
		if number%s.config.Epoch == 0 {
			snap.Votes = nil
			snap.Tally = make(map[common.Address]Tally)
		}

		if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
			delete(snap.Recents, number-limit)
		}

		signer, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}
		if _, ok := snap.Signers[signer]; !ok {
			return nil, errUnauthorized
		}
		for _, recent := range snap.Recents {
			if recent == signer {
				return nil, errUnauthorized
			}
		}
		snap.Recents[number] = signer

		for i, vote := range snap.Votes {
			if vote.Signer == signer && vote.Address == header.Coinbase {

				snap.uncast(vote.Address, vote.Authorize)

				snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
				break 
			}
		}

		var authorize bool
		switch {
		case bytes.Equal(header.Nonce[:], nonceAuthVote):
			authorize = true
		case bytes.Equal(header.Nonce[:], nonceDropVote):
			authorize = false
		default:
			return nil, errInvalidVote
		}
		if snap.cast(header.Coinbase, authorize) {
			snap.Votes = append(snap.Votes, &Vote{
				Signer:    signer,
				Block:     number,
				Address:   header.Coinbase,
				Authorize: authorize,
			})
		}

		if tally := snap.Tally[header.Coinbase]; tally.Votes > len(snap.Signers)/2 {
			if tally.Authorize {
				snap.Signers[header.Coinbase] = struct{}{}
			} else {
				delete(snap.Signers, header.Coinbase)

				if limit := uint64(len(snap.Signers)/2 + 1); number >= limit {
					delete(snap.Recents, number-limit)
				}

				for i := 0; i < len(snap.Votes); i++ {
					if snap.Votes[i].Signer == header.Coinbase {

						snap.uncast(snap.Votes[i].Address, snap.Votes[i].Authorize)

						snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)

						i--
					}
				}
			}

			for i := 0; i < len(snap.Votes); i++ {
				if snap.Votes[i].Address == header.Coinbase {
					snap.Votes = append(snap.Votes[:i], snap.Votes[i+1:]...)
					i--
				}
			}
			delete(snap.Tally, header.Coinbase)
		}
	}
	snap.Number += uint64(len(headers))
	snap.Hash = headers[len(headers)-1].Hash()

	return snap, nil
}

func (s *Snapshot) signers() []common.Address {
	signers := make([]common.Address, 0, len(s.Signers))
	for signer := range s.Signers {
		signers = append(signers, signer)
	}
	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}
	return signers
}

func (s *Snapshot) ableSigners(number uint64) []common.Address {
	recents := make(map[uint64]common.Address)
	for block, recent := range s.Recents {
		recents[block] = recent
	}
	if limit := uint64(len(s.Signers)/2 + 1); number >= limit {
		delete(recents, number-limit)
	}

	signers := make([]common.Address, 0, len(s.Signers))
	for signer := range s.Signers {
		isRecent := false
		for _, recent := range recents {
			if recent == signer {
				isRecent = true
				break
			}
		}
		if (!isRecent) {
			signers = append(signers, signer)
		}
	}
	for i := 0; i < len(signers); i++ {
		for j := i + 1; j < len(signers); j++ {
			if bytes.Compare(signers[i][:], signers[j][:]) > 0 {
				signers[i], signers[j] = signers[j], signers[i]
			}
		}
	}
	return signers
}

func (s *Snapshot) inturn(number uint64, signer common.Address, hash common.Hash, confirmRand uint64) bool {
	var lowHash int64
	b_buf := bytes.NewBuffer(hash[len(hash)-8:])
	binary.Read(b_buf, binary.BigEndian, &lowHash)

	signers, offset := s.ableSigners(number), 0
	sigLen := len(signers)

	var curTurn int
	rand.Seed(lowHash)
	for i := uint64(0); i < confirmRand; i++ {
		curTurn = rand.Intn(sigLen)
	}

	for offset < sigLen && signers[offset] != signer {
		offset++
	}

	ret := (curTurn == offset)

	return ret
}
