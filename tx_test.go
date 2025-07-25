package bt_test

import (
	"bufio"
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"math/big"
	"reflect"
	"testing"

	bec "github.com/bsv-blockchain/go-sdk/primitives/ec"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/bsv-blockchain/go-bt/v2/testing/data"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var FQPoint5SatPerByte = bt.NewFeeQuote().
	AddQuote(bt.FeeTypeStandard, &bt.Fee{
		FeeType: bt.FeeTypeStandard,
		MiningFee: bt.FeeUnit{
			Satoshis: 5,
			Bytes:    10,
		},
		RelayFee: bt.FeeUnit{
			Satoshis: 5,
			Bytes:    10,
		},
	}).AddQuote(bt.FeeTypeData, &bt.Fee{
	FeeType: bt.FeeTypeData,
	MiningFee: bt.FeeUnit{
		Satoshis: 5,
		Bytes:    10,
	},
	RelayFee: bt.FeeUnit{
		Satoshis: 5,
		Bytes:    10,
	},
})

// mustDecodeWIF is a helper function to decode a WIF string and return the WIF object.
func mustDecodeWIF(t testing.TB, wifStr string) *bec.PrivateKey {
	pk, err := bec.PrivateKeyFromWif(wifStr)
	require.NoError(t, err)
	require.NotNil(t, pk)
	return pk
}

// mustFillAllInputs is a helper function to fill all inputs with a transaction using the provided WIF.
func mustFillAllInputs(t testing.TB, tx *bt.Tx, pk *bec.PrivateKey) {
	require.NoError(t, tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk}))
}

// mustPayToAddress is a helper function to pay a specified amount of satoshis to a given address in a transaction.
func mustPayToAddress(t testing.TB, tx *bt.Tx, address string, satoshis uint64) {
	require.NoError(t, tx.PayToAddress(address, satoshis))
}

// mustFrom is a helper function to add an input to a transaction from a previous transaction ID and output index.
func mustFrom(t testing.TB, tx *bt.Tx, txID string, vout uint32, script string, satoshis uint64) {
	require.NoError(t, tx.From(txID, vout, script, satoshis))
}

// TestNewTx tests the creation of a new transaction using the bt package.
func TestNewTx(t *testing.T) {
	t.Parallel()

	t.Run("new tx, defaults", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		assert.IsType(t, &bt.Tx{}, tx)
		assert.Equal(t, uint32(1), tx.Version)
		assert.Equal(t, uint32(0), tx.LockTime)
		assert.Equal(t, 0, tx.InputCount())
		assert.Equal(t, 0, tx.OutputCount())
		assert.Equal(t, uint64(0), tx.TotalOutputSatoshis())
		assert.Equal(t, uint64(0), tx.TotalInputSatoshis())
	})
}

