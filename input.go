package bt

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/pkg/errors"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

/*
Field	                     Description                                                   Size
--------------------------------------------------------------------------------------------------------
Previous Transaction hash  doubled SHA256-hashed of a (previous) to-be-used transaction	 32 bytes
Previous Txout-index       non-negative integer indexing an output of the to-be-used      4 bytes
                           transaction
Txin-script length         non-negative integer VI = VarInt                               1-9 bytes
Txin-script / scriptSig	   Script	                                                        <in-script length>-many bytes
sequence_no	               normally 0xFFFFFFFF; irrelevant unless transaction's           4 bytes
                           lock_time is > 0
*/

// DefaultSequenceNumber is the default starting sequence number
const DefaultSequenceNumber uint32 = 0xFFFFFFFF

// Input is a representation of a transaction input
//
// DO NOT CHANGE ORDER - Optimized for memory via maligned
type Input struct {
	previousTxIDHash   *chainhash.Hash
	PreviousTxSatoshis uint64
	PreviousTxScript   *bscript.Script
	UnlockingScript    *bscript.Script
	PreviousTxOutIndex uint32
	SequenceNumber     uint32
}

// ReadFrom reads from the `io.Reader` into the `bt.Input`.
func (i *Input) ReadFrom(r io.Reader) (int64, error) {
	return i.readFrom(r, false)
}

// ReadFromExtended reads the `io.Reader` into the `bt.Input` when the reader is
// consuming an extended format transaction.
func (i *Input) ReadFromExtended(r io.Reader) (int64, error) {
	return i.readFrom(r, true)
}

// readFrom is a helper function that reads from the `io.Reader` into the `bt.Input`.
func (i *Input) readFrom(r io.Reader, extended bool) (int64, error) {
	return i.readFromWithArena(r, extended, nil)
}

// ReadFromWithArena reads from r into i (standard format) drawing the
// unlocking-script []byte from arena. See bt.Arena for lifetime contract.
// Existing ReadFrom is unchanged.
func (i *Input) ReadFromWithArena(r io.Reader, a *Arena) (int64, error) {
	return i.readFromWithArena(r, false, a)
}

// ReadFromExtendedWithArena is the extended-format counterpart to
// ReadFromWithArena. The PreviousTxScript byte slice is also drawn from
// arena.
func (i *Input) ReadFromExtendedWithArena(r io.Reader, a *Arena) (int64, error) {
	return i.readFromWithArena(r, true, a)
}

// readFromWithArena mirrors readFrom but routes script byte allocations
// through the supplied Arena. Fixed-size fields (previousTxID, prevIndex,
// sequence, prevSatoshis) use stack arrays — they are not the alloc hotspot.
func (i *Input) readFromWithArena(r io.Reader, extended bool, a *Arena) (int64, error) {
	*i = Input{}
	var bytesRead int64

	var previousTxID [32]byte
	n, err := io.ReadFull(r, previousTxID[:])
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "previousTxID(32): got %d bytes", n)
	}

	var prevIndex [4]byte
	n, err = io.ReadFull(r, prevIndex[:])
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "prevIndex(4): got %d bytes", n)
	}

	script, n64, err := readArenaScript(r, a, "unlockingScript")
	bytesRead += n64
	if err != nil {
		return bytesRead, err
	}

	var sequence [4]byte
	n, err = io.ReadFull(r, sequence[:])
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "sequence(4): got %d bytes", n)
	}

	i.previousTxIDHash, err = chainhash.NewHash(previousTxID[:])
	if err != nil {
		return bytesRead, errors.Wrap(err, "could not read hash")
	}
	i.PreviousTxOutIndex = binary.LittleEndian.Uint32(prevIndex[:])
	i.UnlockingScript = bscript.NewFromBytes(script)
	i.SequenceNumber = binary.LittleEndian.Uint32(sequence[:])

	if extended {
		var prevSatoshis [8]byte

		n, err = io.ReadFull(r, prevSatoshis[:])
		bytesRead += int64(n)
		if err != nil {
			return bytesRead, errors.Wrapf(err, "prevSatoshis(8): got %d bytes", n)
		}

		newScript, n64b, err := readArenaScript(r, a, "prevTxScript")
		bytesRead += n64b
		if err != nil {
			return bytesRead, err
		}

		i.PreviousTxSatoshis = binary.LittleEndian.Uint64(prevSatoshis[:])
		i.PreviousTxScript = bscript.NewFromBytes(newScript)
	}

	return bytesRead, nil
}

