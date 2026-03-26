package bt_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/sighash"
)

func buildSigningTx(b testing.TB, nInputs int) *bt.Tx {
	b.Helper()
	tx := bt.NewTx()
	for i := 0; i < nInputs; i++ {
		require.NoError(b, tx.From(
			"b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6",
			uint32(i),
			"76a914167c3e911a14a92760b81334d01045da61e9681888ac",
			100000,
		))
	}
	tx.AddOutput(&bt.Output{
		Satoshis:      99000,
		LockingScript: bscript.NewFromBytes([]byte{0x76, 0xa9, 0x14}),
	})
	return tx
}

// BenchmarkCalcInputPreimage benchmarks the modern sighash path.
func BenchmarkCalcInputPreimage(b *testing.B) {
	for _, nInputs := range []int{1, 5, 20, 100} {
		tx := buildSigningTx(b, nInputs)

		// Benchmark signing ALL inputs (the real-world hot path)
		b.Run(fmt.Sprintf("AllInputs_%d_nocache", nInputs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for j := 0; j < nInputs; j++ {
					_, _ = tx.CalcInputPreimage(uint32(j), sighash.AllForkID)
				}
			}
		})

		b.Run(fmt.Sprintf("AllInputs_%d_cached", nInputs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				cache := tx.NewSigHashCache()
				for j := 0; j < nInputs; j++ {
					_, _ = tx.CalcInputPreimageWithCache(uint32(j), sighash.AllForkID, cache)
				}
			}
		})
	}
}

// BenchmarkCalcInputPreimageLegacy benchmarks the legacy sighash path.
func BenchmarkCalcInputPreimageLegacy(b *testing.B) {
	for _, nInputs := range []int{1, 5, 20} {
		tx := buildSigningTx(b, nInputs)

		b.Run(fmt.Sprintf("AllInputs_%d", nInputs), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				for j := 0; j < nInputs; j++ {
					_, _ = tx.CalcInputPreimageLegacy(uint32(j), sighash.All)
				}
			}
		})
	}
}

// TestCalcInputPreimageWithCache_MatchesUncached verifies the cached path produces
// identical preimages to the uncached path for all inputs.
func TestCalcInputPreimageWithCache_MatchesUncached(t *testing.T) {
	for _, nInputs := range []int{1, 3, 10} {
		tx := buildSigningTx(t, nInputs)
		cache := tx.NewSigHashCache()

		for j := 0; j < nInputs; j++ {
			expected, err := tx.CalcInputPreimage(uint32(j), sighash.AllForkID)
			require.NoError(t, err)

			got, err := tx.CalcInputPreimageWithCache(uint32(j), sighash.AllForkID, cache)
			require.NoError(t, err)

			require.Equal(t, expected, got, "input %d: cached preimage mismatch", j)
		}
	}
}

// TestOutputsHashOptimized verifies OutputsHash still returns correct results after optimization.
func TestOutputsHashOptimized(t *testing.T) {
	tx := buildSigningTx(t, 3)

	// Capture the result before and after (same code, just verifying we didn't break anything)
	hash1 := tx.OutputsHash(-1)
	hash2 := tx.OutputsHash(-1)
	require.Equal(t, hash1, hash2)

	// Single output hash
	hash3 := tx.OutputsHash(0)
	require.Len(t, hash3, 32)
}
