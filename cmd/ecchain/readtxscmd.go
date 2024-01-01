package main

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"github.com/urfave/cli/v2"
	"io"
	"path/filepath"
	"strconv"
)

var (
	readtxcmd = &cli.Command{
		Name:      "readtx",
		Usage:     "Read transactions from csv",
		Action:    readTxFromCsv,
		ArgsUsage: "",
		Description: `
    ecchain readtx /path/to/my.csv`,
	}
)

func readTxFromCsv(ctx *cli.Context) error {
	files := ctx.Args().Slice()

	txCount := 0
	totalFees := 0.0

	for _, file := range files {
		fmt.Println(file)
		fileName := filepath.Base(file)
		ext := filepath.Ext(fileName)
		fileName = fileName[:len(fileName)-len(ext)]
		fmt.Println(fileName)
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

			fmt.Println(oneLine[:18])

			//blockNumber := ToInt(oneLine[0])
			//timestamp := ToInt(oneLine[1])
			//transactionHash := oneLine[2]
			//sender := oneLine[3]
			//to := oneLine[4]
			//toCreate := oneLine[5]
			//fromIsContract := oneLine[6]
			//toIsContract := oneLine[7]
			//value := ToInt(oneLine[8])
			//gasLimit := ToInt(oneLine[9])
			//gasPrice := ToInt(oneLine[10])
			//gasUsed := ToInt(oneLine[11])
			//callingFunction := oneLine[12]
			//isError := oneLine[13]
			//eip2718type := ToInt(oneLine[14])
			//baseFeePerGas := ToInt(oneLine[15])
			//maxFeePerGas := ToInt(oneLine[16])
			//maxPriorityFeePerGas := ToInt(oneLine[17])
			//
			//totalFees += float64(gasPrice * gasUsed)
			//txCount++
			//if txCount%100000 == 0 {
			//	fmt.Println(txCount, totalFees/1e+18)
			//}
		}
	}

	fmt.Println(txCount, totalFees/1e+18)
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
