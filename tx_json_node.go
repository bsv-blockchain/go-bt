package bt

import (
	"encoding/json"
	"errors"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
)

type nodeTxWrapper struct {
	*Tx
}

type nodeTxsWrapper Txs

type nodeOutputWrapper struct {
	*Output
}

type nodeTxJSON struct {
	Version  uint32            `json:"version"`
	LockTime uint32            `json:"locktime"`
	TxID     string            `json:"txid"`
	Hash     string            `json:"hash"`
	Size     int               `json:"size"`
	Hex      string            `json:"hex"`
	Inputs   []*nodeInputJSON  `json:"vin"`
	Outputs  []*nodeOutputJSON `json:"vout"`
}

type nodeInputJSON struct {
	ScriptSig *struct {
		Asm string `json:"asm"`
		Hex string `json:"hex"`
	} `json:"scriptSig,omitempty"`
	TxID               string `json:"txid"`
	Vout               uint32 `json:"vout"`
	Sequence           uint32 `json:"sequence"`
	PreviousTxSatoshis uint64 `json:"previousTxSatoshis,omitempty"`
	PreviousTxScript   *struct {
		Asm     string `json:"asm"`
		Hex     string `json:"hex"`
		ReqSigs int    `json:"reqSigs,omitempty"`
		Type    string `json:"type"`
	} `json:"previousTxScript,omitempty"`
}

type nodeOutputJSON struct {
	Value        float64 `json:"value"`
	Index        int     `json:"n"`
	ScriptPubKey *struct {
		Asm     string `json:"asm"`
		Hex     string `json:"hex"`
		ReqSigs int    `json:"reqSigs,omitempty"`
		Type    string `json:"type"`
	} `json:"scriptPubKey,omitempty"`
}

// MarshalJSON will marshal a transaction that has been marshaled with this library.
func (w *nodeTxWrapper) MarshalJSON() ([]byte, error) {
	if w == nil || w.Tx == nil {
		return nil, errors.New("tx is nil so cannot be marshaled")
	}
	tx := w.Tx
	oo := make([]*nodeOutputJSON, 0, len(tx.Outputs))
	for i, o := range tx.Outputs {
		out := &nodeOutputJSON{}
		if err := out.fromOutput(o); err != nil {
			return nil, err
		}
		out.Index = i
		oo = append(oo, out)
	}
	ii := make([]*nodeInputJSON, 0, len(tx.Inputs))
	for _, i := range tx.Inputs {
		in := &nodeInputJSON{}
		if err := in.fromInput(i); err != nil {
			return nil, err
		}
		ii = append(ii, in)
	}
	txj := nodeTxJSON{
		Version:  tx.Version,
		LockTime: tx.LockTime,
		Inputs:   ii,
		Outputs:  oo,
		TxID:     tx.TxID(),
		Hash:     tx.TxID(),
		Size:     len(tx.Bytes()),
		Hex:      tx.String(),
	}
	return json.Marshal(txj)
}

/*// UnmarshalJSON will unmarshall a transaction that has been marshaled with this library.
func (n *nodeTxWrapper) UnmarshalJSON(b []byte) error {
	tx := n.Tx

	var txj nodeTxJSON
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
	oo := make([]*Output, 0, len(txj.Outputs))
	for _, o := range txj.Outputs {
		out, err := o.toOutput()
		if err != nil {
			return err
		}
		oo = append(oo, out)
	}
	ii := make([]*Input, 0, len(txj.Inputs))
	for _, i := range txj.Inputs {
		in, err := i.toInput()
		if err != nil {
			return err
		}
		ii = append(ii, in)
	}
	tx.Inputs = ii
	tx.Outputs = oo
	tx.LockTime = txj.LockTime
	tx.Version = txj.Version
	return nil
}*/

// UnmarshalJSON will unmarshal a transaction that has been marshaled with this library.
func (w *nodeTxWrapper) UnmarshalJSON(b []byte) error {
	// Ensure we have a Tx to populate
	if w.Tx == nil {
		w.Tx = &Tx{}
	}

	var txj nodeTxJSON
	if err := json.Unmarshal(b, &txj); err != nil {
		return err
	}

	// Fast‑path: is the raw hex present
	if txj.Hex != "" {
		parsed, err := NewTxFromString(txj.Hex)
		if err != nil {
			return err
		}
		w.copyFrom(parsed) // safe deep‑copy; keeps the original pointer
		return nil
	}

	// Build Outputs
	outs := make([]*Output, 0, len(txj.Outputs))
	for _, o := range txj.Outputs {
		out, err := o.toOutput()
		if err != nil {
			return err
		}
		outs = append(outs, out)
	}

	// Build Inputs
	ins := make([]*Input, 0, len(txj.Inputs))
	for _, i := range txj.Inputs {
		in, err := i.toInput()
		if err != nil {
			return err
		}
		ins = append(ins, in)
	}

	tx := w.Tx
	tx.Inputs = ins
	tx.Outputs = outs
	tx.LockTime = txj.LockTime
	tx.Version = txj.Version

	return nil
}

