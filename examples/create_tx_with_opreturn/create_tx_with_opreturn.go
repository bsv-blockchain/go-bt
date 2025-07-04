// Package main demonstrates how to create a transaction with an OP_RETURN output using the go-bt library.
package main

import (
	"context"
	"log"

	primitives "github.com/bsv-blockchain/go-sdk/primitives/ec"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
)

func main() {
	tx := bt.NewTx()

	_ = tx.From(
		"b7b0650a7c3a1bd4716369783876348b59f5404784970192cec1996e86950576",
		0,
		"76a9149cbe9f5e72fa286ac8a38052d1d5337aa363ea7f88ac",
		1000,
	)

	_ = tx.PayToAddress("1C8bzHM8XFBHZ2ZZVvFy2NSoAZbwCXAicL", 900)

	_ = tx.AddOpReturnOutput([]byte("You are using go-bt!"))

	pk, _ := primitives.PrivateKeyFromWif("L3VJH2hcRGYYG6YrbWGmsxQC1zyYixA82YjgEyrEUWDs4ALgk8Vu")

	err := tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk})
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println("tx: ", tx.String())
}