// PreviousTxIDAdd will add the supplied txID bytes to the Input
// if it isn't a valid transaction id an ErrInvalidTxID error will be returned.
func (i *Input) PreviousTxIDAdd(txIDHash *chainhash.Hash) error {
	if !IsValidTxID(txIDHash) {
		return ErrInvalidTxID
	}
	i.previousTxIDHash = txIDHash
	return nil
}

// PreviousTxIDAddStr will validate and add the supplied txID string to the Input,
// if it isn't a valid transaction id an ErrInvalidTxID error will be returned.
func (i *Input) PreviousTxIDAddStr(txID string) error {
	hash, err := chainhash.NewHashFromStr(txID)
	if err != nil {
		return err
	}
	return i.PreviousTxIDAdd(hash)
}

// PreviousTxID will return the PreviousTxID if set.
func (i *Input) PreviousTxID() []byte {
	return i.previousTxIDHash.CloneBytes()
}

// PreviousTxIDStr returns the Previous TxID as a hex string.
func (i *Input) PreviousTxIDStr() string {
	return i.previousTxIDHash.String()
}

// PreviousTxIDChainHash returns the PreviousTxID as a chainhash.Hash.
func (i *Input) PreviousTxIDChainHash() *chainhash.Hash {
	return i.previousTxIDHash
}

// String implements the Stringer interface and returns a string
// representation of a transaction input.
func (i *Input) String() string {
	return fmt.Sprintf(
		`prevTxHash:   %s
prevOutIndex: %d
scriptLen:    %d
script:       %s
sequence:     %x
`,
		i.previousTxIDHash.String(),
		i.PreviousTxOutIndex,
		len(*i.UnlockingScript),
		i.UnlockingScript,
		i.SequenceNumber,
	)
}

// WriteTo writes the serialized Input directly to w without allocating
// an intermediate byte slice. It writes the standard (non-extended) format.
func (i *Input) WriteTo(w io.Writer) (int64, error) {
	p := newPartWriter(w)
	defer p.release()

	i.writeTo(p)

	return p.finish()
}

// writeTo appends this input in standard format to an in-progress transaction
// serialisation.
func (i *Input) writeTo(p *partWriter) {
	if i.previousTxIDHash != nil {
		p.raw(i.previousTxIDHash[:])
	}

	p.u32(i.PreviousTxOutIndex)

	if i.UnlockingScript == nil {
		p.varInt(0)
	} else {
		p.varInt(uint64(len(*i.UnlockingScript)))
		p.raw(*i.UnlockingScript)
	}

	p.u32(i.SequenceNumber)
}

// WriteExtendedTo writes the serialized Input in extended format directly to w.
// Extended format appends PreviousTxSatoshis and PreviousTxScript after the
// standard input fields.
func (i *Input) WriteExtendedTo(w io.Writer) (int64, error) {
	p := newPartWriter(w)
	defer p.release()

	i.writeExtendedTo(p)

	return p.finish()
}

// writeExtendedTo appends this input in extended format, which is the standard
// format followed by the satoshis and locking script of the output it spends.
func (i *Input) writeExtendedTo(p *partWriter) {
	i.writeTo(p)

	p.u64(i.PreviousTxSatoshis)

	if i.PreviousTxScript != nil {
		p.varInt(uint64(len(*i.PreviousTxScript)))
		p.raw(*i.PreviousTxScript)
	} else {
		p.varInt(0)
	}
}

// Size returns the serialized size of the Input in bytes without allocating.
func (i *Input) Size() int {
	// previousTxIDHash(32) + PreviousTxOutIndex(4) + SequenceNumber(4) = 40
	size := 40
	if i.UnlockingScript == nil {
		size += 1 // VarInt(0) = 1 byte
	} else {
		l := len(*i.UnlockingScript)
		size += VarInt(uint64(l)).Length() + l
	}
	return size
}

// ExtendedSize returns the serialized size of the Input in extended format
// without allocating: the standard fields plus PreviousTxSatoshis and
// PreviousTxScript (with a zero-length prev script encoded as a single byte). It
// returns uint64 to compose safely into Tx.ExtendedSize, whose total can exceed
// what an int holds on a 32-bit platform.
func (i *Input) ExtendedSize() uint64 {
	// standard fields + PreviousTxSatoshis(8)
	size := uint64(i.Size()) + 8

	if i.PreviousTxScript == nil {
		size++ // VarInt(0) = 1 byte
	} else {
		l := len(*i.PreviousTxScript)
		size += uint64(VarInt(uint64(l)).Length()) + uint64(l)
	}

	return size
}