// TestNewTxFromString tests the creation of a new transaction from a hex string.
func TestNewTxFromString(t *testing.T) {
	t.Parallel()

	t.Run("valid tx no Inputs", func(t *testing.T) {
		tx, err := bt.NewTxFromString("01000000000100000000000000001a006a07707265666978310c6578616d706c65206461746102133700000000")
		require.NoError(t, err)
		assert.NotNil(t, tx)
	})

	t.Run("invalid tx", func(t *testing.T) {
		tx, err := bt.NewTxFromString("0")
		require.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("invalid tx - too short", func(t *testing.T) {
		tx, err := bt.NewTxFromString("000000")
		require.Error(t, err)
		assert.Nil(t, tx)
	})

	t.Run("valid tx, 1 input, 1 output", func(t *testing.T) {
		rawTx := "02000000011ccba787d421b98904da3329b2c7336f368b62e89bc896019b5eadaa28145b9c000000004847304402205cc711985ce2a6d61eece4f9b6edd6337bad3b7eca3aa3ce59bc15620d8de2a80220410c92c48a226ba7d5a9a01105524097f673f31320d46c3b61d2378e6f05320041ffffffff01c0aff629010000001976a91418392a59fc1f76ad6a3c7ffcea20cfcb17bda9eb88ac00000000"
		tx, err := bt.NewTxFromString(rawTx)
		require.NoError(t, err)
		assert.NotNil(t, tx)

		// Check a version, locktime, Inputs
		assert.Equal(t, uint32(2), tx.Version)
		assert.Equal(t, uint32(0), tx.LockTime)
		assert.Len(t, tx.Inputs, 1)

		// Create a new unlocking script
		// ptid, _ := hex.DecodeString("9c5b1428aaad5e9b0196c89be8628b366f33c7b22933da0489b921d487a7cb1c")
		i := &bt.Input{
			PreviousTxOutIndex: 0,
			SequenceNumber:     bt.DefaultSequenceNumber,
		}
		require.NoError(t, i.PreviousTxIDAdd(tx.InputIdx(0).PreviousTxIDChainHash()))
		i.UnlockingScript, err = bscript.NewFromHexString("47304402205cc711985ce2a6d61eece4f9b6edd6337bad3b7eca3aa3ce59bc15620d8de2a80220410c92c48a226ba7d5a9a01105524097f673f31320d46c3b61d2378e6f05320041")
		require.NoError(t, err)
		assert.NotNil(t, i.UnlockingScript)

		// Check an input type
		assert.Equal(t, tx.InputIdx(0), i)

		// Check output
		assert.Len(t, tx.Outputs, 1)

		// New output
		var lscript *bscript.Script
		lscript, err = bscript.NewFromHexString("76a91418392a59fc1f76ad6a3c7ffcea20cfcb17bda9eb88ac")
		require.NoError(t, err)
		assert.NotNil(t, lscript)

		// Check the type
		o := bt.Output{Satoshis: 4999000000, LockingScript: lscript}
		assert.True(t, reflect.DeepEqual(*tx.Outputs[0], o))
	})
}

// TestNewTxFromBytes tests the creation of a new transaction from a byte slice.
func TestNewTxFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("valid tx", func(t *testing.T) {
		rawTx := "02000000011ccba787d421b98904da3329b2c7336f368b62e89bc896019b5eadaa28145b9c0000000049483045022100c4df63202a9aa2bea5c24ebf4418d145e81712072ef744a4b108174f1ef59218022006eb54cf904707b51625f521f8ed2226f7d34b62492ebe4ddcb1c639caf16c3c41ffffffff0140420f00000000001976a91418392a59fc1f76ad6a3c7ffcea20cfcb17bda9eb88ac00000000"
		b, err := hex.DecodeString(rawTx)
		require.NoError(t, err)

		var tx *bt.Tx
		tx, err = bt.NewTxFromBytes(b)
		require.NoError(t, err)
		assert.NotNil(t, tx)
	})

	t.Run("invalid tx, too short", func(t *testing.T) {
		rawTx := "000000"
		b, err := hex.DecodeString(rawTx)
		require.NoError(t, err)

		var tx *bt.Tx
		tx, err = bt.NewTxFromBytes(b)
		require.Error(t, err)
		assert.Nil(t, tx)
	})
}

// TestTxTxID tests the transaction ID generation for a transaction.
func TestTxTxID(t *testing.T) {
	t.Parallel()

	t.Run("valid tx id", func(t *testing.T) {
		tx, err := bt.NewTxFromString("010000000193a35408b6068499e0d5abd799d3e827d9bfe70c9b75ebe209c91d2507232651000000006b483045022100c1d77036dc6cd1f3fa1214b0688391ab7f7a16cd31ea4e5a1f7a415ef167df820220751aced6d24649fa235132f1e6969e163b9400f80043a72879237dab4a1190ad412103b8b40a84123121d260f5c109bc5a46ec819c2e4002e5ba08638783bfb4e01435ffffffff02404b4c00000000001976a91404ff367be719efa79d76e4416ffb072cd53b208888acde94a905000000001976a91404d03f746652cfcb6cb55119ab473a045137d26588ac00000000")
		require.NoError(t, err)
		assert.NotNil(t, tx)
		assert.Equal(t, "19dcf16ecc9286c3734fdae3d45d4fc4eb6b25f841131e06460f4939bba0026e", tx.TxID())
	})

	t.Run("new tx, no data, but has default tx id", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		assert.Equal(t, "d21633ba23f70118185227be58a63527675641ad37967e2aa461559f577aec43", tx.TxID())
	})
}

// TestVersion tests the version field of a transaction.
func TestVersion(t *testing.T) {
	t.Parallel()

	rawTx := "01000000014c6ec863cf3e0284b407a1a1b8138c76f98280812cb9653231f385a0305fc76f010000006b483045022100f01c1a1679c9437398d691c8497f278fa2d615efc05115688bf2c3335b45c88602201b54437e54fb53bc50545de44ea8c64e9e583952771fcc663c8687dc2638f7854121037e87bbd3b680748a74372640628a8f32d3a841ceeef6f75626ab030c1a04824fffffffff021d784500000000001976a914e9b62e25d4c6f97287dfe62f8063b79a9638c84688ac60d64f00000000001976a914bb4bca2306df66d72c6e44a470873484d8808b8888ac00000000"
	tx, err := bt.NewTxFromString(rawTx)
	require.NoError(t, err)
	assert.NotNil(t, tx)
	assert.Equal(t, uint32(1), tx.Version)
}

