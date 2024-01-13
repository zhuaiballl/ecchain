package main

import "github.com/urfave/cli/v2"

var analysisCmd = &cli.Command{
	Name:   "analyze",
	Usage:  "Analyze the ETH transactions from zip",
	Action: analyze,
	Flags: []cli.Flag{
		zipDirFlag,
	},
	Description: `
    ecchain analyze /path/to/my.zip`,
}

func analyze(ctx *cli.Context) error {
	// TODO do some analysis if you need
	err := processTxFromZip(func(i int) error {
		return nil
	}, func(zip txFromZip) error {
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}
	return nil
}