// Bytes encodes the Input into a hex byte array.
func (i *Input) Bytes(clearLockingScript bool, intoBytes ...[]byte) []byte {
	var h []byte
	if len(intoBytes) > 0 {
		h = intoBytes[0]
	} else {
		h = make([]byte, 0)
	}

	if i.previousTxIDHash != nil {
		h = append(h, i.previousTxIDHash.CloneBytes()...)
	}

	// this is optimized to avoid the memory allocation of LittleEndianBytes
	h = append(h, []byte{
		byte(i.PreviousTxOutIndex),
		byte(i.PreviousTxOutIndex >> 8),
		byte(i.PreviousTxOutIndex >> 16),
		byte(i.PreviousTxOutIndex >> 24),
	}...)

	if clearLockingScript {
		h = append(h, 0x00)
	} else {
		if i.UnlockingScript == nil {
			h = append(h, VarInt(0).Bytes()...)
		} else {
			h = append(h, VarInt(uint64(len(*i.UnlockingScript))).Bytes()...)
			h = append(h, *i.UnlockingScript...)
		}
	}

	// this is optimized to avoid the memory allocation of LittleEndianBytes
	return append(h, []byte{
		byte(i.SequenceNumber),
		byte(i.SequenceNumber >> 8),
		byte(i.SequenceNumber >> 16),
		byte(i.SequenceNumber >> 24),
	}...)
}

// appendTo appends the serialized input to h without allocating.
// Uses direct slice of previousTxIDHash instead of CloneBytes.
func (i *Input) appendTo(h []byte, clearLockingScript bool) []byte {
	if i.previousTxIDHash != nil {
		h = append(h, i.previousTxIDHash[:]...)
	}

	h = append(
		h,
		byte(i.PreviousTxOutIndex),
		byte(i.PreviousTxOutIndex>>8),
		byte(i.PreviousTxOutIndex>>16),
		byte(i.PreviousTxOutIndex>>24),
	)

	if clearLockingScript {
		h = append(h, 0x00)
	} else if i.UnlockingScript == nil {
		h = append(h, 0x00)
	} else {
		h = VarInt(uint64(len(*i.UnlockingScript))).AppendTo(h)
		h = append(h, *i.UnlockingScript...)
	}

	return append(
		h,
		byte(i.SequenceNumber),
		byte(i.SequenceNumber>>8),
		byte(i.SequenceNumber>>16),
		byte(i.SequenceNumber>>24),
	)
}

// appendExtendedTo appends the extended-format serialized input to h without allocating.
func (i *Input) appendExtendedTo(h []byte, clearLockingScript bool) []byte {
	h = i.appendTo(h, clearLockingScript)

	h = append(
		h,
		byte(i.PreviousTxSatoshis),
		byte(i.PreviousTxSatoshis>>8),
		byte(i.PreviousTxSatoshis>>16),
		byte(i.PreviousTxSatoshis>>24),
		byte(i.PreviousTxSatoshis>>32),
		byte(i.PreviousTxSatoshis>>40),
		byte(i.PreviousTxSatoshis>>48),
		byte(i.PreviousTxSatoshis>>56),
	)

	if i.PreviousTxScript != nil {
		l := uint64(len(*i.PreviousTxScript))
		h = VarInt(l).AppendTo(h)
		h = append(h, *i.PreviousTxScript...)
	} else {
		h = append(h, 0x00)
	}

	return h
}

// ExtendedBytes encodes the Input into a hex byte array, including the EF transaction format information.
func (i *Input) ExtendedBytes(clearLockingScript bool, intoBytes ...[]byte) []byte {
	h := i.Bytes(clearLockingScript, intoBytes...)
	h = append(h, []byte{
		byte(i.PreviousTxSatoshis),
		byte(i.PreviousTxSatoshis >> 8),
		byte(i.PreviousTxSatoshis >> 16),
		byte(i.PreviousTxSatoshis >> 24),
		byte(i.PreviousTxSatoshis >> 32),
		byte(i.PreviousTxSatoshis >> 40),
		byte(i.PreviousTxSatoshis >> 48),
		byte(i.PreviousTxSatoshis >> 56),
	}...)

	if i.PreviousTxScript != nil {
		l := uint64(len(*i.PreviousTxScript))
		h = append(h, VarInt(l).Bytes()...)
		h = append(h, *i.PreviousTxScript...)
	} else {
		h = append(h, 0x00) // The length of the script is zero
	}

	return h
}
