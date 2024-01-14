package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"math/big"
	"os/exec"
	"strings"
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

func (g *EcGroup) executeTx(tx txFromZip) {
	sender := common.HexToAddress(tx.sender)
	to := common.HexToAddress(tx.to)
	g.GetNodeForAddress(sender).AddBalance(sender, big.NewInt(1))
	g.GetNodeForAddress(to).AddBalance(to, tx.value)
}

func (g *EcGroup) Commit(height int) error {
	fmt.Print(height)
	for _, n := range g.nodes {
		root, err := n.stateDb.Commit(true)
		if err != nil {
			return err
		}
		err = n.trieDb.Commit(root, false)
		if err != nil {
			return err
		}

		// measure storage cost of the node
		cmdOutput, _ := exec.Command("du", "-s", n.datadir).Output()
		storageCost := string(cmdOutput)
		storageCost = strings.Fields(storageCost)[0]
		fmt.Print(" ", storageCost)
	}
	fmt.Println()
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
)

func ecchain(ctx *cli.Context) error {
	g, err := NewEcGroup(ctx.Int(EcKFlag.Name))
	if err != nil {
		return err
	}
	err = processTxFromZip(func(height int) error {
		return g.Commit(height)
	}, func(tx txFromZip) error {
		g.executeTx(tx)
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}

	return err
}
