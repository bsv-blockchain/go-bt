// Package main demonstrates how to marshal a list of transactions into a Node JSON format using the go-bt library.
package main

import (
	"encoding/json"
	"log"

	"github.com/bsv-blockchain/go-bt/v2"
)

func main() {
	tx, err := bt.NewTxFromString("0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000")
	if err != nil {
		panic(err)
	}

	tx2, err := bt.NewTxFromString("020000000117d2011c2a3b8a309d481930bae86e88017b0f55845ada17f96c464684b3af520000000048473044022014a60c3e84cf0160cb7e4ee7d87a3b78c5efb6dd3b66c76970b680affdb95e8f02207f6d9e3268a934e5e278ae513a3bc6dee3bec7bae37204574480305bfb5dea0e41feffffff0240101024010000001976a9149933e4bad50e7dd4b48c1f0be98436ca7d4392a288ac00e1f505000000001976a914abbe187ad301e4326e59587e43d602edd318364e88ac77000000")
	if err != nil {
		panic(err)
	}

	txs := bt.Txs{tx, tx2}

	bb, err := json.MarshalIndent(txs.NodeJSON(), "", "  ")
	if err != nil {
		panic(err)
	}
	log.Println(string(bb))
}
