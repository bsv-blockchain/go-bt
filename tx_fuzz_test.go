package bt_test

import (
	"testing"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/stretchr/testify/require"
)

// FuzzReverseBytes ensures that reversing arbitrary byte slices is symmetric.
func FuzzReverseBytes(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x02, 0x03})

	f.Fuzz(func(t *testing.T, b []byte) {
		if len(b) > 1000 {
			t.Skip("input too large")
		}

		out := bt.ReverseBytes(b)
		require.Equal(t, b, bt.ReverseBytes(out))
	})
}
