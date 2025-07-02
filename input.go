package bt

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/pkg/errors"
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
// DO NOT CHANGE ORDER - Optimised for memory via maligned
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
	*i = Input{}
	var bytesRead int64

	previousTxID := make([]byte, 32)
	n, err := io.ReadFull(r, previousTxID)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "previousTxID(32): got %d bytes", n)
	}

	prevIndex := make([]byte, 4)
	n, err = io.ReadFull(r, prevIndex)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "previousTxID(4): got %d bytes", n)
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
		return bytesRead, errors.Wrapf(err, "script(%d): got %d bytes", l, n)
	}

	sequence := make([]byte, 4)
	n, err = io.ReadFull(r, sequence)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, errors.Wrapf(err, "sequence(4): got %d bytes", n)
	}

	i.previousTxIDHash, err = chainhash.NewHash(previousTxID)
	if err != nil {
		return bytesRead, errors.Wrap(err, "could not read hash")
	}
	i.PreviousTxOutIndex = binary.LittleEndian.Uint32(prevIndex)
	i.UnlockingScript = bscript.NewFromBytes(script)
	i.SequenceNumber = binary.LittleEndian.Uint32(sequence)

	if extended {
		prevSatoshis := make([]byte, 8)
		var prevTxLockingScript bscript.Script

		n, err = io.ReadFull(r, prevSatoshis)
		bytesRead += int64(n)
		if err != nil {
			return bytesRead, errors.Wrapf(err, "prevSatoshis(8): got %d bytes", n)
		}

		// Read in the prevTxLockingScript
		var scriptLen VarInt
		n64b, err := scriptLen.ReadFrom(r)
		bytesRead += n64b
		if err != nil {
			return bytesRead, err
		}

		newScript := make([]byte, scriptLen)
		nRead, err := io.ReadFull(r, newScript)
		bytesRead += int64(nRead)
		if err != nil {
			return bytesRead, errors.Wrapf(err, "script(%d): got %d bytes", scriptLen.Length(), nRead)
		}

		prevTxLockingScript = *bscript.NewFromBytes(newScript)

		i.PreviousTxSatoshis = binary.LittleEndian.Uint64(prevSatoshis)
		i.PreviousTxScript = bscript.NewFromBytes(prevTxLockingScript)
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
