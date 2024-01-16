package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"math/big"
	"time"
)

type EcGroup struct {
	k         int
	size      int
	threshold int
	nodes     []*EcNode
}

func NewEcGroup(k, threshold int) (*EcGroup, error) {
	g := &EcGroup{
		k:         k,
		size:      (1 << k) + 1, // one extra node to maintain assist data
		threshold: threshold,
	}
	var err error
	g.nodes, err = NewEcNodes(g.size)
	return g, err
}

func (g *EcGroup) IsHot(address common.Address) bool {
	return g.nodes[0].hot.Exist(address)
}

func (g *EcGroup) GetNodeForAddress(address common.Address) *EcNode {
	ind := int(address.Bytes()[0])
	ind >>= 8 - g.k
	return g.nodes[ind]
}

var txCounts [][2]int

func (g *EcGroup) executeTx(tx txFromZip) time.Duration {
	timeBegin := time.Now()
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)
		if g.IsHot(addr) { // the address exists and it's hot
			for i, ecNode := range g.nodes {
				if i == 0 {
					continue
				}
				ecNode.AddBalanceHot(addr, tx.value)
			}
			g.nodes[0].SetNonce(addr, uint64(tx.blockNumber))
		} else {
			if g.GetNodeForAddress(addr).cold.Exist(addr) { // the address exists and it's cold, move it to hot
				// remove addr from cold
				cold := g.GetNodeForAddress(addr).cold
				balance := big.NewInt(0)
				balance.Add(cold.stateDb.GetBalance(addr), tx.value)
				cold.Delete(addr)

				// add addr to hot
				for i, ecNode := range g.nodes {
					if i == 0 {
						continue
					}
					func(ecNode *EcNode) {
						ecNode.AddBalanceHot(addr, balance)
					}(ecNode)
				}
				g.nodes[0].SetNonce(addr, uint64(tx.blockNumber))
			} else { // the address doesn't exist, create it
				for i, ecNode := range g.nodes {
					if i == 0 {
						continue
					}
					func(ecNode *EcNode) {
						ecNode.AddBalanceHot(addr, tx.value)
					}(ecNode)
				}
				g.nodes[0].SetNonce(addr, uint64(tx.blockNumber))

			}
		}

	}
	timeSpent := time.Since(timeBegin)
	return timeSpent
}

func (g *EcGroup) Commit(height int, measureStorage, measureTime bool) error {
	for _, n := range g.nodes {
		if err := n.Commit(); err != nil {
			return err
		}

		if measureStorage {
			fmt.Print(" ", n.StorageCost())
		}
	}
	return nil
}

func (g *EcGroup) Clean() error {
	for _, n := range g.nodes {
		err := n.Clean()
		if err != nil {
			return err
		}
	}
	return nil
}

func ecchain(ctx *cli.Context) error {
	measureTime := ctx.IsSet(MeasureTimeFlag.Name)
	measureStorage := ctx.IsSet(MeasureStorageFlag.Name)
	threshold := ctx.Int(ThresholdFlag.Name)

	g, err := NewEcGroup(ctx.Int(EcKFlag.Name), threshold)
	if err != nil {
		return err
	}
	timeSum := time.Duration(0)
	txCount := 0
	var accountsInCurrentBlock []string
	err = processTxFromZip(func(height int) error {
		// measureTime (average tx execution latency)
		if measureTime {
			fmt.Print(" ")
			if txCount != 0 {
				fmt.Print(float64(timeSum.Nanoseconds()) / float64(txCount))
			} else {
				fmt.Print("-1")
			}
		}
		txCounts = append(txCounts, [2]int{height, txCount})
		timeSum = 0
		txCount = 0

		// colding
		coldHeight := height - threshold
		if coldHeight > 0 && txCounts[0][0] == coldHeight {
			coldAddress := common.BigToAddress(big.NewInt(int64(coldHeight))) // coldAddress is the address that stores accounts in the block[coldHeight]
			for i := 0; i < txCounts[0][1]; i++ {
				account := common.HexToAddress(g.nodes[0].hot.stateDb.GetState(coldAddress, common.BigToHash(big.NewInt(int64(i)))).Hex())
				if g.nodes[0].GetNonce(account) == uint64(coldHeight) {
					// remove addr from hot
					balance := g.nodes[0].hot.stateDb.GetBalance(account)
					for i, ecNode := range g.nodes {
						if i == 0 {
							continue
						}
						ecNode.hot.Delete(account)
					}

					// add addr to cold
					g.GetNodeForAddress(account).AddBalanceCold(account, balance)
				}
			}
			txCounts = txCounts[1:]
			g.nodes[0].hot.Delete(coldAddress)
		}

		err = g.Commit(height, measureStorage, measureTime)
		if err != nil {
			return err
		}
		accountsInCurrentBlock = []string{}
		return nil
	}, func(tx txFromZip) error {
		accountsInCurrentBlock = append(accountsInCurrentBlock, tx.sender, tx.to)
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
