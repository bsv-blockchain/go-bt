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

// TestReadScriptIncremental_AllocationTracksBytesReceived pins the fix for
// bsv-blockchain/go-bt#187: the allocation followed the length the sender
// declared rather than the bytes it sent, so a varint declaring just under
// MaxArenaAlloc bought a ~1 GiB heap allocation off a handful of input bytes.
// Wrapping the stream in an io.LimitedReader did not help, because the
// allocation happened before any byte was delivered.
func TestReadScriptIncremental_AllocationTracksBytesReceived(t *testing.T) {
	const declared = MaxArenaAlloc - 1 // just inside the cap, so the length check passes

	for _, tc := range []struct {
		name string
		sent int
	}{
		{name: "nothing sent", sent: 0},
		{name: "one chunk sent", sent: scriptReadChunk},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := bytes.NewReader(make([]byte, tc.sent))

			var before, after runtime.MemStats

			runtime.GC()
			runtime.ReadMemStats(&before)

			script, n, err := readScriptIncremental(r, declared)

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
	for _, length := range []int{0, 1, scriptReadChunk - 1, scriptReadChunk, scriptReadChunk + 1, 3*scriptReadChunk + 7} {
		t.Run(strconv.Itoa(length), func(t *testing.T) {
			want := make([]byte, length)
			for i := range want {
				want[i] = byte(i)
			}

			// A trailing byte the script must not consume.
			r := bytes.NewReader(append(append([]byte{}, want...), 0xAB))

			script, n, err := readScriptIncremental(r, length)
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

// TestReadScriptIncremental_TruncatedReportsUnexpectedEOF pins the error a
// short script produces. A stream ending exactly on a chunk boundary would
// otherwise report io.EOF, which reads as "nothing there" rather than
// "truncated", where a single ReadFull of the whole script reported
// io.ErrUnexpectedEOF.
func TestReadScriptIncremental_TruncatedReportsUnexpectedEOF(t *testing.T) {
	for _, tc := range []struct {
		name     string
		declared int
		sent     int
		wantErr  error
	}{
		{name: "nothing sent", declared: 10, sent: 0, wantErr: io.EOF},
		{name: "partial first chunk", declared: 10, sent: 4, wantErr: io.ErrUnexpectedEOF},
		{name: "ends on a chunk boundary", declared: scriptReadChunk + 10, sent: scriptReadChunk, wantErr: io.ErrUnexpectedEOF},
		{name: "partial second chunk", declared: 2 * scriptReadChunk, sent: scriptReadChunk + 5, wantErr: io.ErrUnexpectedEOF},
	} {
		t.Run(tc.name, func(t *testing.T) {
			script, n, err := readScriptIncremental(bytes.NewReader(make([]byte, tc.sent)), tc.declared)

			require.ErrorIs(t, err, tc.wantErr)
			require.Nil(t, script)
			require.Equal(t, tc.sent, n, "the byte count must stay exact so the caller's stream accounting holds")
		})
	}
}

// TestReadArenaScript_ArenaAndNonArenaAgree guards the split introduced by the
// incremental read: the arena path still allocates up front, so the two paths
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
// follows. The parse must fail without allocating the declared length.
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

	var before, after runtime.MemStats

	runtime.GC()
	runtime.ReadMemStats(&before)

	tx := NewTx()
	_, err := tx.ReadFrom(bytes.NewReader(data))

	runtime.ReadMemStats(&after)

	require.Error(t, err)

	allocated := after.TotalAlloc - before.TotalAlloc
	require.Less(t, allocated, uint64(1<<20),
		"a declared %d byte unlocking script must not be allocated before its bytes arrive (got %d bytes)",
		declared, allocated)
}
