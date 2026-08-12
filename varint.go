package bt

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"
)

// VarInt (variable integer) is a field used in transaction data to indicate the number of
// upcoming fields, or the length of an upcoming field.
// See http://learnmeabitcoin.com/glossary/varint
type VarInt uint64

// NewVarIntFromBytes takes a byte array in VarInt format and returns the
// decoded unsigned integer value of the length, and it's size in bytes.
// See http://learnmeabitcoin.com/glossary/varint
//
// This function does NOT enforce minimal encoding: `fd 00 00` decodes to 0
// here, whereas (*VarInt).ReadFrom rejects it with ErrNonMinimalVarInt. The
// leniency is retained only because the (VarInt, int) signature has no error
// return, so rejecting would be a breaking API change; it is not a statement
// that the encoding is valid. Nothing on a transaction parse path uses this
// function — every count and length prefix is read through ReadFrom.
//
// Use (*VarInt).ReadFrom for anything decoding untrusted bytes.
func NewVarIntFromBytes(bb []byte) (VarInt, int) {
	switch bb[0] {
	case 0xff:
		return VarInt(binary.LittleEndian.Uint64(bb[1:9])), 9
	case 0xfe:
		return VarInt(binary.LittleEndian.Uint32(bb[1:5])), 5
	case 0xfd:
		return VarInt(binary.LittleEndian.Uint16(bb[1:3])), 3
	default:
		return VarInt(binary.LittleEndian.Uint16([]byte{bb[0], 0x00})), 1
	}
}

// Length return the length of the underlying byte representation of the `bt.VarInt`.
func (v VarInt) Length() int {
	if v < 253 {
		return 1
	}
	if v < 65536 {
		return 3
	}
	if v < 4294967296 {
		return 5
	}
	return 9
}

// Bytes take the underlying unsigned integer and return a byte array in VarInt format.
// See http://learnmeabitcoin.com/glossary/varint
func (v VarInt) Bytes() []byte {
	b := make([]byte, 9)
	if v < 0xfd {
		b[0] = byte(v)
		return b[:1]
	}
	if v < 0x10000 {
		b[0] = 0xfd
		binary.LittleEndian.PutUint16(b[1:3], uint16(v))
		return b[:3]
	}
	if v < 0x100000000 {
		b[0] = 0xfe
		binary.LittleEndian.PutUint32(b[1:5], uint32(v))
		return b[:5]
	}
	b[0] = 0xff
	binary.LittleEndian.PutUint64(b[1:9], uint64(v))
	return b
}

// AppendTo appends the VarInt encoding to dst and returns the extended slice.
// It never allocates when dst has sufficient capacity.
func (v VarInt) AppendTo(dst []byte) []byte {
	if v < 0xfd {
		return append(dst, byte(v))
	}
	if v < 0x10000 {
		return append(dst, 0xfd, byte(v), byte(v>>8))
	}
	if v < 0x100000000 {
		return append(dst, 0xfe, byte(v), byte(v>>8), byte(v>>16), byte(v>>24))
	}
	return append(dst, 0xff, byte(v), byte(v>>8), byte(v>>16), byte(v>>24),
		byte(v>>32), byte(v>>40), byte(v>>48), byte(v>>56))
}

// WriteTo writes the VarInt to w without allocating.
func (v VarInt) WriteTo(w io.Writer) (int64, error) {
	var buf [9]byte
	var n int
	if v < 0xfd {
		buf[0] = byte(v)
		n = 1
	} else if v < 0x10000 {
		buf[0] = 0xfd
		binary.LittleEndian.PutUint16(buf[1:3], uint16(v))
		n = 3
	} else if v < 0x100000000 {
		buf[0] = 0xfe
		binary.LittleEndian.PutUint32(buf[1:5], uint32(v))
		n = 5
	} else {
		buf[0] = 0xff
		binary.LittleEndian.PutUint64(buf[1:9], uint64(v))
		n = 9
	}
	written, err := w.Write(buf[:n])
	return int64(written), err
}

// checkMinimal returns ErrNonMinimalVarInt when v was read from a discriminant
// wider than the value needs, e.g. the value 1 written as `fd 01 00` rather than
// `01`.
//
// Length reports the width Bytes/AppendTo/WriteTo would emit for v, so it is by
// definition the shortest encoding — comparing against the width actually
// consumed is exactly the canonicality rule, and cannot drift from the writers.
func (v VarInt) checkMinimal(read int) error {
	if minimal := v.Length(); minimal != read {
		return fmt.Errorf("%w: %d read as %d bytes, minimal encoding is %d", ErrNonMinimalVarInt, uint64(v), read, minimal)
	}

	return nil
}

// ReadFrom reads the next varint from the io.Reader and assigned it to itself.
//
// A non-minimally encoded varint is rejected with ErrNonMinimalVarInt. Bitcoin
// and SV Node treat an over-long CompactSize prefix as a hard parse error, and
// accepting one here is worse than merely lenient: the over-long form decodes to
// the same value and is then RE-SERIALIZED canonically, so the resulting
// transaction has the canonical txid and every hash-based check downstream
// passes. Consumers cannot detect after the fact what they accepted, which makes
// the parse the only place the divergence is visible. Sibling readers in this
// ecosystem already enforce this — see go-wire's ReadVarInt.
func (v *VarInt) ReadFrom(r io.Reader) (int64, error) {
	b := make([]byte, 1)
	if _, err := io.ReadFull(r, b); err != nil {
		return 0, errors.Wrap(err, "could not read varint type")
	}

	switch b[0] {
	case 0xff:
		bb := make([]byte, 8)
		if n, err := io.ReadFull(r, bb); err != nil {
			return 9, errors.Wrapf(err, "varint(8): got %d bytes", n)
		}
		*v = VarInt(binary.LittleEndian.Uint64(bb))

		return 9, v.checkMinimal(9)

	case 0xfe:
		bb := make([]byte, 4)
		if n, err := io.ReadFull(r, bb); err != nil {
			return 5, errors.Wrapf(err, "varint(4): got %d bytes", n)
		}
		*v = VarInt(binary.LittleEndian.Uint32(bb))

		return 5, v.checkMinimal(5)

	case 0xfd:
		bb := make([]byte, 2)
		if n, err := io.ReadFull(r, bb); err != nil {
			return 3, errors.Wrapf(err, "varint(2): got %d bytes", n)
		}
		*v = VarInt(binary.LittleEndian.Uint16(bb))

		return 3, v.checkMinimal(3)

	default:
		*v = VarInt(binary.LittleEndian.Uint16([]byte{b[0], 0x00}))
		return 1, nil
	}
}

// UpperLimitInc returns true if a number is at the
// upper limit of a VarInt and will result in a VarInt
// length change if incremented. The value returned will
// indicate how many bytes will be increased if the length
//
//	is incremented. -1 will be returned when the upper limit
//
// of VarInt is reached.
func (v VarInt) UpperLimitInc() int {
	switch uint64(v) {
	case 252, 65535:
		return 2
	case 4294967295:
		return 4
	case 18446744073709551615:
		return -1
	}

	return 0
}
