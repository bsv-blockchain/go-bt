package bt

import (
	"bytes"

	crypto "github.com/bsv-blockchain/go-sdk/primitives/hash"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/sighash"
)

// defaultHex is used to fix a bug in the original client (see if statement in the CalcInputSignatureHash func)
var defaultHex = []byte{1, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

type sigHashFunc func(inputIdx uint32, shf sighash.Flag) ([]byte, error)

// SigHashCache holds pre-computed intermediate hash values used during
// signature hashing. When signing multiple inputs of the same transaction,
// these hashes are identical across inputs (for the common SighashAll case)
// and can be computed once instead of O(N) times.
//
// Create with tx.NewSigHashCache() and pass to CalcInputPreimageWithCache.
type SigHashCache struct {
	PrevOutHash  []byte
	SequenceHash []byte
	OutputsHash  []byte
}

// NewSigHashCache pre-computes the intermediate hashes for the modern
// (post-fork) sighash algorithm. The returned cache can be passed to
// CalcInputPreimageWithCache for each input, avoiding O(N²) work.
func (tx *Tx) NewSigHashCache() *SigHashCache {
	return &SigHashCache{
		PrevOutHash:  tx.PreviousOutHash(),
		SequenceHash: tx.SequenceHash(),
		OutputsHash:  tx.OutputsHash(-1),
	}
}

// sigStrat will decide which tx serialization to use.
// The legacy serialization will be used for txs pre-fork
// whereas the new serialization will be used for post-fork
// txs (and they should include the sighash_forkid flag).
func (tx *Tx) sigStrat(shf sighash.Flag) sigHashFunc {
	if shf.Has(sighash.ForkID) {
		return tx.CalcInputPreimage
	}
	return tx.CalcInputPreimageLegacy
}

// CalcInputSignatureHash serialized the transaction and returns the hash digest
// to be signed. BitCoin (SV) uses a different signature hashing algorithm
// after the UAHF fork for replay protection.
//
// see https://github.com/bitcoin-sv/bitcoin-sv/blob/master/doc/abc/replay-protected-sighash.md#digest-algorithm
func (tx *Tx) CalcInputSignatureHash(inputNumber uint32, sigHashFlag sighash.Flag) ([]byte, error) {
	sigHashFn := tx.sigStrat(sigHashFlag)
	buf, err := sigHashFn(inputNumber, sigHashFlag)
	if err != nil {
		return nil, err
	}

	// A bug in the original Satoshi client implementation means specifying
	// an index that is out of range results in a signature hash of 1 (as an
	// uint256 little endian).  The original intent appeared to be to
	// indicate failure, but unfortunately, it was never checked and thus is
	// treated as the actual signature hash.  This buggy behavior is now
	// part of the consensus and a hard fork would be required to fix it.
	//
	// Due to this, if the tx signature returned matches this special case value,
	// we skip the double hashing as to not interfere.
	if bytes.Equal(defaultHex, buf) {
		return buf, nil
	}

	return crypto.Sha256d(buf), nil
}

// CalcInputPreimage serializes the transaction based on the input index and the SIGHASH flag
// and returns the preimage before double hashing (SHA256d).
//
// see https://github.com/bitcoin-sv/bitcoin-sv/blob/master/doc/abc/replay-protected-sighash.md#digest-algorithm
func (tx *Tx) CalcInputPreimage(inputNumber uint32, sigHashFlag sighash.Flag) ([]byte, error) {
	in := tx.InputIdx(int(inputNumber))
	if in == nil {
		return nil, ErrInputNoExist
	}
	if in.previousTxIDHash == nil {
		return nil, ErrEmptyPreviousTxID
	}
	if in.PreviousTxScript == nil {
		return nil, ErrEmptyPreviousTxScript
	}

	var hashPreviousOuts, hashSequence, hashOutputs []byte

	if sigHashFlag&sighash.AnyOneCanPay == 0 {
		hashPreviousOuts = tx.PreviousOutHash()
	}
	if sigHashFlag&sighash.AnyOneCanPay == 0 &&
		(sigHashFlag&31) != sighash.Single &&
		(sigHashFlag&31) != sighash.None {
		hashSequence = tx.SequenceHash()
	}
	if (sigHashFlag&31) != sighash.Single && (sigHashFlag&31) != sighash.None {
		hashOutputs = tx.OutputsHash(-1)
	} else if (sigHashFlag&31) == sighash.Single && inputNumber < uint32(tx.OutputCount()) {
		hashOutputs = tx.OutputsHash(int32(inputNumber))
	}

	return tx.calcInputPreimage(in, sigHashFlag, hashPreviousOuts, hashSequence, hashOutputs)
}

// CalcInputPreimageWithCache is like CalcInputPreimage but uses pre-computed
// hashes from a SigHashCache. Use this when signing multiple inputs to avoid
// redundant O(N) hash computations per input.
func (tx *Tx) CalcInputPreimageWithCache(inputNumber uint32, sigHashFlag sighash.Flag, cache *SigHashCache) ([]byte, error) {
	in := tx.InputIdx(int(inputNumber))
	if in == nil {
		return nil, ErrInputNoExist
	}
	if in.previousTxIDHash == nil {
		return nil, ErrEmptyPreviousTxID
	}
	if in.PreviousTxScript == nil {
		return nil, ErrEmptyPreviousTxScript
	}

	var hashPreviousOuts, hashSequence, hashOutputs []byte

	if sigHashFlag&sighash.AnyOneCanPay == 0 {
		hashPreviousOuts = cache.PrevOutHash
	}
	if sigHashFlag&sighash.AnyOneCanPay == 0 &&
		(sigHashFlag&31) != sighash.Single &&
		(sigHashFlag&31) != sighash.None {
		hashSequence = cache.SequenceHash
	}
	if (sigHashFlag&31) != sighash.Single && (sigHashFlag&31) != sighash.None {
		hashOutputs = cache.OutputsHash
	} else if (sigHashFlag&31) == sighash.Single && inputNumber < uint32(tx.OutputCount()) {
		hashOutputs = tx.OutputsHash(int32(inputNumber))
	}

	return tx.calcInputPreimage(in, sigHashFlag, hashPreviousOuts, hashSequence, hashOutputs)
}

// calcInputPreimage is the internal implementation shared by CalcInputPreimage
// and CalcInputPreimageWithCache. All temp allocations are eliminated by using
// inline byte appends.
func (tx *Tx) calcInputPreimage(in *Input, sigHashFlag sighash.Flag,
	hashPreviousOuts, hashSequence, hashOutputs []byte,
) ([]byte, error) {
	scriptLen := len(*in.PreviousTxScript)
	// 4 (version) + 32+32 (hashPrevOuts+hashSeq) + 32+4 (outpoint) +
	// varint(scriptLen) + scriptLen + 8 (value) + 4 (nSeq) + 32 (hashOutputs) +
	// 4 (locktime) + 4 (sighashtype) = 156 + varint + scriptLen
	bufSize := 156 + VarInt(uint64(scriptLen)).Length() + scriptLen
	buf := make([]byte, 0, bufSize)

	// Version
	buf = append(buf,
		byte(tx.Version), byte(tx.Version>>8),
		byte(tx.Version>>16), byte(tx.Version>>24),
	)

	// Input previousOuts/nSequence (none/all, depending on flags)
	if hashPreviousOuts != nil {
		buf = append(buf, hashPreviousOuts...)
	} else {
		buf = append(buf, make([]byte, 32)...)
	}
	if hashSequence != nil {
		buf = append(buf, hashSequence...)
	} else {
		buf = append(buf, make([]byte, 32)...)
	}

	// outpoint (32-byte hash + 4-byte little endian)
	buf = append(buf, in.previousTxIDHash[:]...)
	buf = append(buf,
		byte(in.PreviousTxOutIndex), byte(in.PreviousTxOutIndex>>8),
		byte(in.PreviousTxOutIndex>>16), byte(in.PreviousTxOutIndex>>24),
	)

	// scriptCode of the input
	buf = VarInt(uint64(scriptLen)).AppendTo(buf)
	buf = append(buf, *in.PreviousTxScript...)

	// value of the output spent by this input (8-byte little endian)
	buf = append(buf,
		byte(in.PreviousTxSatoshis), byte(in.PreviousTxSatoshis>>8),
		byte(in.PreviousTxSatoshis>>16), byte(in.PreviousTxSatoshis>>24),
		byte(in.PreviousTxSatoshis>>32), byte(in.PreviousTxSatoshis>>40),
		byte(in.PreviousTxSatoshis>>48), byte(in.PreviousTxSatoshis>>56),
	)

	// nSequence of the input
	buf = append(buf,
		byte(in.SequenceNumber), byte(in.SequenceNumber>>8),
		byte(in.SequenceNumber>>16), byte(in.SequenceNumber>>24),
	)

	// Outputs (none/one/all, depending on flags)
	if hashOutputs != nil {
		buf = append(buf, hashOutputs...)
	} else {
		buf = append(buf, make([]byte, 32)...)
	}

	// LockTime
	buf = append(buf,
		byte(tx.LockTime), byte(tx.LockTime>>8),
		byte(tx.LockTime>>16), byte(tx.LockTime>>24),
	)

	// sighashType
	shf := uint32(sigHashFlag)
	buf = append(buf,
		byte(shf), byte(shf>>8),
		byte(shf>>16), byte(shf>>24),
	)

	return buf, nil
}

// CalcInputPreimageLegacy serializes the transaction based on the input index and the SIGHASH flag
// and returns the preimage before double hashing (SHA256d), in the legacy format.
//
// see https://wiki.bitcoinsv.io/index.php/Legacy_Sighash_Algorithm
func (tx *Tx) CalcInputPreimageLegacy(inputNumber uint32, shf sighash.Flag) ([]byte, error) {
	in := tx.InputIdx(int(inputNumber))
	if in == nil {
		return nil, ErrInputNoExist
	}
	if in.previousTxIDHash == nil {
		return nil, ErrEmptyPreviousTxID
	}
	if in.PreviousTxScript == nil {
		return nil, ErrEmptyPreviousTxScript
	}

	// The SigHashSingle signature type signs only the corresponding input
	// and output (the output with the same index number as the input).
	//
	// Since transactions can have more inputs than outputs, this means it
	// is improper to use SigHashSingle on input indices that don't have a
	// corresponding output.
	//
	// A bug in the original Satoshi client implementation means specifying
	// an index that is out of range results in a signature hash of 1 (as an
	// uint256 little endian).  The original intent appeared to be to
	// indicate failure, but unfortunately, it was never checked and thus is
	// treated as the actual signature hash.  This buggy behavior is now
	// part of the consensus and a hard fork would be required to fix it.
	//
	// Due to this, care must be taken by software that creates transactions
	// which make use of SigHashSingle because it can lead to an extremely
	// dangerous situation where the invalid inputs will end up signing a
	// hash of 1.  This in turn presents an opportunity for attackers to
	// cleverly construct transactions which can steal those coins provided
	// they can reuse signatures.
	if shf.HasWithMask(sighash.Single) && int(inputNumber) > len(tx.Outputs)-1 {
		return defaultHex, nil
	}

	txCopy := tx.ShallowClone()

	for i := range txCopy.Inputs {
		if i == int(inputNumber) {
			txCopy.Inputs[i].PreviousTxScript = tx.Inputs[inputNumber].PreviousTxScript
		} else {
			txCopy.Inputs[i].UnlockingScript = &bscript.Script{}
			txCopy.Inputs[i].PreviousTxScript = &bscript.Script{}
		}
	}

	if shf.HasWithMask(sighash.None) {
		txCopy.Outputs = txCopy.Outputs[0:0]
		for i := range txCopy.Inputs {
			if i != int(inputNumber) {
				txCopy.Inputs[i].SequenceNumber = 0
			}
		}
	} else if shf.HasWithMask(sighash.Single) {
		txCopy.Outputs = txCopy.Outputs[:inputNumber+1]
		for i := 0; i < int(inputNumber); i++ {
			txCopy.Outputs[i].Satoshis = 18446744073709551615 // -1 but underflowed
			txCopy.Outputs[i].LockingScript = &bscript.Script{}
		}

		for i := range txCopy.Inputs {
			if i != int(inputNumber) {
				txCopy.Inputs[i].SequenceNumber = 0
			}
		}
	}

	if shf&sighash.AnyOneCanPay != 0 {
		txCopy.Inputs = txCopy.Inputs[inputNumber : inputNumber+1]
	}

	// Estimate buffer size: version(4) + varint + N*(32+4+varint+script+4) + varint + outputs + locktime(4) + sighash(4)
	buf := make([]byte, 0, txCopy.Size()+8)

	// Version
	buf = append(buf,
		byte(tx.Version), byte(tx.Version>>8),
		byte(tx.Version>>16), byte(tx.Version>>24),
	)

	buf = VarInt(uint64(len(txCopy.Inputs))).AppendTo(buf)
	for _, in := range txCopy.Inputs {
		if in.previousTxIDHash != nil {
			buf = append(buf, in.previousTxIDHash[:]...)
		}

		buf = append(buf,
			byte(in.PreviousTxOutIndex), byte(in.PreviousTxOutIndex>>8),
			byte(in.PreviousTxOutIndex>>16), byte(in.PreviousTxOutIndex>>24),
		)

		buf = VarInt(uint64(len(*in.PreviousTxScript))).AppendTo(buf)
		buf = append(buf, *in.PreviousTxScript...)

		buf = append(buf,
			byte(in.SequenceNumber), byte(in.SequenceNumber>>8),
			byte(in.SequenceNumber>>16), byte(in.SequenceNumber>>24),
		)
	}

	buf = VarInt(uint64(len(txCopy.Outputs))).AppendTo(buf)
	for _, out := range txCopy.Outputs {
		buf = append(buf,
			byte(out.Satoshis), byte(out.Satoshis>>8),
			byte(out.Satoshis>>16), byte(out.Satoshis>>24),
			byte(out.Satoshis>>32), byte(out.Satoshis>>40),
			byte(out.Satoshis>>48), byte(out.Satoshis>>56),
		)

		buf = VarInt(uint64(len(*out.LockingScript))).AppendTo(buf)
		buf = append(buf, *out.LockingScript...)
	}

	// LockTime
	buf = append(buf,
		byte(tx.LockTime), byte(tx.LockTime>>8),
		byte(tx.LockTime>>16), byte(tx.LockTime>>24),
	)

	// sighash flag
	s := uint32(shf)
	buf = append(buf,
		byte(s), byte(s>>8),
		byte(s>>16), byte(s>>24),
	)

	return buf, nil
}

// OutputsHash returns a bytes slice of the requested output, used for generating
// the txs signature hash. If n is -1, it will create the byte slice from all outputs.
func (tx *Tx) OutputsHash(n int32) []byte {
	if n == -1 {
		// Pre-compute total size for all outputs
		size := 0
		for _, out := range tx.Outputs {
			size += out.Size()
		}
		buf := make([]byte, 0, size)
		for _, out := range tx.Outputs {
			buf = out.appendTo(buf)
		}
		return crypto.Sha256d(buf)
	}

	buf := make([]byte, 0, tx.Outputs[n].Size())
	buf = tx.Outputs[n].appendTo(buf)
	return crypto.Sha256d(buf)
}