// TestTxIsCoinbase tests the IsCoinbase method of a transaction.
func TestTxIsCoinbase(t *testing.T) {
	t.Parallel()

	t.Run("invalid number of Inputs", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)
		assert.False(t, tx.IsCoinbase())
	})

	t.Run("valid coinbase tx, 1 input", func(t *testing.T) {
		rawTx := "02000000010000000000000000000000000000000000000000000000000000000000000000ffffffff0e5101010a2f4542323030302e302fffffffff0100f2052a01000000232103db233bb9fc387d78b133ec904069d46e95ff17da657671b44afa0bc64e89ac18ac00000000"
		tx, err := bt.NewTxFromString(rawTx)
		require.NoError(t, err)
		assert.NotNil(t, tx)

		assert.True(t, tx.IsCoinbase())
		assert.Equal(t, 1, tx.InputCount())
	})

	t.Run("valid coinbase tx", func(t *testing.T) {
		coinbaseTx := "01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4303bfea07322f53696d6f6e204f726469736820616e642053747561727420467265656d616e206d61646520746869732068617070656e2f9a46434790f7dbdea3430000ffffffff018a08ac4a000000001976a9148bf10d323ac757268eb715e613cb8e8e1d1793aa88ac00000000"
		tx, err := bt.NewTxFromString(coinbaseTx)
		require.NoError(t, err)
		assert.NotNil(t, tx)
		assert.True(t, tx.IsCoinbase())
	})

	t.Run("tx is not a coinbase tx", func(t *testing.T) {
		coinbaseTx := "01000000014c6ec863cf3e0284b407a1a1b8138c76f98280812cb9653231f385a0305fc76f010000006b483045022100f01c1a1679c9437398d691c8497f278fa2d615efc05115688bf2c3335b45c88602201b54437e54fb53bc50545de44ea8c64e9e583952771fcc663c8687dc2638f7854121037e87bbd3b680748a74372640628a8f32d3a841ceeef6f75626ab030c1a04824fffffffff021d784500000000001976a914e9b62e25d4c6f97287dfe62f8063b79a9638c84688ac60d64f00000000001976a914bb4bca2306df66d72c6e44a470873484d8808b8888ac00000000"
		tx, err := bt.NewTxFromString(coinbaseTx)
		require.NoError(t, err)
		assert.NotNil(t, tx)
		assert.False(t, tx.IsCoinbase())
	})

	t.Run("tx (2) is not a coinbase tx", func(t *testing.T) {
		coinbaseTx := "010000000159ef0cbb7881f2c934d6fb669f68f7c6a9c632f997152f828d1153806b7ac82b010000006b483045022100e775a21994cc6d6d6bf79d295aeea592e7b4cf8d8ecddaf67bb6626d7af82fd302201921a313de67e23a78c81dd5fe9a19322839c0ea1034b9c54e8206dea3aa9e68412103d1c02ee3522ff58df6c6287e67202a797b562fa8b5a9ed86613fe5ee48fb8821ffffffff02000000000000000011006a0e6d657461737472656d652e636f6dc9990200000000001976a914fa1b02ff7e41975d698fec6fb1b2d7e4656f8e7f88ac00000000"
		tx, err := bt.NewTxFromString(coinbaseTx)
		require.NoError(t, err)
		assert.NotNil(t, tx)
		assert.False(t, tx.IsCoinbase())
	})
}

// TestTxCreateTx tests the creation of a transaction with inputs and outputs.
func TestTxCreateTx(t *testing.T) {
	t.Parallel()

	tx := bt.NewTx()
	assert.NotNil(t, tx)

	mustFrom(t, tx,
		"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
		0,
		"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
		2000000,
	)

	mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1999942)

	pk := mustDecodeWIF(t, "KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
	mustFillAllInputs(t, tx, pk)
}

// TestTxHasDataOutputs tests whether a transaction has data outputs.
func TestTxHasDataOutputs(t *testing.T) {
	t.Parallel()

	t.Run("has data Outputs", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)

		mustFrom(t, tx,
			"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
			0,
			"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
			2000000,
		)

		mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1999942)

		// Add op return data
		type OpReturnData [][]byte
		ops := OpReturnData{[]byte("prefix1"), []byte("example data"), []byte{0x13, 0x37}}

		err := tx.AddOpReturnPartsOutput(ops)
		require.NoError(t, err)

		pk := mustDecodeWIF(t, "KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
		mustFillAllInputs(t, tx, pk)

		assert.True(t, tx.HasDataOutputs())
	})

	t.Run("no data Outputs", func(t *testing.T) {
		tx := bt.NewTx()
		assert.NotNil(t, tx)

		mustFrom(t, tx,
			"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
			0,
			"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
			2000000,
		)

		mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1999942)

		pk := mustDecodeWIF(t, "KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
		mustFillAllInputs(t, tx, pk)

		assert.False(t, tx.HasDataOutputs())
	})
}

// TestTxOutputIdx tests the OutputIdx method of a transaction.
func TestTxOutputIdx(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		tx        *bt.Tx
		idx       int
		expOutput *bt.Output
	}{
		"tx with 3 Outputs and output idx 0 requested should return output": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				mustPayToAddress(t, tx, "myUmQeCYxQECGHXbupe539n41u6BTBz1Eh", 1000)
				mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				return tx
			}(),
			idx: 0,
			expOutput: &bt.Output{
				Satoshis: 1000,
				LockingScript: func() *bscript.Script {
					s, err := bscript.NewP2PKHFromAddress("myUmQeCYxQECGHXbupe539n41u6BTBz1Eh")
					require.NoError(t, err)
					return s
				}(),
			},
		}, "tx with 3 Outputs and output idx 2 requested should return output": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				mustPayToAddress(t, tx, "myUmQeCYxQECGHXbupe539n41u6BTBz1Eh", 1000)
				mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				mustPayToAddress(t, tx, "mywmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				return tx
			}(),
			idx: 2,
			expOutput: &bt.Output{
				Satoshis: 1000,
				LockingScript: func() *bscript.Script {
					s, err := bscript.NewP2PKHFromAddress("mywmGVP89x3DsLNqk3NvctfQy9m9pvt7mz")
					require.NoError(t, err)
					return s
				}(),
			},
		}, "tx with 3 Outputs and output idx 5 requested should return nil": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				mustPayToAddress(t, tx, "myUmQeCYxQECGHXbupe539n41u6BTBz1Eh", 1000)
				mustPayToAddress(t, tx, "n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				mustPayToAddress(t, tx, "mywmGVP89x3DsLNqk3NvctfQy9m9pvt7mz", 1000)
				return tx
			}(),
			idx:       5,
			expOutput: nil,
		}, "tx with 0 Outputs and output idx 5 requested should return nil": {
			tx: func() *bt.Tx {
				return bt.NewTx()
			}(),
			idx:       5,
			expOutput: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			o := test.tx.OutputIdx(test.idx)
			assert.Equal(t, test.expOutput, o)
		})
	}
}

