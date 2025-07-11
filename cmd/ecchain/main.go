package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/console/prompt"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"
	"os"
	"sort"
)

const (
	clientIdentifier = "ecchain" // Client identifier to advertise over the network
)

var app = flags.NewApp("the ec-chain command line interface")

func init() {
	// Initialize the CLI app and start Geth
	app.Action = ecchain
	app.Copyright = "Copyright 2013-2023 The go-ethereum Authors"
	app.Commands = []*cli.Command{
		readtxcmd,
		gethCmd,
		analyzeCmd,
		dbGroupCmd,
	}
	sort.Sort(cli.CommandsByName(app.Commands))

	app.Flags = []cli.Flag{
		zipDirFlag,
		cleanFlag,
		ecKFlag,
		measureStorageFlag,
		measureTimeFlag,
		recencyFlag,
		frequencyFlag,
		debugFlag,
	}

	app.Before = func(ctx *cli.Context) error {
		flags.MigrateGlobalFlags(ctx)
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		prompt.Stdin.Close() // Resets terminal mode.
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
