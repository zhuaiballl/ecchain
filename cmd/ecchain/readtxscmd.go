package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"math/big"
	"path/filepath"
	"strconv"
)

var zips []string = []string{
	"0to999999",
	"1000000to1999999",
	"2000000to2999999",
	"3000000to3999999",
	"4000000to4999999",
	"5000000to5999999",
	"6000000to6999999",
	"7000000to7999999",
	"8000000to8999999",
	"9000000to9999999",
	"10000000to10999999",
	"11000000to11999999",
	"12000000to12999999",
	"13000000to13249999",
	"13250000to13499999",
	"13500000to13749999",
	"13750000to13999999",
	"14000000to14249999",
	"14250000to14499999",
	"14500000to14749999",
	"14750000to14999999",
	"15000000to15249999",
	"15250000to15499999",
	"15500000to15749999",
	"15750000to15999999",
	"16000000to16249999",
	"16250000to16499999",
	"16500000to16749999",
	"16750000to16999999",
	"17000000to17249999",
	"17250000to17499999",
	"17750000to17999999",
	"18000000to18249999",
	"18250000to18499999",
}

var (
	readtxcmd = &cli.Command{
		Name:      "readtx",
		Usage:     "Read transactions from zip",
		Action:    readTxFromZipCmd,
		ArgsUsage: "",
		Description: `
    ecchain readtx /path/to/my.zip`,
	}
	gethCmd = &cli.Command{
		Name:   "geth",
		Usage:  "Execute transactions from zip",
		Action: geth,
		Flags: []cli.Flag{
			cleanFlag,
			zipDirFlag,
			measureTimeFlag,
			measureStorageFlag,
			debugFlag,
		},
		ArgsUsage:   "",
		Description: "ecchain geth /path/to/my.zip",
	}
)

type txFromZip struct {
	txNumber             int
	blockNumber          int
	timestamp            int
	transactionHash      string
	sender               string
	to                   string
	toCreate             string
	fromIsContract       string
	toIsContract         string
	value                *big.Int
	gasLimit             int
	gasPrice             int
	gasUsed              int
	callingFunction      string
	isError              string
	eip2718type          int
	baseFeePerGas        int
	maxFeePerGas         int
	maxPriorityFeePerGas int
}

func processTxFromZip(finishBlock func(int) error, processTx func(txFromZip) error, files ...string) error {
	cntLine := 0
	lastBlockNumber := -1
	for _, file := range files {
		//fmt.Println(file)
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
			record, err := csvReader.Read()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			cntLine++
			// Parse transaction data from record
			tx := txFromZip{
				cntLine,
				ToInt(record[0]),
				ToInt(record[1]),
				record[2],
				record[3],
				record[4],
				record[5],
				record[6],
				record[7],
				big.NewInt(1),
				ToInt(record[9]),
				ToInt(record[10]),
				ToInt(record[11]),
				record[12],
				record[13],
				ToInt(record[14]),
				ToInt(record[15]),
				ToInt(record[16]),
				ToInt(record[17]),
			}
			tx.value.SetString(record[8], 10)

			// If the previous block ends, run finishBlock
			if lastBlockNumber != tx.blockNumber {
				if lastBlockNumber != -1 {
					err = finishBlock(lastBlockNumber)
					if err != nil {
						return err
					}
				}
				lastBlockNumber = tx.blockNumber
			}

			// process tx
			err = processTx(tx)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func readTxFromZipCmd(ctx *cli.Context) error {
	files := ctx.Args().Slice()
	return processTxFromZip(func(i int) error {
		return nil
	}, func(tx txFromZip) error {
		fmt.Println(tx)
		return nil
	}, files...)
}

func prepareFiles(ctx *cli.Context) (files []string) {
	if ctx.IsSet(zipDirFlag.Name) {
		zipDir := ctx.String(zipDirFlag.Name)
		for _, fileName := range zips {
			files = append(files, filepath.Join(zipDir, fileName+"_BlockTransaction.zip"))
		}
	} else {
		// read transactions from zip files
		files = ctx.Args().Slice()
	}
	return
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
