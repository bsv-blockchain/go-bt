package bt

import (
	"encoding/json"
	"slices"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
	"github.com/pkg/errors"
)

type txJSON struct {
	TxID     string    `json:"txid"`
	Hex      string    `json:"hex"`
	Inputs   []*Input  `json:"inputs"`
	Outputs  []*Output `json:"outputs"`
	Version  uint32    `json:"version"`
	LockTime uint32    `json:"lockTime"`
}

type inputJSON struct {
	UnlockingScript    string `json:"unlockingScript"`
	TxID               string `json:"txid"`
	Vout               uint32 `json:"vout"`
	Sequence           uint32 `json:"sequence"`
	PreviousTxSatoshis uint64 `json:"previousTxSatoshis,omitempty"`
	PreviousTxScript   string `json:"previousTxScript,omitempty"`
}

type outputJSON struct {
	Satoshis      uint64 `json:"satoshis"`
	LockingScript string `json:"lockingScript"`
}

// MarshalJSON will serialize a transaction to json.
func (tx *Tx) MarshalJSON() ([]byte, error) {
	if tx == nil {
		return nil, errors.Wrap(ErrTxNil, "cannot marshal tx")
	}
	return json.Marshal(txJSON{
		TxID:     tx.TxID(),
		Hex:      tx.String(),
		Inputs:   tx.Inputs,
		Outputs:  tx.Outputs,
		LockTime: tx.LockTime,
		Version:  tx.Version,
	})
}

/*
// UnmarshalJSON will unmarshall a transaction that has been marshaled with this library.
func (tx *Tx) UnmarshalJSON(b []byte) error {
	var txj txJSON
	if err := json.Unmarshal(b, &txj); err != nil {
		return err
	}
	// quick convert
	if txj.Hex != "" {
		t, err := NewTxFromString(txj.Hex)
		if err != nil {
			return err
		}
		*tx = *t //nolint:govet // this needs to be refactored to use a constructor
		return nil
	}
	tx.LockTime = txj.LockTime
	tx.Version = txj.Version
	return nil
}
*/

// UnmarshalJSON will unmarshal a transaction that has been marshaled with this library.
func (tx *Tx) UnmarshalJSON(b []byte) error {
	var txj txJSON
	if err := json.Unmarshal(b, &txj); err != nil {
		return err
	}

	// fast path: raw hex
	if txj.Hex != "" {
		parsed, err := NewTxFromString(txj.Hex)
		if err != nil {
			return err
		}
		tx.copyFrom(parsed)
		return nil
	}

	// fallback path
	tx.Version = txj.Version
	tx.LockTime = txj.LockTime
	return nil
}

// copyFrom deep-copies the contents of src into tx, avoiding slice aliasing.
// NOTE: Ensure the calling file's import list includes: import "slices" (Go 1.21+)
func (tx *Tx) copyFrom(src *Tx) {
	if src == nil {
		return
	}

	tx.Version = src.Version
	tx.LockTime = src.LockTime

	// deep copy slices
	tx.Inputs = slices.Clone(src.Inputs)
	tx.Outputs = slices.Clone(src.Outputs)

	// add additional deep-copy logic here for new fields if needed
}

// MarshalJSON will convert an input to json, expanding upon the
// input struct to add additional fields.
func (i *Input) MarshalJSON() ([]byte, error) {
	ij := &inputJSON{
		TxID:               i.previousTxIDHash.String(),
		Vout:               i.PreviousTxOutIndex,
		UnlockingScript:    i.UnlockingScript.String(),
		Sequence:           i.SequenceNumber,
		PreviousTxSatoshis: i.PreviousTxSatoshis,
	}

	if i.PreviousTxScript != nil {
		ij.PreviousTxScript = i.PreviousTxScript.String()
	}

	return json.Marshal(ij)
}

// UnmarshalJSON will convert a JSON input to an input.
func (i *Input) UnmarshalJSON(b []byte) error {
	var ij inputJSON
	if err := json.Unmarshal(b, &ij); err != nil {
		return err
	}
	ptxID, err := chainhash.NewHashFromStr(ij.TxID)
	if err != nil {
		return err
	}
	s, err := bscript.NewFromHexString(ij.UnlockingScript)
	if err != nil {
		return err
	}
	i.UnlockingScript = s
	i.previousTxIDHash = ptxID
	i.PreviousTxOutIndex = ij.Vout
	i.SequenceNumber = ij.Sequence

	if ij.PreviousTxSatoshis != 0 {
		i.PreviousTxSatoshis = ij.PreviousTxSatoshis
	}

	if ij.PreviousTxScript != "" {
		s, err := bscript.NewFromHexString(ij.PreviousTxScript)
		if err != nil {
			return err
		}
		i.PreviousTxScript = s
	}

	return nil
}

// MarshalJSON will serialize an output to json.
func (o *Output) MarshalJSON() ([]byte, error) {
	return json.Marshal(&outputJSON{
		Satoshis:      o.Satoshis,
		LockingScript: o.LockingScriptHexString(),
	})
}

// UnmarshalJSON will convert a json serialized output to a bt Output.
func (o *Output) UnmarshalJSON(b []byte) error {
	var oj outputJSON
	if err := json.Unmarshal(b, &oj); err != nil {
		return err
	}
	s, err := bscript.NewFromHexString(oj.LockingScript)
	if err != nil {
		return err
	}
	o.Satoshis = oj.Satoshis
	o.LockingScript = s
	return nil
}