// fromOutput converts an Output to a nodeOutputJSON.
func (o *nodeOutputJSON) fromOutput(out *Output) error {
	asm, err := out.LockingScript.ToASM()
	if err != nil {
		return err
	}
	addresses, err := out.LockingScript.Addresses()
	if err != nil {
		return err
	}

	*o = nodeOutputJSON{
		Value: float64(out.Satoshis) / 100000000,
		Index: 0,
		ScriptPubKey: &struct {
			Asm     string `json:"asm"`
			Hex     string `json:"hex"`
			ReqSigs int    `json:"reqSigs,omitempty"`
			Type    string `json:"type"`
		}{
			Asm:     asm,
			Hex:     out.LockingScriptHexString(),
			ReqSigs: len(addresses),
			Type:    out.LockingScript.ScriptType(),
		},
	}

	return nil
}

// toOutput converts a nodeOutputJSON to an Output.
func (o *nodeOutputJSON) toOutput() (*Output, error) {
	out := &Output{}
	s, err := bscript.NewFromHexString(o.ScriptPubKey.Hex)
	if err != nil {
		return nil, err
	}
	out.Satoshis = uint64(o.Value * 100000000)
	out.LockingScript = s
	return out, nil
}

// toInput converts a nodeInputJSON to an Input.
func (i *nodeInputJSON) toInput() (*Input, error) {
	input := &Input{}
	s, err := bscript.NewFromHexString(i.ScriptSig.Hex)
	if err != nil {
		return nil, err
	}

	input.UnlockingScript = s
	input.PreviousTxOutIndex = i.Vout
	input.SequenceNumber = i.Sequence
	if err = input.PreviousTxIDAddStr(i.TxID); err != nil {
		return nil, err
	}

	return input, nil
}

// fromInput converts an Input to a nodeInputJSON.
func (i *nodeInputJSON) fromInput(input *Input) error {
	asm, err := input.UnlockingScript.ToASM()
	if err != nil {
		return err
	}

	i.ScriptSig = &struct {
		Asm string `json:"asm"`
		Hex string `json:"hex"`
	}{
		Asm: asm,
		Hex: input.UnlockingScript.String(),
	}

	i.Vout = input.PreviousTxOutIndex
	i.Sequence = input.SequenceNumber
	i.TxID = input.PreviousTxIDStr()

	if input.PreviousTxSatoshis != 0 {
		i.PreviousTxSatoshis = input.PreviousTxSatoshis
	}

	if input.PreviousTxScript != nil {
		asm, err := input.PreviousTxScript.ToASM()
		if err != nil {
			return err
		}
		i.PreviousTxScript = &struct {
			Asm     string `json:"asm"`
			Hex     string `json:"hex"`
			ReqSigs int    `json:"reqSigs,omitempty"`
			Type    string `json:"type"`
		}{
			Asm:     asm,
			Hex:     input.PreviousTxScript.String(),
			ReqSigs: 1,
			Type:    input.PreviousTxScript.ScriptType(),
		}

	}

	return nil
}

// MarshalJSON will marshal a transaction that has been marshaled with this library.
func (nn nodeTxsWrapper) MarshalJSON() ([]byte, error) {
	txs := make([]*nodeTxWrapper, len(nn))
	for i, n := range nn {
		txs[i] = n.NodeJSON().(*nodeTxWrapper)
	}
	return json.Marshal(txs)
}

// UnmarshalJSON will unmarshal a transaction that has been marshaled with this library.
func (nn *nodeTxsWrapper) UnmarshalJSON(b []byte) error {
	var jj []json.RawMessage
	if err := json.Unmarshal(b, &jj); err != nil {
		return err
	}

	*nn = make(nodeTxsWrapper, 0)
	for _, j := range jj {
		tx := NewTx()
		if err := json.Unmarshal(j, tx.NodeJSON()); err != nil {
			return err
		}
		*nn = append(*nn, tx)
	}
	return nil
}

// MarshalJSON will marshal the Output to JSON.
func (n *nodeOutputWrapper) MarshalJSON() ([]byte, error) {
	oj := &nodeOutputJSON{}
	if err := oj.fromOutput(n.Output); err != nil {
		return nil, err
	}
	return json.Marshal(oj)
}

// UnmarshalJSON will unmarshal the Output from JSON.
func (n *nodeOutputWrapper) UnmarshalJSON(b []byte) error {
	oj := &nodeOutputJSON{}
	if err := json.Unmarshal(b, &oj); err != nil {
		return err
	}

	o, err := oj.toOutput()
	if err != nil {
		return err
	}

	*n.Output = *o

	return nil
}
