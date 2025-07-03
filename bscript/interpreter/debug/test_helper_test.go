package debug_test

import (
	"encoding/hex"
	"testing"

	"github.com/bsv-blockchain/go-bt/v2/bscript"
	"github.com/bsv-blockchain/go-bt/v2/bscript/interpreter"
	"github.com/bsv-blockchain/go-bt/v2/bscript/interpreter/debug"
	"github.com/stretchr/testify/require"
)

type stateHistory struct {
	dstack  [][]string
	astack  [][]string
	opcodes []string
	entries []string
}

func parseScripts(t *testing.T, lHex, uHex string) (*bscript.Script, *bscript.Script) {
	t.Helper()
	l, err := bscript.NewFromHexString(lHex)
	require.NoError(t, err)
	u, err := bscript.NewFromHexString(uHex)
	require.NoError(t, err)
	return l, u
}

func runEngine(t *testing.T, lscript, uscript *bscript.Script, dbg debug.DefaultDebugger) error {
	t.Helper()
	return interpreter.NewEngine().Execute(
		interpreter.WithScripts(lscript, uscript),
		interpreter.WithAfterGenesis(),
		interpreter.WithDebugger(dbg),
	)
}

func snapshot(state *interpreter.State) []string {
	stack := make([]string, len(state.DataStack))
	for i, d := range state.DataStack {
		stack[i] = hex.EncodeToString(d)
	}
	return stack
}

func recordState(h *stateHistory, state *interpreter.State) {
	h.dstack = append(h.dstack, snapshot(state))
	h.opcodes = append(h.opcodes, state.Opcode().Name())
}
