package bt

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"io"
	"log"
	"sync/atomic"

	crypto "github.com/bsv-blockchain/go-sdk/primitives/hash"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

/*
General format of a Bitcoin transaction (inside a block)
--------------------------------------------------------
Field            Description                                                               Size

Version no	     currently 1	                                                           4 bytes

In-counter  	 positive integer VI = VarInt                                              1 - 9 bytes

list of Inputs	 the first input of the first transaction is also called "coinbase"        <in-counter>-many Inputs
                 (its content was ignored in earlier versions)

Out-counter    	 positive integer VI = VarInt                                              1 - 9 bytes

list of Outputs  the Outputs of the first transaction spend the mined                      <out-counter>-many Outputs
								 bitcoins for the block

lock_time        if non-zero and sequence numbers are < 0xFFFFFFFF: block height or        4 bytes
                 timestamp when transaction is final
--------------------------------------------------------
*/

// Tx wraps a bitcoin transaction
//
// DO NOT CHANGE ORDER - Optimized memory via malign
type Tx struct {
	Inputs   []*Input  `json:"inputs"`
	Outputs  []*Output `json:"outputs"`
	Version  uint32    `json:"version"`
	LockTime uint32    `json:"locktime"`
	extended bool

	// local cache of the txid
	txHash atomic.Pointer[chainhash.Hash]
}

// ExtendedTx wraps a *bt.Tx and provides a method to convert it back to *bt.Tx.
type ExtendedTx struct {
	*Tx
}

// ToTx returns the underlying *bt.Tx from the ExtendedTx.
func (etx *ExtendedTx) ToTx() *Tx {
	return etx.Tx
}

// Txs a collection of *bt.Tx.
type Txs []*Tx

// NewTx creates a new transaction object with default values.
func NewTx() *Tx {
	return &Tx{Version: 1, LockTime: 0, Inputs: make([]*Input, 0)}
}

// NewTxFromString takes a toBytesHelper string representation of a bitcoin transaction
// and returns a Tx object.
func NewTxFromString(str string) (*Tx, error) {
	bb, err := hex.DecodeString(str)
	if err != nil {
		return nil, err
	}

	return NewTxFromBytes(bb)
}

// NewTxFromBytes takes an array of bytes, constructs a Tx and returns it.
// This function assumes that the byte slice contains exactly 1 transaction.
func NewTxFromBytes(b []byte) (*Tx, error) {
	tx, used, err := NewTxFromStream(b)
	if err != nil {
		return nil, err
	}

	if used != len(b) {
		return nil, ErrNLockTimeLength
	}

	return tx, nil
}

// NewTxFromStream takes an array of bytes and constructs a Tx from it, returning the Tx and the bytes used.
// Despite the name, this is not reading a stream in the true sense: it is a byte slice that contains
// many transactions one after another.
func NewTxFromStream(b []byte) (*Tx, int, error) {
	tx := Tx{}

	bytesRead, err := tx.ReadFrom(bytes.NewReader(b))

	return &tx, int(bytesRead), err
}

