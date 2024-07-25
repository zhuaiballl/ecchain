package main

import (
	"github.com/urfave/cli/v2"
	"math"
	"strconv"
)

var analyzeCmd = &cli.Command{
	Name:   "analyze",
	Usage:  "Analyze the ETH transactions from zip",
	Action: analyze,
	Flags: []cli.Flag{
		zipDirFlag,
		recencyFlag,
		frequencyFlag,
	},
	Description: `
    ecchain analyze /path/to/my.zip`,
}

var (
	coldReadCount    int
	hotTrieSize      int
	coldTrieSize     int
	hotAccounts      map[string]bool
	coldAccounts     map[string]bool
	blockToExpire    map[string]int
	createdHeight    map[string]int
	accessTime       map[string]int
	accountsToExpire map[int]map[string]bool
)

type block struct {
	height    int
	addresses []string
}

func (b *block) appendAddr(addr ...string) {
	b.addresses = append(b.addresses, addr...)
}

// BEGIN cold read vs. threshold
func encoldAccounts(height int) error {
	for addr := range accountsToExpire[height] {
		coldAccounts[addr] = true
		delete(hotAccounts, addr)
	}
	delete(accountsToExpire, height)
	if len(coldAccounts) > coldTrieSize {
		coldTrieSize = len(coldAccounts)
	}
	return nil
}

func updateWithTx(tx txFromZip, recency int, frequency float64) error {
	// update hot and cold tries
	for _, addr := range []string{tx.sender, tx.to} {
		if _, ok := hotAccounts[addr]; !ok {
			coldReadCount++
			if _, okk := coldAccounts[addr]; okk {
				delete(coldAccounts, addr)
			} else {
				createdHeight[addr] = tx.blockNumber
			}
		}
		hotAccounts[addr] = true
		if _, ok := accessTime[addr]; !ok {
			accessTime[addr] = 1
		} else {
			accessTime[addr]++
			delete(accountsToExpire[blockToExpire[addr]], addr)
		}
		newBlockToExpire := func(a, b int) int {
			if a > b {
				return a
			}
			return b
		}(tx.blockNumber+recency, createdHeight[addr]+int(math.Ceil(float64(accessTime[addr])/frequency)))
		blockToExpire[addr] = newBlockToExpire
		if _, ok := accountsToExpire[newBlockToExpire]; !ok {
			accountsToExpire[newBlockToExpire] = make(map[string]bool)
		}
		accountsToExpire[newBlockToExpire][addr] = true
	}
	if len(hotAccounts) > hotTrieSize {
		hotTrieSize = len(hotAccounts)
	}

	return nil
}

// END cold read vs. threshold

func analyze(ctx *cli.Context) error {
	hotAccounts = make(map[string]bool)
	coldAccounts = make(map[string]bool)
	blockToExpire = make(map[string]int)
	createdHeight = make(map[string]int)
	accessTime = make(map[string]int)
	accountsToExpire = make(map[int]map[string]bool)

	coldReadCount = 0
	hotTrieSize = 0
	coldTrieSize = 0
	lstBlock := -1

	recency := ctx.Int(recencyFlag.Name)
	frequency := ctx.Float64(frequencyFlag.Name)
	gasSum := 0
	txCount := 0
	err := processTxFromZip(func(height int) error {
		if err := encoldAccounts(height); err != nil {
			return err
		}

		if height/10000 != lstBlock/10000 {
			println(height, coldReadCount, txCount, strconv.FormatFloat(float64(coldReadCount)/float64(txCount), 'f', -1, 64))
			coldReadCount = 0
			txCount = 0
		}
		lstBlock = height

		return nil
	}, func(tx txFromZip) error {
		gasSum += tx.gasUsed
		txCount++
		return updateWithTx(tx, recency, frequency)
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}
	return nil
}
