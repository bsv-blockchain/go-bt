package bt_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	primitives "github.com/bsv-blockchain/go-sdk/primitives/ec"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// TestTx_JSON tests the JSON marshaling and unmarshaling of transactions.
func TestTx_JSON(t *testing.T) {
	tests := map[string]struct {
		tx  *bt.Tx
		err error
	}{
		"standard tx should marshal and unmarshal correctly": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				require.NoError(t, tx.PayToAddress("n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1000))
				pk, err := primitives.PrivateKeyFromWif("KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
				require.NoError(t, err)
				assert.NotNil(t, pk)

				err = tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk})
				require.NoError(t, err)
				return tx
			}(),
		}, "data tx should marshall correctly": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					2000000,
				))
				require.NoError(t, tx.PayToAddress("n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1000))
				pk, err := primitives.PrivateKeyFromWif("KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
				require.NoError(t, err)
				assert.NotNil(t, pk)
				s := &bscript.Script{}
				require.NoError(t, s.AppendPushDataString("test"))
				tx.AddOutput(&bt.Output{
					LockingScript: s,
				})
				err = tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk})
				require.NoError(t, err)
				return tx
			}(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			bb, err := json.Marshal(test.tx)
			require.NoError(t, err)
			if err != nil {
				return
			}
			var tx *bt.Tx
			require.NoError(t, json.Unmarshal(bb, &tx))
			assert.Equal(t, test.tx.String(), tx.String())
		})
	}
}

// TestTx_MarshallJSON tests the JSON marshaling of transactions.
func TestTx_MarshallJSON(t *testing.T) {
	tests := map[string]struct {
		tx      *bt.Tx
		expJSON string
	}{
		"transaction with 1 input 1 p2pksh output 1 data output should create valid json": {
			tx: func() *bt.Tx {
				tx, err := bt.NewTxFromString("0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000")
				require.NoError(t, err)
				return tx
			}(),
			expJSON: `{
	"txid": "aec245f27b7640c8b1865045107731bfb848115c573f7da38166074b1c9e475d",
	"hex": "0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000",
	"inputs": [
		{
			"unlockingScript": "4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8",
			"txid": "a2a55ecc61f418e300888b1f82eaf84024496b34e3e538f3d32d342fd753adab",
			"vout": 1,
			"sequence": 4294967295
		}
	],
	"outputs": [
		{
			"satoshis": 0,
			"lockingScript": "006a0548656c6c6f"
		},
		{
			"satoshis": 895,
			"lockingScript": "76a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac"
		}
	],
	"version": 1,
	"lockTime": 0
}`,
		}, "transaction with multiple Inputs": {
			tx: func() *bt.Tx {
				tx := bt.NewTx()
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					0,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					10000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					2,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					10000,
				))
				require.NoError(t, tx.From(
					"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
					114,
					"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac",
					10000,
				))
				require.NoError(t, tx.PayToAddress("n2wmGVP89x3DsLNqk3NvctfQy9m9pvt7mk", 1000))
				pk, err := primitives.PrivateKeyFromWif("KznvCNc6Yf4iztSThoMH6oHWzH9EgjfodKxmeuUGPq5DEX5maspS")
				require.NoError(t, err)
				assert.NotNil(t, pk)
				err = tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: pk})
				require.NoError(t, err)
				return tx
			}(),
			expJSON: `{
	"txid": "41741af6fb64839c69f2385987eb3770c55c42eb6f7900fa2af9d667c42ceb20",
	"hex": "0100000003d5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c000000006b48304502210081214df575da1e9378f1d5a29dfd6811e93466a7222fb010b7c50dd2d44d7f2e0220399bb396336d2e294049e7db009926b1b30018ac834ee0cbca20b9d99f488038412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffffd5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c0200000069463043021f7059426d6aeb7d74275e52819a309b2bf903bd18b2b4d942d0e8e037681df702203f851f8a45aabfefdca5822f457609600f5d12a173adc09c6e7e2d4fdff7620a412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffffd5da6f960610cc65153521fd16dbe96b499143ac8d03222c13a9b97ce2dd8e3c720000006b483045022100e7b3837f2818fe00a05293e0f90e9005d59b0c5c8890f22bd31c36190a9b55e9022027de4b77b78139ea21b9fd30876a447bbf29662bd19d7914028c607bccd772e4412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66ffffffff01e8030000000000001976a914eb0bd5edba389198e73f8efabddfc61666969ff788ac00000000",
	"inputs": [
		{
			"unlockingScript": "48304502210081214df575da1e9378f1d5a29dfd6811e93466a7222fb010b7c50dd2d44d7f2e0220399bb396336d2e294049e7db009926b1b30018ac834ee0cbca20b9d99f488038412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66",
			"txid": "3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
			"vout": 0,
			"sequence": 4294967295
		},
		{
			"unlockingScript": "463043021f7059426d6aeb7d74275e52819a309b2bf903bd18b2b4d942d0e8e037681df702203f851f8a45aabfefdca5822f457609600f5d12a173adc09c6e7e2d4fdff7620a412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66",
			"txid": "3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
			"vout": 2,
			"sequence": 4294967295
		},
		{
			"unlockingScript": "483045022100e7b3837f2818fe00a05293e0f90e9005d59b0c5c8890f22bd31c36190a9b55e9022027de4b77b78139ea21b9fd30876a447bbf29662bd19d7914028c607bccd772e4412102798913bc057b344de675dac34faafe3dc2f312c758cd9068209f810877306d66",
			"txid": "3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5",
			"vout": 114,
			"sequence": 4294967295
		}
	],
	"outputs": [
		{
			"satoshis": 1000,
			"lockingScript": "76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac"
		}
	],
	"version": 1,
	"lockTime": 0
}`,
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// decouple extended info for this test
			testTx, err := bt.NewTxFromBytes(test.tx.Bytes())
			require.NoError(t, err)
			bb, err := json.MarshalIndent(testTx, "", "\t")
			require.NoError(t, err)
			assert.Equal(t, test.expJSON, string(bb))
		})
	}
	// TODO add tests for extended format
}