// ReadFrom reads from the `io.Reader` into the `bt.Tx`.
func (tx *Tx) ReadFrom(r io.Reader) (int64, error) {
	*tx = Tx{}
	var bytesRead int64

	// Define n64 and err here to avoid linter complaining about shadowing variables.
	var n64 int64
	var err error

	version := make([]byte, 4)
	n, err := io.ReadFull(r, version)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, err
	}

	tx.Version = binary.LittleEndian.Uint32(version)

	var inputCount VarInt

	n64, err = inputCount.ReadFrom(r)
	bytesRead += n64
	if err != nil {
		return bytesRead, err
	}

	var outputCount VarInt
	locktime := make([]byte, 4)

	// ----------------------------------------------------------------------------------
	// If the inputCount is 0, we may be parsing an incomplete transaction, or we may be
	// both of these cases without needing to rewind (peek) the incoming stream of bytes.
	// ----------------------------------------------------------------------------------
	if inputCount == 0 {
		n64, err = outputCount.ReadFrom(r)
		bytesRead += n64
		if err != nil {
			return bytesRead, err
		}

		if outputCount == 0 {
			// Read in lock time
			n, err = io.ReadFull(r, locktime)
			bytesRead += int64(n)
			if err != nil {
				return bytesRead, err
			}

			if binary.BigEndian.Uint32(locktime) != 0xEF {
				tx.LockTime = binary.LittleEndian.Uint32(locktime)
				return bytesRead, nil
			}

			tx.extended = true

			n64, err = inputCount.ReadFrom(r)
			bytesRead += n64
			if err != nil {
				return bytesRead, err
			}
		}
	}
	// ----------------------------------------------------------------------------------
	// If we have not returned from the previous block of code, we will have detected
	// a sane transaction, and we will know if it is extended format or not.
	// We can now proceed with reading the rest of the transaction.
	// ----------------------------------------------------------------------------------

	// create Inputs
	for i := uint64(0); i < uint64(inputCount); i++ {
		input := &Input{}
		n64, err = input.readFrom(r, tx.extended)
		bytesRead += n64
		if err != nil {
			return bytesRead, err
		}
		tx.Inputs = append(tx.Inputs, input)
	}

	if inputCount > 0 || tx.extended {
		// Re-read the actual output count...
		n64, err = outputCount.ReadFrom(r)
		bytesRead += n64
		if err != nil {
			return bytesRead, err
		}
	}

	for i := uint64(0); i < uint64(outputCount); i++ {
		output := new(Output)
		n64, err = output.ReadFrom(r)
		bytesRead += n64
		if err != nil {
			return bytesRead, err
		}

		tx.Outputs = append(tx.Outputs, output)
	}

	n, err = io.ReadFull(r, locktime)
	bytesRead += int64(n)
	if err != nil {
		return bytesRead, err
	}
	tx.LockTime = binary.LittleEndian.Uint32(locktime)

	return bytesRead, nil
}

// ReadFrom txs from a block in a `bt.Txs`. This assumes a preceding varint detailing
// the total number of txs that the reader will provide.
func (tt *Txs) ReadFrom(r io.Reader) (int64, error) {
	var bytesRead int64

	var txCount VarInt
	n, err := txCount.ReadFrom(r)
	bytesRead += n
	if err != nil {
		return bytesRead, err
	}

	*tt = make([]*Tx, txCount)

	for i := uint64(0); i < uint64(txCount); i++ {
		tx := new(Tx)
		n, err := tx.ReadFrom(r)
		bytesRead += n
		if err != nil {
			return bytesRead, err
		}

		(*tt)[i] = tx
	}

	return bytesRead, nil
}

// HasDataOutputs returns true if the transaction has
// at least one data (OP_RETURN) output in it.
func (tx *Tx) HasDataOutputs() bool {
	for _, out := range tx.Outputs {
		if out.LockingScript.IsData() {
			return true
		}
	}
	return false
}

// InputIdx will return the input at the specified index.
//
// This will consume an overflow error and simply return nil if the input
// isn't found at the index.
func (tx *Tx) InputIdx(i int) *Input {
	if i > tx.InputCount()-1 {
		return nil
	}
	return tx.Inputs[i]
}

// OutputIdx will return the output at the specified index.
//
// This will consume an overflow error and simply return nil if the output
// isn't found at the index.
func (tx *Tx) OutputIdx(i int) *Output {
	if i > tx.OutputCount()-1 {
		return nil
	}
	return tx.Outputs[i]
}

// IsCoinbase determines if this transaction is a coinbase by
// checking if the tx input is a standard coinbase input.
func (tx *Tx) IsCoinbase() bool {
	if len(tx.Inputs) != 1 {
		return false
	}

	cbi := make([]byte, 32)

	if !bytes.Equal(tx.Inputs[0].PreviousTxID(), cbi) {
		return false
	}

	if tx.Inputs[0].PreviousTxOutIndex == DefaultSequenceNumber || tx.Inputs[0].SequenceNumber == DefaultSequenceNumber {
		return true
	}

	return false
}

// IsExtended checks if the transaction is in extended format.
func (tx *Tx) IsExtended() bool {
	if tx == nil || tx.Inputs == nil {
		return false
	}

	if tx.extended {
		return true
	}

	for _, input := range tx.Inputs {
		if input.PreviousTxScript == nil {
			return false
		}
	}

	return true
}

