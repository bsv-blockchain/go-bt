package bt

import (
	"bytes"
	"encoding/binary"
	"io"
	"runtime"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

// declaredScriptHeader returns the varint encoding of a script length, i.e. the
// bytes a sender uses to declare how long its script is.
func declaredScriptHeader(t *testing.T, length uint64) []byte {
	t.Helper()

	return VarInt(length).Bytes()
}

// scriptReaders lists both incremental readers, so the tests below hold the
// arena path to the same allocation, content and error contract as the heap
// path. The arena starts empty, so every byte it holds was allocated by the read.
var scriptReaders = []struct {
	name string
	read func(r io.Reader, length int) ([]byte, int, error)
}{
	{name: "heap", read: readScriptIncremental},
	{name: "arena", read: func(r io.Reader, length int) ([]byte, int, error) {
		return readArenaScriptIncremental(r, NewArena(0), length)
	}},
}

// TestReadScriptIncremental_AllocationTracksBytesReceived pins the fix for
// bsv-blockchain/go-bt#187: the allocation followed the length the sender
// declared rather than the bytes it sent, so a varint declaring just under
// MaxArenaAlloc bought a ~1 GiB heap allocation, or grew the arena slab to
// ~1 GiB, off a handful of input bytes. Wrapping the stream in an
// io.LimitedReader did not help, because the allocation happened before any
// byte was delivered.
func TestReadScriptIncremental_AllocationTracksBytesReceived(t *testing.T) {
	const declared = MaxArenaAlloc - 1 // just inside the cap, so the length check passes

	for _, reader := range scriptReaders {
		for _, tc := range []struct {
			name string
			sent int
		}{
			{name: "nothing sent", sent: 0},
			{name: "one chunk sent", sent: scriptReadChunk},
		} {
			t.Run(reader.name+"/"+tc.name, func(t *testing.T) {
				r := bytes.NewReader(make([]byte, tc.sent))

				var before, after runtime.MemStats

				runtime.GC()
				runtime.ReadMemStats(&before)

				script, n, err := reader.read(r, declared)

				runtime.ReadMemStats(&after)

				require.Error(t, err, "a truncated script must fail")
				require.Nil(t, script)
				require.Equal(t, tc.sent, n, "the byte count must reflect what arrived")

				// Generous next to the gigabyte the declaration used to buy, and still two
				// orders of magnitude below it. -race inflates the per-allocation overhead,
				// so this is not a tight accounting of the chunks themselves.
				allocated := after.TotalAlloc - before.TotalAlloc
				require.Less(t, allocated, uint64(1<<20),
					"allocation must track the %d bytes received, not the %d declared (got %d bytes)",
					tc.sent, declared, allocated)
			})
		}
	}
}

// TestOutput_ReadFrom_AllocationTracksBytesReceived exercises the same defect
// through the public non-arena path Teranode reaches when parsing a coinbase
// transaction out of a peer-supplied block message.
func TestOutput_ReadFrom_AllocationTracksBytesReceived(t *testing.T) {
	const declared = MaxArenaAlloc - 1

	// satoshis(8) + a script length nobody intends to send, then nothing.
	data := append(make([]byte, 0, 16), make([]byte, 8)...)
	data = append(data, declaredScriptHeader(t, declared)...)

	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	o := &Output{}
	_, err := o.ReadFrom(bytes.NewReader(data))

	runtime.ReadMemStats(&after)

	require.Error(t, err)
	require.Contains(t, err.Error(), "lockingScript")

	allocated := after.TotalAlloc - before.TotalAlloc
	require.Less(t, allocated, uint64(1<<20),
		"an output declaring a %d byte script and sending none must not allocate it (got %d bytes)",
		declared, allocated)
}

// TestReadScriptIncremental_ReadsExactScripts covers the sizes either side of
// the chunk boundary, where a chunked read is most likely to go wrong.
func TestReadScriptIncremental_ReadsExactScripts(t *testing.T) {
	for _, reader := range scriptReaders {
		for _, length := range []int{0, 1, scriptReadChunk - 1, scriptReadChunk, scriptReadChunk + 1, 3*scriptReadChunk + 7, 1<<20 + 3} {
			t.Run(reader.name+"/"+strconv.Itoa(length), func(t *testing.T) {
				want := make([]byte, length)
				for i := range want {
					want[i] = byte(i)
				}

				// A trailing byte the script must not consume.
				r := bytes.NewReader(append(append([]byte{}, want...), 0xAB))

				script, n, err := reader.read(r, length)
				require.NoError(t, err)
				require.Equal(t, length, n)
				require.Equal(t, want, script)
				require.NotNil(t, script, "an empty script must stay non-nil so it compares equal across the arena and non-arena paths")

				rest, err := io.ReadAll(r)
				require.NoError(t, err)
				require.Equal(t, []byte{0xAB}, rest, "the read must stop at the declared length")
			})
		}
	}
}

// TestReadScriptIncremental_TruncatedReportsUnexpectedEOF pins the error a
// short script produces. A stream ending exactly where a piece of the read
// starts would otherwise report io.EOF, which reads as "nothing there" rather
// than "truncated", where a single ReadFull of the whole script reported
// io.ErrUnexpectedEOF.
func TestReadScriptIncremental_TruncatedReportsUnexpectedEOF(t *testing.T) {
	for _, tc := range []struct {
		name     string
		declared int
		sent     int
		wantErr  error
	}{
		{name: "nothing sent", declared: 10, sent: 0, wantErr: io.EOF},
		{name: "partly sent", declared: 10, sent: 4, wantErr: io.ErrUnexpectedEOF},
		{name: "ends where the second chunk of the quarter starts", declared: 1 << 20, sent: scriptReadChunk, wantErr: io.ErrUnexpectedEOF},
		{name: "ends exactly at the quarter", declared: 1 << 20, sent: (1 << 20) / 4, wantErr: io.ErrUnexpectedEOF},
		{name: "ends after the quarter", declared: scriptReadChunk + 10, sent: scriptReadChunk, wantErr: io.ErrUnexpectedEOF},
	} {
		for _, reader := range scriptReaders {
			t.Run(reader.name+"/"+tc.name, func(t *testing.T) {
				script, n, err := reader.read(bytes.NewReader(make([]byte, tc.sent)), tc.declared)

				require.ErrorIs(t, err, tc.wantErr)
				require.Nil(t, script)
				require.Equal(t, tc.sent, n, "the byte count must stay exact so the caller's stream accounting holds")
			})
		}
	}
}

// TestReadArenaScript_ArenaAndNonArenaAgree guards the split between the two
// incremental reads: the arena and heap paths grow different buffers, so they
// must keep producing identical scripts and byte counts.
func TestReadArenaScript_ArenaAndNonArenaAgree(t *testing.T) {
	for _, length := range []int{0, 1, scriptReadChunk + 1} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			payload := make([]byte, length)
			for i := range payload {
				payload[i] = byte(i % 251)
			}

			data := append(declaredScriptHeader(t, uint64(length)), payload...)

			withArena, arenaBytes, err := readArenaScript(bytes.NewReader(data), NewArena(0), "lockingScript")
			require.NoError(t, err)

			withoutArena, plainBytes, err := readArenaScript(bytes.NewReader(data), nil, "lockingScript")
			require.NoError(t, err)

			require.Equal(t, withArena, withoutArena)
			require.Equal(t, arenaBytes, plainBytes)
			require.Equal(t, int64(len(data)), plainBytes)
			require.NotNil(t, withoutArena)
		})
	}
}

