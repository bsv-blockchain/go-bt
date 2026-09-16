package bt_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

// TestExtendedSize_MatchesExtendedBytes is the core invariant behind this whole
// change: the arithmetic ExtendedSize() must equal the length of the bytes
// ExtendedBytes() actually produces, for every transaction shape.
//
// It is a genuine oracle rather than the same code checked twice - ExtendedBytes()
// is built by the slice-appending path (toBytesHelper -> appendExtendedTo) while
// ExtendedSize() is an independent structural calculation. The invariant holds
// even for the non-extended corpus shapes, because both methods always
// emit/measure the extended representation regardless of what IsExtended() reports.
func TestExtendedSize_MatchesExtendedBytes(t *testing.T) {
	shapes := txShapeTests(t)
	shapes = append(shapes, []struct {
		name string
		tx   *bt.Tx
	}{
		{"extended 1 in 1 out", extendedShapeTx(t, 1, 1)},
		{"extended 2 in 2 out", extendedShapeTx(t, 2, 2)},
		{"extended 20 in 3 out", extendedShapeTx(t, 20, 3)},
	}...)

	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			want := uint64(len(s.tx.ExtendedBytes()))
			require.Equal(t, want, s.tx.ExtendedSize())
		})
	}
}

// TestExtendedSize_MatchesWriteExtendedTo pins ExtendedSize() to the streaming
// serialiser as well: the byte count WriteExtendedTo reports (and writes) must
// equal the pre-computed size, so a caller can trust ExtendedSize() as the
// content length of a stream it has not yet produced.
func TestExtendedSize_MatchesWriteExtendedTo(t *testing.T) {
	for _, n := range []int{1, 2, 20} {
		t.Run(fmt.Sprintf("%d_inputs", n), func(t *testing.T) {
			tx := extendedShapeTx(t, n, 2)

			var buf bytes.Buffer

			written, err := tx.WriteExtendedTo(&buf)
			require.NoError(t, err)
			require.Equal(t, tx.ExtendedSize(), uint64(written), "reported byte count")
			require.Equal(t, tx.ExtendedSize(), uint64(buf.Len()), "bytes written")
		})
	}
}

// TestExtendedSize_VarIntThresholds walks the PreviousTxScript length across
// every boundary where its varint length prefix changes width (1 -> 3 -> 5
// bytes). These are exactly the spans the auditor used to confirm the formula
// held; a mistake in the varint accounting would surface here as a one, two or
// four byte drift between ExtendedSize() and len(ExtendedBytes()).
func TestExtendedSize_VarIntThresholds(t *testing.T) {
	for _, n := range []int{0, 1, 252, 253, 254, 65535, 65536, 65537, 70000} {
		t.Run(fmt.Sprintf("prevscript_%d", n), func(t *testing.T) {
			tx := extendedShapeTx(t, 1, 1)
			tx.Inputs[0].PreviousTxScript = bscript.NewFromBytes(make([]byte, n))

			require.Equal(t, uint64(len(tx.ExtendedBytes())), tx.ExtendedSize())
		})
	}
}

// TestExtendedSize_Formula documents and pins the shape of the calculation:
// the bare size, plus the six-byte marker, plus per input the eight satoshi
// bytes and the length-prefixed previous locking script.
func TestExtendedSize_Formula(t *testing.T) {
	tx := extendedShapeTx(t, 3, 2)

	want := uint64(tx.Size()) + 6 // bare tx size + extended marker
	for _, in := range tx.Inputs {
		want += 8 // PreviousTxSatoshis
		l := len(*in.PreviousTxScript)
		want += uint64(bt.VarInt(uint64(l)).Length()) + uint64(l)
	}

	require.Equal(t, want, tx.ExtendedSize())
	require.Equal(t, want, uint64(len(tx.ExtendedBytes())))
}

// TestInput_ExtendedSize checks the per-input building block against its own
// serialiser, including the nil previous-script case that encodes as a single
// zero byte.
func TestInput_ExtendedSize(t *testing.T) {
	newInput := func(t *testing.T, prevScript *bscript.Script) *bt.Input {
		t.Helper()

		in := &bt.Input{
			UnlockingScript:    bscript.NewFromBytes(bytes.Repeat([]byte{0x51}, 107)),
			PreviousTxOutIndex: 0,
			SequenceNumber:     0xffffffff,
			PreviousTxSatoshis: 100000,
			PreviousTxScript:   prevScript,
		}

		require.NoError(t, in.PreviousTxIDAdd(&chainhash.Hash{}))

		return in
	}

	t.Run("nil previous script", func(t *testing.T) {
		in := newInput(t, nil)
		require.Equal(t, uint64(len(in.ExtendedBytes(false))), in.ExtendedSize())
	})

	for _, n := range []int{0, 1, 252, 253, 65535, 65536} {
		t.Run(fmt.Sprintf("prevscript_%d", n), func(t *testing.T) {
			in := newInput(t, bscript.NewFromBytes(make([]byte, n)))
			require.Equal(t, uint64(len(in.ExtendedBytes(false))), in.ExtendedSize())
		})
	}
}

