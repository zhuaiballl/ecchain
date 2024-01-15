package main

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"math/big"
	"os"
)

type DbNode struct {
	ind     int
	datadir string
	stateDb *state.StateDB
	trieDb  *trie.Database
}

func NewDbNode(ind int) (n *DbNode, err error) {
	datadir, err := os.MkdirTemp("", "ecchain")
	if err != nil {
		return
	}
	//fmt.Println(datadir)
	nodeConfig := &node.Config{
		Name:    "geth-ec",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
		UseLightweightKDF: true,
	}
	tempNode, err := node.New(nodeConfig)

	chainDb, err := tempNode.OpenDatabaseWithFreezer("chaindata", 256, 256, "", "eth/db/chaindata/", false)
	trieDb := trie.NewDatabase(chainDb)

	// prepare snaps
	snapconfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    false,
		AsyncBuild: false,
	}

	snaps, _ := snapshot.New(snapconfig, chainDb, trieDb, common.HexToHash("hellomynameisghc")) // TODO I'm not sure about this code

	stateDB, err := state.New(common.Hash{}, state.NewDatabaseWithNodeDB(chainDb, trieDb), snaps)

	if err != nil {
		return
	}

	return &DbNode{
		ind:     ind,
		datadir: datadir,
		stateDb: stateDB,
		trieDb:  trieDb,
	}, nil
}

func NewDbNodes(n int) (nodes []*DbNode, err error) {
	for i := 0; i < n; i++ {
		newNode, nerr := NewDbNode(i)
		if nerr != nil {
			err = nerr
			return
		}
		nodes = append(nodes, newNode)
	}
	return
}

func (n *DbNode) AddBalance(address common.Address, value *big.Int) {
	n.stateDb.AddBalance(address, value)
}

func (n *DbNode) Clean() error {
	err := os.RemoveAll(n.datadir)
	if err != nil {
		return err
	}
	return nil
}