// TestReadArenaScript_StillRejectsOversizedDeclaration keeps the cap check in
// front of the incremental read: a length beyond MaxArenaAlloc must fail on the
// declaration alone, without reading the payload.
func TestReadArenaScript_StillRejectsOversizedDeclaration(t *testing.T) {
	data := declaredScriptHeader(t, MaxArenaAlloc+1)

	script, n, err := readArenaScript(bytes.NewReader(data), nil, "lockingScript")

	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxArenaAlloc")
	require.Nil(t, script)
	require.Equal(t, int64(len(data)), n, "only the varint may be consumed")
}

// TestTx_ReadFrom_HugeDeclaredCoinbaseScript is the shape Teranode hits: a
// transaction whose input declares a script far larger than the body that
// follows. The parse must fail without allocating the declared length, with or
// without an arena; Teranode decodes peer-supplied subtree data through a
// pooled, pre-sized arena.
func TestTx_ReadFrom_HugeDeclaredCoinbaseScript(t *testing.T) {
	const declared = MaxArenaAlloc - 1

	data := make([]byte, 0, 64)
	data = append(data, 0x01, 0x00, 0x00, 0x00) // version
	data = append(data, 0x01)                   // input count
	data = append(data, make([]byte, 32)...)
	data = append(data, 0xff, 0xff, 0xff, 0xff)               // vout
	data = append(data, declaredScriptHeader(t, declared)...) // unlocking script length
	// A few real script bytes, then the stream simply ends.
	data = append(data, binary.LittleEndian.AppendUint32(nil, 1)...)

	for _, tc := range []struct {
		name  string
		arena *Arena
	}{
		{name: "no arena"},
		{name: "arena", arena: NewArena(2 << 20)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var before, after runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&before)

			_, err := NewTx().ReadFromWithArena(bytes.NewReader(data), tc.arena)

			runtime.ReadMemStats(&after)

			require.Error(t, err)

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, uint64(1<<20),
				"a declared %d byte unlocking script must not be allocated before its bytes arrive (got %d bytes)",
				declared, allocated)
		})
	}
}

// TestReadScriptIncremental_HonestScriptCostsAQuarterMore bounds what a large
// script that does arrive costs. Growing one buffer against the bytes received
// reallocated it many times over: an 80 MiB script allocated 457 MiB on the heap
// path, and a 320 MiB script grew a 2 MiB arena to a 512 MiB slab. Reading a
// quarter first and then allocating the script once costs about 1.25 times its
// length.
func TestReadScriptIncremental_HonestScriptCostsAQuarterMore(t *testing.T) {
	const length = 8 << 20

	for _, reader := range scriptReaders {
		t.Run(reader.name, func(t *testing.T) {
			r := bytes.NewReader(make([]byte, length))

			var before, after runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&before)

			script, _, err := reader.read(r, length)

			runtime.ReadMemStats(&after)

			require.NoError(t, err)
			require.Len(t, script, length)

			allocated := after.TotalAlloc - before.TotalAlloc
			require.Less(t, allocated, uint64(length+length/2),
				"a %d byte script must cost about 1.25 times its length (got %d bytes)", length, allocated)
		})
	}
}

