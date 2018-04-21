package ddmdpos

import (
	"bytes"
	"encoding/json"

	"github.com/ddmchain/go-ddmchain/common"
	"github.com/ddmchain/go-ddmchain/core/types"
	"github.com/ddmchain/go-ddmchain/ddmdb"
	"github.com/ddmchain/go-ddmchain/params"
	lru "github.com/hashicorp/golang-lru"
)

type Tally struct {
	Authorize bool `json:"authorize"`
	Votes     int  `json:"votes"`
}

type Vote struct {
	Signer    common.Address `json:"signer"`
	Block     uint64         `json:"block"`
	Address   common.Address `json:"address"`
	Authorize bool           `json:"authorize"`
}

type Archive struct {
	config   *params.DPosConfig
	sigcache *lru.ARCCache

	Number  uint64                      `json:"number"`
	Hash    common.Hash                 `json:"hash"`
	Signers map[common.Address]struct{} `json:"signers"`
	Recents map[uint64]common.Address   `json:"recents"`
	Votes   []*Vote                     `json:"votes"`
	Tally   map[common.Address]Tally    `json:"tally"`
}

func newArchive(config *params.DPosConfig, sigcache *lru.ARCCache, number uint64, hash common.Hash, signers []common.Address) *Archive {
	archive := &Archive{
		config:   config,
		sigcache: sigcache,
		Number:   number,
		Hash:     hash,
		Signers:  make(map[common.Address]struct{}),
		Recents:  make(map[uint64]common.Address),
		Tally:    make(map[common.Address]Tally),
	}
	for _, signer := range signers {
		archive.Signers[signer] = struct{}{}
	}
	return archive
}

func loadArchive(config *params.DPosConfig, sigcache *lru.ARCCache, db ddmdb.Database, hash common.Hash) (*Archive, error) {
	blob, err := db.Get(append([]byte("ddmdpos-"), hash[:]...))
	if err != nil {
		return nil, err
	}
	archive := new(Archive)
	if err := json.Unmarshal(blob, archive); err != nil {
		return nil, err
	}
	archive.config = config
	archive.sigcache = sigcache

	return archive, nil
}

func (s *Archive) store(db ddmdb.Database) error {
	blob, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return db.Put(append([]byte("ddmdpos-"), s.Hash[:]...), blob)
}

func (s *Archive) copy() *Archive {
	cpy := &Archive{
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

func (s *Archive) validVote(address common.Address, authorize bool) bool {
	_, signer := s.Signers[address]
	return (signer && !authorize) || (!signer && authorize)
}

func (s *Archive) cast(address common.Address, authorize bool) bool {
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

func (s *Archive) uncast(address common.Address, authorize bool) bool {
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

func (s *Archive) apply(headers []*types.Header) (*Archive, error) {
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
	archive := s.copy()

	for _, header := range headers {
		number := header.Number.Uint64()
		if number%s.config.Epoch == 0 {
			archive.Votes = nil
			archive.Tally = make(map[common.Address]Tally)
		}
		if limit := uint64(len(archive.Signers)/2 + 1); number >= limit {
			delete(archive.Recents, number-limit)
		}
		signer, err := ecrecover(header, s.sigcache)
		if err != nil {
			return nil, err
		}
		if _, ok := archive.Signers[signer]; !ok {
			return nil, errUnauthorized
		}
		for _, recent := range archive.Recents {
			if recent == signer {
				return nil, errUnauthorized
			}
		}
		archive.Recents[number] = signer

		for i, vote := range archive.Votes {
			if vote.Signer == signer && vote.Address == header.Coinbase {
				archive.uncast(vote.Address, vote.Authorize)

				archive.Votes = append(archive.Votes[:i], archive.Votes[i+1:]...)
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
		if archive.cast(header.Coinbase, authorize) {
			archive.Votes = append(archive.Votes, &Vote{
				Signer:    signer,
				Block:     number,
				Address:   header.Coinbase,
				Authorize: authorize,
			})
		}
		if tally := archive.Tally[header.Coinbase]; tally.Votes > len(archive.Signers)/2 {
			if tally.Authorize {
				archive.Signers[header.Coinbase] = struct{}{}
			} else {
				delete(archive.Signers, header.Coinbase)

				if limit := uint64(len(archive.Signers)/2 + 1); number >= limit {
					delete(archive.Recents, number-limit)
				}
				for i := 0; i < len(archive.Votes); i++ {
					if archive.Votes[i].Signer == header.Coinbase {
						archive.uncast(archive.Votes[i].Address, archive.Votes[i].Authorize)

						archive.Votes = append(archive.Votes[:i], archive.Votes[i+1:]...)

						i--
					}
				}
			}
			for i := 0; i < len(archive.Votes); i++ {
				if archive.Votes[i].Address == header.Coinbase {
					archive.Votes = append(archive.Votes[:i], archive.Votes[i+1:]...)
					i--
				}
			}
			delete(archive.Tally, header.Coinbase)
		}
	}
	archive.Number += uint64(len(headers))
	archive.Hash = headers[len(headers)-1].Hash()

	return archive, nil
}

func (s *Archive) ableSigners(number uint64) []common.Address {
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

func (s *Archive) inturn(number uint64, signer common.Address, hash common.Hash) bool {
	var lowHash int64
	b_buf := bytes.NewBuffer(hash[len(hash)-8:])
	binary.Read(b_buf, binary.BigEndian, &lowHash)

	signers, offset := s.ableSigners(number), 0
	sigLen := len(signers)

	rand.Seed(lowHash)
 	curTurn := rand.Intn(sigLen)

	for offset < sigLen && signers[offset] != signer {
		offset++
	}

	return (curTurn == offset)
}