// TestTxInputIdx tests the InputIdx method of a transaction.
func TestTxInputIdx(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		tx       *bt.Tx
		idx      int
		expInput *bt.Input
	}{
		"tx with 3 Inputs and input idx 0 requested should return correct input": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					1000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				return tx
			}(),
			idx: 0,
			expInput: func() *bt.Input {
				in := &bt.Input{
					PreviousTxSatoshis: 1000,
					PreviousTxScript: func() *bscript.Script {
						b, err := bscript.NewFromHexString("76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac")
						require.NoError(t, err)
						return b
					}(),
					PreviousTxOutIndex: 0,
					SequenceNumber:     bt.DefaultSequenceNumber,
				}
				_ = in.PreviousTxIDAddStr("3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5")
				return in
			}(),
		}, "tx with 3 Outputs and output idx 2 requested should return output": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					1000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdac4",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					999,
				))
				return tx
			}(),
			idx: 2,
			expInput: func() *bt.Input {
				in := &bt.Input{
					PreviousTxSatoshis: 999,
					PreviousTxScript: func() *bscript.Script {
						b, err := bscript.NewFromHexString("76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac")
						require.NoError(t, err)
						return b
					}(),
					PreviousTxOutIndex: 0,
					SequenceNumber:     bt.DefaultSequenceNumber,
				}
				_ = in.PreviousTxIDAddStr("3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdac4")
				return in
			}(),
		}, "tx with 3 Outputs and output idx 5 requested should return nil": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					1000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					999,
				))
				return tx
			}(),
			idx:      5,
			expInput: nil,
		}, "tx with 0 Outputs and output idx 5 requested should return nil": {
			tx: func() *bt.Tx {
				return bt.NewTx()
			}(),
			idx:      5,
			expInput: nil,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			o := test.tx.InputIdx(test.idx)
			assert.Equal(t, test.expInput, o)
		})
	}
}

// TestIsValidTxID tests the IsValidTxID function for validating transaction IDs.
func TestIsValidTxID(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		txid string
		exp  bool
	}{
		"valid txID should return true": {
			txid: "a2a55ecc61f418e300888b1f82eaf84024496b34e3e538f3d32d342fd753adab",
			exp:  true,
		},
		"invalid txID should return false": {
			txid: "a2a55ecc61f418e300888b1f82eaf84024496b34e3e538f3d32d342fd753adZZ",
			exp:  false,
		}, "empty txID should return false": {
			txid: "",
			exp:  true, // this is because the empty string gets converted to all 0's
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bb, _ := chainhash.NewHashFromStr(test.txid)
			assert.Equal(t, test.exp, bt.IsValidTxID(bb))
		})
	}
}

// TestTxClone tests the Clone method of a transaction.
func TestTxClone(t *testing.T) {
	t.Parallel()

	tx, err := bt.NewTxFromString("0200000003a9bc457fdc6a54d99300fb137b23714d860c350a9d19ff0f571e694a419ff3a0010000006b48304502210086c83beb2b2663e4709a583d261d75be538aedcafa7766bd983e5c8db2f8b2fc02201a88b178624ab0ad1748b37c875f885930166237c88f5af78ee4e61d337f935f412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff0092bb9a47e27bf64fc98f557c530c04d9ac25e2f2a8b600e92a0b1ae7c89c20010000006b483045022100f06b3db1c0a11af348401f9cebe10ae2659d6e766a9dcd9e3a04690ba10a160f02203f7fbd7dfcfc70863aface1a306fcc91bbadf6bc884c21a55ef0d32bd6b088c8412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff9d0d4554fa692420a0830ca614b6c60f1bf8eaaa21afca4aa8c99fb052d9f398000000006b483045022100d920f2290548e92a6235f8b2513b7f693a64a0d3fa699f81a034f4b4608ff82f0220767d7d98025aff3c7bd5f2a66aab6a824f5990392e6489aae1e1ae3472d8dffb412103e8be830d98bb3b007a0343ee5c36daa48796ae8bb57946b1e87378ad6e8a090dfeffffff02807c814a000000001976a9143a6bf34ebfcf30e8541bbb33a7882845e5a29cb488ac76b0e60e000000001976a914bd492b67f90cb85918494767ebb23102c4f06b7088ac67000000")
	require.NoError(t, err)

	for i, ipt := range tx.Inputs {
		n, _ := rand.Int(rand.Reader, big.NewInt(0).Lsh(big.NewInt(1), 63))
		ipt.PreviousTxSatoshis = n.Uint64()

		// ipt.PreviousTxSatoshis = rand.Uint64()
		var script *bscript.Script
		script, err = bscript.NewFromASM(fmt.Sprintf("OP_%d OP_IF OP_ENDIF", i+1))
		require.NoError(t, err)

		ipt.PreviousTxScript = script
	}

	t.Run("all fields cloned", func(t *testing.T) {
		clone := tx.Clone()
		assert.Equal(t, tx.TxID(), clone.TxID())
		assert.Equal(t, tx.Bytes(), clone.Bytes())
		assert.Equal(t, tx.Version, clone.Version)
		assert.Equal(t, tx.LockTime, clone.LockTime)

		assert.Equal(t, tx.InputCount(), clone.InputCount())
		for i, input := range tx.Inputs {
			cloneInput := clone.InputIdx(i)
			assert.NotEqual(t, fmt.Sprintf("%p", input), fmt.Sprintf("%p", cloneInput))
			assert.Equal(t, input.Bytes(true), cloneInput.Bytes(true))
			assert.Equal(t, input.PreviousTxID(), cloneInput.PreviousTxID())
			assert.Equal(t, input.SequenceNumber, cloneInput.SequenceNumber)
			assert.Equal(t, input.PreviousTxOutIndex, cloneInput.PreviousTxOutIndex)
			assert.Equal(t, *input.UnlockingScript, *cloneInput.UnlockingScript)
			assert.NotEqual(t, fmt.Sprintf("%p", input.UnlockingScript), fmt.Sprintf("%p", cloneInput.UnlockingScript))
			assert.Equal(t, input.PreviousTxSatoshis, cloneInput.PreviousTxSatoshis)
			assert.Equal(t, *input.PreviousTxScript, *cloneInput.PreviousTxScript)
			assert.NotEqual(t, fmt.Sprintf("%p", input.PreviousTxScript), fmt.Sprintf("%p", cloneInput.PreviousTxScript))
		}

		assert.Equal(t, tx.OutputCount(), clone.OutputCount())
		for i, output := range tx.Outputs {
			cloneOutput := clone.OutputIdx(i)
			assert.NotEqual(t, fmt.Sprintf("%p", output), fmt.Sprintf("%p", cloneOutput))
			assert.Equal(t, output.Bytes(), cloneOutput.Bytes())
			assert.Equal(t, output.BytesForSigHash(), cloneOutput.BytesForSigHash())
			assert.Equal(t, *output.LockingScript, *cloneOutput.LockingScript)
			assert.NotEqual(t, fmt.Sprintf("%p", output.LockingScript), fmt.Sprintf("%p", cloneOutput.LockingScript))
			assert.Equal(t, output.Satoshis, cloneOutput.Satoshis)
		}
	})
}

