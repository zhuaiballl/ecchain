package main

import (
	"github.com/ethereum/go-ethereum/common"
	"math/big"
)

type EcNode struct {
	k         int
	threshold int
	ind       int
	hot       *DbNode
	cold      *DbNode
}

func NewEcNode(k, threshold, ind int) (*EcNode, error) {
	hot, err := NewDbNode(1)
	if err != nil {
		return nil, err
	}
	cold, err := NewDbNode(2)
	if err != nil {
		return nil, err
	}
	return &EcNode{
		k, threshold, ind,
		hot, cold,
	}, nil
}

func NewEcNodes(k, threshold int) (nodes []*EcNode, err error) {
	size := 1 << k
	for i := 0; i < size; i++ {
		newNode, nerr := NewEcNode(k, threshold, i)
		if nerr != nil {
			err = nerr
			return
		}
		nodes = append(nodes, newNode)
	}
	return
}

func (ecNode *EcNode) AddBalanceHot(address common.Address, value *big.Int) {
	ecNode.hot.AddBalance(address, value)
}

func (ecNode *EcNode) AddBalanceCold(address common.Address, value *big.Int) {
	ecNode.cold.AddBalance(address, value)
}

func (ecNode *EcNode) SetNonce(address common.Address, nonce uint64) {
	ecNode.hot.SetNonce(address, nonce)
}

func (ecNode *EcNode) GetNonce(address common.Address) uint64 {
	return ecNode.hot.GetNonce(address)
}

func (ecNode *EcNode) Commit() error {
	for _, n := range []*DbNode{ecNode.hot, ecNode.cold} {
		root, err := n.stateDb.Commit(true)
		if err != nil {
			return err
		}
		err = n.trieDb.Commit(root, false)
		if err != nil {
			return err
		}
	}
	return nil
}

func (ecNode *EcNode) Clean() error {
	if err := ecNode.hot.Clean(); err != nil {
		return err
	}
	if err := ecNode.cold.Clean(); err != nil {
		return err
	}
	return nil
}

func (ecNode *EcNode) StorageCost() int {
	return ecNode.hot.StorageCost() + ecNode.cold.StorageCost()
}
