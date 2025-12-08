// Package main demonstrates how to create a bid for an ordinal inscription using the go-bt library.
package main

import (
	"context"
	"encoding/hex"
	"log"

	primitives "github.com/bsv-blockchain/go-sdk/primitives/ec"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/bsv-blockchain/go-bt/v2/ord"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
)

func main() {
	fundingPk, _ := primitives.PrivateKeyFromWif("L5W2nyKUCsDStVUBwZj2Q3Ph5vcae4bgdzprZDYqDpvZA8AFguFH") // 19NfKd8aTwvb5ngfP29RxgfQzZt8KAYtQo
	fundingAddr, _ := bscript.NewAddressFromPublicKeyString(hex.EncodeToString(fundingPk.PubKey().Compressed()), true)
	fundingScript, _ := bscript.NewP2PKHFromAddress(fundingAddr.AddressString)
	fundingUnlockerGetter := unlocker.Getter{PrivateKey: fundingPk}
	fundingUnlocker, _ := fundingUnlockerGetter.Unlocker(context.Background(), fundingScript)

	bidAmount := 100000000

	us := []*bt.UTXO{
		{
			TxIDHash: func() *chainhash.Hash {
				t, _ := chainhash.NewHashFromStr("411084d83d4f380cfc331ed849946bd7f354ca17138dbd723a6420ec9f5f4bd1")
				return t
			}(),
			Vout:          uint32(0),
			LockingScript: fundingScript,
			Satoshis:      20,
			Unlocker:      &fundingUnlocker,
		},
		{
			TxIDHash: func() *chainhash.Hash {
				t, _ := chainhash.NewHashFromStr("411084d83d4f380cfc331ed849946bd7f354ca17138dbd723a6420ec9f5f4bd1")
				return t
			}(),
			Vout:          uint32(1),
			LockingScript: fundingScript,
			Satoshis:      20,
			Unlocker:      &fundingUnlocker,
		},
		{
			TxIDHash: func() *chainhash.Hash {
				t, _ := chainhash.NewHashFromStr("fc136d44114bdaa99f2d7d06a0fee514d376d974af53a3909fc43a79a3644653")
				return t
			}(),
			Vout:          uint32(0),
			LockingScript: fundingScript,
			Satoshis:      100027971,
			Unlocker:      &fundingUnlocker,
		},
	}

	mba := &ord.MakeBidArgs{
		BidAmount:   uint64(bidAmount), //nolint:gosec // example bid amount
		OrdinalTxID: "e17d7856c375640427943395d2341b6ed75f73afc8b22bb3681987278978a584",
		OrdinalVOut: 81,
		BidderUTXOs: us,
		BuyerReceiveOrdinalScript: func() *bscript.Script {
			s, _ := bscript.NewP2PKHFromAddress("1JPxYgWSYCb3ZEBBkcum84AHHdPWQzHGXj")
			return s
		}(),
		DummyOutputScript: func() *bscript.Script {
			s, _ := bscript.NewP2PKHFromAddress("19NfKd8aTwvb5ngfP29RxgfQzZt8KAYtQo") // L1JWiLZtCkkqin41XtQ2Jxo1XGxj1R4ydT2zmxPiaeQfuyUK631D
			return s
		}(),
		ChangeScript: func() *bscript.Script {
			s, _ := bscript.NewP2PKHFromAddress("19NfKd8aTwvb5ngfP29RxgfQzZt8KAYtQo") // L1JWiLZtCkkqin41XtQ2Jxo1XGxj1R4ydT2zmxPiaeQfuyUK631D
			return s
		}(),
		FQ: bt.NewFeeQuote(),
	}

	pstx, err := ord.MakeBidToBuy1SatOrdinal(context.Background(), mba)
	if err != nil {
		log.Fatal(err.Error())
	}
	log.Println(pstx.String())
}
