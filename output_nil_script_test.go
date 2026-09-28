package bt

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/sighash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A nil LockingScript behaves exactly like an empty one (&bscript.Script{}):
// the same size, bytes, hex, JSON, TxID and preimage, and no panic (#162).
func TestOutput_NilLockingScriptMatchesEmpty(t *testing.T) {
	t.Parallel()

	nilOut := &Output{Satoshis: 1000}
	emptyOut := &Output{Satoshis: 1000, LockingScript: &bscript.Script{}}

	type result struct {
		size  int
		bytes []byte
		hex   string
		str   string
		wrote []byte
	}
	get := func(o *Output) result {
		var buf bytes.Buffer
		_, err := o.WriteTo(&buf)
		require.NoError(t, err)
		return result{o.Size(), o.Bytes(), o.LockingScriptHexString(), o.String(), buf.Bytes()}
	}

	var got result
	require.NotPanics(t, func() { got = get(nilOut) })
	assert.Equal(t, get(emptyOut), got)
}

func TestTx_NilLockingScriptMatchesEmpty(t *testing.T) {
	t.Parallel()

	build := func(script *bscript.Script) *Tx {
		tx := NewTx()
		require.NoError(t, tx.From(
			"3c8edde27cb9a9132c22038dac4391496be9db16fd21351565cc1006966fdad5", 0,
			"76a914eb0bd5edba389198e73f8efabddfc61666969ff788ac", 2000))
		// Input.MarshalJSON needs an unlocking script; the nil being tested
		// here is the output's.
		tx.Inputs[0].UnlockingScript = &bscript.Script{}
		tx.Outputs = []*Output{{Satoshis: 1000, LockingScript: script}}
		return tx
	}
	nilTx, emptyTx := build(nil), build(&bscript.Script{})

	type result struct {
		size     int
		bytes    []byte
		txid     string
		json     []byte
		nodeJSON []byte
		preimage []byte
	}
	get := func(tx *Tx) result {
		j, err := json.Marshal(tx)
		require.NoError(t, err)
		nj, err := json.Marshal(tx.NodeJSON())
		require.NoError(t, err)
		p, err := tx.CalcInputPreimage(0, sighash.AllForkID)
		require.NoError(t, err)
		return result{tx.Size(), tx.Bytes(), tx.TxID(), j, nj, p}
	}

	var got result
	require.NotPanics(t, func() { got = get(nilTx) })
	assert.Equal(t, get(emptyTx), got)
}

// Script.IsData on a nil script reports false instead of panicking.
func TestScript_IsDataNil(t *testing.T) {
	t.Parallel()

	var s *bscript.Script
	require.NotPanics(t, func() { assert.False(t, s.IsData()) })
}
