package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/urfave/cli/v2"
	"io"
	"math/big"
	"os"
	"path/filepath"
	"strconv"
	"time"
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
		Action:      executeTxFromZip,
		ArgsUsage:   "",
		Description: "ecchain execute /path/to/my.zip",
	}
)

func readTxFromZip(files ...string) ([][]string, error) {
	var records [][]string

	for _, file := range files {
		fmt.Println(file)
		fileName := filepath.Base(file)

		// remove file name extension
		ext := filepath.Ext(fileName)
		fileName = fileName[:len(fileName)-len(ext)]

		zipFilePath := file
		theZIP, err := zip.OpenReader(zipFilePath)
		if err != nil {
			return nil, err
		}
		defer theZIP.Close()

		theCSVFile, err := theZIP.Open(fileName + ".csv")
		if err != nil {
			return nil, err
		}
		defer theCSVFile.Close()

		csvReader := csv.NewReader(theCSVFile)
		_, err = csvReader.Read() // Skip header row

		for {
			oneLine, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return nil, err
			}
			records = append(records, oneLine[:18])
		}
	}
	return records, nil
}

func readTxFromZipCmd(ctx *cli.Context) error {
	files := ctx.Args().Slice()
	records, err := readTxFromZip(files...)
	if err != nil {
		return err
	}
	for _, record := range records {
		fmt.Println(record)
	}
	return nil
}

func executeTxs(s *state.StateDB, t *trie.Trie, records [][]string) error {
	for i, record := range records {
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

		s.SubBalance(common.HexToAddress(sender), value)
		s.AddBalance(common.HexToAddress(to), value)
		root, err := s.Commit(false)
		if err != nil {
			return err
		}
		fmt.Println(i, root)
	}
	return nil
}

func executeTxFromZip(ctx *cli.Context) error {
	// read transactions from zip files
	files := ctx.Args().Slice()
	records, err := readTxFromZip(files...)
	if err != nil {
		return err
	}

	// Instantiate an empty MPT
	db := trie.NewDatabase(nil)
	mpt, err := trie.New(trie.TrieID(common.Hash{}), db)
	if err != nil {
		return err
	}

	// Initialize the state
	datadir, _ := os.MkdirTemp("", "")
	config := &node.Config{
		Name:    "geth",
		Version: params.Version,
		DataDir: datadir,
		P2P: p2p.Config{
			ListenAddr:  "0.0.0.0:0",
			NoDiscovery: true,
			MaxPeers:    25,
		},
		UseLightweightKDF: true,
	}
	stack, err := node.New(config)
	genesis := core.DefaultGenesisBlock()
	genesis.Difficulty = params.MinimumDifficulty
	genesis.GasLimit = 25000000

	genesis.BaseFee = big.NewInt(params.InitialBaseFee)
	genesis.Config = params.AllEthashProtocolChanges
	genesis.Config.TerminalTotalDifficulty = new(big.Int).Mul(big.NewInt(20), params.MinimumDifficulty)

	genesis.Alloc = core.GenesisAlloc{}
	econfig := &ethconfig.Config{
		Genesis:         genesis,
		NetworkId:       genesis.Config.ChainID.Uint64(),
		SyncMode:        downloader.FullSync,
		DatabaseCache:   256,
		DatabaseHandles: 256,
		TxPool:          txpool.DefaultConfig,
		GPO:             ethconfig.Defaults.GPO,
		Ethash:          ethconfig.Defaults.Ethash,
		Miner: miner.Config{
			GasFloor: genesis.GasLimit * 9 / 10,
			GasCeil:  genesis.GasLimit * 11 / 10,
			GasPrice: big.NewInt(1),
			Recommit: 1 * time.Second,
		},
		LightServ:        100,
		LightPeers:       10,
		LightNoSyncServe: true,
	}
	chainDb, err := stack.OpenDatabaseWithFreezer("chaindata", econfig.DatabaseCache, econfig.DatabaseHandles, econfig.DatabaseFreezer, "eth/db/chaindata/", false)
	stateDB, err := state.New(common.Hash{}, state.NewDatabaseWithNodeDB(chainDb, db), nil)

	err = executeTxs(stateDB, mpt, records)
	if err != nil {
		return err
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
