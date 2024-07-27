package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/urfave/cli/v2"
	"math"
	"math/big"
	"time"
)

type EcGroup struct {
	k                    int
	size                 int
	recency              int
	frequency            float64
	nodes                []*EcNode
	blockToExpireNode    *DbNode
	createdHeightNode    *DbNode
	accessTimeNode       *DbNode
	accountsToExpireNode *DbNode
}

func NewEcGroup(k, recency int, frequency float64) (*EcGroup, error) {
	g := &EcGroup{
		k:         k,
		size:      1 << k,
		recency:   recency,
		frequency: frequency,
	}
	var err error
	g.nodes, err = NewEcNodes(k, recency, frequency)
	if err != nil {
		return nil, err
	}
	g.blockToExpireNode, err = NewDbNode(g.size)
	if err != nil {
		return nil, err
	}
	g.createdHeightNode, err = NewDbNode(g.size + 1)
	if err != nil {
		return nil, err
	}
	g.accessTimeNode, err = NewDbNode(g.size + 2)
	if err != nil {
		return nil, err
	}
	g.accountsToExpireNode, err = NewDbNode(g.size + 3)
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
	accountsToExpire = make(map[int]map[string]bool)
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)
		if g.IsHot(addr) { // the address exists and it's hot
			for _, ecNode := range g.nodes {
				ecNode.AddBalanceHot(addr, tx.value)
			}
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
			} else { // the address doesn't exist, create it
				for _, ecNode := range g.nodes {
					func(ecNode *EcNode) {
						ecNode.AddBalanceHot(addr, tx.value)
					}(ecNode)
				}
			}
		}
		if _, ok := accessTime[addrString]; !ok {
			g.accessTimeNode.SetBalance(addr, big.NewInt(1))
		} else {
			g.accessTimeNode.AddBalance(addr, big.NewInt(1))
			delete(accountsToExpire[int(g.blockToExpireNode.GetBalance(addr).Int64())], addrString)
		}
		newBlockToExpire := func(a, b int) int {
			if a > b {
				return a
			}
			return b
		}(tx.blockNumber+g.recency, int(g.createdHeightNode.GetBalance(addr).Int64())+int(math.Ceil(float64(g.accessTimeNode.GetBalance(addr).Int64())/g.frequency)))
		if newBlockToExpire < 4000000 {
			g.blockToExpireNode.SetBalance(addr, big.NewInt(int64(newBlockToExpire)))
			if _, ok := accountsToExpire[newBlockToExpire]; !ok {
				accountsToExpire[newBlockToExpire] = make(map[string]bool)
			}
			accountsToExpire[newBlockToExpire][addrString] = true
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
	if err := g.blockToExpireNode.Clean(); err != nil {
		return err
	}
	if err := g.createdHeightNode.Clean(); err != nil {
		return err
	}
	if err := g.accessTimeNode.Clean(); err != nil {
		return err
	}
	if err := g.accountsToExpireNode.Clean(); err != nil {
		return err
	}
	return nil
}

func ecchain(ctx *cli.Context) error {
	measureTime := ctx.IsSet(measureTimeFlag.Name)
	measureStorage := ctx.IsSet(measureStorageFlag.Name)
	recency := ctx.Int(recencyFlag.Name)
	frequency := ctx.Float64(frequencyFlag.Name)
	debugging := ctx.IsSet(debugFlag.Name)

	g, err := NewEcGroup(ctx.Int(ecKFlag.Name), recency, frequency)
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
		for addrString := range accountsToExpire[height] {
			addr := common.HexToAddress(addrString)
			//fmt.Println("Colding", account)
			// remove addr from hot
			balance := g.nodes[0].hot.stateDb.GetBalance(addr)
			for _, ecNode := range g.nodes {
				ecNode.hot.Delete(addr)
			}

			// add addr to cold
			g.GetNodeForAddress(addr).SetBalanceCold(addr, balance)
		}
		delete(accountsToExpire, height)

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
