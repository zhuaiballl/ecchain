package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/mpvl/unique"
	"github.com/urfave/cli/v2"
	"math/big"
	"time"
)

type EcGroup struct {
	k         int
	size      int
	threshold int
	nodes     []*EcNode
	metaNode  *EcNode
}

func NewEcGroup(k, threshold int) (*EcGroup, error) {
	g := &EcGroup{
		k:         k,
		size:      1 << k, // one extra node to maintain assist data
		threshold: threshold,
	}
	var err error
	g.nodes, err = NewEcNodes(k, threshold)
	if err != nil {
		return nil, err
	}
	g.metaNode, err = NewEcNode(k, threshold, g.size)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (g *EcGroup) IsHot(address common.Address) bool {
	return g.nodes[0].hot.Exist(address)
}

func GetIndForAddress(k int, address common.Address) int {
	ind := int(address.Bytes()[0])
	ind >>= 8 - k
	return ind
}

func (g *EcGroup) GetNodeForAddress(address common.Address) *EcNode {
	return g.nodes[GetIndForAddress(g.k, address)]
}

var accountCounts [][2]int

func (g *EcGroup) executeTx(tx txFromZip) time.Duration {
	timeBegin := time.Now()
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)
		if g.IsHot(addr) { // the address exists and it's hot
			for _, ecNode := range g.nodes {
				ecNode.AddBalanceHot(addr, tx.value)
			}
			g.metaNode.SetBalance(addr, big.NewInt(int64(tx.blockNumber)))
		} else {
			if g.GetNodeForAddress(addr).cold.Exist(addr) { // the address exists and is cold, move it to hot
				fmt.Println("cold")
				// remove addr from cold
				cold := g.GetNodeForAddress(addr).cold
				balance := big.NewInt(0)
				balance.Add(cold.stateDb.GetBalance(addr), tx.value)
				cold.Delete(addr)

				// add addr to hot
				for _, ecNode := range g.nodes {
					func(ecNode *EcNode) {
						ecNode.AddBalanceHot(addr, balance)
					}(ecNode)
				}
				g.metaNode.SetBalance(addr, big.NewInt(int64(tx.blockNumber)))
			} else { // the address doesn't exist, create it
				for _, ecNode := range g.nodes {
					func(ecNode *EcNode) {
						ecNode.AddBalanceHot(addr, tx.value)
					}(ecNode)
				}
				g.metaNode.SetBalance(addr, big.NewInt(int64(tx.blockNumber)))

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
	}
	//err := g.metaNode.Commit()
	//if err != nil {
	//	return err
	//}
	return nil
}

func (g *EcGroup) Clean() error {
	for _, n := range g.nodes {
		err := n.Clean()
		if err != nil {
			return err
		}
	}
	if err := g.metaNode.Clean(); err != nil {
		return err
	}
	return nil
}

func ecchain(ctx *cli.Context) error {
	measureTime := ctx.IsSet(measureTimeFlag.Name)
	measureStorage := ctx.IsSet(measureStorageFlag.Name)
	threshold := ctx.Int(recencyFlag.Name)
	debugging := ctx.IsSet(debugFlag.Name)

	g, err := NewEcGroup(ctx.Int(ecKFlag.Name), threshold)
	if err != nil {
		return err
	}
	timeSum := time.Duration(0)
	txCount := 0
	var accountsInCurrentBlock []string
	lstBlock := -1
	err = processTxFromZip(func(height int) error {
		if debugging || height/10000 != lstBlock/10000 {
			fmt.Print(height, " ")
			defer fmt.Println("")
		}

		// measureTime (average tx execution latency)
		if measureTime && (debugging || height/10000 != lstBlock/10000) {
			fmt.Print(" ")
			if txCount != 0 {
				fmt.Print(float64(timeSum.Nanoseconds()) / float64(txCount))
			} else {
				fmt.Print("-1")
			}
		}
		timeSum = 0
		txCount = 0

		// colding
		coldHeight := height - threshold
		for len(accountCounts) > 0 && accountCounts[0][0] <= coldHeight {
			//fmt.Println("colding", accountCounts[0][0])
			coldAddress := common.BigToAddress(big.NewInt(int64(accountCounts[0][0]))) // coldAddress is the address that stores accounts in the block[coldHeight]
			for i := 0; i < accountCounts[0][1]; i++ {
				account := common.HexToAddress(g.metaNode.hot.stateDb.GetState(coldAddress, common.BigToHash(big.NewInt(int64(i)))).Hex())
				//fmt.Println("Is", g.metaNode.hot.stateDb.GetState(coldAddress, common.BigToHash(big.NewInt(int64(i)))), "colding?")
				//fmt.Println("Is", coldAddress, common.BigToHash(big.NewInt(int64(i))), "colding?")
				if g.metaNode.hot.GetBalance(account).Int64() <= int64(coldHeight) {
					//fmt.Println("Colding", account)
					// remove addr from hot
					balance := g.nodes[0].hot.stateDb.GetBalance(account)
					for _, ecNode := range g.nodes {
						ecNode.hot.Delete(account)
					}

					// add addr to cold
					g.GetNodeForAddress(account).SetBalanceCold(account, balance)
				}
			}
			accountCounts = accountCounts[1:]
			g.nodes[0].hot.Delete(coldAddress)
		}

		// write accountsInCurrentBlock into metaNode
		unique.Strings(&accountsInCurrentBlock)
		heightAddress := common.BigToAddress(big.NewInt(int64(height)))
		for i, account := range accountsInCurrentBlock {
			g.metaNode.hot.stateDb.SetState(heightAddress, common.BigToHash(big.NewInt(int64(i))), common.HexToHash(account))

		}
		accountCounts = append(accountCounts, [2]int{height, len(accountsInCurrentBlock)})

		accountsInCurrentBlock = []string{}

		if err = g.Commit(height, measureStorage, measureTime); err != nil {
			return err
		}
		if measureStorage && height/10000 != lstBlock/10000 {
			for _, n := range g.nodes {
				fmt.Print(" ", n.StorageCost())
			}
		}
		lstBlock = height
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
