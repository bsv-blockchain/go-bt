package bt_test

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bsv-blockchain/go-bt/v2"
)

// canonicalTxHex is a standard 1-input, 2-output transaction. Its input count is
// a single 0x01 byte at offset 4, immediately after the 4-byte version.
const canonicalTxHex = "010000000110ee96aa946338cfd0b2ed0603259cfe2f5458c32ee4bd7b88b583769c6b046e010000006b483045022100e5e4749d539a163039769f52e1ebc8e6f62e39387d61e1a305bd722116cded6c022014924b745dd02194fe6b5cb8ac88ee8e9a2aede89e680dcea6169ea696e24d52012102b4b754609b46b5d09644c2161f1767b72b93847ce8154d795f95d31031a08aa2ffffffff028098f34c010000001976a914a134408afa258a50ed7a1d9817f26b63cc9002cc88ac8028bb13010000001976a914fec5b1145596b35f59f8be1daf169f375942143388ac00000000"

func TestVarIntReadFromRejectsNonMinimal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		value uint64
	}{
		{"1 as 3 bytes", []byte{0xfd, 0x01, 0x00}, 1},
		{"0 as 3 bytes", []byte{0xfd, 0x00, 0x00}, 0},
		{"252 as 3 bytes", []byte{0xfd, 0xfc, 0x00}, 252},
		{"1 as 5 bytes", []byte{0xfe, 0x01, 0x00, 0x00, 0x00}, 1},
		{"65535 as 5 bytes", []byte{0xfe, 0xff, 0xff, 0x00, 0x00}, 65535},
		{"1 as 9 bytes", []byte{0xff, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}, 1},
		{"4294967295 as 9 bytes", []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0x00, 0x00, 0x00, 0x00}, 4294967295},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			var v bt.VarInt

			n, err := v.ReadFrom(bytes.NewReader(test.input))

			require.Error(t, err)
			require.ErrorIs(t, err, bt.ErrNonMinimalVarInt)

			// The full prefix is still consumed, so a caller that chooses to
			// continue stays aligned with the stream.
			require.Equal(t, int64(len(test.input)), n)

			// The decoded value is reported, so the message can name it.
			require.Contains(t, err.Error(), fmt.Sprintf("%d", test.value))
		})
	}
}

func TestVarIntReadFromAcceptsMinimal(t *testing.T) {
	t.Parallel()

	// Every value that sits on a width boundary, plus the widths either side.
	values := []uint64{
		0, 1, 251, 252,
		253, 254, 65534, 65535,
		65536, 65537, 4294967294, 4294967295,
		4294967296, 4294967297, 18446744073709551615,
	}

	for _, value := range values {
		t.Run(fmt.Sprintf("%d", value), func(t *testing.T) {
			t.Parallel()

			encoded := bt.VarInt(value).Bytes()

			var v bt.VarInt

			n, err := v.ReadFrom(bytes.NewReader(encoded))

			require.NoError(t, err)
			require.Equal(t, value, uint64(v))
			require.Equal(t, int64(len(encoded)), n)
			require.Len(t, encoded, bt.VarInt(value).Length())
		})
	}
}

// TestVarIntReadFromRejectsEveryOverlongWidth walks every value/width pair and
// asserts the reader accepts exactly one width per value — the one the writers
// emit. This pins checkMinimal against Length/Bytes rather than against a
// hand-written table of thresholds.
func TestVarIntReadFromRejectsEveryOverlongWidth(t *testing.T) {
	t.Parallel()

	values := []uint64{0, 1, 252, 253, 65535, 65536, 4294967295, 4294967296}
	widths := map[int]byte{3: 0xfd, 5: 0xfe, 9: 0xff}

	for _, value := range values {
		minimal := bt.VarInt(value).Length()

		for width, discriminant := range widths {
			if width == minimal {
				continue
			}

			if width < minimal {
				// Cannot represent the value at all; not a minimality question.
				continue
			}

			encoded := make([]byte, width)
			encoded[0] = discriminant

			for i := 0; i < width-1; i++ {
				encoded[1+i] = byte(value >> (8 * i))
			}

			var v bt.VarInt

			_, err := v.ReadFrom(bytes.NewReader(encoded))
			require.ErrorIsf(t, err, bt.ErrNonMinimalVarInt,
				"value %d in %d bytes should be rejected (minimal is %d)", value, width, minimal)
		}
	}
}

