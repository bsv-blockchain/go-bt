package bt

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func inputFixtures(t testing.TB) []struct {
	name string
	data []byte
} {
	t.Helper()

	// Standard P2PKH unlocking script input
	// prevTxID(32) + vout(4) + script_len(1) + script(0) + sequence(4)
	emptyScript := []byte{
		// prev txid (32 zeros)
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0,
		// vout
		0x00, 0x00, 0x00, 0x00,
		// script_len = 0
		0x00,
		// sequence
		0xff, 0xff, 0xff, 0xff,
	}

	// 200-byte unlocking script
	bigScript := make([]byte, 200)
	for i := range bigScript {
		bigScript[i] = byte(i & 0xff)
	}
	bigData := append([]byte{}, emptyScript[:36]...) // prev + vout
	bigData = append(bigData, 0xc8)                  // varint = 200
	bigData = append(bigData, bigScript...)
	bigData = append(bigData, 0xff, 0xff, 0xff, 0xff) // sequence

	return []struct {
		name string
		data []byte
	}{
		{name: "empty_script", data: emptyScript},
		{name: "200b_script", data: bigData},
	}
}

func TestInput_ReadFromWithArena_Equivalence(t *testing.T) {
	for _, tt := range inputFixtures(t) {
		t.Run(tt.name, func(t *testing.T) {
			refIn := &Input{}
			_, err := refIn.ReadFrom(bytes.NewReader(tt.data))
			require.NoError(t, err)

			arena := NewArena(0)
			gotIn := &Input{}
			_, err = gotIn.ReadFromWithArena(bytes.NewReader(tt.data), arena)
			require.NoError(t, err)

			require.Equal(t, refIn.PreviousTxOutIndex, gotIn.PreviousTxOutIndex)
			require.Equal(t, refIn.SequenceNumber, gotIn.SequenceNumber)
			require.True(t, bytes.Equal(refIn.PreviousTxID(), gotIn.PreviousTxID()))
			require.Equal(t, []byte(*refIn.UnlockingScript), []byte(*gotIn.UnlockingScript))
		})
	}
}

func TestInput_ReadFromWithArena_RejectsOversizedScript(t *testing.T) {
	// prev txid (32) + vout (4) + varint=0xFE 0xFF 0xFF 0xFF 0xFF (huge, ~4 GiB)
	data := append(make([]byte, 36), 0xfe, 0xff, 0xff, 0xff, 0xff)
	arena := NewArena(0)
	in := &Input{}
	_, err := in.ReadFromWithArena(bytes.NewReader(data), arena)
	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxArenaAlloc")
}

