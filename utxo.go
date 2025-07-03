package bt

import (
	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/chainhash"
)

// UTXO an unspent transaction output, used for creating inputs
type UTXO struct {
	TxIDHash       *chainhash.Hash `json:"txid"`
	Vout           uint32          `json:"vout"`
	LockingScript  *bscript.Script `json:"locking_script"`
	Satoshis       uint64          `json:"satoshis"`
	SequenceNumber uint32          `json:"sequence_number"`
	Unlocker       *Unlocker       `json:"-"`
}

// UTXOs a collection of *bt.UTXO.
type UTXOs []*UTXO

// NodeJSON returns a wrapped *bt.UTXO for marshaling/unmarshaling into a node utxo format.
//
// Marshaling usage example:
//
//	bb, err := json.Marshal(utxo.NodeJSON())
//
// Unmarshalling usage example:
//
//	utxo := &bt.UTXO{}
//	if err := json.Unmarshal(bb, utxo.NodeJSON()); err != nil {}
func (u *UTXO) NodeJSON() interface{} {
	return &nodeUTXOWrapper{UTXO: u}
}

// NodeJSON returns a wrapped bt.UTXOs for marshaling/unmarshalling into a node utxo format.
//
// Marshaling usage example:
//
//	bb, err := json.Marshal(utxos.NodeJSON())
//
// Unmarshalling usage example:
//
//	var txs bt.UTXOs
//	if err := json.Unmarshal(bb, utxos.NodeJSON()); err != nil {}
func (u *UTXOs) NodeJSON() interface{} {
	return (*nodeUTXOsWrapper)(u)
}

// TxIDStr return the tx id as a string.
func (u *UTXO) TxIDStr() string {
	return u.TxIDHash.String()
}

// LockingScriptHexString return the locking script in hex format.
func (u *UTXO) LockingScriptHexString() string {
	return u.LockingScript.String()
}
