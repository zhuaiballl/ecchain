package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"math/big"
	"os/exec"
	"strings"
	"time"
)

type EcGroup struct {
	k     int // size = 2^k
	size  int
	nodes []*DbNode
}

func NewEcGroup(k int) (*EcGroup, error) {
	g := &EcGroup{
		k:    k,
		size: 1 << k,
	}
	var err error
	g.nodes, err = NewDbNodes(g.size)
	return g, err
}

func (g *EcGroup) GetNodeForAddress(address common.Address) *DbNode {
	ind := int(address.Bytes()[0])
	ind >>= 8 - g.k
	return g.nodes[ind]
}

type payload struct {
	object common.Address
	value  big.Int
}

func (g *EcGroup) Size() int {
	return g.size
}

func (g *EcGroup) executeTx(tx txFromZip) time.Duration {
	timeBegin := time.Now()
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)

		g.GetNodeForAddress(addr).AddBalance(addr, tx.value)
	}
	timeSpent := time.Since(timeBegin)
	return timeSpent
}

func (g *EcGroup) Commit(height int, measureStorage, measureTime bool) error {
	for _, n := range g.nodes {
		root, err := n.stateDb.Commit(true)
		if err != nil {
			return err
		}
		err = n.trieDb.Commit(root, false)
		if err != nil {
			return err
		}

		if measureStorage {
			// measure storage cost of the node
			cmdOutput, _ := exec.Command("du", "-s", n.datadir).Output()
			storageCost := string(cmdOutput)
			storageCost = strings.Fields(storageCost)[0]
			fmt.Print(" ", storageCost)
		}
	}
	return nil
}

var (
	EcKFlag *cli.IntFlag = &cli.IntFlag{
		Name:  "k",
		Usage: "EC group size is 2^k",
		Value: 2,
	}
	ThresholdFlag *cli.IntFlag = &cli.IntFlag{
		Name:  "threshold",
		Usage: "Threshold between cold/hot tries",
		Value: 10000,
	}
	MeasureTimeFlag *cli.BoolFlag = &cli.BoolFlag{
		Name:  "time",
		Usage: "Output time information",
	}
	MeasureStorageFlag *cli.BoolFlag = &cli.BoolFlag{
		Name:  "storage",
		Usage: "Output storage usage information",
	}
)

func ecchain(ctx *cli.Context) error {
	measureTime := ctx.IsSet(MeasureTimeFlag.Name)
	measureStorage := ctx.IsSet(MeasureStorageFlag.Name)

	g, err := NewEcGroup(ctx.Int(EcKFlag.Name))
	if err != nil {
		return err
	}
	txCount := 0
	timeSum := time.Duration(0)
	err = processTxFromZip(func(height int) error {
		fmt.Print(height)
		if measureTime {
			fmt.Println(" ", float64(timeSum.Nanoseconds())/float64(txCount))
		}
		return g.Commit(height, measureStorage, measureTime)
	}, func(tx txFromZip) error {
		timeSum += g.executeTx(tx)
		txCount++
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}

	return err
}
