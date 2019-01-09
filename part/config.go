
package params

import (
	"fmt"
	"math/big"

	"github.com/ddmchain/go-ddmchain/general"
)

var (
	MainnetGenesisHash = common.HexToHash("0x23707e8feaf2e1698ec6250aa6eaa51f2a06fddf3beecbe4fe3c18f35184bbe3")
	TestnetGenesisHash = common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d")
)

var (

	MainnetChainConfig = &ChainConfig{
		ChainId:        big.NewInt(101),
		HomesteadBlock: big.NewInt(1),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(2),
		EIP150Hash:     common.HexToHash("0x05ff48f6a66afc733b6e33147b3db10445990cefafa18270578ea0a761c0de7a"),
		EIP155Block:    big.NewInt(3),
		EIP158Block:    big.NewInt(3),
		ByzantiumBlock: big.NewInt(4),

		Dpos: &DposConfig{
			Period: 5,
			Epoch:  30000,
		},
	}

	TestnetChainConfig = &ChainConfig{
		ChainId:        big.NewInt(3),
		HomesteadBlock: big.NewInt(0),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(0),
		EIP150Hash:     common.HexToHash("0x41941023680923e0fe4d74a34bdac8141f2540e3ae90623718e47d66d1ca4a2d"),
		EIP155Block:    big.NewInt(10),
		EIP158Block:    big.NewInt(10),
		ByzantiumBlock: big.NewInt(11),

		DDMhash: new(DDMhashConfig),
	}

	RinkebyChainConfig = &ChainConfig{
		ChainId:        big.NewInt(4),
		HomesteadBlock: big.NewInt(1),
		DAOForkBlock:   nil,
		DAOForkSupport: true,
		EIP150Block:    big.NewInt(2),
		EIP150Hash:     common.HexToHash("0x9b095b36c15eaf13044373aef8ee0bd3a382a5abb92e402afa44b8249c3a90e9"),
		EIP155Block:    big.NewInt(3),
		EIP158Block:    big.NewInt(3),
		ByzantiumBlock: big.NewInt(4),

		Dpos: &DposConfig{
			Period: 15,
			Epoch:  30000,
		},
	}

	AllDDMhashProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), new(DDMhashConfig), nil}

	AllDposProtocolChanges = &ChainConfig{big.NewInt(1337), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), nil, &DposConfig{Period: 0, Epoch: 30000}}

	TestChainConfig = &ChainConfig{big.NewInt(1), big.NewInt(0), nil, false, big.NewInt(0), common.Hash{}, big.NewInt(0), big.NewInt(0), big.NewInt(0), new(DDMhashConfig), nil}
	TestRules       = TestChainConfig.Rules(new(big.Int))
)

type ChainConfig struct {
	ChainId *big.Int `json:"chainId"`

	HomesteadBlock *big.Int `json:"homesteadBlock,omitempty"`

	DAOForkBlock   *big.Int `json:"daoForkBlock,omitempty"`
	DAOForkSupport bool     `json:"daoForkSupport,omitempty"`

	EIP150Block *big.Int    `json:"eip150Block,omitempty"`
	EIP150Hash  common.Hash `json:"eip150Hash,omitempty"`

	EIP155Block *big.Int `json:"eip155Block,omitempty"`
	EIP158Block *big.Int `json:"eip158Block,omitempty"`

	ByzantiumBlock *big.Int `json:"byzantiumBlock,omitempty"`

	DDMhash *DDMhashConfig `json:"ddmhash,omitempty"`
	Dpos *DposConfig `json:"dpos,omitempty"`
}

type DDMhashConfig struct{}

func (c *DDMhashConfig) String() string {
	return "ddmhash"
}

type DposConfig struct {
	Period uint64 `json:"period"`
	Epoch  uint64 `json:"epoch"`
}

func (c *DposConfig) String() string {
	return "dpos"
}

func (c *ChainConfig) String() string {
	var engine interface{}
	switch {
	case c.DDMhash != nil:
		engine = c.DDMhash
	case c.Dpos != nil:
		engine = c.Dpos
	default:
		engine = "unknown"
	}
	return fmt.Sprintf("{ChainID: %v Homestead: %v DAO: %v DAOSupport: %v EIP150: %v EIP155: %v EIP158: %v Byzantium: %v Engine: %v}",
		c.ChainId,
		c.HomesteadBlock,
		c.DAOForkBlock,
		c.DAOForkSupport,
		c.EIP150Block,
		c.EIP155Block,
		c.EIP158Block,
		c.ByzantiumBlock,
		engine,
	)
}

func (c *ChainConfig) IsHomestead(num *big.Int) bool {
	return isForked(c.HomesteadBlock, num)
}

func (c *ChainConfig) IsDAOFork(num *big.Int) bool {
	return isForked(c.DAOForkBlock, num)
}

func (c *ChainConfig) IsEIP150(num *big.Int) bool {
	return isForked(c.EIP150Block, num)
}