// TestTxReadFromRejectsNonMinimalInputCount is the reason this check belongs in
// the reader rather than in each consumer.
//
// Before this change go-bt accepted the over-long input-count prefix and then
// re-serialized it canonically, so the parsed transaction was byte-identical to
// the canonical one and carried the SAME txid. Nothing downstream — not the txid,
// not a merkle root built from it — could tell that bytes SV Node rejects at
// parse had been accepted. Only the parse can see it.
func TestTxReadFromRejectsNonMinimalInputCount(t *testing.T) {
	t.Parallel()

	canonical, err := hex.DecodeString(canonicalTxHex)
	require.NoError(t, err)

	// Sanity: the canonical form parses, and its input count really is at offset 4.
	good, err := bt.NewTxFromBytes(canonical)
	require.NoError(t, err)
	require.Len(t, good.Inputs, 1)
	require.Equal(t, byte(0x01), canonical[4])

	// Rewrite the 1-byte input count `01` as the 3-byte form `fd 01 00`.
	nonMinimal := make([]byte, 0, len(canonical)+2)
	nonMinimal = append(nonMinimal, canonical[:4]...)
	nonMinimal = append(nonMinimal, 0xfd, 0x01, 0x00)
	nonMinimal = append(nonMinimal, canonical[5:]...)

	require.Len(t, nonMinimal, len(canonical)+2)

	_, err = bt.NewTxFromBytes(nonMinimal)
	require.Error(t, err)
	require.ErrorIs(t, err, bt.ErrNonMinimalVarInt)
}

// TestTxReadFromRejectsNonMinimalScriptLength covers a length prefix nested
// inside an input rather than a top-level count, so the rejection does not
// depend on which varint in the transaction was rewritten.
func TestTxReadFromRejectsNonMinimalScriptLength(t *testing.T) {
	t.Parallel()

	canonical, err := hex.DecodeString(canonicalTxHex)
	require.NoError(t, err)

	// version(4) + inputCount(1) + prevTxID(32) + prevIndex(4) = 41
	const scriptLenOffset = 41

	scriptLen := canonical[scriptLenOffset]
	require.Less(t, scriptLen, byte(0xfd), "fixture's unlocking script must use a 1-byte length")

	nonMinimal := make([]byte, 0, len(canonical)+2)
	nonMinimal = append(nonMinimal, canonical[:scriptLenOffset]...)
	nonMinimal = append(nonMinimal, 0xfd, scriptLen, 0x00)
	nonMinimal = append(nonMinimal, canonical[scriptLenOffset+1:]...)

	_, err = bt.NewTxFromBytes(nonMinimal)
	require.Error(t, err)
	require.ErrorIs(t, err, bt.ErrNonMinimalVarInt)
}

// TestErrNonMinimalVarIntIsMatchable pins the sentinel as part of the public
// contract: consumers need to tell a malformed encoding apart from a truncated
// or otherwise unreadable stream.
func TestErrNonMinimalVarIntIsMatchable(t *testing.T) {
	t.Parallel()

	var v bt.VarInt

	_, err := v.ReadFrom(bytes.NewReader([]byte{0xfd, 0x01, 0x00}))
	require.ErrorIs(t, err, bt.ErrNonMinimalVarInt)

	// A truncated prefix is a different failure and must not match.
	_, err = v.ReadFrom(bytes.NewReader([]byte{0xfd, 0x01}))
	require.Error(t, err)
	require.NotErrorIs(t, err, bt.ErrNonMinimalVarInt)
}
