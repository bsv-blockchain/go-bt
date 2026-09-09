package bt

import (
	"io"
	"sync"
)

// scratchLen is the size of the reusable buffer a streaming serialise call
// borrows. It holds the fixed-size fields (amounts, output indexes, sequence
// numbers, length prefixes) and any script short enough to fit alongside them,
// so a standard pay-to-public-key-hash transaction reaches the underlying
// writer in one call rather than fifteen.
//
// 512 bytes holds a two-input two-output extended transaction whole. A larger
// buffer would not serialise more transactions in one write, because the shapes
// that overflow it are the ones carrying scripts of thousands of bytes, and
// those are written straight through without being copied.
const scratchLen = 512

// scratchPool hands out the buffers. A pointer to a slice is pooled rather than
// a slice, because putting a slice into the pool's any-typed argument boxes the
// three-word header and allocates, which is the cost this whole file exists to
// remove.
var scratchPool = sync.Pool{
	New: func() any {
		b := make([]byte, 0, scratchLen)

		return &b
	},
}

// partWriter accumulates a transaction's small fields in a borrowed buffer and
// flushes them to the underlying writer.
//
// It exists because the obvious way to write eight bytes to an io.Writer, a
// local `var buf [8]byte` handed to w.Write, allocates. The compiler cannot see
// where an interface method takes the bytes, so it moves the array to the heap;
// `go build -gcflags=-m` reports exactly that for output.go, input.go and
// varint.go. One allocation per field per input and per output is a large
// number on a caller streaming a whole block, and on a caller whose heap is at
// its GOMEMLIMIT each one first pays off sweep debt before it may proceed.
//
// Every method is a no-op once err is set, so callers read as a straight run of
// appends with a single error check at the end.
type partWriter struct {
	w     io.Writer
	buf   []byte
	held  *[]byte
	total int64
	err   error
}

// newPartWriter borrows a buffer and wraps w.
func newPartWriter(w io.Writer) *partWriter {
	held, _ := scratchPool.Get().(*[]byte)

	return &partWriter{w: w, buf: (*held)[:0], held: held}
}

// release returns the buffer to the pool. The partWriter must not be used
// afterwards.
func (p *partWriter) release() {
	if p.held != nil {
		*p.held = p.buf[:0]
		scratchPool.Put(p.held)
		p.held = nil
	}

	p.buf = nil
}

// space reports how many more bytes fit before a flush is needed.
func (p *partWriter) space() int {
	return cap(p.buf) - len(p.buf)
}

// reserve flushes if fewer than n bytes remain.
func (p *partWriter) reserve(n int) {
	if p.err == nil && p.space() < n {
		p.flush()
	}
}

// u32 appends a 32-bit value in little-endian order.
func (p *partWriter) u32(v uint32) {
	p.reserve(4)

	if p.err != nil {
		return
	}

	p.buf = append(p.buf, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
}

// u64 appends a 64-bit value in little-endian order.
func (p *partWriter) u64(v uint64) {
	p.reserve(8)

	if p.err != nil {
		return
	}

	p.buf = append(p.buf,
		byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

// varInt appends a Bitcoin variable-length integer.
func (p *partWriter) varInt(v uint64) {
	p.reserve(9)

	if p.err != nil {
		return
	}

	p.buf = VarInt(v).AppendTo(p.buf)
}

// raw appends b when it fits in what is left of the buffer, and otherwise
// flushes and writes it straight to the underlying writer. Copying a
// hundred-kilobyte script into a scratch buffer would cost more than the
// allocation this file removes, so large scripts are never copied.
func (p *partWriter) raw(b []byte) {
	if p.err != nil || len(b) == 0 {
		return
	}

	if len(b) <= p.space() {
		p.buf = append(p.buf, b...)
		return
	}

	p.flush()

	if p.err != nil {
		return
	}

	n, err := p.w.Write(b)
	p.total += int64(n)

	if err != nil {
		p.err = err
		return
	}

	if n < len(b) {
		p.err = io.ErrShortWrite
	}
}

// flush writes whatever has accumulated and empties the buffer.
func (p *partWriter) flush() {
	if p.err != nil || len(p.buf) == 0 {
		return
	}

	want := len(p.buf)

	n, err := p.w.Write(p.buf)
	p.total += int64(n)
	p.buf = p.buf[:0]

	if err != nil {
		p.err = err
		return
	}

	if n < want {
		p.err = io.ErrShortWrite
	}
}

// finish flushes and reports the bytes the underlying writer accepted, which is
// the same quantity the per-field w.Write calls used to sum to.
func (p *partWriter) finish() (int64, error) {
	p.flush()

	return p.total, p.err
}

// extendedMarker is the six-byte sequence that follows the version in an
// extended transaction. It is a package-level array so that appending it costs
// nothing; a `[]byte{...}` literal inside the writer would allocate on every
// transaction, which is the cost this file removes everywhere else.
var extendedMarker = [6]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0xEF}