func (c *ChainConfig) IsEIP155(num *big.Int) bool {
	return isForked(c.EIP155Block, num)
}

func (c *ChainConfig) IsEIP158(num *big.Int) bool {
	return isForked(c.EIP158Block, num)
}

func (c *ChainConfig) IsByzantium(num *big.Int) bool {
	return isForked(c.ByzantiumBlock, num)
}

func (c *ChainConfig) GasTable(num *big.Int) GasTable {
	if num == nil {
		return GasTableHomestead
	}
	switch {
	case c.IsEIP158(num):
		return GasTableEIP158
	case c.IsEIP150(num):
		return GasTableEIP150
	default:
		return GasTableHomestead
	}
}

func (c *ChainConfig) CheckCompatible(newcfg *ChainConfig, height uint64) *ConfigCompatError {
	bhead := new(big.Int).SetUint64(height)

	var lasterr *ConfigCompatError
	for {
		err := c.checkCompatible(newcfg, bhead)
		if err == nil || (lasterr != nil && err.RewindTo == lasterr.RewindTo) {
			break
		}
		lasterr = err
		bhead.SetUint64(err.RewindTo)
	}
	return lasterr
}

func (c *ChainConfig) checkCompatible(newcfg *ChainConfig, head *big.Int) *ConfigCompatError {
	if isForkIncompatible(c.HomesteadBlock, newcfg.HomesteadBlock, head) {
		return newCompatError("Homestead fork block", c.HomesteadBlock, newcfg.HomesteadBlock)
	}
	if isForkIncompatible(c.DAOForkBlock, newcfg.DAOForkBlock, head) {
		return newCompatError("DAO fork block", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if c.IsDAOFork(head) && c.DAOForkSupport != newcfg.DAOForkSupport {
		return newCompatError("DAO fork support flag", c.DAOForkBlock, newcfg.DAOForkBlock)
	}
	if isForkIncompatible(c.EIP150Block, newcfg.EIP150Block, head) {
		return newCompatError("EIP150 fork block", c.EIP150Block, newcfg.EIP150Block)
	}
	if isForkIncompatible(c.EIP155Block, newcfg.EIP155Block, head) {
		return newCompatError("EIP155 fork block", c.EIP155Block, newcfg.EIP155Block)
	}
	if isForkIncompatible(c.EIP158Block, newcfg.EIP158Block, head) {
		return newCompatError("EIP158 fork block", c.EIP158Block, newcfg.EIP158Block)
	}
	if c.IsEIP158(head) && !configNumEqual(c.ChainId, newcfg.ChainId) {
		return newCompatError("EIP158 chain ID", c.EIP158Block, newcfg.EIP158Block)
	}
	if isForkIncompatible(c.ByzantiumBlock, newcfg.ByzantiumBlock, head) {
		return newCompatError("Byzantium fork block", c.ByzantiumBlock, newcfg.ByzantiumBlock)
	}
	return nil
}

func isForkIncompatible(s1, s2, head *big.Int) bool {
	return (isForked(s1, head) || isForked(s2, head)) && !configNumEqual(s1, s2)
}

func isForked(s, head *big.Int) bool {
	if s == nil || head == nil {
		return false
	}
	return s.Cmp(head) <= 0
}

func configNumEqual(x, y *big.Int) bool {
	if x == nil {
		return y == nil
	}
	if y == nil {
		return x == nil
	}
	return x.Cmp(y) == 0
}

type ConfigCompatError struct {
	What string

	StoredConfig, NewConfig *big.Int

	RewindTo uint64
}

func newCompatError(what string, storedblock, newblock *big.Int) *ConfigCompatError {
	var rew *big.Int
	switch {
	case storedblock == nil:
		rew = newblock
	case newblock == nil || storedblock.Cmp(newblock) < 0:
		rew = storedblock
	default:
		rew = newblock
	}
	err := &ConfigCompatError{what, storedblock, newblock, 0}
	if rew != nil && rew.Sign() > 0 {
		err.RewindTo = rew.Uint64() - 1
	}
	return err
}

func (err *ConfigCompatError) Error() string {
	return fmt.Sprintf("mismatching %s in database (have %d, want %d, rewindto %d)", err.What, err.StoredConfig, err.NewConfig, err.RewindTo)
}

type Rules struct {
	ChainId                                   *big.Int
	IsHomestead, IsEIP150, IsEIP155, IsEIP158 bool
	IsByzantium                               bool
}

func (c *ChainConfig) Rules(num *big.Int) Rules {
	chainId := c.ChainId
	if chainId == nil {
		chainId = new(big.Int)
	}
	return Rules{ChainId: new(big.Int).Set(chainId), IsHomestead: c.IsHomestead(num), IsEIP150: c.IsEIP150(num), IsEIP155: c.IsEIP155(num), IsEIP158: c.IsEIP158(num), IsByzantium: c.IsByzantium(num)}
}
