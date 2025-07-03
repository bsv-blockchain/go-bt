package bt_test

import (
	"context"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2"
	"github.com/bsv-blockchain/go-bt/v2/unlocker"
	wifpkg "github.com/libsv/go-bk/wif"
	"github.com/stretchr/testify/require"
)

// newTxWithInput creates a transaction with a single input.
//
//nolint:unparam // idx is kept for future flexibility
func newTxWithInput(t *testing.T, txID string, idx uint32, script string, satoshis uint64) *bt.Tx {
	t.Helper()
	tx := bt.NewTx()
	require.NoError(t, tx.From(txID, idx, script, satoshis))
	return tx
}

// signAllInputs signs all transaction inputs using the provided WIF key.
func signAllInputs(t *testing.T, tx *bt.Tx, wifStr string) {
	t.Helper()
	wif, err := wifpkg.DecodeWIF(wifStr)
	require.NoError(t, err)
	require.NoError(t, tx.FillAllInputs(context.Background(), &unlocker.Getter{PrivateKey: wif.PrivKey}))
}

const testWIF = "L3MhnEn1pLWcggeYLk9jdkvA2wUK1iWwwrGkBbgQRqv6HPCdRxuw"
