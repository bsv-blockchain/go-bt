package bt_test

import (
	"encoding/hex"
	"os"
	"sync"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// FuzzNewTxFromBytes ensures that parsing arbitrary byte slices never panics
// and that valid transactions roundtrip correctly.
func FuzzNewTxFromBytes(f *testing.F) {
	// Seed corpus with a known valid transaction to exercise the successful
	// parsing path. This transaction is taken from the unit tests.
	seedHex := "02000000011ccba787d421b98904da3329b2c7336f368b62e89bc896019b5eadaa28145b9c0000000049483045022100c4df63202a9aa2bea5c24ebf4418d145e81712072ef744a4b108174f1ef59218022006eb54cf904707b51625f521f8ed2226f7d34b62492ebe4ddcb1c639caf16c3c41ffffffff0140420f00000000001976a91418392a59fc1f76ad6a3c7ffcea20cfcb17bda9eb88ac00000000"
	seedBytes, _ := hex.DecodeString(seedHex)
	f.Add(seedBytes)
	f.Add([]byte{0x00}) // minimal invalid input

	var mu sync.Mutex

	f.Fuzz(func(t *testing.T, txBytes []byte) {
		if os.Getenv("BT_ENABLE_FUZZ") != "1" {
			t.Skip("fuzzing disabled")
		}
		mu.Lock()
		defer mu.Unlock()

		if len(txBytes) > 10000 {
			t.Skip("input too large")
		}

		if len(txBytes) >= 9 {
			v, _ := bt.NewVarIntFromBytes(txBytes)
			if v > 1_000_000 {
				t.Skip("varint too big")
			}
		}

		tx, err := bt.NewTxFromBytes(txBytes)
		if err != nil {
			// On invalid input an error is expected and the tx should be nil
			assert.Nil(t, tx)
			return
		}

		require.NotNil(t, tx)

		// Re-encode and decode to ensure the bytes roundtrip without mutation.
		clone, err := bt.NewTxFromBytes(tx.Bytes())
		require.NoError(t, err)
		assert.Equal(t, tx.Bytes(), clone.Bytes())
	})
}
