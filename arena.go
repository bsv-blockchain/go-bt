package bt

import (
	"io"

	"github.com/pkg/errors"
)

// MaxArenaAlloc is the maximum single allocation size accepted by Arena.Alloc
// and by *.ReadFromWithArena script-length guards. Set to 1 GiB to admit
// legitimate large BSV transactions (e.g. ~320 MB data carriers) while still
// rejecting obviously-bogus varint lengths.
//
// Implementation note: Arena.Alloc computes `a.pos + n` as int. With
// MaxArenaAlloc = 1<<30 and a 32-bit int, `pos + n` could overflow into
// negative values and bypass the grow check. go-bt targets 64-bit Go (all
// supported BSV node deployments are 64-bit); 32-bit platforms are not
// supported.
const MaxArenaAlloc = 1 << 30

// Arena is a bump-pointer allocator used by *.ReadFromWithArena to amortize
// per-script []byte allocations across a batch of transaction decodes.
//
// All slices returned by Alloc remain valid until Reset (or ResetAndShrink)
// is called. The caller must not retain any slice returned by Alloc past a
// Reset call — the backing array will be reused for subsequent allocations.
//
// Arena is NOT safe for concurrent use. A single goroutine must own the
// Arena between Reset calls.
//
// The zero value Arena{} is ready to use; the first Alloc allocates the
// backing slab.
type Arena struct {
	slab []byte
	pos  int
}

// NewArena returns an Arena with initialCap bytes pre-allocated. Passing
// initialCap <= 0 is equivalent to using a zero-value Arena.
func NewArena(initialCap int) *Arena {
	a := &Arena{}
	if initialCap > 0 {
		a.slab = make([]byte, initialCap)
	}
	return a
}

// Alloc returns a slice of length n drawn from the arena's slab. The
// returned slice has cap == len (3-arg slicing) so writes cannot overflow
// into subsequent allocations.
//
// Panics if n < 0 or n > MaxArenaAlloc.
func (a *Arena) Alloc(n int) []byte {
	if n < 0 || n > MaxArenaAlloc {
		panic("bt.Arena.Alloc: size out of range")
	}
	if a.pos+n > cap(a.slab) {
		newCap := 2 * cap(a.slab)
		if newCap < a.pos+n {
			newCap = a.pos + n
		}
		next := make([]byte, newCap)
		copy(next, a.slab[:a.pos])
		a.slab = next
	}
	// len(a.slab) == cap(a.slab) is maintained by every write to a.slab
	// (NewArena and the grow path both use make([]byte, n) which sets len==cap),
	// so the 3-arg slice below is always within bounds.
	out := a.slab[a.pos : a.pos+n : a.pos+n]
	a.pos += n
	return out
}

// Reset rewinds the arena's cursor to 0. The current backing slab is retained
// for reuse. O(1).
//
// Lifetime contract: any slice returned by Alloc that was allocated from the
// CURRENT slab becomes invalid for future reads after Reset — the backing
// memory will be overwritten by subsequent Alloc calls. Slices returned from
// an OLDER slab (i.e. before the most recent grow event) remain readable for
// as long as a caller holds a reference — the older slabs are not overwritten,
// they are simply orphaned for GC. Repeated grow-then-reset cycles can
// therefore retain pre-grow allocations longer than callers expect; use
// ResetAndShrink to drop oversized slabs explicitly.
func (a *Arena) Reset() {
	a.pos = 0
}

// ResetAndShrink behaves like Reset but additionally drops the backing slab
// if its capacity exceeds maxKeep. Use to bound the idle footprint of a
// pooled arena when a recent decode required an unusually large allocation.
// Passing maxKeep <= 0 retains the slab unchanged.
func (a *Arena) ResetAndShrink(maxKeep int) {
	a.pos = 0
	if maxKeep > 0 && cap(a.slab) > maxKeep {
		a.slab = nil
	}
}

// Cap returns the current backing slab capacity.
func (a *Arena) Cap() int { return cap(a.slab) }

// Used returns the number of bytes currently allocated since the last Reset.
func (a *Arena) Used() int { return a.pos }

