package main

import (
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/params/ethernova"
	"github.com/urfave/cli/v2"
)

var sanitycheckCommand = &cli.Command{
	Name:  "sanitycheck",
	Usage: "Verify Ethernova genesis/datadir without starting networking",
	Flags: []cli.Flag{
		utils.DataDirFlag,
		utils.DBEngineFlag,
		utils.AncientFlag,
		utils.CachePreimagesFlag,
		utils.OverrideCancun,
		utils.OverrideVerkle,
		utils.NetworkIdFlag,
	},
	Action: sanitycheck,
}

var printGenesisCommand = &cli.Command{
	Name:   "print-genesis",
	Usage:  "Print expected genesis and embedded genesis SHA256",
	Action: printGenesis,
}

func sanitycheck(ctx *cli.Context) error {
	info, err := loadEthernovaGenesis(ctx)
	if err != nil {
		fmt.Printf("FAIL: %v\n", err)
		return cli.Exit(err.Error(), 1)
	}
	printEthernovaStartup(info)
	fmt.Printf("genesis_sha256=%s\n", info.GenesisSHA256)

	if _, err := ensureEthernovaGenesis(ctx, info); err != nil {
		fmt.Printf("FAIL: %v\n", err)
		return cli.Exit(err.Error(), 1)
	}
	fmt.Println("PASS")
	return nil
}

func printGenesis(ctx *cli.Context) error {
	genesis := ethernova.MustGenesis()
	chainID := genesis.Config.GetChainID()
	var chainIDValue uint64
	if chainID != nil {
		chainIDValue = chainID.Uint64()
	}
	networkID := chainIDValue
	if n := genesis.GetNetworkID(); n != nil {
		networkID = *n
	}

	fmt.Printf("expected_genesis_hash=%s\n", ethernova.ExpectedGenesisHashHex)
	fmt.Printf("chain_id=%d\n", chainIDValue)
	fmt.Printf("network_id=%d\n", networkID)
	fmt.Printf("embedded_genesis_sha256=%s\n", ethernova.EmbeddedGenesisSHA256Hex())
	return nil
}
