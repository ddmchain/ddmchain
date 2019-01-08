
package light

import (
	"encoding/binary"
	"errors"
	"math/big"
	"time"

	"github.com/ddmchain/go-ddmchain/general"
	"github.com/ddmchain/go-ddmchain/general/bitutil"
	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/major/types"
	"github.com/ddmchain/go-ddmchain/ddmpv"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/part"
	"github.com/ddmchain/go-ddmchain/ptl"
	"github.com/ddmchain/go-ddmchain/tree"
)

const (

	CHTFrequencyClient = 32768

	CHTFrequencyServer = 4096

	HelperTrieConfirmations        = 2048 
	HelperTrieProcessConfirmations = 256  
)

type trustedCheckpoint struct {
	name                                string
	sectionIdx                          uint64
	sectionHead, chtRoot, bloomTrieRoot common.Hash
}

var (
	mainnetCheckpoint = trustedCheckpoint{
		name:          "mainnet",
		sectionIdx:    153,
		sectionHead:   common.HexToHash("04c2114a8cbe49ba5c37a03cc4b4b8d3adfc0bd2c78e0e726405dd84afca1d63"),
		chtRoot:       common.HexToHash("d7ec603e5d30b567a6e894ee7704e4603232f206d3e5a589794cec0c57bf318e"),
		bloomTrieRoot: common.HexToHash("0b139b8fb692e21f663ff200da287192201c28ef5813c1ac6ba02a0a4799eef9"),
	}

	ropstenCheckpoint = trustedCheckpoint{
		name:          "ropsten",
		sectionIdx:    79,
		sectionHead:   common.HexToHash("1b1ba890510e06411fdee9bb64ca7705c56a1a4ce3559ddb34b3680c526cb419"),
		chtRoot:       common.HexToHash("71d60207af74e5a22a3e1cfbfc89f9944f91b49aa980c86fba94d568369eaf44"),
		bloomTrieRoot: common.HexToHash("70aca4b3b6d08dde8704c95cedb1420394453c1aec390947751e69ff8c436360"),
	}
)

var trustedCheckpoints = map[common.Hash]trustedCheckpoint{
	params.MainnetGenesisHash: mainnetCheckpoint,
	params.TestnetGenesisHash: ropstenCheckpoint,
}

var (
	ErrNoTrustedCht       = errors.New("No trusted canonical hash trie")
	ErrNoTrustedBloomTrie = errors.New("No trusted bloom trie")
	ErrNoHeader           = errors.New("Header not found")
	chtPrefix             = []byte("chtRoot-") 
	ChtTablePrefix        = "cht-"
)

type ChtNode struct {
	Hash common.Hash
	Td   *big.Int
}

func GetChtRoot(db ddmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

func GetChtV2Root(db ddmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	return GetChtRoot(db, (sectionIdx+1)*(CHTFrequencyClient/CHTFrequencyServer)-1, sectionHead)
}

func StoreChtRoot(db ddmdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(chtPrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

type ChtIndexerBackend struct {
	diskdb               ddmdb.Database
	triedb               *trie.Database
	section, sectionSize uint64
	lastHash             common.Hash
	trie                 *trie.Trie
}

func NewChtIndexer(db ddmdb.Database, clientMode bool) *core.ChainIndexer {
	var sectionSize, confirmReq uint64
	if clientMode {
		sectionSize = CHTFrequencyClient
		confirmReq = HelperTrieConfirmations
	} else {
		sectionSize = CHTFrequencyServer
		confirmReq = HelperTrieProcessConfirmations
	}
	idb := ddmdb.NewTable(db, "chtIndex-")
	backend := &ChtIndexerBackend{
		diskdb:      db,
		triedb:      trie.NewDatabase(ddmdb.NewTable(db, ChtTablePrefix)),
		sectionSize: sectionSize,
	}
	return core.NewChainIndexer(db, idb, backend, sectionSize, confirmReq, time.Millisecond*100, "cht")
}

func (c *ChtIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetChtRoot(c.diskdb, section-1, lastSectionHead)
	}
	var err error
	c.trie, err = trie.New(root, c.triedb)
	c.section = section
	return err
}

func (c *ChtIndexerBackend) Process(header *types.Header) {
	hash, num := header.Hash(), header.Number.Uint64()
	c.lastHash = hash

	td := core.GetTd(c.diskdb, hash, num)
	if td == nil {
		panic(nil)
	}
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], num)
	data, _ := rlp.EncodeToBytes(ChtNode{hash, td})
	c.trie.Update(encNumber[:], data)
}

