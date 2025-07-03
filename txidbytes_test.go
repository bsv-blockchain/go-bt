package bt

import (
	"encoding/hex"
	"testing"
)

// TestTxIDBytes tests the TxIDBytes method of the Tx struct.
func TestTxIDBytes(t *testing.T) {
	rawTx := "02000000011ccba787d421b98904da3329b2c7336f368b62e89bc896019b5eadaa28145b9c000000004847304402205cc711985ce2a6d61eece4f9b6edd6337bad3b7eca3aa3ce59bc15620d8de2a80220410c92c48a226ba7d5a9a01105524097f673f31320d46c3b61d2378e6f05320041ffffffff01c0aff629010000001976a91418392a59fc1f76ad6a3c7ffcea20cfcb17bda9eb88ac00000000"
	tx, err := NewTxFromString(rawTx)
	if err != nil {
		t.Fatalf("failed to parse tx: %v", err)
	}

	txidBytes := tx.TxIDBytes()
	if len(txidBytes) != 32 {
		t.Errorf("TxIDBytes() should return 32 bytes, got %d", len(txidBytes))
	}

	txidChainHash := tx.TxIDChainHash()
	if txidChainHash == nil {
		t.Fatal("TxIDChainHash() returned nil")
	}
	reversed := ReverseBytes(txidChainHash[:])
	if hex.EncodeToString(txidBytes) != hex.EncodeToString(reversed) {
		t.Errorf("TxIDBytes() does not match reversed TxIDChainHash: got %x, want %x", txidBytes, reversed)
	}
}