// TestEstimateIsFeePaidEnough tests the IsFeePaidEnough method of a transaction.
func TestEstimateIsFeePaidEnough(t *testing.T) {
	tests := map[string]struct {
		tx         *bt.Tx
		dataLength uint64
		expSize    *bt.TxSize
		isEnough   bool
	}{
		"unsigned transaction (1 input 1 P2PKHOutput + no change) paying less by 1 satoshi": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 905))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    85,
				TotalStdBytes: 85,
			},
			isEnough: false,
		}, "unsigned transaction (1 input 1 P2PKHOutput + change) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834709,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: true,
		}, "unsigned transaction (0 input 1 P2PKHOutput) should not pay": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     44,
				TotalStdBytes:  44,
				TotalDataBytes: 0,
			},
			isEnough: false,
		}, "unsigned transaction (1 input 2 P2PKHOutputs) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834763,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 578091))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: true,
		}, "unsigned transaction (1 input 2 P2PKHOutputs) should fail paying less by 1 sat": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834763,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256560))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 578091))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: false,
		}, "226B signed transaction (1 input 1 P2PKHOutput + change) no data should return 113 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 834709,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    226,
				TotalStdBytes: 226,
			},
			isEnough: true,
		}, "192B signed transaction (1 input 1 P2PKHOutput + no change) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				require.NoError(t, tx.From(
					"a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000,
				))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 904))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    192,
				TotalStdBytes: 192,
			},
			isEnough: true,
		}, "214B signed transaction (1 input, 1 change output, 1 opreturn) should pay exact amount": {
			tx: func() *bt.Tx {
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"160f06232540dcb0e9b6db9b36a27f01da1e7e473989df67859742cf098d498f",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000,
				))
				require.NoError(t, tx.AddOpReturnOutput([]byte("hellohello")))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 89))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     214,
				TotalStdBytes:  201,
				TotalDataBytes: 13,
			},
			isEnough: true,
		}, "214B signed transaction (1 input, 1 change output, 1 opreturn) should fail paying less by 1 sat": {
			tx: func() *bt.Tx {
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"160f06232540dcb0e9b6db9b36a27f01da1e7e473989df67859742cf098d498f",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000,
				))
				require.NoError(t, tx.AddOpReturnOutput([]byte("hellohello")))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 895))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     213,
				TotalStdBytes:  200,
				TotalDataBytes: 13,
			},
			isEnough: false,
		},
		// TODO: add tests for different fee type values
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			isEnough, err := test.tx.EstimateIsFeePaidEnough(FQPoint5SatPerByte)
			require.NoError(t, err)
			assert.Equal(t, test.isEnough, isEnough)

			swt := test.tx.SizeWithTypes()
			assert.Equal(t, test.expSize, swt)
		})
	}
}

