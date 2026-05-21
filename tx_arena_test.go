package bt_test

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
)

// TestTx_ReadFromWithArena_Equivalence checks that ReadFromWithArena produces a
// tx that serialises to the same bytes as ReadFrom for every canonical shape.
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
				"arena-decoded tx must serialise to identical bytes")
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
