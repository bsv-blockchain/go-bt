package bt

import (
	"io"

	"github.com/pkg/errors"
)

// MaxArenaAlloc is the maximum single allocation size accepted by Arena.Alloc
// and by *.ReadFromWithArena script-length guards. Set to 1 GiB to admit
// legitimate large BSV transactions (e.g. ~320 MB data carriers) while still
// rejecting obviously-bogus varint lengths.
const MaxArenaAlloc = 1 << 30

// Arena is a bump-pointer allocator used by *.ReadFromWithArena to amortise
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

// Reset rewinds the arena's cursor to 0. The slab is retained for reuse.
// O(1). All slices previously returned by Alloc become invalid for future
// reads — the backing memory will be overwritten by subsequent Alloc calls.
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
// script bytes. When a is nil, the bytes go to make([]byte, n) — matching
// the non-arena ReadFrom path. When a is non-nil, bytes come from
// arena.Alloc for n > 0; n == 0 uses a non-nil []byte{} sentinel so wrapped
// *bscript.Script values compare equal across both paths.
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

	var script []byte
	switch {
	case a == nil:
		script = make([]byte, l)
	case l > 0:
		script = a.Alloc(int(l))
	default:
		script = []byte{}
	}
	n, err := io.ReadFull(r, script)
	bytesRead += int64(n)
	if err != nil {
		return nil, bytesRead, errors.Wrapf(err, "%s(%d): got %d bytes", label, l, n)
	}

	return script, bytesRead, nil
}
