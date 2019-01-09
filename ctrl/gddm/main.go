
package main

import (
	"fmt"
	"os"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ddmchain/go-ddmchain/user"
	"github.com/ddmchain/go-ddmchain/user/keystore"
	"github.com/ddmchain/go-ddmchain/ctrl/utils"
	"github.com/ddmchain/go-ddmchain/cle"
	"github.com/ddmchain/go-ddmchain/ddm"
	"github.com/ddmchain/go-ddmchain/ddmct"
	"github.com/ddmchain/go-ddmchain/ddmin/debug"
	"github.com/ddmchain/go-ddmchain/sign"
	"github.com/ddmchain/go-ddmchain/rhythm"
	"github.com/ddmchain/go-ddmchain/pitch"
	"gopkg.in/urfave/cli.v1"
)

const (
	clientIdentifier = "gddm"
)

var (
	gitCommit = ""
	app = utils.NewApp(gitCommit, "the go-ddmchain command line interface")

	nodeFlags = []cli.Flag{
		utils.IdentityFlag,
		utils.UnlockedAccountFlag,
		utils.PasswordFileFlag,

		utils.DataDirFlag,
		utils.KeyStoreDirFlag,
		utils.NoUSBFlag,
		utils.DashboardEnabledFlag,
		utils.DashboardAddrFlag,
		utils.DashboardPortFlag,
		utils.DashboardRefreshFlag,
		utils.DashboardAssetsFlag,

		utils.TxPoolNoLocalsFlag,
		utils.TxPoolJournalFlag,
		utils.TxPoolRejournalFlag,
		utils.TxPoolPriceLimitFlag,
		utils.TxPoolPriceBumpFlag,
		utils.TxPoolAccountSlotsFlag,
		utils.TxPoolGlobalSlotsFlag,
		utils.TxPoolAccountQueueFlag,
		utils.TxPoolGlobalQueueFlag,
		utils.TxPoolLifetimeFlag,

		utils.GCModeFlag,

		utils.CacheFlag,
		utils.CacheDatabaseFlag,
		utils.CacheGCFlag,
		utils.TrieCacheGenFlag,
		utils.ListenPortFlag,
		utils.MaxPeersFlag,
		utils.MaxPendingPeersFlag,

		utils.NATFlag,

		utils.NoDiscoverFlag,

		utils.NetrestrictFlag,
		utils.NodeKeyFileFlag,

		utils.VMEnableDebugFlag,
		utils.NetworkIdFlag,
		utils.RPCCORSDomainFlag,
		utils.RPCVirtualHostsFlag,
		utils.DDMStatsURLFlag,
		utils.MetricsEnabledFlag,

		utils.GpoBlocksFlag,
		utils.GpoPercentileFlag,

		configFileFlag,
	}

	rpcFlags = []cli.Flag{
		utils.RPCEnabledFlag,
		utils.RPCListenAddrFlag,
		utils.RPCPortFlag,
		utils.RPCApiFlag,

	}

	whisperFlags = []cli.Flag{

	}
)

func init() {

	app.Action = gddm
	app.HideVersion = true
	app.Copyright = "Copyright The go-ddmchain Authors"
	app.Commands = []cli.Command{

		initCommand,
		importCommand,
		exportCommand,
		copydbCommand,
		removedbCommand,
		dumpCommand,

		monitorCommand,

		accountCommand,
		walletCommand,

		consoleCommand,
		attachCommand,
		javascriptCommand,

		versionCommand,
		bugCommand,
		licenseCommand,

		dumpConfigCommand,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = append(app.Flags, nodeFlags...)
	app.Flags = append(app.Flags, rpcFlags...)
	app.Flags = append(app.Flags, consoleFlags...)
	app.Flags = append(app.Flags, debug.Flags...)
	app.Flags = append(app.Flags, whisperFlags...)

	app.Before = func(ctx *cli.Context) error {
		runtime.GOMAXPROCS(runtime.NumCPU())
		if err := debug.Setup(ctx); err != nil {
			return err
		}

		go metrics.CollectProcessMetrics(3 * time.Second)

		utils.SetupNetwork(ctx)
		return nil
	}

	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		console.Stdin.Close()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func gddm(ctx *cli.Context) error {
	node := makeFullNode(ctx)
	startNode(ctx, node)
	node.Wait()
	return nil
}

func startNode(ctx *cli.Context, stack *node.Node) {

	utils.StartNode(stack)

	ks := stack.AccountManager().Backends(keystore.KeyStoreType)[0].(*keystore.KeyStore)

	passwords := utils.MakePasswordList(ctx)
	unlocks := strings.Split(ctx.GlobalString(utils.UnlockedAccountFlag.Name), ",")
	for i, account := range unlocks {
		if trimmed := strings.TrimSpace(account); trimmed != "" {
			unlockAccount(ctx, ks, trimmed, i, passwords)
		}
	}

	events := make(chan accounts.WalletEvent, 16)
	stack.AccountManager().Subscribe(events)

	go func() {

		rpcClient, err := stack.Attach()
		if err != nil {
			utils.Fatalf("Failed to attach to self: %v", err)
		}
		stateReader := ddmclient.NewClient(rpcClient)

		for _, wallet := range stack.AccountManager().Wallets() {
			if err := wallet.Open(""); err != nil {
				log.Warn("Failed to open wallet", "url", wallet.URL(), "err", err)
			}
		}

		for event := range events {
			switch event.Kind {
			case accounts.WalletArrived:
				if err := event.Wallet.Open(""); err != nil {
					log.Warn("New wallet appeared, failed to open", "url", event.Wallet.URL(), "err", err)
				}
			case accounts.WalletOpened:
				status, _ := event.Wallet.Status()
				log.Info("New wallet appeared", "url", event.Wallet.URL(), "status", status)

				if event.Wallet.URL().Scheme == "ledger" {
					event.Wallet.SelfDerive(accounts.DefaultLedgerBaseDerivationPath, stateReader)
				} else {
					event.Wallet.SelfDerive(accounts.DefaultBaseDerivationPath, stateReader)
				}

			case accounts.WalletDropped:
				log.Info("Old wallet dropped", "url", event.Wallet.URL())
				event.Wallet.Close()
			}
		}
	}()

	if ctx.GlobalBool(utils.MiningEnabledFlag.Name) || ctx.GlobalBool(utils.DeveloperFlag.Name) {

		if ctx.GlobalBool(utils.LightModeFlag.Name) || ctx.GlobalString(utils.SyncModeFlag.Name) == "light" {
			utils.Fatalf("Light clients do not support mining")
		}
		var ddmchain *ddm.DDMchain
		if err := stack.Service(&ddmchain); err != nil {
			utils.Fatalf("DDMchain service not running: %v", err)
		}

		if threads := ctx.GlobalInt(utils.MinerThreadsFlag.Name); threads > 0 {
			type threaded interface {
				SetThreads(threads int)
			}
			if th, ok := ddmchain.Engine().(threaded); ok {
				th.SetThreads(threads)
			}
		}

		ddmchain.TxPool().SetGasPrice(utils.GlobalBig(ctx, utils.GasPriceFlag.Name))
		if err := ddmchain.StartMining(true); err != nil {
			utils.Fatalf("Failed to start mining: %v", err)
		}
	}
}
