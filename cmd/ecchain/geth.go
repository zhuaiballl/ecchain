package main

import (
	"fmt"
	"github.com/urfave/cli/v2"
	"time"
)

func geth(ctx *cli.Context) error {
	measureTime := ctx.IsSet(measureTimeFlag.Name)
	measureStorage := ctx.IsSet(measureStorageFlag.Name)
	debugging := ctx.IsSet(debugFlag.Name)

	dbNode, err := NewDbNode(0)
	if err != nil {
		return err
	}

	txCount := 0
	lstBlock := -1

	timeSum := time.Duration(0)
	err = processTxFromZip(func(height int) error {
		if debugging || height/10000 != lstBlock/10000 {
			fmt.Print(height, " ")
			defer fmt.Println("")
		}
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
		if err := dbNode.Commit(); err != nil {
			return err
		}
		if measureStorage && (debugging || height/10000 != lstBlock/10000) {
			fmt.Print(" ", dbNode.StorageCost())
		}
		lstBlock = height
		return nil
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