func (c *ChtIndexerBackend) Commit() error {
	root, err := c.trie.Commit(nil)
	if err != nil {
		return err
	}
	c.triedb.Commit(root, false)

	if ((c.section+1)*c.sectionSize)%CHTFrequencyClient == 0 {
		log.Info("Storing CHT", "section", c.section*c.sectionSize/CHTFrequencyClient, "head", c.lastHash, "root", root)
	}
	StoreChtRoot(c.diskdb, c.section, c.lastHash, root)
	return nil
}

const (
	BloomTrieFrequency        = 32768
	ddmBloomBitsSection       = 4096
	ddmBloomBitsConfirmations = 256
)

var (
	bloomTriePrefix      = []byte("bltRoot-") 
	BloomTrieTablePrefix = "blt-"
)

func GetBloomTrieRoot(db ddmdb.Database, sectionIdx uint64, sectionHead common.Hash) common.Hash {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	data, _ := db.Get(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...))
	return common.BytesToHash(data)
}

func StoreBloomTrieRoot(db ddmdb.Database, sectionIdx uint64, sectionHead, root common.Hash) {
	var encNumber [8]byte
	binary.BigEndian.PutUint64(encNumber[:], sectionIdx)
	db.Put(append(append(bloomTriePrefix, encNumber[:]...), sectionHead.Bytes()...), root.Bytes())
}

type BloomTrieIndexerBackend struct {
	diskdb                                     ddmdb.Database
	triedb                                     *trie.Database
	section, parentSectionSize, bloomTrieRatio uint64
	trie                                       *trie.Trie
	sectionHeads                               []common.Hash
}

func NewBloomTrieIndexer(db ddmdb.Database, clientMode bool) *core.ChainIndexer {
	backend := &BloomTrieIndexerBackend{
		diskdb: db,
		triedb: trie.NewDatabase(ddmdb.NewTable(db, BloomTrieTablePrefix)),
	}
	idb := ddmdb.NewTable(db, "bltIndex-")

	var confirmReq uint64
	if clientMode {
		backend.parentSectionSize = BloomTrieFrequency
		confirmReq = HelperTrieConfirmations
	} else {
		backend.parentSectionSize = ddmBloomBitsSection
		confirmReq = HelperTrieProcessConfirmations
	}
	backend.bloomTrieRatio = BloomTrieFrequency / backend.parentSectionSize
	backend.sectionHeads = make([]common.Hash, backend.bloomTrieRatio)
	return core.NewChainIndexer(db, idb, backend, BloomTrieFrequency, confirmReq-ddmBloomBitsConfirmations, time.Millisecond*100, "bloomtrie")
}

func (b *BloomTrieIndexerBackend) Reset(section uint64, lastSectionHead common.Hash) error {
	var root common.Hash
	if section > 0 {
		root = GetBloomTrieRoot(b.diskdb, section-1, lastSectionHead)
	}
	var err error
	b.trie, err = trie.New(root, b.triedb)
	b.section = section
	return err
}

func (b *BloomTrieIndexerBackend) Process(header *types.Header) {
	num := header.Number.Uint64() - b.section*BloomTrieFrequency
	if (num+1)%b.parentSectionSize == 0 {
		b.sectionHeads[num/b.parentSectionSize] = header.Hash()
	}
}

func (b *BloomTrieIndexerBackend) Commit() error {
	var compSize, decompSize uint64

	for i := uint(0); i < types.BloomBitLength; i++ {
		var encKey [10]byte
		binary.BigEndian.PutUint16(encKey[0:2], uint16(i))
		binary.BigEndian.PutUint64(encKey[2:10], b.section)
		var decomp []byte
		for j := uint64(0); j < b.bloomTrieRatio; j++ {
			data, err := core.GetBloomBits(b.diskdb, i, b.section*b.bloomTrieRatio+j, b.sectionHeads[j])
			if err != nil {
				return err
			}
			decompData, err2 := bitutil.DecompressBytes(data, int(b.parentSectionSize/8))
			if err2 != nil {
				return err2
			}
			decomp = append(decomp, decompData...)
		}
		comp := bitutil.CompressBytes(decomp)

		decompSize += uint64(len(decomp))
		compSize += uint64(len(comp))
		if len(comp) > 0 {
			b.trie.Update(encKey[:], comp)
		} else {
			b.trie.Delete(encKey[:])
		}
	}
	root, err := b.trie.Commit(nil)
	if err != nil {
		return err
	}
	b.triedb.Commit(root, false)

	sectionHead := b.sectionHeads[b.bloomTrieRatio-1]
	log.Info("Storing bloom trie", "section", b.section, "head", sectionHead, "root", root, "compression", float64(compSize)/float64(decompSize))
	StoreBloomTrieRoot(b.diskdb, b.section, sectionHead, root)

	return nil
}