// TestIsFeePaidEnough tests the IsFeePaidEnough method of a transaction.
func TestIsFeePaidEnough(t *testing.T) {
	tests := map[string]struct {
		tx         *bt.Tx
		dataLength uint64
		expSize    *bt.TxSize
		isEnough   bool
	}{
		"unsigned transaction (1 input 1 P2PKHOutput + no change) paying less by 1 satoshi": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 959))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    85,
				TotalStdBytes: 85,
			},
			isEnough: false,
		}, "unsigned transaction (1 input 1 P2PKHOutput + change) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834709))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: true,
		}, "unsigned transaction (0 input 1 P2PKHOutput) should not pay": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     44,
				TotalStdBytes:  44,
				TotalDataBytes: 0,
			},
			isEnough: false,
		}, "unsigned transaction (1 input 2 P2PKHOutputs) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834709))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 578091))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: true,
		}, "unsigned transaction (1 input 2 P2PKHOutputs) should fail paying less by 1 sat": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 834709))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256560))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 578091))
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     119,
				TotalStdBytes:  119,
				TotalDataBytes: 0,
			},
			isEnough: false,
		}, "226B signed transaction (1 input 1 P2PKHOutput + change) no data should return 113 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 834709))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 256559))
				require.NoError(t, tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    226,
				TotalStdBytes: 226,
			},
			isEnough: true,
		}, "192B signed transaction (1 input 1 P2PKHOutput + no change) should pay exact amount": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 904))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:    192,
				TotalStdBytes: 192,
			},
			isEnough: true,
		}, "214B signed transaction (1 input, 1 change output, 1 opreturn) should pay exact amount": {
			tx: func() *bt.Tx {
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				tx := bt.NewTx()
				require.NoError(t, tx.From("160f06232540dcb0e9b6db9b36a27f01da1e7e473989df67859742cf098d498f",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000))
				require.NoError(t, tx.AddOpReturnOutput([]byte("hellohello")))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 893))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     213,
				TotalStdBytes:  200,
				TotalDataBytes: 13,
			},
			isEnough: true,
		}, "214B signed transaction (1 input, 1 change output, 1 opreturn) should fail paying less by 1 sat": {
			tx: func() *bt.Tx {
				pk := mustDecodeWIF(t, "cRhdUmZx4MbsjxVxGH4bM4geNLzQEPxspnhGtDCvMmfCLcED8Q6G")
				tx := bt.NewTx()
				require.NoError(t, tx.From("160f06232540dcb0e9b6db9b36a27f01da1e7e473989df67859742cf098d498f",
					0, "76a914ff8c9344d4e76c0580420142f697e5fc2ce5c98e88ac", 1000))
				require.NoError(t, tx.AddOpReturnOutput([]byte("hellohello")))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 895))
				mustFillAllInputs(t, tx, pk)
				return tx
			}(),
			expSize: &bt.TxSize{
				TotalBytes:     213,
				TotalStdBytes:  200,
				TotalDataBytes: 13,
			},
			isEnough: false,
		},
		// TODO: add tests for different fee type values
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			isEnough, err := test.tx.IsFeePaidEnough(FQPoint5SatPerByte)
			require.NoError(t, err)
			assert.Equal(t, test.isEnough, isEnough)

			swt := test.tx.SizeWithTypes()
			assert.Equal(t, test.expSize, swt)
		})
	}
}

// TestEstimateFeesPaid tests the EstimateFeesPaid method of a transaction.
func TestEstimateFeesPaid(t *testing.T) {
	tests := map[string]struct {
		tx         *bt.Tx
		dataLength uint64
		expFees    *bt.TxFees
		expSize    *bt.TxSize
	}{
		"226B transaction (1 input 1 P2PKHOutput + no change) no data should return 113 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 96,
				StdFeePaid:   96,
				DataFeePaid:  0,
			},
			expSize: &bt.TxSize{
				TotalBytes:    192,
				TotalStdBytes: 192,
			},
		}, "226B transaction (1 input 1 P2PKHOutput + change) no data should return 113 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))

				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				require.NoError(t, tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte))
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 113,
				StdFeePaid:   113,
				DataFeePaid:  0,
			},
			expSize: &bt.TxSize{
				TotalBytes:     226,
				TotalStdBytes:  226,
				TotalDataBytes: 0,
			},
		}, "214B unsigned transaction (1 input, 1 opreturn, no change) 10 byte of data should return 100 sats fee 6 data fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))
				require.NoError(t, tx.AddOpReturnOutput([]byte("hellohello")))
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 89,
				StdFeePaid:   83,
				DataFeePaid:  6,
			},
			expSize: &bt.TxSize{
				TotalBytes:     180,
				TotalStdBytes:  167,
				TotalDataBytes: 13,
			},
		}, "556B unsigned transaction (3 inputs + 2 outputs + no change) no data should return 261 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				err := tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000)
				require.NoError(t, err)
				err = tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000)
				require.NoError(t, err)
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 261,
				StdFeePaid:   261,
				DataFeePaid:  0,
			},
			expSize: &bt.TxSize{
				TotalBytes:     522,
				TotalStdBytes:  522,
				TotalDataBytes: 0,
			},
		}, "556B unsigned transaction (3 inputs + 2 outputs + 1 change) no data should return 278 sats fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				err := tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000)
				require.NoError(t, err)
				err = tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000)
				require.NoError(t, err)
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				err = tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte)
				require.NoError(t, err)
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 278,
				StdFeePaid:   278,
				DataFeePaid:  0,
			},
			expSize: &bt.TxSize{
				TotalBytes:     556,
				TotalStdBytes:  556,
				TotalDataBytes: 0,
			},
		}, "565B unsigned transaction 100B data should return 63 sats std fee, 50 data fee": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 100))
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 100))
				require.NoError(t, tx.From("a4c76f8a7c05a91dcf5699b95b54e856298e50c1ceca9a8a5569c8532c500c11",
					0, "76a91455b61be43392125d127f1780fb038437cd67ef9c88ac", 1000))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				require.NoError(t, tx.AddP2PKHOutputFromAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", 100))
				require.NoError(t, tx.AddOpReturnOutput(make([]byte, 0x64)))
				err := tx.ChangeToAddress("mtestD3vRB7AoYWK2n6kLdZmAMLbLhDsLr", FQPoint5SatPerByte)
				require.NoError(t, err)
				return tx
			}(),
			expFees: &bt.TxFees{
				TotalFeePaid: 334,
				StdFeePaid:   282,
				DataFeePaid:  52,
			},
			expSize: &bt.TxSize{
				TotalBytes:     669,
				TotalStdBytes:  565,
				TotalDataBytes: 104,
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			resp, err := test.tx.EstimateFeesPaid(FQPoint5SatPerByte)
			require.NoError(t, err)
			assert.Equal(t, test.expFees, resp)

			swt, err := test.tx.EstimateSizeWithTypes()
			require.NoError(t, err)
			assert.Equal(t, test.expSize, swt)
		})
	}
}

