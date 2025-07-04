package bt_test

import (
	"context"
	"encoding/hex"
	"errors"
	"math"
	"testing"

	primitives "github.com/bsv-blockchain/go-sdk/primitives/ec"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/bsv-blockchain/go-bt/v2/sighash"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddInputFromTx(t *testing.T) {
	pubkey1, _ := hex.DecodeString("0280f642908697e8068c2e921bd998d6c2b90553064656f91b9cb9e98f443aac30")
	pubkey2, _ := hex.DecodeString("02434dc3db4281c0895d7a126bb266e7648caca7d0e2e487bc41f954722d4ee397")

	prvTx := bt.NewTx()
	err := prvTx.AddP2PKHOutputFromPubKeyBytes(pubkey1, uint64(100000))
	require.NoError(t, err)
	err = prvTx.AddP2PKHOutputFromPubKeyBytes(pubkey1, uint64(100000))
	require.NoError(t, err)
	err = prvTx.AddP2PKHOutputFromPubKeyBytes(pubkey2, uint64(100000))
	require.NoError(t, err)

	newTx := bt.NewTx()
	err = newTx.AddP2PKHInputsFromTx(prvTx, pubkey1)
	require.NoError(t, err)
	assert.Equal(t, 2, newTx.InputCount()) // only 2 utxos added
	assert.Equal(t, uint64(200000), newTx.TotalInputSatoshis())
}

func TestTx_InputCount(t *testing.T) {
	t.Run("get input count", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		err := tx.From(
			"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
			0,
			"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
			4000000,
		)
		require.NoError(t, err)
		assert.Equal(t, 1, tx.InputCount())
	})
}

func TestTx_From(t *testing.T) {
	t.Run("invalid locking script (hex decode failed)", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		err := tx.From(
			"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
			0,
			"0",
			4000000,
		)
		require.Error(t, err)

		err = tx.From(
			"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
			0,
			"76a914af2590a45ae4016",
			4000000,
		)
		require.Error(t, err)
	})

	t.Run("valid script and tx", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		err := tx.From(
			"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
			0,
			"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
			4000000,
		)
		require.NoError(t, err)

		inputs := tx.Inputs
		assert.Len(t, inputs, 1)
		assert.Equal(t, "07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b", inputs[0].PreviousTxIDStr())
		assert.Equal(t, uint32(0), inputs[0].PreviousTxOutIndex)
		assert.Equal(t, uint64(4000000), inputs[0].PreviousTxSatoshis)
		assert.Equal(t, bt.DefaultSequenceNumber, inputs[0].SequenceNumber)
		assert.Equal(t, "76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac", inputs[0].PreviousTxScript.String())
	})
}

func TestTx_FromUTXOs(t *testing.T) {
	t.Parallel()

	t.Run("one utxo", func(t *testing.T) {
		tx := bt.NewTx()
		script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
		require.NoError(t, err)

		txID, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
		require.NoError(t, err)

		require.NoError(t, tx.FromUTXOs(&bt.UTXO{
			TxIDHash:      txID,
			LockingScript: script,
			Vout:          0,
			Satoshis:      1000,
		}))

		input := tx.Inputs[0]
		assert.Len(t, tx.Inputs, 1)
		assert.Equal(t, "07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b", input.PreviousTxIDStr())
		assert.Equal(t, uint32(0), input.PreviousTxOutIndex)
		assert.Equal(t, uint64(1000), input.PreviousTxSatoshis)
		assert.Equal(t, "76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac", input.PreviousTxScript.String())
	})

	t.Run("multiple utxos", func(t *testing.T) {
		tx := bt.NewTx()
		script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
		require.NoError(t, err)
		txID, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
		require.NoError(t, err)

		script2, err := bscript.NewFromHexString("76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac")
		require.NoError(t, err)
		txID2, err := chainhash.NewHashFromStr("3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5")
		require.NoError(t, err)

		require.NoError(t, tx.FromUTXOs(&bt.UTXO{
			TxIDHash:      txID,
			LockingScript: script,
			Vout:          0,
			Satoshis:      1000,
		}, &bt.UTXO{
			TxIDHash:      txID2,
			LockingScript: script2,
			Vout:          1,
			Satoshis:      2000,
		}))

		assert.Len(t, tx.Inputs, 2)

		input := tx.Inputs[0]
		assert.Equal(t, "07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b", input.PreviousTxIDStr())
		assert.Equal(t, uint32(0), input.PreviousTxOutIndex)
		assert.Equal(t, uint64(1000), input.PreviousTxSatoshis)
		assert.Equal(t, "76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac", input.PreviousTxScript.String())

		input = tx.Inputs[1]
		assert.Equal(t, "3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5", input.PreviousTxIDStr())
		assert.Equal(t, uint32(1), input.PreviousTxOutIndex)
		assert.Equal(t, uint64(2000), input.PreviousTxSatoshis)
		assert.Equal(t, "76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac", input.PreviousTxScript.String())
	})
}

