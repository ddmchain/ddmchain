
// +build none

package main

import (
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"sort"
	"strconv"

	"github.com/ddmchain/go-ddmchain/major"
	"github.com/ddmchain/go-ddmchain/ptl"
)

type allocItem struct{ Addr, Balance *big.Int }

type allocList []allocItem

func (a allocList) Len() int           { return len(a) }
func (a allocList) Less(i, j int) bool { return a[i].Addr.Cmp(a[j].Addr) < 0 }
func (a allocList) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func makelist(g *core.Genesis) allocList {
	a := make(allocList, 0, len(g.Alloc))
	for addr, account := range g.Alloc {
		if len(account.Storage) > 0 || len(account.Code) > 0 || account.Nonce != 0 {
			panic(fmt.Sprintf("can't encode account %x", addr))
		}
		a = append(a, allocItem{addr.Big(), account.Balance})
	}
	sort.Sort(a)
	return a
}

func makealloc(g *core.Genesis) string {
	a := makelist(g)
	data, err := rlp.EncodeToBytes(a)
	if err != nil {
		panic(err)
	}
	return strconv.QuoteToASCII(string(data))
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "Usage: mkalloc genesis.json")
		os.Exit(1)
	}

	g := new(core.Genesis)
	file, err := os.Open(os.Args[1])
	if err != nil {
		panic(err)
	}
	if err := json.NewDecoder(file).Decode(g); err != nil {
		panic(err)
	}
	fmt.Println("const allocData =", makealloc(g))
}