// TestTxEstimateFeesPaidTotal tests the EstimateFeesPaidTotal method of a transaction.
func TestTxEstimateFeesPaidTotal(t *testing.T) {
	tests := map[string]struct {
		tx      *bt.Tx
		fees    *bt.FeeQuote
		expFees uint64
		err     error
	}{
		"Transaction with one input one output should return 96": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
					0,
					"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
					1000))
				mustPayToAddress(t, tx, "mxAoAyZFXX6LZBWhoam3vjm6xt9NxPQ15f", 500)
				return tx
			}(),
			fees: func() *bt.FeeQuote {
				std := &bt.Fee{
					FeeType: bt.FeeTypeStandard,
					MiningFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
					RelayFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
				}
				dataVal := &bt.Fee{
					FeeType: bt.FeeTypeData,
					MiningFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
					RelayFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
				}
				return bt.NewFeeQuote().
					AddQuote(bt.FeeTypeStandard, std).
					AddQuote(bt.FeeTypeData, dataVal)
			}(),
			expFees: 96,
		}, "Transaction with one input 4 Outputs should return 147": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"07912972e42095fe58daaf09161c5a5da57be47c2054dc2aaa52b30fefa1940b",
					0,
					"76a914af2590a45ae401651fdbdf59a76ad43d1862534088ac",
					2500,
				))
				mustPayToAddress(t, tx, "mxAoAyZFXX6LZBWhoam3vjm6xt9NxPQ15f", 500)
				mustPayToAddress(t, tx, "mxAoAyZFXX6LZBWhoam3vjm6xt9NxPQ15f", 500)
				mustPayToAddress(t, tx, "mxAoAyZFXX6LZBWhoam3vjm6xt9NxPQ15f", 500)
				mustPayToAddress(t, tx, "mxAoAyZFXX6LZBWhoam3vjm6xt9NxPQ15f", 500)
				return tx
			}(),
			fees: func() *bt.FeeQuote {
				std := &bt.Fee{
					FeeType: bt.FeeTypeStandard,
					MiningFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
					RelayFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
				}
				dataVal := &bt.Fee{
					FeeType: bt.FeeTypeData,
					MiningFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
					RelayFee: bt.FeeUnit{
						Satoshis: 5,
						Bytes:    10,
					},
				}
				return bt.NewFeeQuote().
					AddQuote(bt.FeeTypeStandard, std).
					AddQuote(bt.FeeTypeData, dataVal)
			}(),
			expFees: 147,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fee, err := test.tx.EstimateFeesPaid(test.fees)
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.expFees, fee.TotalFeePaid)
		})
	}
}

// This test reads a sample block from a file, but normally
// we would get the block directly from the node via a REST GET...
/*
	resp, err := http.Get(fmt.Sprintf("%s/rest/block/%s.bin", b.client.serverAddr, blockHash))
	if err != nil {
		return nil, fmt.Errorf("Could not GET block: %v", err)
	}
	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read response body: %w", err)
		}
		return nil, fmt.Errorf("ERROR: code %d: %s", resp.StatusCode, data)
	}
	Then use resp.Body as in the test below
*/

// TestTxReadFrom reads a transaction from a binary file and checks its ID and size.
func TestTxReadFrom(t *testing.T) {
	f, err := data.TxBinData.Open("block.bin")
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()
	require.NoError(t, err)

	r := bufio.NewReader(f)

	header := make([]byte, 80)
	_, err = io.ReadFull(f, header)
	require.NoError(t, err)

	var txCount bt.VarInt
	_, err = txCount.ReadFrom(r)
	require.NoError(t, err)
	assert.Equal(t, uint64(648), uint64(txCount))

	tx := new(bt.Tx)
	var bytesRead int64
	for i := uint64(0); i < uint64(txCount); i++ {
		n, err := tx.ReadFrom(r)
		bytesRead += n
		require.NoError(t, err)
	}

	assert.Equal(t, "b7c59d7fa17a74bbe0a05e5381f42b9ac7fe23b8a1ca40005a74802fe5b8bb5a", tx.TxID())
	assert.Equal(t, int64(340216), bytesRead)
}

