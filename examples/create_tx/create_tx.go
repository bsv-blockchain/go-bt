// Package main demonstrates how to create a transaction using the go-bt library.
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
		"11b476ad8e0a48fcd40807a111a050af51114877e09283bfa7f3505081a1819d",
		0,
		"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac6a0568656c6c6f",
		1500,
	)

	_ = tx.PayToAddress("1NRoySJ9Lvby6DuE2UQYnyT67AASwNZxGb", 1000)

	pk, _ := primitives.PrivateKeyFromWif("KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")

	if err := tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk}); err != nil {
		log.Fatal(err.Error())
	}
	log.Printf("tx: %s\n", tx)
}