func TestInput_ReadFromExtendedWithArena_Equivalence(t *testing.T) {
	// Hand-built extended-format input:
	// prev txid (32) + vout (4) + unlocking_script_len=0 + sequence(4)
	// + prev satoshis (8) + prev_script_len=2 + prev_script(2)
	data := make([]byte, 0, 52)
	data = append(data, make([]byte, 32)...)                            // prev txid
	data = append(data, 0x00, 0x00, 0x00, 0x00)                         // vout
	data = append(data, 0x00)                                           // unlocking script len = 0
	data = append(data, 0xff, 0xff, 0xff, 0xff)                         // sequence
	data = append(data, 0xe8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // prevSatoshis = 1000
	data = append(data, 0x02, 0x76, 0xa9)                               // prevScript len=2, payload OP_DUP OP_HASH160

	refIn := &Input{}
	_, err := refIn.ReadFromExtended(bytes.NewReader(data))
	require.NoError(t, err)

	arena := NewArena(0)
	gotIn := &Input{}
	_, err = gotIn.ReadFromExtendedWithArena(bytes.NewReader(data), arena)
	require.NoError(t, err)

	require.Equal(t, refIn.PreviousTxOutIndex, gotIn.PreviousTxOutIndex)
	require.Equal(t, refIn.SequenceNumber, gotIn.SequenceNumber)
	require.Equal(t, refIn.PreviousTxSatoshis, gotIn.PreviousTxSatoshis)
	require.True(t, bytes.Equal(refIn.PreviousTxID(), gotIn.PreviousTxID()))
	require.Equal(t, []byte(*refIn.UnlockingScript), []byte(*gotIn.UnlockingScript))
	require.Equal(t, []byte(*refIn.PreviousTxScript), []byte(*gotIn.PreviousTxScript))
}

func TestInput_ReadFrom_RejectsOversizedScript(t *testing.T) {
	// prev txid (32) + vout (4) + varint = 0xFE 0xFF 0xFF 0xFF 0xFF
	data := append(make([]byte, 36), 0xfe, 0xff, 0xff, 0xff, 0xff)
	in := &Input{}
	_, err := in.ReadFrom(bytes.NewReader(data))
	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxArenaAlloc")
}

func TestInput_ReadFromExtended_RejectsOversizedPrevTxScript(t *testing.T) {
	// Standard input header (no unlocking script) + sequence + prev satoshis +
	// PreviousTxScript varint that overflows.
	data := make([]byte, 0, 54)
	data = append(data, make([]byte, 32)...)                            // prev txid
	data = append(data, 0x00, 0x00, 0x00, 0x00)                         // vout
	data = append(data, 0x00)                                           // unlocking script len = 0
	data = append(data, 0xff, 0xff, 0xff, 0xff)                         // sequence
	data = append(data, 0xe8, 0x03, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00) // prevSatoshis = 1000
	data = append(data, 0xfe, 0xff, 0xff, 0xff, 0xff)                   // prevScript varint, oversize
	in := &Input{}
	_, err := in.ReadFromExtended(bytes.NewReader(data))
	require.Error(t, err)
	require.Contains(t, err.Error(), "MaxArenaAlloc")
}

func TestNewInputFromReader(t *testing.T) {
	t.Parallel()

	t.Run("valid tx", func(t *testing.T) {
		rawHex := "4c6ec863cf3e0284b407a1a1b8138c76f98280812cb9653231f385a0305fc76f010000006b483045022100f01c1a1679c9437398d691c8497f278fa2d615efc05115688bf2c3335b45c88602201b54437e54fb53bc50545de44ea8c64e9e583952771fcc663c8687dc2638f7854121037e87bbd3b680748a74372640628a8f32d3a841ceeef6f75626ab030c1a04824fffffffff"
		b, err := hex.DecodeString(rawHex)
		require.NoError(t, err)

		i := &Input{}
		var s int64
		s, err = i.readFrom(bytes.NewReader(b), false)

		require.NoError(t, err)
		assert.NotNil(t, i)
		assert.Equal(t, int64(148), s)
		assert.Equal(t, uint32(1), i.PreviousTxOutIndex)
		assert.Len(t, *i.UnlockingScript, 107)
		assert.Equal(t, DefaultSequenceNumber, i.SequenceNumber)
	})

	t.Run("empty bytes", func(t *testing.T) {
		i := &Input{}

		s, err := i.readFrom(bytes.NewReader([]byte("")), false)
		require.Error(t, err)
		assert.Equal(t, int64(0), s)
	})

	t.Run("invalid input, too short", func(t *testing.T) {
		i := &Input{}
		s, err := i.readFrom(bytes.NewReader([]byte("invalid")), false)
		require.Error(t, err)
		assert.Equal(t, int64(7), s)
	})

	t.Run("invalid input, too short + script", func(t *testing.T) {
		i := &Input{}
		s, err := i.readFrom(bytes.NewReader([]byte("000000000000000000000000000000000000000000000000000000000000000000000000")), false)
		require.Error(t, err)
		assert.Equal(t, int64(72), s)
	})
}

func TestInput_String(t *testing.T) {
	t.Run("valid tx", func(t *testing.T) {
		rawHex := "4c6ec863cf3e0284b407a1a1b8138c76f98280812cb9653231f385a0305fc76f010000006b483045022100f01c1a1679c9437398d691c8497f278fa2d615efc05115688bf2c3335b45c88602201b54437e54fb53bc50545de44ea8c64e9e583952771fcc663c8687dc2638f7854121037e87bbd3b680748a74372640628a8f32d3a841ceeef6f75626ab030c1a04824fffffffff"
		b, err := hex.DecodeString(rawHex)
		require.NoError(t, err)

		i := &Input{}
		var s int64

		s, err = i.readFrom(bytes.NewReader(b), false)
		require.NoError(t, err)
		assert.NotNil(t, i)
		assert.Equal(t, int64(148), s)

		assert.Equal(
			t,
			"prevTxHash:   6fc75f30a085f3313265b92c818082f9768c13b8a1a107b484023ecf63c86e4c\nprevOutIndex: 1\nscriptLen:    107\nscript:       483045022100f01c1a1679c9437398d691c8497f278fa2d615efc05115688bf2c3335b45c88602201b54437e54fb53bc50545de44ea8c64e9e583952771fcc663c8687dc2638f7854121037e87bbd3b680748a74372640628a8f32d3a841ceeef6f75626ab030c1a04824f\nsequence:     ffffffff\n",
			i.String(),
		)
	})
}