// TestTxUnmarshalJSON tests the JSON unmarshaling of transactions.
func TestTxUnmarshalJSON(t *testing.T) {
	t.Parallel()
	tests := map[string]struct {
		json  string
		expTX *bt.Tx
	}{
		"our json with hex should map correctly": {
			json: `{
				"version": 1,
				"lockTime": 0,
				"hex": "0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000",
				"inputs": [
					{
						"unlockingScript":"4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8",
						"txid": "a2a55ecc61f418e300888b1f82eaf84024496b34e3e538f3d32d342fd753adab",
						"vout": 1,
						"sequence": 4294967295
					}
				],
				"vout": [
					{
						"satoshis": 0,
						"lockingScript": "006a0548656c6c6f"
					},
					{
						"satoshis": 895,
						"lockingScript":"76a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac"
					}
				]
			}`,
			expTX: func() *bt.Tx {
				tx, err := bt.NewTxFromString("0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000")
				require.NoError(t, err)
				return tx
			}(),
		}, "ONLY hex should map correctly": {
			json: `{
				"hex": "0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000"
			}`,
			expTX: func() *bt.Tx {
				tx, err := bt.NewTxFromString("0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000")
				require.NoError(t, err)
				return tx
			}(),
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			var tx *bt.Tx
			err := json.Unmarshal([]byte(test.json), &tx)
			require.NoError(t, err)
			assert.Equal(t, test.expTX, tx)
		})
	}
}

// TestTx_ToJson tests the JSON marshaling of a transaction to ensure it can be converted to JSON format.
func TestTx_ToJson(t *testing.T) {
	tx, _ := bt.NewTxFromString("0100000001abad53d72f342dd3f338e5e3346b492440f8ea821f8b8800e318f461cc5ea5a2010000006a4730440220042edc1302c5463e8397120a56b28ea381c8f7f6d9bdc1fee5ebca00c84a76e2022077069bbdb7ed701c4977b7db0aba80d41d4e693112256660bb5d674599e390cf41210294639d6e4249ea381c2e077e95c78fc97afe47a52eb24e1b1595cd3fdd0afdf8ffffffff02000000000000000008006a0548656c6c6f7f030000000000001976a914b85524abf8202a961b847a3bd0bc89d3d4d41cc588ac00000000")

	_, err := json.MarshalIndent(tx, "", "\t")
	require.NoError(t, err)
}

// TestSuite is a test suite for the transaction JSON tests.
type TestSuite struct {
	suite.Suite
}

// TestRunTestSuite runs the test suite for transaction JSON tests.
func TestRunTestSuite(t *testing.T) {
	suite.Run(t, new(TestSuite))
}

// TestTxUnmarshalJSON tests the unmarshaling of transactions from JSON format.
func (ts *TestSuite) TestTxUnmarshalJSON() {
	ts.Run("hex path", func() {
		const sampleHex = "01000000000000000000" // TODO: replace with a valid tx hex
		data := fmt.Sprintf(`{"hex":"%s"}`, sampleHex)

		var got bt.Tx
		ts.Require().NoError(json.Unmarshal([]byte(data), &got))

		want, err := bt.NewTxFromString(sampleHex)
		ts.Require().NoError(err)
		ts.Require().Equal(want, &got)
	})

	ts.Run("fallback fields", func() {
		const (
			version  = uint32(2) // ← match Tx.Version’s type
			lockTime = uint32(0)
		)
		data := fmt.Sprintf(`{"version":%d,"lockTime":%d}`, version, lockTime)

		var got bt.Tx
		ts.Require().NoError(json.Unmarshal([]byte(data), &got))
		ts.Require().Equal(version, got.Version) // types now align
		ts.Require().Equal(lockTime, got.LockTime)
	})
}
