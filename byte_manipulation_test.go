package bt_test

import (
	"encoding/hex"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2"
	crypto "github.com/bsv-blockchain/go-sdk/primitives/hash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLittleEndianBytes tests the LittleEndianBytes function.
func TestLittleEndianBytes(_ *testing.T) {
	// todo: add test coverage
}

// TestReverseBytes tests the ReverseBytes function.
func TestReverseBytes(t *testing.T) {
	t.Parallel()

	t.Run("genesis hash", func(t *testing.T) {
		b, err := hex.DecodeString("01000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4d04ffff001d0104455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73ffffffff0100f2052a01000000434104678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5fac00000000")
		require.NoError(t, err)

		h := bt.ReverseBytes(crypto.Sha256d(b))

		assert.Equal(t,
			"4a5e1e4baab89f3a32518a88c31bc87f618f76673e2cc77ab2127b7afdeda33b",
			hex.EncodeToString(h),
		)
	})

	t.Run("genesis block hash", func(t *testing.T) {
		b, err := hex.DecodeString("0100000000000000000000000000000000000000000000000000000000000000000000003ba3edfd7a7b12b27ac72c3e67768f617fc81bc3888a51323a9fb8aa4b1e5e4a29ab5f49ffff001d1dac2b7c0101000000010000000000000000000000000000000000000000000000000000000000000000ffffffff4d04ffff001d0104455468652054696d65732030332f4a616e2f32303039204368616e63656c6c6f72206f6e206272696e6b206f66207365636f6e64206261696c6f757420666f722062616e6b73ffffffff0100f2052a01000000434104678afdb0fe5548271967f1a67130b7105cd6a828e03909a67962e0ea1f61deb649f6bc3f4cef38c4f35504e51ec112de5c384df7ba0b8d578a4c702b6bf11d5fac00000000")
		require.NoError(t, err)

		h := bt.ReverseBytes(crypto.Sha256d(b[0:80]))

		assert.Equal(t,
			"000000000019d6689c085ae165831e934ff763ae46a2a6c172b3f1b60a8ce26f",
			hex.EncodeToString(h),
		)
	})
}
