package bt

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
)

/*
General format (inside a block) of each output of a transaction - Txout
Field	                        Description	                                Size
-----------------------------------------------------------------------------------------------------
value                         non-negative integer giving the number of   8 bytes
                              Satoshis(BTC/10^8) to be transferred
Txout-script length           non-negative integer                        1 - 9 bytes VI = VarInt
Txout-script / scriptPubKey   Script                                      <out-script length>-many bytes
(lockingScript)

*/

// Output is a representation of a transaction output
type Output struct {
	Satoshis      uint64          `json:"satoshis"`
	LockingScript *bscript.Script `json:"locking_script"`
}

// ReadFrom reads from the `io.Reader` into the `bt.Output`.
func (o *Output) ReadFrom(r io.Reader) (int64, error) {
	*o = Output{}
	var bytesRead int64

	satoshis := make([]byte, 8)
	n, err := io.ReadFull(r, satoshis)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "satoshis(8): got %d bytes", n)
	}

	var l VarInt
	n64, err := l.ReadFrom(r)
	bytesRead += n64
	if err != nil {
		return bytesRead, err
	}

	script := make([]byte, l)
	n, err = io.ReadFull(r, script)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "lockingScript(%d): got %d bytes", l, n)
	}

	o.Satoshis = binary.LittleEndian.Uint64(satoshis)
	o.LockingScript = bscript.NewFromBytes(script)

	return bytesRead, nil
}

// LockingScriptHexString returns the locking script
// of an output encoded as a hex string.
func (o *Output) LockingScriptHexString() string {
	return hex.EncodeToString(*o.LockingScript)
}

func (o *Output) String() string {
	return fmt.Sprintf(`value:     %d
scriptLen: %d
script:    %s
`, o.Satoshis, len(*o.LockingScript), o.LockingScript)
}

// WriteTo writes the serialized Output directly to w without allocating
// an intermediate byte slice.
func (o *Output) WriteTo(w io.Writer) (int64, error) {
	var total int64

	// Satoshis (8 bytes LE)
	var buf [8]byte
	buf[0] = byte(o.Satoshis)
	buf[1] = byte(o.Satoshis >> 8)
	buf[2] = byte(o.Satoshis >> 16)
	buf[3] = byte(o.Satoshis >> 24)
	buf[4] = byte(o.Satoshis >> 32)
	buf[5] = byte(o.Satoshis >> 40)
	buf[6] = byte(o.Satoshis >> 48)
	buf[7] = byte(o.Satoshis >> 56)
	n, err := w.Write(buf[:])
	total += int64(n)
	if err != nil {
		return total, err
	}

	// LockingScript length (varint) + script bytes
	n64, err := VarInt(uint64(len(*o.LockingScript))).WriteTo(w)
	total += n64
	if err != nil {
		return total, err
	}
	n, err = w.Write(*o.LockingScript)
	total += int64(n)
	return total, err
}

// Size returns the serialized size of the Output in bytes without allocating.
func (o *Output) Size() int {
	// Satoshis(8)
	l := len(*o.LockingScript)
	return 8 + VarInt(uint64(l)).Length() + l
}

// Bytes encodes the Output into a byte array.
func (o *Output) Bytes(inBytes ...[]byte) []byte {
	var h []byte
	if len(inBytes) > 0 {
		h = inBytes[0]
		// append the satoshis without allocating a new slice
		// this is much faster as we do not need to malloc
		h = append(h, []byte{
			byte(o.Satoshis),
			byte(o.Satoshis >> 8),
			byte(o.Satoshis >> 16),
			byte(o.Satoshis >> 24),
			byte(o.Satoshis >> 32),
			byte(o.Satoshis >> 40),
			byte(o.Satoshis >> 48),
			byte(o.Satoshis >> 56),
		}...)
	} else {
		b := make([]byte, 8)
		binary.LittleEndian.PutUint64(b, o.Satoshis)

		h = make([]byte, 0, len(b)+9+len(*o.LockingScript))
		h = append(h, b...)
	}

	h = append(h, VarInt(uint64(len(*o.LockingScript))).Bytes()...)
	h = append(h, *o.LockingScript...)

	return h
}

// BytesForSigHash returns the proper serialization
// of an output to be hashed and signed (sighash).
func (o *Output) BytesForSigHash() []byte {
	buf := make([]byte, 0, 8+9+len(*o.LockingScript))

	satoshis := make([]byte, 8)
	binary.LittleEndian.PutUint64(satoshis, o.Satoshis)
	buf = append(buf, satoshis...)

	buf = append(buf, VarInt(uint64(len(*o.LockingScript))).Bytes()...)
	buf = append(buf, *o.LockingScript...)

	return buf
}

// NodeJSON returns a wrapped *bt.Output for marshaling/unmarshalling into a node output format.
//
// Marshaling usage example:
//
//	bb, err := json.Marshal(output.NodeJSON())
//
// Unmarshalling usage example:
//
//	output := &bt.Output{}
//	if err := json.Unmarshal(bb, output.NodeJSON()); err != nil {}
func (o *Output) NodeJSON() interface{} {
	return &nodeOutputWrapper{Output: o}
}
