package bt_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

// TestTx_ReadFromWithArena_Equivalence checks that ReadFromWithArena produces a
// tx that serializes to the same bytes as ReadFrom for every canonical shape.
func TestTx_ReadFromWithArena_Equivalence(t *testing.T) {
	for _, tt := range txShapeTests(t) {
		t.Run(tt.name, func(t *testing.T) {
			expected := tt.tx.Bytes()

			refTx := &bt.Tx{}
			_, err := refTx.ReadFrom(bytes.NewReader(expected))
			require.NoError(t, err)

			arena := bt.NewArena(0)
			gotTx := &bt.Tx{}
			_, err = gotTx.ReadFromWithArena(bytes.NewReader(expected), arena)
			require.NoError(t, err)

			require.Equal(t, refTx.Bytes(), gotTx.Bytes(),
				"arena-decoded tx must serialize to identical bytes")
			require.Equal(t, refTx.TxID(), gotTx.TxID())
		})
	}
}

// TestTx_ReadFromWithArena_320MBOutput checks that a single OP_RETURN output
// of 320 MiB round-trips correctly through the arena decoder.
func TestTx_ReadFromWithArena_320MBOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping in short mode")
	}
	const scriptSize = 320 << 20

	bigScript := make([]byte, scriptSize)
	bigScript[0] = 0x6a // OP_RETURN

	tx := bt.NewTx()
	tx.AddOutput(&bt.Output{
		Satoshis:      0,
		LockingScript: bscript.NewFromBytes(bigScript),
	})
	raw := tx.Bytes()

	arena := bt.NewArena(0)
	got := &bt.Tx{}
	_, err := got.ReadFromWithArena(bytes.NewReader(raw), arena)
	require.NoError(t, err)
	require.Len(t, *got.Outputs[0].LockingScript, scriptSize)
}

func TestTx_HashTxIDInto_Equivalence(t *testing.T) {
	for _, tt := range txShapeTests(t) {
		t.Run(tt.name, func(t *testing.T) {
			expected := tt.tx.TxIDChainHash()

			var scratch []byte
			got, _ := tt.tx.HashTxIDInto(scratch)
			require.Equal(t, *expected, got)
		})
	}
}

func TestTx_HashTxIDInto_UsesCachedHash(t *testing.T) {
	tx := mustParseTx()
	cached := *tx.TxIDChainHash() // compute reference
	tx.SetTxHash(&cached)         // populate the cache

	var scratch []byte
	got, sc := tx.HashTxIDInto(scratch)
	require.Equal(t, cached, got)
	// When cache hits, scratch must be passed through unchanged.
	require.Nil(t, sc)
}

func TestTx_HashTxIDInto_ZeroAllocAfterWarmup(t *testing.T) {
	tx := mustParseTx()
	scratch := make([]byte, 0, tx.Size())

	allocs := testing.AllocsPerRun(100, func() {
		tx.SetTxHash(nil) // bust cache so we actually serialize each iteration
		var h chainhash.Hash
		h, scratch = tx.HashTxIDInto(scratch)
		_ = h
	})
	// HashTxIDInto + DoubleHashH should not allocate the serialization buffer
	// once scratch is sized. DoubleHashH on a byte slice is stack-only.
	require.LessOrEqual(t, allocs, 0.0)
}