// readArenaScript reads a varint length prefix from r followed by that many
// script bytes. When a is nil, the bytes go to a heap buffer — matching the
// non-arena ReadFrom path. When a is non-nil, bytes come from arena.Alloc.
// Either way the declared length is trusted only once a quarter of the script
// has arrived, unless the script fits in a scriptReadChunk heap buffer or in the
// arena's free space. A zero length yields a non-nil []byte{} sentinel so
// wrapped *bscript.Script values compare equal across both paths.
//
// Rejects lengths greater than MaxArenaAlloc with a wrapped error containing
// "MaxArenaAlloc". The label is included in both the cap-exceeded and the
// short-read error messages.
//
// Returns the script slice and the total bytes consumed from r (varint +
// script payload).
func readArenaScript(r io.Reader, a *Arena, label string) ([]byte, int64, error) {
	var bytesRead int64

	var l VarInt
	n64, err := l.ReadFrom(r)
	bytesRead += n64
	if err != nil {
		return nil, bytesRead, err
	}
	if uint64(l) > uint64(MaxArenaAlloc) {
		return nil, bytesRead, errors.Errorf("%s length %d exceeds MaxArenaAlloc", label, l)
	}

	var (
		script []byte
		n      int
	)

	if a != nil {
		script, n, err = readArenaScriptIncremental(r, a, int(l))
	} else {
		script, n, err = readScriptIncremental(r, int(l))
	}
	bytesRead += int64(n)

	if err != nil {
		return nil, bytesRead, errors.Wrapf(err, "%s(%d): got %d bytes", label, l, n)
	}

	return script, bytesRead, nil
}

// scriptReadChunk is the first chunk of a script's quarter-first read, and the
// size up to which the heap path allocates a script outright. Any declared
// length therefore buys at most this much memory before bytes arrive.
const scriptReadChunk = 64 * 1024

// readScriptIncremental reads length bytes from r onto the heap without
// trusting the declared length up front: a script larger than scriptReadChunk
// is read through readScriptQuarterFirst. A sender that declares a length near
// MaxArenaAlloc and then sends nothing therefore holds scriptReadChunk bytes,
// not a gigabyte (bsv-blockchain/go-bt#187).
//
// Returns the bytes read so far even on failure, so the caller can keep its
// stream accounting exact.
func readScriptIncremental(r io.Reader, length int) ([]byte, int, error) {
	prefixLen := 0
	if length > scriptReadChunk {
		prefixLen = length / 4
	}

	// make, not nil: callers compare wrapped *bscript.Script values across the
	// arena and non-arena paths, and a nil script is not equal to an empty one.
	return readScriptQuarterFirst(r, length, prefixLen, func(n int) []byte { return make([]byte, n) })
}

// readArenaScriptIncremental is readScriptIncremental for the arena. A script
// that fits in the slab costs no new memory whatever its declared length, so it
// is read in place. One that does not fit would grow the slab, so the slab grows
// only once a quarter of the script has arrived, and then once, to fit it.
//
// A failed read gives back the arena space it took. Returns the bytes read so
// far even on failure, so the caller can keep its stream accounting exact.
func readArenaScriptIncremental(r io.Reader, a *Arena, length int) ([]byte, int, error) {
	if length == 0 {
		// Non-nil, so it compares equal to the non-arena path's empty script.
		return []byte{}, 0, nil
	}

	prefixLen := 0
	if length > cap(a.slab)-a.pos {
		prefixLen = length / 4
	}

	start := a.pos

	script, n, err := readScriptQuarterFirst(r, length, prefixLen, a.Alloc)
	if err != nil {
		a.pos = start
	}

	return script, n, err
}

// readScriptQuarterFirst reads the first prefixLen bytes of a length-byte script
// into heap chunks that start at scriptReadChunk and double as bytes arrive, then
// takes the whole script from alloc, copies the prefix in and reads the rest
// straight into it. Memory follows the bytes received until the prefix is in;
// after that the declared length is trusted. Each byte is copied at most once,
// so a script that arrives costs about length + prefixLen.
//
// Returns the bytes read so far even on failure.
func readScriptQuarterFirst(r io.Reader, length, prefixLen int, alloc func(int) []byte) ([]byte, int, error) {
	var chunks [][]byte

	read := 0

	for size := scriptReadChunk; read < prefixLen; size *= 2 {
		chunk := make([]byte, min(size, prefixLen-read))

		n, err := io.ReadFull(r, chunk)
		read += n

		if err != nil {
			return nil, read, truncatedScriptError(err, read)
		}

		chunks = append(chunks, chunk)
	}

	script := alloc(length)

	off := 0
	for _, chunk := range chunks {
		off += copy(script[off:], chunk)
	}

	n, err := io.ReadFull(r, script[read:])
	read += n

	if err != nil {
		return nil, read, truncatedScriptError(err, read)
	}

	return script, read, nil
}

// truncatedScriptError reports what a single io.ReadFull of the whole script
// would have: io.EOF only when nothing arrived. A read split into pieces sees
// io.EOF on a piece that starts exactly where the stream ends, but from the
// caller's point of view the script is truncated, not absent.
func truncatedScriptError(err error, read int) error {
	if errors.Is(err, io.EOF) && read > 0 {
		return io.ErrUnexpectedEOF
	}

	return err
}