func TestTx_Fund(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		tx                      *bt.Tx
		utxos                   []*bt.UTXO
		utxoGetterFuncOverrider func([]*bt.UTXO) bt.UTXOGetterFunc
		expTotalInputs          int
		expErr                  error
	}{
		"tx with exact inputs and surplus inputs is covered": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 1500))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 2,
		},
		"tx with extra inputs and surplus inputs is covered with all utxos": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 1500))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 3,
		},
		"tx with extra inputs and surplus inputs that returns correct amount is covered with minimum needed utxos": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 1500))
				return tx
			}(),
			utxoGetterFuncOverrider: func(utxos []*bt.UTXO) bt.UTXOGetterFunc {
				return func(_ context.Context, _ uint64) ([]*bt.UTXO, error) {
					return utxos[:2], nil
				}
			},
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 2,
		},
		"tx with exact input satshis is covered": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 1500))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 2,
		},
		"tx with large amount of satoshis is covered with all utxos": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 500, 0xffffff, nil,
				}, {
					txid, 0, script, 670, 0xffffff, nil,
				}, {
					txid, 0, script, 700, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 650, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 8,
		},
		"tx with large amount of satoshis is covered with needed utxos": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))
				return tx
			}(),
			utxoGetterFuncOverrider: func(utxos []*bt.UTXO) bt.UTXOGetterFunc {
				utxosCopy := make([]*bt.UTXO, len(utxos))
				copy(utxosCopy, utxos)
				return func(_ context.Context, _ uint64) ([]*bt.UTXO, error) {
					defer func() { utxosCopy = utxosCopy[1:] }()
					return utxosCopy[:1], nil
				}
			},
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 500, 0xffffff, nil,
				}, {
					txid, 0, script, 670, 0xffffff, nil,
				}, {
					txid, 0, script, 700, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 650, 0xffffff, nil,
				}}
			}(),
			expTotalInputs: 7,
		},
		"getter with no utxos error": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 1500))
				return tx
			}(),
			utxos:  []*bt.UTXO{},
			expErr: bt.ErrInsufficientFunds,
		},
		"getter with insufficient utxos errors": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 25400))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 500, 0xffffff, nil,
				}, {
					txid, 0, script, 670, 0xffffff, nil,
				}, {
					txid, 0, script, 700, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 650, 0xffffff, nil,
				}}
			}(),
			expErr: bt.ErrInsufficientFunds,
		},
		"error is returned to the user": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				return tx
			}(),
			utxoGetterFuncOverrider: func([]*bt.UTXO) bt.UTXOGetterFunc {
				return func(context.Context, uint64) ([]*bt.UTXO, error) {
					return nil, errors.New("custom error")
				}
			},
			expErr: errors.New("custom error"),
		},
		"tx with large amount of satoshis is covered, with multiple iterations": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))
				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 500, 0xffffff, nil,
				}, {
					txid, 0, script, 670, 0xffffff, nil,
				}, {
					txid, 0, script, 700, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 1000, 0xffffff, nil,
				}, {
					txid, 0, script, 650, 0xffffff, nil,
				}}
			}(),
			utxoGetterFuncOverrider: func(utxos []*bt.UTXO) bt.UTXOGetterFunc {
				idx := 0
				return func(context.Context, uint64) ([]*bt.UTXO, error) {
					defer func() { idx++ }()
					return utxos[idx : idx+1], nil
				}
			},
			expTotalInputs: 7,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			iptFn := func() bt.UTXOGetterFunc {
				idx := 0
				return func(_ context.Context, _ uint64) ([]*bt.UTXO, error) {
					if idx == len(test.utxos) {
						return nil, bt.ErrNoUTXO
					}
					defer func() { idx += len(test.utxos) }()
					return test.utxos, nil
				}
			}()
			if test.utxoGetterFuncOverrider != nil {
				iptFn = test.utxoGetterFuncOverrider(test.utxos)
			}

			err := test.tx.Fund(context.Background(), FQPoint5SatPerByte, iptFn)
			if test.expErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, test.expErr.Error())
				return
			}

			require.NoError(t, err)
			assert.Equal(t, test.expTotalInputs, test.tx.InputCount())
		})
	}
}