// SetExtended sets the extended flag for the transaction.
func (tx *Tx) SetExtended(extended bool) {
	tx.extended = extended
}

// ToExtendedTx converts the Tx to an ExtendedTx if it is already extended.
func (tx *Tx) ToExtendedTx() *ExtendedTx {
	if tx == nil {
		return nil
	}

	if !tx.IsExtended() {
		return nil
	}

	return &ExtendedTx{
		Tx: tx,
	}
}

// TxID returns the transaction ID of the transaction
// (which is also the transaction hash).
func (tx *Tx) TxID() string {
	return tx.TxIDChainHash().String()
}

// SetTxHash should only be used when the transaction hash is known and the transaction will not change,
// this can be used to optimize processes that depend on the txid and avoid recalculating it
func (tx *Tx) SetTxHash(hash *chainhash.Hash) {
	tx.txHash.Store(hash)
}

// TxIDChainHash returns the transaction ID as a chainhash.Hash.
func (tx *Tx) TxIDChainHash() *chainhash.Hash {
	txHash := tx.txHash.Load()
	if txHash != nil {
		return txHash
	}

	hash := chainhash.DoubleHashH(tx.Bytes())
	return &hash
}

// String encodes the transaction into a hex string.
func (tx *Tx) String() string {
	return hex.EncodeToString(tx.Bytes())
}

// IsValidTxID will check that the txid bytes are valid.
//
// A txid should be of 32 bytes length.
func IsValidTxID(txid *chainhash.Hash) bool {
	return txid != nil && len(txid) == 32
}

// Bytes encodes the transaction into a byte array.
// See https://chainquery.com/bitcoin-cli/decoderawtransaction
func (tx *Tx) Bytes() []byte {
	return tx.toBytesHelper(0, nil, false)
}

// ExtendedBytes outputs the transaction into a byte array in extended format
// (with PreviousTxSatoshis and PreviousTXScript included)
func (tx *Tx) ExtendedBytes() []byte {
	return tx.toBytesHelper(0, nil, true)
}

// SerializeBytes returns the serialized byte representation of the transaction.
// If the transaction is in extended format, it includes additional data such as
// PreviousTxSatoshis and PreviousTxScript.
func (tx *Tx) SerializeBytes() []byte {
	if tx.IsExtended() {
		return tx.ExtendedBytes()
	}

	return tx.Bytes()
}

// BytesWithClearedInputs encodes the transaction into a byte array but clears its Inputs first.
// This is used when signing transactions.
func (tx *Tx) BytesWithClearedInputs(index int, lockingScript []byte) []byte {
	return tx.toBytesHelper(index, lockingScript, false)
}

// CloneTx returns a clone of the tx by bytes
func (tx *Tx) CloneTx() *Tx {
	// Ignore erring as byte slice passed in is created from valid tx
	clone, err := NewTxFromBytes(tx.Bytes())
	if err != nil {
		log.Fatal(err)
	}

	for i, input := range tx.Inputs {
		clone.Inputs[i].PreviousTxSatoshis = input.PreviousTxSatoshis
		if input.PreviousTxScript != nil {
			clone.Inputs[i].PreviousTxScript = &bscript.Script{}
			*clone.Inputs[i].PreviousTxScript = *input.PreviousTxScript
		}
	}

	return clone
}

// Clone returns a clone of the tx
func (tx *Tx) Clone() *Tx {
	// Creating a new Tx from scratch is much faster than cloning from bytes
	// ~ 420ns/op vs. 2200ns/op of the above function in benchmarking
	// this matters as we clone txs a couple of times when verifying signatures
	clone := &Tx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
		Inputs:   make([]*Input, len(tx.Inputs)),
		Outputs:  make([]*Output, len(tx.Outputs)),
	}

	for i, input := range tx.Inputs {
		clone.Inputs[i] = &Input{
			previousTxIDHash:   (*chainhash.Hash)(input.previousTxIDHash[:]),
			PreviousTxSatoshis: input.PreviousTxSatoshis,
			PreviousTxOutIndex: input.PreviousTxOutIndex,
			SequenceNumber:     input.SequenceNumber,
		}
		if input.UnlockingScript != nil {
			clone.Inputs[i].UnlockingScript = bscript.NewFromBytes(*input.UnlockingScript)
		}
		if input.PreviousTxScript != nil {
			clone.Inputs[i].PreviousTxScript = bscript.NewFromBytes(*input.PreviousTxScript)
		}
	}

	for i, output := range tx.Outputs {
		clone.Outputs[i] = &Output{
			Satoshis: output.Satoshis,
		}
		if output.LockingScript != nil {
			clone.Outputs[i].LockingScript = bscript.NewFromBytes(*output.LockingScript)
		}
	}

	return clone
}

