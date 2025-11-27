package ord_test

import (
	"context"
	"encoding/hex"
	"testing"

	primitives "github.com/bsv-blockchain/go-sdk/primitives/ec"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/bsv-blockchain/go-bt/v2/ord"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
)

func TestOfferToSellPSBTNoErrors(t *testing.T) {
	ordPk, _ := primitives.PrivateKeyFromWif("L42PyNwEKE4XRaa8PzPh7JZurSAWJmx49nbVfaXYuiQg3RCubwn7") // 1JijRHzVfub38S2hizxkxEcVKQwuCTZmxJ
	ordPrefixAddr, _ := bscript.NewAddressFromPublicKeyString(hex.EncodeToString(ordPk.PubKey().Compressed()), true)
	ordPrefixScript, _ := bscript.NewP2PKHFromAddress(ordPrefixAddr.AddressString)

	ordUnlockerGetter := unlocker.Getter{PrivateKey: ordPk}
	ordUnlocker, _ := ordUnlockerGetter.Unlocker(context.Background(), ordPrefixScript)

	ordUTXO := &bt.UTXO{
		TxIDHash: func() *chainhash.Hash {
			t, _ := chainhash.NewHashFromStr("8f027fb1361ae46ac165e1d90e5436ed9c11d4eeaa60669ab90386a3abd9ce6a")
			return t
		}(),
		Vout: uint32(0),
		LockingScript: func() *bscript.Script {
			// hello world (text/plain) test inscription
			s, _ := bscript.NewFromHexString("76a914c25e9a2b70ec83d7b4fbd0f36f00a86723a48e6b88ac0063036f72645118746578742f706c61696e3b636861727365743d7574662d38000d48656c6c6f2c20776f726c642168")
			return s
		}(),
		Satoshis: 1,
	}

	pstx, CreateListingError := ord.ListOrdinalForSale(context.Background(), &ord.ListOrdinalArgs{
		SellerReceiveOutput: &bt.Output{
			Satoshis: 1000,
			LockingScript: func() *bscript.Script {
				s, _ := bscript.NewP2PKHFromAddress("1C3V9TTJefP8Hft96sVf54mQyDJh8Ze4w4") // L1JWiLZtCkkqin41XtQ2Jxo1XGxj1R4ydT2zmxPiaeQfuyUK631D
				return s
			}(),
		},
		OrdinalUTXO:     ordUTXO,
		OrdinalUnlocker: ordUnlocker,
	})

	t.Run("no errors creating PSBT to make an offer to sell ordinal", func(t *testing.T) {
		require.NoError(t, CreateListingError)
	})

	t.Run("validate PSBT to make an offer to sell ordinal", func(t *testing.T) {
		vla := &ord.ValidateListingArgs{
			ListedOrdinalUTXO: ordUTXO,
		}
		assert.True(t, vla.Validate(pstx))
	})

	us := []*bt.UTXO{
		{
			TxIDHash: func() *chainhash.Hash {
				t, _ := chainhash.NewHashFromStr("8f027fb1361ae46ac165e1d90e5436ed9c11d4eeaa60669ab90386a3abd9ce6a")
				return t
			}(),
			Vout:          uint32(1),
			LockingScript: ordPrefixScript,
			Satoshis:      953,
			Unlocker:      &ordUnlocker,
		},
		{
			TxIDHash: func() *chainhash.Hash {
				t, _ := chainhash.NewHashFromStr("fcc55cd1a4275e5750070381028d3e3edf99b238bdc56199ff8bdc17dfb599d1")
				return t
			}(),
			Vout:          uint32(3),
			LockingScript: ordPrefixScript,
			Satoshis:      27601,
			Unlocker:      &ordUnlocker,
		},
	}
	buyerOrdS, _ := bscript.NewP2PKHFromAddress("1HebepswCi6huw1KJ7LvkrgemAV63TyVUs") // KwQq67d4Jds3wxs3kQHB8PPwaoaBQfNKkzAacZeMesb7zXojVYpj
	dummyS, _ := bscript.NewP2PKHFromAddress("19NfKd8aTwvb5ngfP29RxgfQzZt8KAYtQo")    // L5W2nyKUCsDStVUBwZj2Q3Ph5vcae4bgdzprZDYqDpvZA8AFguFH
	changeS, _ := bscript.NewP2PKHFromAddress("19NfKd8aTwvb5ngfP29RxgfQzZt8KAYtQo")   // L5W2nyKUCsDStVUBwZj2Q3Ph5vcae4bgdzprZDYqDpvZA8AFguFH

	t.Run("no errors when accepting listing", func(t *testing.T) {
		_, err := ord.AcceptOrdinalSaleListing(context.Background(), &ord.ValidateListingArgs{
			ListedOrdinalUTXO: ordUTXO,
		},
			&ord.AcceptListingArgs{
				PSTx:                      pstx,
				UTXOs:                     us,
				BuyerReceiveOrdinalScript: buyerOrdS,
				DummyOutputScript:         dummyS,
				ChangeScript:              changeS,
				FQ:                        bt.NewFeeQuote(),
			})
		require.NoError(t, err)
	})

	// TODO: are 2 dummies useful or to be removed?
	t.Run("no errors when accepting listing using 2 dummies", func(t *testing.T) {
		us = append([]*bt.UTXO{
			{
				TxIDHash: func() *chainhash.Hash {
					t, _ := chainhash.NewHashFromStr("61dfcc313763eb5332c036131facdf92c2ca9d663ffb96e4b997086a0643d635")
					return t
				}(),
				Vout:          uint32(0),
				LockingScript: ordPrefixScript,
				Satoshis:      10,
				Unlocker:      &ordUnlocker,
			},
			{
				TxIDHash: func() *chainhash.Hash {
					t, _ := chainhash.NewHashFromStr("61dfcc313763eb5332c036131facdf92c2ca9d663ffb96e4b997086a0643d635")
					return t
				}(),
				Vout:          uint32(1),
				LockingScript: ordPrefixScript,
				Satoshis:      10,
				Unlocker:      &ordUnlocker,
			},
		}, us...)

		_, err := ord.AcceptOrdinalSaleListing2Dummies(context.Background(), &ord.ValidateListingArgs{
			ListedOrdinalUTXO: ordUTXO,
		},
			&ord.AcceptListingArgs{
				PSTx:                      pstx,
				UTXOs:                     us,
				BuyerReceiveOrdinalScript: buyerOrdS,
				DummyOutputScript:         dummyS,
				ChangeScript:              changeS,
				FQ:                        bt.NewFeeQuote(),
			})
		require.NoError(t, err)
	})
	//
}