// TestTxReadFrom reads a transaction from a binary file and checks its ID and size.
func TestTxsReadFrom(t *testing.T) {
	f, err := data.TxBinData.Open("block.bin")
	defer func() {
		if f != nil {
			_ = f.Close()
		}
	}()
	require.NoError(t, err)

	r := bufio.NewReader(f)

	header := make([]byte, 80)
	_, err = io.ReadFull(f, header)
	require.NoError(t, err)

	txs := bt.Txs{}
	bytesRead, err := txs.ReadFrom(r)
	require.NoError(t, err)

	assert.Equal(t, "b7c59d7fa17a74bbe0a05e5381f42b9ac7fe23b8a1ca40005a74802fe5b8bb5a", txs[len(txs)-1].TxID())
	assert.Equal(t, int64(340219), bytesRead)
}

// TestExtendedFormat tests the ExtendedBytes and NewTxFromBytes methods for transactions.
func TestExtendedFormat(t *testing.T) {
	tx, err := bt.NewTxFromString("0100000001478a4ac0c8e4dae42db983bc720d95ed2099dec4c8c3f2d9eedfbeb74e18cdbb1b0100006b483045022100b05368f9855a28f21d3cb6f3e278752d3c5202f1de927862bbaaf5ef7d67adc50220728d4671cd4c34b1fa28d15d5cd2712b68166ea885522baa35c0b9e399fe9ed74121030d4ad284751daf629af387b1af30e02cf5794139c4e05836b43b1ca376624f7fffffffff01000000000000000070006a0963657274696861736822314c6d763150594d70387339594a556e374d3948565473446b64626155386b514e4a406164386337373536356335363935353261626463636634646362353537376164633936633866613933623332663630373865353664666232326265623766353600000000")
	if err != nil {
		t.Error(err)
		return
	}

	require.Equal(t, "e6adcaf6b86fb5d690a3bade36011cd02f80dd364f1ecf2bb04902aa1b6bf455", tx.TxID())

	tx.Inputs[0].PreviousTxSatoshis = 16
	s, _ := hex.DecodeString("76a9140c77a935b45abdcf3e472606d3bc647c5cc0efee88ac")
	tx.Inputs[0].PreviousTxScript = bscript.NewFromBytes(s)

	tx2, err := bt.NewTxFromBytes(tx.ExtendedBytes())
	if err != nil {
		t.Error(err)
		return
	}

	require.Equal(t, "e6adcaf6b86fb5d690a3bade36011cd02f80dd364f1ecf2bb04902aa1b6bf455", tx2.TxID())

	assert.Equal(t, uint64(16), tx2.Inputs[0].PreviousTxSatoshis)
	assert.Equal(t, s, []byte(*tx2.Inputs[0].PreviousTxScript))
}

// TestFromNodeJS tests if a transaction can be created from a Node.js serialized string.
func TestFromNodeJS(t *testing.T) {
	_, err := bt.NewTxFromString("010000000000000000ef01478a4ac0c8e4dae42db983bc720d95ed2099dec4c8c3f2d9eedfbeb74e18cdbb1b0100006b483045022100b05368f9855a28f21d3cb6f3e278752d3c5202f1de927862bbaaf5ef7d67adc50220728d4671cd4c34b1fa28d15d5cd2712b68166ea885522baa35c0b9e399fe9ed74121030d4ad284751daf629af387b1af30e02cf5794139c4e05836b43b1ca376624f7fffffffff10000000000000001976a9140c77a935b45abdcf3e472606d3bc647c5cc0efee88ac01000000000000000070006a0963657274696861736822314c6d763150594d70387339594a556e374d3948565473446b64626155386b514e4a406164386337373536356335363935353261626463636634646362353537376164633936633866613933623332663630373865353664666232326265623766353600000000")

	require.NoError(t, err)
}

// TestSetTxHash tests if the SetTxHash method correctly sets the transaction hash to zero.
func TestSetTxHash(t *testing.T) {
	tx, err := bt.NewTxFromString("010000000000000000ef01478a4ac0c8e4dae42db983bc720d95ed2099dec4c8c3f2d9eedfbeb74e18cdbb1b0100006b483045022100b05368f9855a28f21d3cb6f3e278752d3c5202f1de927862bbaaf5ef7d67adc50220728d4671cd4c34b1fa28d15d5cd2712b68166ea885522baa35c0b9e399fe9ed74121030d4ad284751daf629af387b1af30e02cf5794139c4e05836b43b1ca376624f7fffffffff10000000000000001976a9140c77a935b45abdcf3e472606d3bc647c5cc0efee88ac01000000000000000070006a0963657274696861736822314c6d763150594d70387339594a556e374d3948565473446b64626155386b514e4a406164386337373536356335363935353261626463636634646362353537376164633936633866613933623332663630373865353664666232326265623766353600000000")
	require.NoError(t, err)

	assert.Equal(t, "e6adcaf6b86fb5d690a3bade36011cd02f80dd364f1ecf2bb04902aa1b6bf455", tx.TxID())

	txHash := chainhash.Hash{}
	tx.SetTxHash(&txHash)

	assert.Equal(t, "0000000000000000000000000000000000000000000000000000000000000000", tx.TxID())
}