// ShallowClone returns a clone of the tx, but only clones the elements of the tx
// that are mutated in the signing process.
func (tx *Tx) ShallowClone() *Tx {
	// Creating a new Tx from scratch is much faster than cloning from bytes
	// ~ 420ns/op vs. 2200ns/op of the above function in benchmarking
	// this matters as we clone txs a couple of times when verifying signatures
	clone := &Tx{
		Version:  tx.Version,
		LockTime: tx.LockTime,
		Inputs:   make([]*Input, len(tx.Inputs)),
		Outputs:  make([]*Output, len(tx.Outputs)),
	}

	for i, input := range tx.Inputs {
		clone.Inputs[i] = &Input{
			previousTxIDHash:   (*chainhash.Hash)(input.previousTxIDHash[:]),
			PreviousTxSatoshis: input.PreviousTxSatoshis,
			PreviousTxOutIndex: input.PreviousTxOutIndex,
			SequenceNumber:     input.SequenceNumber,
		}
		if input.UnlockingScript != nil {
			clone.Inputs[i].UnlockingScript = input.UnlockingScript
		}
		if input.PreviousTxScript != nil {
			// previousTxScript needs to be cloned as it is mutated in the signing process
			clone.Inputs[i].PreviousTxScript = bscript.NewFromBytes(*input.PreviousTxScript)
		}
	}

	for i, output := range tx.Outputs {
		clone.Outputs[i] = &Output{
			Satoshis: output.Satoshis,
		}
		if output.LockingScript != nil {
			clone.Outputs[i].LockingScript = output.LockingScript
		}
	}

	return clone
}

// NodeJSON returns a wrapped *bt.Tx for marshaling/unmarshalling into a node tx format.
//
// Marshaling usage example:
//
//	bb, err := json.Marshal(tx.NodeJSON())
//
// Unmarshalling usage example:
//
//	tx := bt.NewTx()
//	if err := json.Unmarshal(bb, tx.NodeJSON()); err != nil {}
func (tx *Tx) NodeJSON() interface{} {
	return &nodeTxWrapper{Tx: tx}
}

// NodeJSON returns a wrapped bt.Txs for marshaling/unmarshalling into a node tx format.
//
// Marshaling usage example:
//
//	bb, err := json.Marshal(txs.NodeJSON())
//
// Unmarshalling usage example:
//
//	var txs bt.Txs
//	if err := json.Unmarshal(bb, txs.NodeJSON()); err != nil {}
func (tt *Txs) NodeJSON() interface{} {
	return (*nodeTxsWrapper)(tt)
}

// toBytesHelper encodes the transaction into a byte array.
func (tx *Tx) toBytesHelper(index int, lockingScript []byte, extended bool) []byte {
	h := make([]byte, 0, 1024)
	// this is faster than using LittleEndianBytes, since we do not malloc a new byte slice
	h = append(h, []byte{
		byte(tx.Version),
		byte(tx.Version >> 8),
		byte(tx.Version >> 16),
		byte(tx.Version >> 24),
	}...)

	if extended {
		h = append(h, []byte{0x00, 0x00, 0x00, 0x00, 0x00, 0xEF}...)
	}

	h = append(h, VarInt(uint64(len(tx.Inputs))).Bytes()...)

	for i, in := range tx.Inputs {
		if i == index && lockingScript != nil {
			h = append(h, VarInt(uint64(len(lockingScript))).Bytes()...)
			h = append(h, lockingScript...)
		} else {
			if extended {
				h = in.ExtendedBytes(lockingScript != nil, h)
			} else {
				h = in.Bytes(lockingScript != nil, h)
			}
		}
	}

	h = append(h, VarInt(uint64(len(tx.Outputs))).Bytes()...)
	for _, out := range tx.Outputs {
		h = out.Bytes(h)
	}

	return append(h, []byte{
		byte(tx.LockTime),
		byte(tx.LockTime >> 8),
		byte(tx.LockTime >> 16),
		byte(tx.LockTime >> 24),
	}...)
}