// TestExtendedBytes_ExactAlloc guards the allocation fix. toBytesHelper now
// pre-sizes the buffer with ExtendedSize() for the extended path, so
// ExtendedBytes() must allocate its slice exactly once at the right capacity
// instead of re-growing (and copying) as it appends. A regression here would
// re-introduce the multi-copy behaviour the audit flagged as costly.
func TestExtendedBytes_ExactAlloc(t *testing.T) {
	tx := extendedShapeTx(t, 2, 2)

	b := tx.ExtendedBytes()
	require.Equal(t, len(b), cap(b),
		"ExtendedBytes() should allocate exact capacity, got len=%d cap=%d", len(b), cap(b))
	require.Equal(t, uint64(len(b)), tx.ExtendedSize())

	allocs := testing.AllocsPerRun(100, func() {
		_ = tx.ExtendedBytes()
	})
	require.InDelta(t, 1, allocs, 0, "ExtendedBytes() should make exactly one allocation")
}

// TestExtendedBytesMax covers the bounded serialiser: rejected below the size,
// allowed at or above it, and always producing bytes identical to ExtendedBytes()
// when it does not reject.
func TestExtendedBytesMax(t *testing.T) {
	tx := extendedShapeTx(t, 2, 2)
	size := tx.ExtendedSize()

	t.Run("below size is rejected without allocating", func(t *testing.T) {
		b, err := tx.ExtendedBytesMax(size - 1)
		require.ErrorIs(t, err, bt.ErrTxTooLarge, "error must wrap ErrTxTooLarge")
		require.Nil(t, b)
	})

	t.Run("exact size is allowed", func(t *testing.T) {
		b, err := tx.ExtendedBytesMax(size)
		require.NoError(t, err)
		require.Equal(t, tx.ExtendedBytes(), b)
	})

	t.Run("above size is allowed", func(t *testing.T) {
		b, err := tx.ExtendedBytesMax(size + 1024)
		require.NoError(t, err)
		require.Equal(t, tx.ExtendedBytes(), b)
	})
}

// TestSerializeBytesMax covers the format-selecting bounded serialiser and, in
// particular, that it bounds against the correct size for each format.
func TestSerializeBytesMax(t *testing.T) {
	t.Run("extended tx is bounded by extended size", func(t *testing.T) {
		tx := extendedShapeTx(t, 2, 2)
		require.True(t, tx.IsExtended())

		// The bare Size() is strictly smaller than the extended size. A limit set
		// to the bare size must still be rejected, proving the extended path (and
		// its larger size) is the one being checked.
		_, err := tx.SerializeBytesMax(uint64(tx.Size()))
		require.ErrorIs(t, err, bt.ErrTxTooLarge)

		b, err := tx.SerializeBytesMax(tx.ExtendedSize())
		require.NoError(t, err)
		require.Equal(t, tx.SerializeBytes(), b)
	})

	t.Run("standard tx is bounded by bare size", func(t *testing.T) {
		tx := mustParseTx()
		require.False(t, tx.IsExtended())

		size := uint64(tx.Size())

		_, err := tx.SerializeBytesMax(size - 1)
		require.ErrorIs(t, err, bt.ErrTxTooLarge)

		b, err := tx.SerializeBytesMax(size)
		require.NoError(t, err)
		require.Equal(t, tx.SerializeBytes(), b)
		require.Equal(t, tx.Bytes(), b)
	})
}

// FuzzExtendedSizeMatchesBytes keeps the ExtendedSize() arithmetic in lockstep
// with ExtendedBytes() across arbitrary decodable transactions, so no unusual
// input count, output count or script length can drift the two apart.
func FuzzExtendedSizeMatchesBytes(f *testing.F) {
	for _, hexStr := range []string{testTxHex, coinbaseTxHex} {
		raw, err := hex.DecodeString(hexStr)
		require.NoError(f, err)
		f.Add(raw)
	}

	f.Add(extendedShapeTx(f, 3, 2).ExtendedBytes())

	f.Fuzz(func(t *testing.T, data []byte) {
		tx, err := bt.NewTxFromBytes(data)
		if err != nil {
			t.Skip()
		}

		require.Equal(t, uint64(len(tx.ExtendedBytes())), tx.ExtendedSize())
	})
}

// BenchmarkExtendedBytes measures ExtendedBytes() at the shapes a node writes,
// confirming the pre-sized buffer keeps it to a single allocation.
func BenchmarkExtendedBytes(b *testing.B) {
	shapes := []struct {
		name string
		tx   *bt.Tx
	}{
		{"extended_2in_2out", extendedShapeTx(b, 2, 2)},
		{"extended_20in_2out", extendedShapeTx(b, 20, 2)},
	}

	for _, s := range shapes {
		b.Run(s.name, func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_ = s.tx.ExtendedBytes()
			}
		})
	}
}

// BenchmarkExtendedSize measures the zero-allocation size calculation.
func BenchmarkExtendedSize(b *testing.B) {
	tx := extendedShapeTx(b, 20, 2)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = tx.ExtendedSize()
	}
}
