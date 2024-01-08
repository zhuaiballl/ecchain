package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"math/big"
)

type ecNode struct {
	addr common.Address
	db   *state.StateDB
}

type ecGroup struct {
	n      int
	dbList []*state.StateDB
}

func NewEcGroup(n int) ecGroup {
	return ecGroup{
		n:      0,
		dbList: nil,
	}
}

type payload struct {
	object common.Address
	value  big.Int
}

func (g *ecGroup) Size() int {
	return g.n
}

func (g *ecGroup) operate(opt string, p payload) {
	// get the targetDB by the object address
	obj := p.object
	ind := int(obj[0])
	var targetDB *state.StateDB = g.dbList[ind]

	switch opt {
	case "AddBalance":
		targetDB.AddBalance(obj, &p.value)
	case "SubBalance":
		targetDB.SubBalance(obj, &p.value)
	}
}