// TxSize contains the size breakdown of a transaction
// including the breakdown of data bytes vs. standard bytes.
// This information can be used when calculating fees.
type TxSize struct {
	// TotalBytes are the number of bytes for the entire tx.
	TotalBytes uint64
	// TotalStdBytes are the number of bytes for the tx minus the data bytes.
	TotalStdBytes uint64
	// TotalDataBytes is the size in bytes of the op_return / data outputs.
	TotalDataBytes uint64
}

// Size will return the size of tx in bytes.
func (tx *Tx) Size() int {
	return len(tx.Bytes())
}

// SizeWithTypes will return the size of tx in bytes
// and include the different data types (std/data/etc.).
func (tx *Tx) SizeWithTypes() *TxSize {
	totBytes := tx.Size()

	// calculate data outputs
	dataLen := 0
	for _, d := range tx.Outputs {
		if d.LockingScript.IsData() {
			dataLen += len(*d.LockingScript)
		}
	}
	return &TxSize{
		TotalBytes:     uint64(totBytes),           //nolint:gosec // G115 - totBytes is bounded by tx size
		TotalStdBytes:  uint64(totBytes - dataLen), //nolint:gosec // G115 - totBytes is bounded by tx size
		TotalDataBytes: uint64(dataLen),            //nolint:gosec // G115 - dataLen is bounded by tx size
	}
}

// EstimateSize will return the size of tx in bytes and will add 107 bytes
// to the unlocking script of any unsigned inputs (only P2PKH for now) found
// to give a final size estimate of the tx size.
func (tx *Tx) EstimateSize() (int, error) {
	tempTx, err := tx.estimatedFinalTx()
	if err != nil {
		return 0, err
	}

	return tempTx.Size(), nil
}

// EstimateSizeWithTypes will return the size of tx in bytes, including the
// different data types (std/data/etc.), and will add 107 bytes to the unlocking
// script of any unsigned inputs (only P2PKH for now) found to give a final size
// estimate of the tx size.
func (tx *Tx) EstimateSizeWithTypes() (*TxSize, error) {
	tempTx, err := tx.estimatedFinalTx()
	if err != nil {
		return nil, err
	}

	return tempTx.SizeWithTypes(), nil
}

// estimatedFinalTx will return a clone of the tx with all inputs
func (tx *Tx) estimatedFinalTx() (*Tx, error) {
	tempTx := tx.Clone()

	for i, in := range tempTx.Inputs {
		if in.PreviousTxScript == nil {
			return nil, fmt.Errorf("%w at index %d in order to calc expected UnlockingScript", ErrEmptyPreviousTxScript, i)
		}
		if !in.PreviousTxScript.IsP2PKH() && !in.PreviousTxScript.IsP2PKHInscription() {
			return nil, ErrUnsupportedScript
		}
		if in.UnlockingScript == nil || len(*in.UnlockingScript) == 0 {
			//nolint:lll // insert dummy p2pkh unlocking script (sig + pubkey)
			dummyUnlockingScript, _ := hex.DecodeString("4830450221009c13cbcbb16f2cfedc7abf3a4af1c3fe77df1180c0e7eee30d9bcc53ebda39da02207b258005f1bc3cf9dffa06edb358d6db2bcfc87f50516fac8e3f4686fc2a03df412103107feff22788a1fc8357240bf450fd7bca4bd45d5f8bac63818c5a7b67b03876")
			in.UnlockingScript = bscript.NewFromBytes(dummyUnlockingScript)
		}
	}
	return tempTx, nil
}

// TxFees is returned when CalculateFee is called and contains
// a breakdown of the fees including the total and the size breakdown of
// the tx in bytes.
type TxFees struct {
	// TotalFeePaid is the total number of fees this tx will pay.
	TotalFeePaid uint64
	// StdFeePaid is the amount of fee to cover the standard inputs and outputs, etc.
	StdFeePaid uint64
	// DataFeePaid is the amount of fee to cover the op_return data outputs.
	DataFeePaid uint64
}

