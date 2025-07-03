package bt

import (
	"encoding/json"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

type nodeUTXOWrapper struct {
	*UTXO
}

type nodeUTXOsWrapper UTXOs

type utxoJSON struct {
	TxID          string `json:"txid"`
	Vout          uint32 `json:"vout"`
	LockingScript string `json:"lockingScript"`
	Satoshis      uint64 `json:"satoshis"`
}

type utxoNodeJSON struct {
	TxID         string  `json:"txid"`
	Vout         uint32  `json:"vout"`
	ScriptPubKey string  `json:"scriptPubKey"`
	Amount       float64 `json:"amount"`
}

// UnmarshalJSON will convert a JSON serialized utxo to a bt.UTXO.
func (u *UTXO) UnmarshalJSON(body []byte) error {
	var j utxoJSON
	if err := json.Unmarshal(body, &j); err != nil {
		return err
	}

	txID, err := chainhash.NewHashFromStr(j.TxID)
	if err != nil {
		return err
	}

	lscript, err := bscript.NewFromHexString(j.LockingScript)
	if err != nil {
		return err
	}

	u.TxIDHash = txID
	u.LockingScript = lscript
	u.Vout = j.Vout
	u.Satoshis = j.Satoshis

	return nil
}

// MarshalJSON will serialize an utxo to JSON.
func (u *UTXO) MarshalJSON() ([]byte, error) {
	return json.Marshal(utxoJSON{
		TxID:          u.TxIDStr(),
		Satoshis:      u.Satoshis,
		Vout:          u.Vout,
		LockingScript: u.LockingScriptHexString(),
	})
}

// MarshalJSON will marshal a transaction that has been marshaled with this library.
func (n *nodeUTXOWrapper) MarshalJSON() ([]byte, error) {
	utxo := n.UTXO
	return json.Marshal(utxoNodeJSON{
		TxID:         utxo.TxIDStr(),
		Amount:       float64(utxo.Satoshis) / 100000000,
		ScriptPubKey: utxo.LockingScriptHexString(),
		Vout:         utxo.Vout,
	})
}

// UnmarshalJSON will unmarshal a transaction that has been marshaled with this library.
func (n *nodeUTXOWrapper) UnmarshalJSON(b []byte) error {
	var uj utxoNodeJSON
	if err := json.Unmarshal(b, &uj); err != nil {
		return err
	}

	txID, err := chainhash.NewHashFromStr(uj.TxID)
	if err != nil {
		return err
	}

	lscript, err := bscript.NewFromHexString(uj.ScriptPubKey)
	if err != nil {
		return err
	}

	n.Satoshis = uint64(uj.Amount * 100000000)
	n.Vout = uj.Vout
	n.LockingScript = lscript
	n.TxIDHash = txID

	return nil
}

// MarshalJSON will marshal a transaction that has been marshaled with this library.
func (nn nodeUTXOsWrapper) MarshalJSON() ([]byte, error) {
	utxos := make([]*nodeUTXOWrapper, len(nn))
	for i, n := range nn {
		utxos[i] = n.NodeJSON().(*nodeUTXOWrapper)
	}
	return json.Marshal(utxos)
}

// UnmarshalJSON will unmarshal a transaction that has been marshaled with this library.
func (nn *nodeUTXOsWrapper) UnmarshalJSON(b []byte) error {
	var jj []json.RawMessage
	if err := json.Unmarshal(b, &jj); err != nil {
		return err
	}

	*nn = make(nodeUTXOsWrapper, 0)
	for _, j := range jj {
		var utxo UTXO
		if err := json.Unmarshal(j, utxo.NodeJSON()); err != nil {
			return err
		}
		*nn = append(*nn, &utxo)
	}
	return nil
}
