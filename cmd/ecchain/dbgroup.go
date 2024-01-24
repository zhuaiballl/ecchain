package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"math/rand"
	"time"
)

type DbGroup struct {
	k     int // size = 2^k
	size  int
	nodes []*DbNode
}

func NewDbGroup(k int) (*DbGroup, error) {
	g := &DbGroup{
		k:    k,
		size: 1 << k,
	}
	var err error
	g.nodes, err = NewDbNodes(g.size)
	return g, err
}

func (g *DbGroup) GetNodeForAddress(address common.Address) *DbNode {
	ind := int(address.Bytes()[0])
	ind >>= 8 - g.k
	return g.nodes[ind]
}

func (g *DbGroup) executeTx(tx txFromZip) time.Duration {
	timeBegin := time.Now()
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)
		g.GetNodeForAddress(addr).AddBalance(addr, tx.value)
	}
	timeSpent := time.Since(timeBegin)
	return timeSpent
}

func (g *DbGroup) Commit(height int, measureStorage, measureTime bool) error {
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
			fmt.Print(" ", n.StorageCost())
		}
	}
	return nil
}

func (g *DbGroup) Clean() error {
	for _, n := range g.nodes {
		err := n.Clean()
		if err != nil {
			return err
		}
	}
	return nil
}

var (
	EcKFlag = &cli.IntFlag{
		Name:  "k",
		Usage: "EC group size is 2^k",
		Value: 2,
	}
	ThresholdFlag = &cli.IntFlag{
		Name:  "threshold",
		Usage: "Threshold between cold/hot tries",
		Value: 100,
	}
	MeasureTimeFlag = &cli.BoolFlag{
		Name:  "time",
		Usage: "Output time information",
	}
	MeasureStorageFlag = &cli.BoolFlag{
		Name:  "storage",
		Usage: "Output storage usage information",
	}
	IndFlag = &cli.IntFlag{
		Name:  "ind",
		Usage: "Designate the index of the ecnode in the ecgroup",
		Value: 0,
	}
)

var dbGroupCmd = &cli.Command{
	Name:   "dbgroup",
	Usage:  "Execute transactions with dbgroup",
	Action: dbGroup,
	Flags: []cli.Flag{
		cleanFlag,
		zipDirFlag,
		ecKFlag,
		measureTimeFlag,
		measureStorageFlag,
		IndFlag,
	},
	Description: "ecchain dbgroup /path/to/my.zip",
}

func dbGroup(ctx *cli.Context) error {
	if ctx.IsSet(IndFlag.Name) {
		return oneNodeFromDBGroup(ctx)
	}

	measureTime := ctx.IsSet(measureTimeFlag.Name)
	measureStorage := ctx.IsSet(measureStorageFlag.Name)

	g, err := NewDbGroup(ctx.Int(ecKFlag.Name))
	if err != nil {
		return err
	}
	txCount := 0
	timeSum := time.Duration(0)
	err = processTxFromZip(func(height int) error {
		if measureTime {
			fmt.Print(" ")
			if txCount != 0 {
				fmt.Print(float64(timeSum.Nanoseconds()) / float64(txCount))
			} else {
				fmt.Print("-1")
			}
		}
		timeSum = 0
		txCount = 0
		return g.Commit(height, measureStorage, measureTime)
	}, func(tx txFromZip) error {
		timeSum += g.executeTx(tx)
		txCount++
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}
	if ctx.IsSet(cleanFlag.Name) {
		err = g.Clean()
		if err != nil {
			return err
		}
	}

	return nil
}

func oneNodeFromDBGroup(ctx *cli.Context) error {
	measureTime := ctx.IsSet(MeasureTimeFlag.Name)
	measureStorage := ctx.IsSet(MeasureStorageFlag.Name)
	eck := ctx.Int(EcKFlag.Name)

	n, err := NewDbNode(ctx.Int(IndFlag.Name))
	if err != nil {
		return err
	}
	txCount := 0
	timeSum := time.Duration(0)
	err = processTxFromZip(func(height int) error {
		if measureTime {
			fmt.Print(" ")
			if txCount != 0 {
				fmt.Print(float64(timeSum.Nanoseconds()) / float64(txCount))
			} else {
				fmt.Print("-1")
			}
		}
		timeSum = 0
		txCount = 0
		err := n.Commit()
		if err != nil {
			return err
		}
		if measureStorage {
			fmt.Print(n.StorageCost())
		}
		return nil
	}, func(tx txFromZip) error {
		cntAddr := 0
		beginTime := time.Now()
		for _, account := range []string{tx.sender, tx.to} {
			addr := common.HexToAddress(account)
			if GetIndForAddress(eck, addr) == n.ind {
				cntAddr++
				n.AddBalance(addr, tx.value)
			}
		}
		if cntAddr == 2 {
			txCount++
		} else if cntAddr == 1 {
			txCount += rand.Int() % 2
		}
		timeSum += time.Since(beginTime)
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}
	if ctx.IsSet(cleanFlag.Name) {
		err = n.Clean()
		if err != nil {
			return err
		}
	}

	return nil
}
