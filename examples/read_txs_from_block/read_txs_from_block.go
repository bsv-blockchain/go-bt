// Package main demonstrates how to read transactions from a block file in a memory-efficient way using the go-bt library.
package main

import (
	"bufio"
	"io"
	"log"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/testing/data"
)

// In this example, all txs from a block are being read in via chunking, so at no point
// does the entire block have to be held in memory, and instead can be streamed.
//
// We represent the block by interactively reading a file, however it could be any data
// stream that satisfies the io.Reader interface.

func main() {
	// Open file container block data.
	f, err := data.TxBinData.Open("block.bin")
	if err != nil {
		panic(err)
	}
	defer func() {
		_ = f.Close()
	}()

	// Create buffered reader for this file.
	r := bufio.NewReader(f)

	// Read file header. This step is specific to file reading and
	// may need omitted or modified for other implementations.
	_, err = io.ReadFull(f, make([]byte, 80))
	if err != nil {
		panic(err)
	}

	txs := bt.Txs{}
	if _, err = txs.ReadFrom(r); err != nil {
		panic(err)
	}
	for _, tx := range txs {
		log.Println(tx.TxID())
	}
}
