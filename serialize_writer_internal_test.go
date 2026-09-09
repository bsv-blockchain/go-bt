package bt

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestPartWriterNeverGrowsItsBuffer protects the size guard in raw().
//
// Without it, a script larger than the scratch buffer is appended anyway, the
// buffer grows to hold it, and that grown buffer goes back into the shared
// pool. Every later transaction then borrows a buffer sized for the largest
// script the process has ever seen, and the pool quietly retains it. The
// allocation test cannot see this: the pool serves the grown buffer back, so
// the second and later runs allocate nothing and it passes.
func TestPartWriterNeverGrowsItsBuffer(t *testing.T) {
	p := newPartWriter(io.Discard)
	defer p.release()

	require.Equal(t, scratchLen, cap(p.buf), "a fresh scratch buffer")

	p.raw(bytes.Repeat([]byte{0x01}, 100_000))
	require.NoError(t, p.err)
	require.Equal(t, scratchLen, cap(p.buf), "the buffer must not grow to hold a large script")

	// A script that fits is still copied, which is the point of the buffer.
	p.raw(bytes.Repeat([]byte{0x02}, 16))
	require.Len(t, p.buf, 16)
	require.Equal(t, scratchLen, cap(p.buf))
}

// TestPartWriterCountsWhatTheWriterAccepted pins the returned byte count to the
// bytes the underlying writer took, which is what the per-field w.Write calls
// used to sum to. A buffered writer makes this easy to get wrong, because the
// bytes are counted at flush rather than at append.
func TestPartWriterCountsWhatTheWriterAccepted(t *testing.T) {
	var buf bytes.Buffer

	p := newPartWriter(&buf)
	defer p.release()

	p.u32(1)
	p.u64(2)
	p.varInt(0xffff)
	p.raw(bytes.Repeat([]byte{0x03}, 100_000))
	p.raw([]byte{0x04})

	n, err := p.finish()
	require.NoError(t, err)
	require.Equal(t, int64(buf.Len()), n)
	require.Equal(t, int64(4+8+3+100_000+1), n)
}

// TestPartWriterStopsAtTheFirstError checks that a failing writer is reported
// and that nothing is written after it, since every method is a no-op once err
// is set and the callers have no error check between fields.
func TestPartWriterStopsAtTheFirstError(t *testing.T) {
	fw := &failingWriter{failAfter: 4}

	p := newPartWriter(fw)
	defer p.release()

	// Enough to force several flushes, the first of which fails.
	for i := 0; i < 200; i++ {
		p.u64(uint64(i))
		p.raw(bytes.Repeat([]byte{0x05}, 100_000))
	}

	n, err := p.finish()
	require.Error(t, err)
	require.Equal(t, int64(4), n, "only the bytes the writer accepted are counted")
	require.Equal(t, 1, fw.calls, "no writes after the first failure")
}

type failingWriter struct {
	failAfter int
	calls     int
}

func (f *failingWriter) Write(b []byte) (int, error) {
	f.calls++

	return f.failAfter, io.ErrClosedPipe
}
