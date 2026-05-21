package bt

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const outputHexStr = "8a08ac4a000000001976a9148bf10d323ac757268eb715e613cb8e8e1d1793aa88ac00000000"

func TestNewOutputFromBytes(t *testing.T) {
	t.Parallel()

	t.Run("invalid output, too short", func(t *testing.T) {
		o, s, err := newOutputFromBytes([]byte(""))
		require.Error(t, err)
		assert.Nil(t, o)
		assert.Equal(t, 0, s)
	})

	t.Run("invalid output, too short + script", func(t *testing.T) {
		o, s, err := newOutputFromBytes([]byte("0000000000000"))
		require.Error(t, err)
		assert.Nil(t, o)
		assert.Equal(t, 0, s)
	})

	t.Run("valid output", func(t *testing.T) {
		bytes, err := hex.DecodeString(outputHexStr)
		require.NoError(t, err)

		var o *Output
		var s int
		o, s, err = newOutputFromBytes(bytes)
		require.NoError(t, err)
		assert.NotNil(t, o)

		assert.Equal(t, 34, s)
		assert.Equal(t, uint64(1252788362), o.Satoshis)
		assert.Len(t, *o.LockingScript, 25)
		assert.Equal(t, "76a9148bf10d323ac757268eb715e613cb8e8e1d1793aa88ac", o.LockingScriptHexString())
	})
}

func TestOutput_String(t *testing.T) {
	t.Run("compare string output", func(t *testing.T) {
		bytes, err := hex.DecodeString(outputHexStr)
		require.NoError(t, err)

		var o *Output
		o, _, err = newOutputFromBytes(bytes)
		require.NoError(t, err)
		assert.NotNil(t, o)

		assert.Equal(t, "value:     1252788362\nscriptLen: 25\nscript:    76a9148bf10d323ac757268eb715e613cb8e8e1d1793aa88ac\n", o.String())
	})
}

// outputFixtures returns a set of representative Output payloads for
// equivalence + arena tests.
func outputFixtures(t testing.TB) []struct {
	name string
	data []byte
} {
	t.Helper()

	// 1 satoshi, OP_RETURN, 4-byte data
	small := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // satoshis = 1
		0x06,                                            // script len = 6
		0x6a, 0x04, 0xde, 0xad, 0xbe, 0xef,             // OP_RETURN 4-byte
	}

	// 5000-byte script
	bigScript := make([]byte, 5000)
	for i := range bigScript {
		bigScript[i] = byte(i)
	}
	big := []byte{0x05, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xfd, 0x88, 0x13} // satoshis=5, varint=5000
	big = append(big, bigScript...)

	return []struct {
		name string
		data []byte
	}{
		{name: "small_opreturn", data: small},
		{name: "5kb_script", data: big},
		{name: "empty_script", data: []byte{
			0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // satoshis = 7
			0x00, // script len = 0
		}},
	}
}

func TestOutput_ReadFromWithArena_Equivalence(t *testing.T) {
	for _, tt := range outputFixtures(t) {
		t.Run(tt.name, func(t *testing.T) {
			refOut := &Output{}
			_, err := refOut.ReadFrom(bytes.NewReader(tt.data))
			require.NoError(t, err)

			arena := NewArena(0)
			gotOut := &Output{}
			_, err = gotOut.ReadFromWithArena(bytes.NewReader(tt.data), arena)
			require.NoError(t, err)

			require.Equal(t, refOut.Satoshis, gotOut.Satoshis)
			require.Equal(t, []byte(*refOut.LockingScript), []byte(*gotOut.LockingScript))
		})
	}
}

func TestOutput_ReadFromWithArena_PostResetInvalidatesScript(t *testing.T) {
	data := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // satoshis=1
		0x04, 0xde, 0xad, 0xbe, 0xef,                    // script_len=4, payload
	}

	arena := NewArena(0)
	out := &Output{}
	_, err := out.ReadFromWithArena(bytes.NewReader(data), arena)
	require.NoError(t, err)

	first := make([]byte, len(*out.LockingScript))
	copy(first, *out.LockingScript)

	arena.Reset()
	// Next Alloc reuses the same backing memory — confirms slab is shared.
	overwrite := arena.Alloc(len(first))
	for i := range overwrite {
		overwrite[i] = 0xFF
	}

	// out.LockingScript now points into the slab — must read 0xFF.
	require.NotEqual(t, first, []byte(*out.LockingScript),
		"after Reset + reallocation, prior script bytes must be invalidated")
}

func TestOutput_ReadFromWithArena_RejectsOversizedScript(t *testing.T) {
	// satoshis(8) + varint=0xFE 0x00 0x00 0x00 0x80 (2^31 = 2 GiB) + (no data)
	data := []byte{
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0xfe, 0x00, 0x00, 0x00, 0x80,
	}
	arena := NewArena(0)
	out := &Output{}
	_, err := out.ReadFromWithArena(bytes.NewReader(data), arena)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxArenaAlloc")
}
