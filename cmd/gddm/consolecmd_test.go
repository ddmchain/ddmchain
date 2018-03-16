// Copyright 2016 The go-ddmchain Authors
// This file is part of go-ddmchain.
//
// go-ddmchain is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ddmchain is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ddmchain. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"crypto/rand"
	"math/big"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/ddmchain/go-ddmchain/params"
)

const (
	ipcAPIs  = "admin:1.0 debug:1.0 ddm:1.0 miner:1.0 net:1.0 personal:1.0 rpc:1.0 shh:1.0 txpool:1.0 web3:1.0"
	httpAPIs = "ddm:1.0 net:1.0 rpc:1.0 web3:1.0"
)

// Tests that a node embedded within a console can be started up properly and
// then terminated by closing the input stream.
func TestConsoleWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"

	// Start a gddm console, make sure it's cleaned up and terminate the console
	gddm := runGddm(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--ddmxbase", coinbase, "--shh",
		"console")

	// Gather all the infos the welcome message needs to contain
	gddm.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	gddm.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	gddm.SetTemplateFunc("gover", runtime.Version)
	gddm.SetTemplateFunc("gddmver", func() string { return params.Version })
	gddm.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	gddm.SetTemplateFunc("apis", func() string { return ipcAPIs })

	// Verify the actual welcome message to the required template
	gddm.Expect(`
Welcome to the Gddm JavaScript console!

instance: Gddm/v{{gddmver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{.DDMXbase}}
at block: 0 ({{niltime}})
 datadir: {{.Datadir}}
 modules: {{apis}}

> {{.InputLine "exit"}}
`)
	gddm.ExpectExit()
}

// Tests that a console can be attached to a running node via various means.
func TestIPCAttachWelcome(t *testing.T) {
	// Configure the instance for IPC attachement
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	var ipc string
	if runtime.GOOS == "windows" {
		ipc = `\\.\pipe\gddm` + strconv.Itoa(trulyRandInt(100000, 999999))
	} else {
		ws := tmpdir(t)
		defer os.RemoveAll(ws)
		ipc = filepath.Join(ws, "gddm.ipc")
	}
	// Note: we need --shh because testAttachWelcome checks for default
	// list of ipc modules and shh is included there.
	gddm := runGddm(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--ddmxbase", coinbase, "--shh", "--ipcpath", ipc)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gddm, "ipc:"+ipc, ipcAPIs)

	gddm.Interrupt()
	gddm.ExpectExit()
}

func TestHTTPAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P
	gddm := runGddm(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--ddmxbase", coinbase, "--rpc", "--rpcport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gddm, "http://localhost:"+port, httpAPIs)

	gddm.Interrupt()
	gddm.ExpectExit()
}

func TestWSAttachWelcome(t *testing.T) {
	coinbase := "0x8605cdbbdb6d264aa742e77020dcbc58fcdce182"
	port := strconv.Itoa(trulyRandInt(1024, 65536)) // Yeah, sometimes this will fail, sorry :P

	gddm := runGddm(t,
		"--port", "0", "--maxpeers", "0", "--nodiscover", "--nat", "none",
		"--ddmxbase", coinbase, "--ws", "--wsport", port)

	time.Sleep(2 * time.Second) // Simple way to wait for the RPC endpoint to open
	testAttachWelcome(t, gddm, "ws://localhost:"+port, httpAPIs)

	gddm.Interrupt()
	gddm.ExpectExit()
}

func testAttachWelcome(t *testing.T, gddm *testgddm, endpoint, apis string) {
	// Attach to a running gddm note and terminate immediately
	attach := runGddm(t, "attach", endpoint)
	defer attach.ExpectExit()
	attach.CloseStdin()

	// Gather all the infos the welcome message needs to contain
	attach.SetTemplateFunc("goos", func() string { return runtime.GOOS })
	attach.SetTemplateFunc("goarch", func() string { return runtime.GOARCH })
	attach.SetTemplateFunc("gover", runtime.Version)
	attach.SetTemplateFunc("gddmver", func() string { return params.Version })
	attach.SetTemplateFunc("ddmxbase", func() string { return gddm.DDMXbase })
	attach.SetTemplateFunc("niltime", func() string { return time.Unix(0, 0).Format(time.RFC1123) })
	attach.SetTemplateFunc("ipc", func() bool { return strings.HasPrefix(endpoint, "ipc") })
	attach.SetTemplateFunc("datadir", func() string { return gddm.Datadir })
	attach.SetTemplateFunc("apis", func() string { return apis })

	// Verify the actual welcome message to the required template
	attach.Expect(`
Welcome to the Gddm JavaScript console!

instance: Gddm/v{{gddmver}}/{{goos}}-{{goarch}}/{{gover}}
coinbase: {{ddmxbase}}
at block: 0 ({{niltime}}){{if ipc}}
 datadir: {{datadir}}{{end}}
 modules: {{apis}}

> {{.InputLine "exit" }}
`)
	attach.ExpectExit()
}

// trulyRandInt generates a crypto random integer used by the console tests to
// not clash network ports with other tests running cocurrently.
func trulyRandInt(lo, hi int) int {
	num, _ := rand.Int(rand.Reader, big.NewInt(int64(hi-lo)))
	return int(num.Int64()) + lo
}