// IsFeePaidEnough will calculate the fees that this transaction is paying
// including the individual fee types (std/data/etc.).
func (tx *Tx) IsFeePaidEnough(fees *FeeQuote) (bool, error) {
	expFeesPaid, err := tx.feesPaid(tx.SizeWithTypes(), fees)
	if err != nil {
		return false, err
	}
	totalInputSatoshis := tx.TotalInputSatoshis()
	totalOutputSatoshis := tx.TotalOutputSatoshis()

	if totalInputSatoshis < totalOutputSatoshis {
		return false, nil
	}

	actualFeePaid := totalInputSatoshis - totalOutputSatoshis
	return actualFeePaid >= expFeesPaid.TotalFeePaid, nil
}

// EstimateIsFeePaidEnough will calculate the fees that this transaction is paying
// including the individual fee types (std/data/etc.), and will add 107 bytes to the unlocking
// script of any unsigned inputs (only P2PKH for now) found to give a final size
// estimate of the tx size for fee calculation.
func (tx *Tx) EstimateIsFeePaidEnough(fees *FeeQuote) (bool, error) {
	tempTx, err := tx.estimatedFinalTx()
	if err != nil {
		return false, err
	}
	expFeesPaid, err := tempTx.feesPaid(tempTx.SizeWithTypes(), fees)
	if err != nil {
		return false, err
	}
	totalInputSatoshis := tempTx.TotalInputSatoshis()
	totalOutputSatoshis := tempTx.TotalOutputSatoshis()

	if totalInputSatoshis < totalOutputSatoshis {
		return false, nil
	}

	actualFeePaid := totalInputSatoshis - totalOutputSatoshis
	return actualFeePaid >= expFeesPaid.TotalFeePaid, nil
}

// EstimateFeesPaid will estimate how big the tx will be when finalized
// by estimating input unlocking scripts that have not yet been filled
// including the individual fee types (std/data/etc.).
func (tx *Tx) EstimateFeesPaid(fees *FeeQuote) (*TxFees, error) {
	size, err := tx.EstimateSizeWithTypes()
	if err != nil {
		return nil, err
	}
	return tx.feesPaid(size, fees)
}

// feesPaid will calculate the fees that this transaction is paying
func (tx *Tx) feesPaid(size *TxSize, fees *FeeQuote) (*TxFees, error) {
	// get fees
	stdFee, err := fees.Fee(FeeTypeStandard)
	if err != nil {
		return nil, err
	}
	dataFee, err := fees.Fee(FeeTypeData)
	if err != nil {
		return nil, err
	}

	txFees := &TxFees{
		StdFeePaid:  size.TotalStdBytes * uint64(stdFee.MiningFee.Satoshis) / uint64(stdFee.MiningFee.Bytes),    //nolint:gosec // G115 - satoshi values bounded by protocol
		DataFeePaid: size.TotalDataBytes * uint64(dataFee.MiningFee.Satoshis) / uint64(dataFee.MiningFee.Bytes), //nolint:gosec // G115 - satoshi values bounded by protocol
	}
	txFees.TotalFeePaid = txFees.StdFeePaid + txFees.DataFeePaid
	return txFees, nil
}

// estimateDeficit estimates the deficit of the transaction
func (tx *Tx) estimateDeficit(fees *FeeQuote) (uint64, error) {
	totalInputSatoshis := tx.TotalInputSatoshis()
	totalOutputSatoshis := tx.TotalOutputSatoshis()

	expFeesPaid, err := tx.EstimateFeesPaid(fees)
	if err != nil {
		return 0, err
	}

	if totalInputSatoshis > totalOutputSatoshis+expFeesPaid.TotalFeePaid {
		return 0, nil
	}

	return totalOutputSatoshis + expFeesPaid.TotalFeePaid - totalInputSatoshis, nil
}

// TxIDBytes returns the transaction ID of the transaction as bytes
// (which is also the transaction hash).
func (tx *Tx) TxIDBytes() []byte {
	return ReverseBytes(crypto.Sha256d(tx.Bytes()))
}
