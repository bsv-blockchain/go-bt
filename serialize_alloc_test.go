package bt_test

import (
	"bufio"
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

// extendedShapeTx builds a transaction in extended format: every input carries
// the satoshis and locking script of the output it spends. The shared corpus in
// tx_benchmark_test.go has no extended shape, and the extended path writes six
// more fields per input than the standard one, so it is where the allocations
// concentrate.
func extendedShapeTx(t testing.TB, inputs, outputs int) *bt.Tx {
	t.Helper()

	tx := bt.NewTx()

	for i := 0; i < inputs; i++ {
		in := &bt.Input{
			UnlockingScript:    bscript.NewFromBytes(bytes.Repeat([]byte{0x51}, 107)),
			PreviousTxOutIndex: uint32(i),
			SequenceNumber:     0xffffffff,
			PreviousTxSatoshis: 100000,
			PreviousTxScript:   bscript.NewFromBytes(bytes.Repeat([]byte{0x76}, 25)),
		}

		h := &chainhash.Hash{}
		h[0] = byte(i)
		require.NoError(t, in.PreviousTxIDAdd(h))

		tx.Inputs = append(tx.Inputs, in)
	}

	for i := 0; i < outputs; i++ {
		tx.AddOutput(&bt.Output{
			Satoshis:      1000,
			LockingScript: bscript.NewFromBytes(bytes.Repeat([]byte{0x76}, 25)),
		})
	}

	require.True(t, tx.IsExtended(), "the shape under test must be extended")

	return tx
}

// TestSerializeToDoesNotAllocate holds the streaming serialisers to zero heap
// allocations per transaction.
//
// Why it is worth a test. Every one of these methods used to declare a small
// fixed-size array for a length or an amount and hand it to w.Write. Because w
// is an interface, the compiler cannot see where the bytes go, so each of those
// arrays was moved to the heap: `go build -gcflags=-m` reported "moved to heap:
// buf" for output.go, and previousTxID, prevIndex, sequence and prevSatoshis
// for input.go. A two-input two-output extended transaction cost 16
// allocations to serialise, and a twenty-input one cost 88.
//
// That is not free anywhere, and it is expensive on a caller whose heap sits at
// its GOMEMLIMIT: a CPU profile of a Teranode node writing block 760431 spent
// 91% of the whole subtree-write phase inside runtime.newobject beneath
// Output.WriteTo, because each allocation first pays off sweep debt through
// runtime.deductSweepCredit before it may proceed.
//
// The writer is a bufio.Writer over io.Discard, which is what a streaming
// caller actually passes, and bufio.Write of a small slice allocates nothing of
// its own, so every allocation counted here belongs to this package.
func TestSerializeToDoesNotAllocate(t *testing.T) {
	shapes := txShapeTests(t)
	shapes = append(shapes, struct {
		name string
		tx   *bt.Tx
	}{"extended 2 in 2 out", extendedShapeTx(t, 2, 2)})
	shapes = append(shapes, struct {
		name string
		tx   *bt.Tx
	}{"extended 20 in 2 out", extendedShapeTx(t, 20, 2)})

	for _, s := range shapes {
		t.Run(s.name, func(t *testing.T) {
			w := bufio.NewWriterSize(io.Discard, 1<<20)

			// One call outside the measurement, so any one-off cost in the
			// writer or in a lazily built field is not charged to the loop.
			_, err := s.tx.SerializeTo(w)
			require.NoError(t, err)

			got := testing.AllocsPerRun(50, func() {
				if _, err := s.tx.SerializeTo(w); err != nil {
					t.Fatal(err)
				}

				if w.Buffered() > (1<<20)-(1<<18) {
					w.Reset(io.Discard)
				}
			})

			require.Zero(t, got, "SerializeTo made %.0f heap allocation(s) for %q", got, s.name)
		})
	}
}

// TestSerializeToMatchesBytesExtended is the correctness half. The existing
// TestWriteTo covers the standard shapes against Bytes(); this covers the
// extended path against ExtendedBytes(), which is built by a completely
// separate slice-appending implementation (toBytesHelper) and so is a genuine
// oracle rather than the same code checked twice.
func TestSerializeToMatchesBytesExtended(t *testing.T) {
	for _, n := range []int{1, 2, 20} {
		tx := extendedShapeTx(t, n, 2)

		want := tx.ExtendedBytes()

		var buf bytes.Buffer

		got, err := tx.SerializeTo(&buf)
		require.NoError(t, err)
		require.Equal(t, int64(len(want)), got, "reported byte count")
		require.Equal(t, want, buf.Bytes(), "serialised bytes for %d inputs", n)
	}
}

// TestPartWritersMatchTheirBytes pins the three exported part writers, which
// callers use on their own, to the slice builders beside them.
func TestPartWritersMatchTheirBytes(t *testing.T) {
	tx := extendedShapeTx(t, 3, 3)

	for i, in := range tx.Inputs {
		var buf bytes.Buffer

		n, err := in.WriteTo(&buf)
		require.NoError(t, err)
		require.Equal(t, in.Bytes(false), buf.Bytes(), "input %d", i)
		require.Equal(t, int64(buf.Len()), n, "input %d byte count", i)
	}

	for i, out := range tx.Outputs {
		var buf bytes.Buffer

		n, err := out.WriteTo(&buf)
		require.NoError(t, err)
		require.Equal(t, out.Bytes(), buf.Bytes(), "output %d", i)
		require.Equal(t, int64(buf.Len()), n, "output %d byte count", i)
	}

	for _, v := range []uint64{0, 1, 0xfc, 0xfd, 0xffff, 0x10000, 0xffffffff, 0x100000000} {
		var buf bytes.Buffer

		n, err := bt.VarInt(v).WriteTo(&buf)
		require.NoError(t, err)
		require.Equal(t, bt.VarInt(v).Bytes(), buf.Bytes(), "varint %d", v)
		require.Equal(t, int64(buf.Len()), n, "varint %d byte count", v)
	}
}

// BenchmarkSerializeToStreaming measures the streaming serialiser at the shapes
// a node actually writes: a standard pay-to-public-key-hash transaction, the
// same one in extended format, and a twenty-input transaction. The writer is a
// buffered one over io.Discard, so what is measured is this package's cost and
// not the disk's.
func BenchmarkSerializeToStreaming(b *testing.B) {
	shapes := []struct {
		name string
		tx   *bt.Tx
	}{
		{"extended_2in_2out", extendedShapeTx(b, 2, 2)},
		{"extended_20in_2out", extendedShapeTx(b, 20, 2)},
	}

	for _, s := range shapes {
		b.Run(s.name, func(b *testing.B) {
			w := bufio.NewWriterSize(io.Discard, 1<<20)

			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				if _, err := s.tx.SerializeTo(w); err != nil {
					b.Fatal(err)
				}

				if w.Buffered() > (1<<20)-(1<<18) {
					w.Reset(io.Discard)
				}
			}
		})
	}
}
