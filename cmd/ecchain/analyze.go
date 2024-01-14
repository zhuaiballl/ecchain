package main

import (
	"github.com/urfave/cli/v2"
)

var analyzeCmd = &cli.Command{
	Name:   "analyze",
	Usage:  "Analyze the ETH transactions from zip",
	Action: analyze,
	Flags: []cli.Flag{
		zipDirFlag,
		ThresholdFlag,
	},
	Description: `
    ecchain analyze /path/to/my.zip`,
}

var (
	coldReadCount int
	hotTrieSize   int
	coldTrieSize  int
	blockQueue    []block
)

type block struct {
	height    int
	addresses []string
}

func (b *block) appendAddr(addr ...string) {
	b.addresses = append(b.addresses, addr...)
}

// BEGIN cold read vs. threshold
func encoldAccounts(height, threshold int, hot map[string]int, cold map[string]interface{}) error {
	if len(blockQueue) == 0 {
		return nil
	}
	if blockQueue[0].height >= height-threshold {
		return nil
	}
	for _, addr := range blockQueue[0].addresses {
		if hot[addr] < height-threshold {
			cold[addr] = 1
			delete(hot, addr)
		}
	}
	if len(cold) > coldTrieSize {
		coldTrieSize = len(cold)
	}
	blockQueue = blockQueue[1:]
	return nil
}

func updateWithTx(tx txFromZip, hot map[string]int, cold map[string]interface{}) error {
	// update hot and cold tries
	for _, addr := range []string{tx.sender, tx.to} {
		if _, ok := hot[addr]; !ok {
			coldReadCount++
			if _, okk := cold[addr]; okk {
				delete(cold, addr)
			}
		}
		hot[addr] = tx.blockNumber
	}
	if len(hot) > hotTrieSize {
		hotTrieSize = len(hot)
	}

	// update blockQueue
	if len(blockQueue) > 0 && tx.blockNumber == blockQueue[len(blockQueue)-1].height {
		blockQueue[len(blockQueue)-1].appendAddr(tx.sender, tx.to)
	} else {
		blockQueue = append(blockQueue, block{tx.blockNumber, []string{tx.sender, tx.to}})
	}
	return nil
}

// END cold read vs. threshold

func analyze(ctx *cli.Context) error {
	hot := make(map[string]int)
	cold := make(map[string]interface{})
	coldReadCount = 0
	hotTrieSize = 0
	coldTrieSize = 0

	threshold := ctx.Int(ThresholdFlag.Name)
	err := processTxFromZip(func(height int) error {
		return encoldAccounts(height, threshold, hot, cold)
	}, func(tx txFromZip) error {
		return updateWithTx(tx, hot, cold)
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}
	println(threshold, coldReadCount, coldTrieSize, hotTrieSize)
	return nil
}
