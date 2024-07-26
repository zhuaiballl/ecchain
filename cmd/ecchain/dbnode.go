package main

import (
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"math/big"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
	fmt.Println("Created a node, its datadir is:", datadir)
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

func (dbNode *DbNode) Exist(address common.Address) bool {
	return dbNode.stateDb.Exist(address)
}

func (dbNode *DbNode) Delete(address common.Address) {
	obj := dbNode.stateDb.GetOrNewStateObject(address)
	dbNode.stateDb.DeleteStateObject(obj)
}

func (dbNode *DbNode) AddBalance(address common.Address, value *big.Int) {
	dbNode.stateDb.AddBalance(address, value)
}

func (dbNode *DbNode) SetBalance(address common.Address, amount *big.Int) {
	dbNode.stateDb.SetBalance(address, amount)
}

func (dbNode *DbNode) GetBalance(address common.Address) *big.Int {
	return dbNode.stateDb.GetBalance(address)
}

func (dbNode *DbNode) GetNonce(address common.Address) uint64 {
	return dbNode.stateDb.GetNonce(address)
}

func (dbNode *DbNode) Clean() error {
	err := os.RemoveAll(dbNode.datadir)
	if err != nil {
		return err
	}
	return nil
}

func (dbNode *DbNode) Commit() error {
	root, err := dbNode.stateDb.Commit(true)
	if err != nil {
		return err
	}
	err = dbNode.trieDb.Commit(root, false)
	if err != nil {
		return err
	}
	return nil
}

func (dbNode *DbNode) executeTx(tx txFromZip) time.Duration {
	timeBegin := time.Now()
	for _, addrString := range []string{tx.sender, tx.to} {
		addr := common.HexToAddress(addrString)
		dbNode.AddBalance(addr, tx.value)
	}
	return time.Since(timeBegin)
}

func (dbNode *DbNode) StorageCost() int {
	cmdOutput, _ := exec.Command("du", "-s", dbNode.datadir).Output()
	storageCost := string(cmdOutput)
	storageCost = strings.Fields(storageCost)[0]
	costInt, _ := strconv.Atoi(storageCost)
	return costInt
}