func TestTx_Fund_Deficit(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		utxos       []*bt.UTXO
		expDeficits []uint64
		iteration   int
		tx          *bt.Tx
	}{
		"1 output worth 5000, 3 utxos worth 6000": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))

				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}}
			}(),
			iteration:   1,
			expDeficits: []uint64{5022, 3096, 1170},
		},
		"1 output worth 5000, 3 utxos worth 6000, iterations of 2": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))

				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}}
			}(),
			iteration:   2,
			expDeficits: []uint64{5022, 1170},
		},
		"5 outputs worth 35000, 12 utxos worth 37000": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 10000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 7000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 3000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 10000))

				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 4000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 6000, 0xffffff, nil,
				}, {
					txid, 0, script, 4000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 8000, 0xffffff, nil,
				}, {
					txid, 0, script, 3000, 0xffffff, nil,
				}}
			}(),
			iteration:   1,
			expDeficits: []uint64{35090, 33164, 31238, 29312, 27386, 23460, 21534, 15608, 11682, 9756, 1830},
		},
		"5 outputs worth 35000, 12 utxos worth 37000, iteration of 3": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 5000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 10000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 7000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 3000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 10000))

				return tx
			}(),
			utxos: func() []*bt.UTXO {
				txid, err := chainhash.NewHashFromStr("07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b")
				require.NoError(t, err)
				script, err := bscript.NewFromHexString("76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac")
				require.NoError(t, err)
				return []*bt.UTXO{{
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 4000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 6000, 0xffffff, nil,
				}, {
					txid, 0, script, 4000, 0xffffff, nil,
				}, {
					txid, 0, script, 2000, 0xffffff, nil,
				}, {
					txid, 0, script, 8000, 0xffffff, nil,
				}, {
					txid, 0, script, 3000, 0xffffff, nil,
				}}
			}(),
			iteration:   3,
			expDeficits: []uint64{35090, 29312, 21534, 9756},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			deficits := make([]uint64, 0)
			_ = test.tx.Fund(context.Background(), FQPoint5SatPerByte, func(_ context.Context, deficit uint64) ([]*bt.UTXO, error) {
				if len(test.utxos) == 0 {
					return nil, bt.ErrNoUTXO
				}
				step := int(math.Min(float64(test.iteration), float64(len(test.utxos))))
				defer func() {
					test.utxos = test.utxos[step:]
				}()

				deficits = append(deficits, deficit)
				return test.utxos[:step], nil
			})

			assert.Equal(t, test.expDeficits, deficits)
		})
	}
}

