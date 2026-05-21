package bt_test

import (
	"bytes"
	"encoding/hex"
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

func TestTx_Clone_IsolatesFromArena(t *testing.T) {
	// Build a tx with non-trivial unlocking and locking scripts so we
	// can detect corruption.
	unlockingPayload := bytes.Repeat([]byte{0xAB}, 64)
	lockingPayload := bytes.Repeat([]byte{0xCD}, 64)

	src := bt.NewTx()
	require.NoError(t, src.From(
		"b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0,
		hex.EncodeToString(lockingPayload), 100000,
	))
	// Set unlocking script directly so it is present in the wire serialisation.
	src.Inputs[0].UnlockingScript = bscript.NewFromBytes(unlockingPayload)
	src.AddOutput(&bt.Output{
		Satoshis:      99000,
		LockingScript: bscript.NewFromBytes(lockingPayload),
	})
	raw := src.Bytes()

	arena := bt.NewArena(0)
	decoded := &bt.Tx{}
	_, err := decoded.ReadFromWithArena(bytes.NewReader(raw), arena)
	require.NoError(t, err)

	clone := decoded.Clone()

	// Capture pre-corruption snapshots of clone's scripts.
	preUnlocking := append([]byte(nil), []byte(*clone.Inputs[0].UnlockingScript)...)
	preLocking := append([]byte(nil), []byte(*clone.Outputs[0].LockingScript)...)

	// Reset the arena and overwrite its backing with sentinel bytes.
	usedBefore := arena.Used()
	arena.Reset()
	overwrite := arena.Alloc(usedBefore)
	for i := range overwrite {
		overwrite[i] = 0xEE
	}

	// Clone's scripts MUST be unaffected by the arena overwrite.
	require.Equal(t, preUnlocking, []byte(*clone.Inputs[0].UnlockingScript),
		"Clone must deep-copy UnlockingScript")
	require.Equal(t, preLocking, []byte(*clone.Outputs[0].LockingScript),
		"Clone must deep-copy LockingScript")
}

func TestTx_ShallowClone_IsolatesFromArena(t *testing.T) {
	unlockingPayload := bytes.Repeat([]byte{0xAB}, 64)
	lockingPayload := bytes.Repeat([]byte{0xCD}, 64)

	src := bt.NewTx()
	require.NoError(t, src.From(
		"b7b0650a7c3a1bd4f7571b4c1e38f05171b565b8e28b2e337031ee31e9fa8eb6", 0,
		hex.EncodeToString(lockingPayload), 100000,
	))
	src.Inputs[0].UnlockingScript = bscript.NewFromBytes(unlockingPayload)
	src.AddOutput(&bt.Output{
		Satoshis:      99000,
		LockingScript: bscript.NewFromBytes(lockingPayload),
	})
	raw := src.Bytes()

	arena := bt.NewArena(0)
	decoded := &bt.Tx{}
	_, err := decoded.ReadFromWithArena(bytes.NewReader(raw), arena)
	require.NoError(t, err)

	clone := decoded.ShallowClone()

	preUnlocking := append([]byte(nil), []byte(*clone.Inputs[0].UnlockingScript)...)
	preLocking := append([]byte(nil), []byte(*clone.Outputs[0].LockingScript)...)

	usedBefore := arena.Used()
	arena.Reset()
	overwrite := arena.Alloc(usedBefore)
	for i := range overwrite {
		overwrite[i] = 0xEE
	}

	require.Equal(t, preUnlocking, []byte(*clone.Inputs[0].UnlockingScript),
		"ShallowClone must deep-copy UnlockingScript")
	require.Equal(t, preLocking, []byte(*clone.Outputs[0].LockingScript),
		"ShallowClone must deep-copy LockingScript")
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
