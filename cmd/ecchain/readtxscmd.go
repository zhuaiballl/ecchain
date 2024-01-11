package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

var (
	readtxcmd = &cli.Command{
		Name:      "readtx",
		Usage:     "Read transactions from zip",
		Action:    readTxFromZipCmd,
		ArgsUsage: "",
		Description: `
    ecchain readtx /path/to/my.zip`,
	}
	executetxcmd = &cli.Command{
		Name:        "executetx",
		Usage:       "Execute transactions from zip",
		Action:      executeTxFromZipCmd,
		ArgsUsage:   "",
		Description: "ecchain execute /path/to/my.zip",
	}
)

func processTxFromZip(f func(int, []string) error, files ...string) error {
	cntLine := 0
	for _, file := range files {
		fmt.Println(file)
		fileName := filepath.Base(file)

		// remove file name extension
		ext := filepath.Ext(fileName)
		fileName = fileName[:len(fileName)-len(ext)]

		zipFilePath := file
		theZIP, err := zip.OpenReader(zipFilePath)
		if err != nil {
			return err
		}
		defer theZIP.Close()

		theCSVFile, err := theZIP.Open(fileName + ".csv")
		if err != nil {
			return err
		}
		defer theCSVFile.Close()

		csvReader := csv.NewReader(theCSVFile)
		_, err = csvReader.Read() // Skip header row

		for {
			oneLine, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			cntLine++
			err = f(cntLine, oneLine[:18])
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readTxFromZipCmd(ctx *cli.Context) error {
	files := ctx.Args().Slice()
	return processTxFromZip(func(i int, strings []string) error {
		fmt.Println(i, strings)
		return nil
	}, files...)
}

func executeTx(s *state.StateDB, ind int, record []string) error {
	// Parse transaction data from record
	//blockNumber := ToInt(record[0])
	//timestamp := ToInt(record[1])
	//transactionHash := record[2]
	sender := record[3]
	to := record[4]
	//toCreate := record[5]
	//fromIsContract := record[6]
	//toIsContract := record[7]
	value := new(big.Int)
	value.SetString(record[8], 10)
	//gasLimit := ToInt(record[9])
	//gasPrice := ToInt(record[10])
	//gasUsed := ToInt(record[11])
	//callingFunction := record[12]
	//isError := record[13]
	//eip2718type := ToInt(record[14])
	//baseFeePerGas := ToInt(record[15])
	//maxFeePerGas := ToInt(record[16])
	//maxPriorityFeePerGas := ToInt(record[17])

	s.AddBalance(common.HexToAddress(sender), big.NewInt(1))
	s.AddBalance(common.HexToAddress(to), value)
	if ind%10000 == 0 {
		root, err := s.Commit(true)
		if err != nil {
			return err
		}
		fmt.Println(ind, root)
	}
	return nil
}

func prepareDatabase() (*state.StateDB, string, error) {
	datadir, err := os.MkdirTemp("", "ecchain")
	if err != nil {
		return nil, "", err
	}
	fmt.Println(datadir)
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

	chainDb, err := tempNode.OpenDatabaseWithFreezer("babadata", 256, 256, "", "eth/db/chaindata/", false)
	db := trie.NewDatabase(chainDb)
	stateDB, err := state.New(common.Hash{}, state.NewDatabaseWithNodeDB(chainDb, db), nil)

	//mpt, err := trie.New(trie.TrieID(common.Hash{}), db)
	if err != nil {
		return nil, "", err
	}

	return stateDB, datadir, nil
}

func executeTxFromZipCmd(ctx *cli.Context) error {
	// read transactions from zip files
	files := ctx.Args().Slice()

	stateDB, datadir, err := prepareDatabase()

	err = processTxFromZip(func(ind int, strings []string) error {
		return executeTx(stateDB, ind, strings)
	}, files...)

	// measure storage cost
	if err != nil {
		return err
	}
	fmt.Println(datadir)
	storageCost, _ := exec.Command("du", "-sh", datadir).Output()
	fmt.Println(string(storageCost))

	if ctx.Bool("clean") {
		err = os.RemoveAll(datadir)
		if err != nil {
			return err
		}
	}
	return nil
}

func ToInt(str string) int {
	value, err := strconv.Atoi(str)
	if err != nil {
		return 0
	}
	return value
}

func ToFloat(str string) float64 {
	value, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return 0.0
	}
	return value
}