func TestTx_FillInput(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		inputIdx uint32
		shf      sighash.Flag
		unlocker bt.Unlocker
		expHex   string
		expErr   error
	}{
		"standard unlock": {
			inputIdx: 0,
			shf:      sighash.AllForkID,
			unlocker: func() bt.Unlocker {
				pk, err := primitives.PrivateKeyFromWif("L3MhnEn1pLWcggeYLk9jdkvA2wUK1iWwwrGkBbgQRqv6HPCdRxuw")
				require.NoError(t, err)

				return &unlocker.Simple{PrivateKey: pk}
			}(),
			expHex: "01000000010b94a1ef0fb352aa2adc54207ce47ba55d5a1c1609afda58fe9520e472299107000000006a473044022049ee0c0f26c00e6a6b3af5990fc8296c66eab3e3e42ab075069b89b1be6fefec02206079e49dd8c9e1117ef06fbe99714d822620b1f0f5d19f32a1128f5d29b7c3c4412102c8803fdd437d902f08e3c2344cb33065c99d7c99982018ff9f7219c3dd352ff0ffffffff01a0083d00000000001976a914af2590a45ae401651fdbdf59a76ad43d1862534088ac00000000",
		},
		"sighash all is used as default": {
			inputIdx: 0,
			unlocker: func() bt.Unlocker {
				pk, err := primitives.PrivateKeyFromWif("L3MhnEn1pLWcggeYLk9jdkvA2wUK1iWwwrGkBbgQRqv6HPCdRxuw")
				require.NoError(t, err)

				return &unlocker.Simple{PrivateKey: pk}
			}(),
			expHex: "01000000010b94a1ef0fb352aa2adc54207ce47ba55d5a1c1609afda58fe9520e472299107000000006a473044022049ee0c0f26c00e6a6b3af5990fc8296c66eab3e3e42ab075069b89b1be6fefec02206079e49dd8c9e1117ef06fbe99714d822620b1f0f5d19f32a1128f5d29b7c3c4412102c8803fdd437d902f08e3c2344cb33065c99d7c99982018ff9f7219c3dd352ff0ffffffff01a0083d00000000001976a914af2590a45ae401651fdbdf59a76ad43d1862534088ac00000000",
		},
		"no unlocker errors": {
			inputIdx: 0,
			shf:      sighash.AllForkID,
			expErr:   bt.ErrNoUnlocker,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			tx := bt.NewTx()
			require.NoError(t, tx.From(
				"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
				0,
				"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
				4000000,
			))
			require.NoError(t, tx.ChangeToAddress("mwV3YgnowbJJB3LcyCuqiKpdivvNNFiK7M", FQPoint5SatPerByte))

			err := tx.FillInput(context.Background(), test.unlocker, bt.UnlockerParams{
				InputIdx:     test.inputIdx,
				SigHashFlags: test.shf,
			})
			if test.expErr != nil {
				require.Error(t, err)
				require.EqualError(t, err, test.expErr.Error())
			} else {
				require.NoError(t, err)
				assert.Equal(t, test.expHex, tx.String())
			}
		})
	}
}

func TestTx_FillAllInputs(t *testing.T) {
	t.Parallel()

	t.Run("valid tx (basic)", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)

		err := tx.From(
			"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
			0,
			"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
			4000000)
		require.NoError(t, err)

		err = tx.ChangeToAddress("mwV3YgnowbJJB3LcyCuqiKpdivvNNFiK7M", FQPoint5SatPerByte)
		require.NoError(t, err)

		pk, err := primitives.PrivateKeyFromWif("L3MhnEn1pLWcggeYLk9jdkvA2wUK1iWwwrGkBbgQRqv6HPCdRxuw")
		require.NoError(t, err)
		assert.NotNil(t, pk)

		rawTxBefore := tx.String()

		require.NoError(t, tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk}))

		assert.NotEqual(t, rawTxBefore, tx.String())
	})

	t.Run("no input or output", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)

		rawTxBefore := tx.String()

		require.NoError(t, tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: nil}))

		assert.Equal(t, rawTxBefore, tx.String())
	})
}
