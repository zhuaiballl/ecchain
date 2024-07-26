package main

import "github.com/urfave/cli/v2"

var (
	cleanFlag = &cli.BoolFlag{
		Name:  "clean",
		Usage: "Remove the temp folder after run",
	}
	zipDirFlag = &cli.StringFlag{
		Name:  "zipdir",
		Usage: "Directory of zip files",
	}
	debugFlag = &cli.BoolFlag{
		Name:  "debug",
		Usage: "Tell EC-Chain I'm debugging",
	}
	ecKFlag = &cli.IntFlag{
		Name:  "k",
		Usage: "EC group size is 2^k",
		Value: 2,
	}
	recencyFlag = &cli.IntFlag{
		Name:  "recency",
		Usage: "Recency recency between cold/hot tries",
		Value: 10000,
	}
	frequencyFlag = &cli.Float64Flag{
		Name:  "frequency",
		Usage: "Frequency recency between cold/hot tries",
		Value: 1,
	}
	indFlag = &cli.IntFlag{
		Name:  "ind",
		Usage: "Designate the index of the ecnode in the ecgroup",
		Value: 0,
	}
	measureTimeFlag = &cli.BoolFlag{
		Name:  "time",
		Usage: "Output time information",
	}
	measureStorageFlag = &cli.BoolFlag{
		Name:  "storage",
		Usage: "Output storage usage information",
	}
)
