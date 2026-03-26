package bt_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-bt/v2"
)

// convertIntToBytes converts a uint64 integer to a byte slice
func convertIntToBytes(val uint64) []byte {
	buf := new(bytes.Buffer)
	if err := binary.Write(buf, binary.LittleEndian, val); err != nil {
		return nil
	}
	return buf.Bytes()
}

func TestDecodeVarInt(t *testing.T) {
	t.Parallel()
	tests := []struct {
		testName       string
		input          []byte
		expectedResult uint64
		expectedSize   int
	}{
		{"0xff", convertIntToBytes(0xff), 0, 9},
		{"0xfe", convertIntToBytes(0xfe), 0, 5},
		{"0xfd", convertIntToBytes(0xfd), 0, 3},
		{"1", convertIntToBytes(1), 1, 1},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			r, s := bt.NewVarIntFromBytes(test.input)
			assert.Equal(t, test.expectedResult, uint64(r))
			assert.Equal(t, test.expectedSize, s)
		})
	}
}

func TestVarIntUpperLimitInc(t *testing.T) {
	t.Parallel()

	tests := []struct {
		testName       string
		input          uint64
		expectedResult int
	}{
		{"0", 0, 0},
		{"10", 10, 0},
		{"100", 100, 0},
		{"252", 252, 2},
		{"65535", 65535, 2},
		{"4294967295", 4294967295, 4},
		{"18446744073709551615", 18446744073709551615, -1},
	}

	for _, test := range tests {
		t.Run(test.testName, func(t *testing.T) {
			r := bt.VarInt(test.input).UpperLimitInc()
			assert.Equal(t, test.expectedResult, r)
		})
	}
}

func TestVarInt(t *testing.T) {
	t.Parallel()

	varIntTests := []struct {
		testName    string
		input       uint64
		expectedLen int
	}{
		{"VarInt 1 byte Lower", 0, 1},
		{"VarInt 1 byte Upper", 252, 1},
		{"VarInt 3 byte Lower", 253, 3},
		{"VarInt 3 byte Upper", 65535, 3},
		{"VarInt 5 byte Lower", 65536, 5},
		{"VarInt 5 byte Upper", 4294967295, 5},
		{"VarInt 9 byte Lower", 4294967296, 9},
		{"VarInt 9 byte Upper", 18446744073709551615, 9},
	}

	for _, varIntTest := range varIntTests {
		t.Run(varIntTest.testName, func(t *testing.T) {
			assert.Len(t, bt.VarInt(varIntTest.input).Bytes(), varIntTest.expectedLen)
		})
	}
}

func TestVarInt_Size(t *testing.T) {
	tests := map[string]struct {
		v       bt.VarInt
		expSize int
	}{
		"252 returns 1": {
			v:       bt.VarInt(252),
			expSize: 1,
		},
		"253 returns 3": {
			v:       bt.VarInt(253),
			expSize: 3,
		},
		"65535 returns 3": {
			v:       bt.VarInt(65535),
			expSize: 3,
		},
		"65536 returns 5": {
			v:       bt.VarInt(65536),
			expSize: 5,
		},
		"4294967295 returns 5": {
			v:       bt.VarInt(4294967295),
			expSize: 5,
		},
		"4294967296 returns 9": {
			v:       bt.VarInt(4294967296),
			expSize: 9,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expSize, test.v.Length())
		})
	}
}

func TestVarInt_AppendTo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input uint64
	}{
		{"zero", 0},
		{"one byte max", 252},
		{"three byte min", 253},
		{"three byte max", 65535},
		{"five byte min", 65536},
		{"five byte max", 4294967295},
		{"nine byte min", 4294967296},
		{"nine byte max", 18446744073709551615},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v := bt.VarInt(tt.input)
			expected := v.Bytes()

			// AppendTo on nil should produce same bytes
			got := v.AppendTo(nil)
			assert.Equal(t, expected, got)

			// AppendTo on existing prefix should preserve prefix
			prefix := []byte{0xDE, 0xAD}
			got = v.AppendTo(prefix)
			assert.Equal(t, prefix, got[:2])
			assert.Equal(t, expected, got[2:])
		})
	}
}

func TestVarInt_AppendTo_ZeroAllocs(t *testing.T) {
	v := bt.VarInt(1000)
	buf := make([]byte, 0, 16)

	allocs := testing.AllocsPerRun(100, func() {
		buf = v.AppendTo(buf[:0])
	})
	assert.InDelta(t, 0, allocs, 0, "AppendTo should not allocate when buffer has capacity")
}
