package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"time"
)

func geth(ctx *cli.Context) error {
	measureTime := ctx.IsSet(MeasureTimeFlag.Name)
	measureStorage := ctx.IsSet(MeasureStorageFlag.Name)
	dbNode, err := NewDbNode(0)
	if err != nil {
		return err
	}

	txCount := 0
	timeSum := time.Duration(0)
	err = processTxFromZip(func(height int) error {
		if measureTime {
			fmt.Print(" ")
			if txCount != 0 {
				fmt.Print(float64(timeSum.Nanoseconds()) / float64(txCount))
			} else {
				fmt.Print("-1")
			}
		}
		timeSum = 0
		txCount = 0
		return dbNode.finishBlock(height, measureStorage, measureTime)
	}, func(tx txFromZip) error {
		timeSum += dbNode.executeTx(tx)
		txCount++
		return nil
	}, prepareFiles(ctx)...)
	if err != nil {
		return err
	}

	if ctx.IsSet(cleanFlag.Name) {
		err = dbNode.Clean()
		if err != nil {
			return err
		}
	}
	return nil
}