// TestReadScriptIncremental_FullAllocationWaitsForAQuarter pins where the
// declared length is first trusted: once a quarter of the script has arrived,
// so a sender holds at most about four times what it has delivered.
func TestReadScriptIncremental_FullAllocationWaitsForAQuarter(t *testing.T) {
	const declared = 64 << 20

	for _, reader := range scriptReaders {
		for _, tc := range []struct {
			name           string
			sent           int
			fullyAllocated bool
		}{
			{name: "just short of a quarter", sent: declared/4 - 1024},
			{name: "just past a quarter", sent: declared/4 + 1024, fullyAllocated: true},
		} {
			t.Run(reader.name+"/"+tc.name, func(t *testing.T) {
				r := bytes.NewReader(make([]byte, tc.sent))

				var before, after runtime.MemStats

				runtime.GC()
				runtime.ReadMemStats(&before)

				_, n, err := reader.read(r, declared)

				runtime.ReadMemStats(&after)

				require.ErrorIs(t, err, io.ErrUnexpectedEOF)
				require.Equal(t, tc.sent, n)

				allocated := after.TotalAlloc - before.TotalAlloc
				if tc.fullyAllocated {
					require.GreaterOrEqual(t, allocated, uint64(declared),
						"once a quarter has arrived the script is allocated whole (got %d bytes)", allocated)
				} else {
					require.Less(t, allocated, uint64(declared/2),
						"before a quarter has arrived the declared length must not be allocated (got %d bytes)", allocated)
				}
			})
		}
	}
}

// TestReadArenaScript_ArenaGrowsOnceForLargeScript reads a script that does not
// fit the slab, with an earlier allocation already in it. The slab must grow
// once, to exactly what the script needs, keep the earlier bytes at their
// offsets, and hand back a script that cannot be appended into the next
// allocation.
func TestReadArenaScript_ArenaGrowsOnceForLargeScript(t *testing.T) {
	const (
		slab    = scriptReadChunk + 100
		earlier = 50
		length  = 3*scriptReadChunk + 7 // past twice the slab, so Alloc sizes the slab to fit
	)

	a := NewArena(slab)
	copy(a.Alloc(earlier), bytes.Repeat([]byte{0xEE}, earlier))

	want := make([]byte, length)
	for i := range want {
		want[i] = byte(i % 251)
	}

	// A trailing byte the script must not consume.
	r := bytes.NewReader(append(append(declaredScriptHeader(t, length), want...), 0xAB))

	script, _, err := readArenaScript(r, a, "lockingScript")
	require.NoError(t, err)
	require.Equal(t, want, script)
	require.Equal(t, len(script), cap(script), "the script must not share capacity with later allocations")
	require.Equal(t, earlier+length, a.Cap(), "the slab must grow once, to what the script needs")
	require.Equal(t, bytes.Repeat([]byte{0xEE}, earlier), a.slab[:earlier], "the grow must keep the earlier allocation")

	copy(a.Alloc(8), bytes.Repeat([]byte{0xFF}, 8))
	require.Equal(t, want, script, "a later allocation must not overwrite the script")

	rest, err := io.ReadAll(r)
	require.NoError(t, err)
	require.Equal(t, []byte{0xAB}, rest, "the read must stop at the declared length")
}

// TestReadArenaScript_TruncatedScriptGivesBackArenaSpace covers a pooled arena
// that is nearly full when a truncated script arrives. The declaration alone
// must not grow the slab, and a failed read must give back the cursor space it
// took, so a decoder that skips a bad transaction does not lose it. A slab that
// grew once a quarter had arrived stays grown.
func TestReadArenaScript_TruncatedScriptGivesBackArenaSpace(t *testing.T) {
	const (
		slab = 1 << 20
		used = slab - 100
	)

	for _, tc := range []struct {
		name      string
		declared  int
		sent      int
		slabGrows bool
	}{
		{name: "fits, partly sent", declared: 50, sent: 10},
		{name: "does not fit, nothing sent", declared: MaxArenaAlloc - 1},
		{name: "does not fit, a quarter sent", declared: 4 * scriptReadChunk, sent: scriptReadChunk + 5, slabGrows: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a := NewArena(slab)
			a.Alloc(used)

			data := append(declaredScriptHeader(t, uint64(tc.declared)), make([]byte, tc.sent)...)

			script, _, err := readArenaScript(bytes.NewReader(data), a, "lockingScript")
			require.Error(t, err)
			require.Nil(t, script)
			require.Equal(t, used, a.Used(), "a failed read must give back the space it took")

			if !tc.slabGrows {
				require.Equal(t, slab, a.Cap(), "the slab must not grow before a quarter of the script has arrived")
			}
		})
	}
}
