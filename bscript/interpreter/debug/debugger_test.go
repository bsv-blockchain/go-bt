package debug_test

import (
	"encoding/hex"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/bsv-blockchain/go-bt/v2/bscript/interpreter"
	"github.com/bsv-blockchain/go-bt/v2/bscript/interpreter/debug"
)

const (
	testSimpleScript  = "simple script"
	testComplexScript = "complex script"
	testErrorScript   = "error script"

	scriptHexComplex = "76a97ca8a687"

	opOP0         = "OP_0"
	opOP2         = "OP_2"
	opOP3         = "OP_3"
	opOP4         = "OP_4"
	opOP6         = "OP_6"
	opOP7         = "OP_7"
	opADD         = "OP_ADD"
	opDUP         = "OP_DUP"
	opEQUAL       = "OP_EQUAL"
	opEQUALVERIFY = "OP_EQUALVERIFY"
	opHASH160     = "OP_HASH160"
	opMUL         = "OP_MUL"
	opRIPEMD160   = "OP_RIPEMD160"
	opSHA256      = "OP_SHA256"
	opSWAP        = "OP_SWAP"

	hashB472 = "b472a266d0bd89c13706a4132ccfb16f7c3b9fcb"
)

func TestDebugger_BeforeExecute(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStack           []string
		expOpcode          string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStack:           []string{},
			expOpcode:          opOP4,
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStack:           []string{},
			expOpcode:          opOP0,
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStack:           []string{},
			expOpcode:          opOP4,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			var timesCalled int
			debugger := debug.NewDebugger()
			debugger.AttachBeforeExecute(func(state *interpreter.State) {
				timesCalled++
				assert.Equal(t, test.expStack, snapshot(state))
				assert.Equal(t, test.expOpcode, state.Opcode().Name())
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, 1, timesCalled)
		})
	}
}

func TestDebugger_BeforeStep(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06", "02", "03"},
				{"04", "06", "06"},
				{"04"},
				{"04", "02"},
				{"04", "02", "02"},
				{"04", "04"},
			},
			expOpcodes: []string{
				opOP4, opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{},
				{""},
				{"", ""},
				{"", hashB472},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
			},
			expOpcodes: []string{opOP0, opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07", "02", "03"},
				{"04", "07", "06"},
			},
			expOpcodes: []string{opOP4, opOP7, opOP2, opOP3, opMUL, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachBeforeStep(func(state *interpreter.State) {
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
		})
	}
}

func TestDebugger_AfterStep(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06", "02", "03"},
				{"04", "06", "06"},
				{"04"},
				{"04", "02"},
				{"04", "02", "02"},
				{"04", "04"},
				{"01"},
			},
			expOpcodes: []string{
				opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{"", ""},
				{"", hashB472},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
				{"01"},
			},
			expOpcodes: []string{opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07", "02", "03"},
				{"04", "07", "06"},
			},
			expOpcodes: []string{opOP7, opOP2, opOP3, opMUL, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachAfterStep(func(state *interpreter.State) {
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
		})
	}
}

func TestDebugger_BeforeExecuteOpcode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06", "02", "03"},
				{"04", "06", "06"},
				{"04"},
				{"04", "02"},
				{"04", "02", "02"},
				{"04", "04"},
			},
			expOpcodes: []string{
				opOP4, opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{},
				{""},
				{"", ""},
				{"", hashB472},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
			},
			expOpcodes: []string{opOP0, opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07", "02", "03"},
				{"04", "07", "06"},
			},
			expOpcodes: []string{opOP4, opOP7, opOP2, opOP3, opMUL, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachBeforeExecuteOpcode(func(state *interpreter.State) {
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
		})
	}
}

func TestDebugger_AfterExecuteOpcode(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06", "02", "03"},
				{"04", "06", "06"},
				{"04"},
				{"04", "02"},
				{"04", "02", "02"},
				{"04", "04"},
				{"01"},
			},
			expOpcodes: []string{
				opOP4, opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{"", ""},
				{"", hashB472},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
				{"01"},
			},
			expOpcodes: []string{opOP0, opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07", "02", "03"},
				{"04", "07", "06"},
			},
			expOpcodes: []string{opOP4, opOP7, opOP2, opOP3, opMUL},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachAfterExecuteOpcode(func(state *interpreter.State) {
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
		})
	}
}

func TestDebugger_BeforeScriptChange(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
		exptimesCalled     int
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04", "06"},
				{"01"},
			},
			expOpcodes:     []string{opOP6, opEQUAL},
			exptimesCalled: 2,
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{"01"},
			},
			expOpcodes:     []string{opOP0, opEQUAL},
			exptimesCalled: 2,
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04", "07"},
			},
			expOpcodes:     []string{opOP7},
			exptimesCalled: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{
				dstack:  make([][]string, 0),
				astack:  make([][]string, 0),
				opcodes: make([]string, 0),
			}

			debugger := debug.NewDebugger()

			var timesCalled int
			debugger.AttachBeforeScriptChange(func(state *interpreter.State) {
				timesCalled++
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
			assert.Equal(t, test.exptimesCalled, timesCalled)
		})
	}
}

func TestDebugger_AfterScriptChange(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
		exptimesCalled     int
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04", "06"},
				{"01"},
			},
			expOpcodes:     []string{opOP2, opEQUAL},
			exptimesCalled: 2,
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{"01"},
			},
			expOpcodes:     []string{opDUP, opEQUAL},
			exptimesCalled: 2,
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04", "07"},
			},
			expOpcodes:     []string{opOP2},
			exptimesCalled: 1,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{
				dstack:  make([][]string, 0),
				astack:  make([][]string, 0),
				opcodes: make([]string, 0),
			}

			debugger := debug.NewDebugger()

			var timesCalled int
			debugger.AttachAfterScriptChange(func(state *interpreter.State) {
				timesCalled++
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
			assert.Equal(t, test.exptimesCalled, timesCalled)
		})
	}
}

func TestDebugger_AfterExecution(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStack           []string
		expOpcode          string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStack:           []string{"01"},
			expOpcode:          opEQUAL,
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStack:           []string{"01"},
			expOpcode:          opEQUAL,
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStack:           []string{"04"},
			expOpcode:          opEQUALVERIFY,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			stack := make([]string, 0)
			var opcode string

			debugger := debug.NewDebugger()
			debugger.AttachAfterExecute(func(state *interpreter.State) {
				stack = append(stack, snapshot(state)...)
				opcode = state.Opcode().Name()
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStack, stack)
			assert.Equal(t, test.expOpcode, opcode)
		})
	}
}

func TestDebugger_AfterError(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStack           []string
		expOpcode          string
		expCalled          bool
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStack:           []string{"04"},
			expOpcode:          opEQUALVERIFY,
			expCalled:          true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			stack := make([]string, 0)
			var opcode string
			var called bool

			debugger := debug.NewDebugger()
			debugger.AttachAfterError(func(state *interpreter.State, _ error) {
				called = true
				stack = append(stack, snapshot(state)...)
				opcode = state.Opcode().Name()
			})

			_ = runEngine(t, lscript, uscript, debugger)

			// This produces an error... This needs to be reviewed in the future.
			// require.NoError(t, err)

			assert.Equal(t, test.expCalled, called)
			if called {
				assert.Equal(t, test.expStack, stack)
				assert.Equal(t, test.expOpcode, opcode)
			}
		})
	}
}

func TestDebugger_AfterSuccess(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStack           []string
		expOpcode          string
		expCalled          bool
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStack:           []string{},
			expOpcode:          opEQUAL,
			expCalled:          true,
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStack:           []string{},
			expOpcode:          opEQUAL,
			expCalled:          true,
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			stack := make([]string, 0)
			var opcode string
			var called bool

			debugger := debug.NewDebugger()
			debugger.AttachAfterSuccess(func(state *interpreter.State) {
				called = true
				stack = append(stack, snapshot(state)...)
				opcode = state.Opcode().Name()
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expCalled, called)
			if called {
				assert.Equal(t, test.expStack, stack)
				assert.Equal(t, test.expOpcode, opcode)
			}
		})
	}
}

func TestDebugger_BeforeStackPush(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
		expPushData        []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06"},
				{"04"},
				{"04"},
				{"04", "02"},
				{"04"},
				{},
			},
			expPushData: []string{"04", "06", "02", "03", "06", "01", "02", "02", "04", "01"},
			expOpcodes: []string{
				opOP4, opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{},
				{""},
				{""},
				{hashB472},
				{hashB472},
				{hashB472},
				{},
			},
			expPushData: []string{"", "", hashB472, "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hashB472, "01"},
			expOpcodes:  []string{opOP0, opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{},
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07"},
				{"04"},
			},
			expPushData: []string{"04", "07", "02", "03", "06", ""},
			expOpcodes:  []string{opOP4, opOP7, opOP2, opOP3, opMUL, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachBeforeStackPush(func(state *interpreter.State, data []byte) {
				recordState(history, state)
				history.entries = append(history.entries, hex.EncodeToString(data))
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
			assert.Equal(t, test.expPushData, history.entries)
		})
	}
}

func TestDebugger_AfterStackPush(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
		expPushData        []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04"},
				{"04", "06"},
				{"04", "06", "02"},
				{"04", "06", "02", "03"},
				{"04", "06", "06"},
				{"04", "01"},
				{"04", "02"},
				{"04", "02", "02"},
				{"04", "04"},
				{"01"},
			},
			expPushData: []string{"04", "06", "02", "03", "06", "01", "02", "02", "04", "01"},
			expOpcodes: []string{
				opOP4, opOP6,
				opOP2, opOP3, opMUL, opEQUALVERIFY,
				opOP2, opOP2, opADD, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{"", ""},
				{"", hashB472},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
				{"01"},
			},
			expPushData: []string{"", "", hashB472, "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hashB472, "01"},
			expOpcodes:  []string{opOP0, opDUP, opHASH160, opSWAP, opSHA256, opRIPEMD160, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04"},
				{"04", "07"},
				{"04", "07", "02"},
				{"04", "07", "02", "03"},
				{"04", "07", "06"},
				{"04", ""},
			},
			expPushData: []string{"04", "07", "02", "03", "06", ""},
			expOpcodes:  []string{opOP4, opOP7, opOP2, opOP3, opMUL, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachAfterStackPush(func(state *interpreter.State, data []byte) {
				recordState(history, state)
				history.entries = append(history.entries, hex.EncodeToString(data))
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
			assert.Equal(t, test.expPushData, history.entries)
		})
	}
}

func TestDebugger_BeforeStackPop(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04", "06", "02", "03"},
				{"04", "06", "02"},
				{"04", "06", "06"},
				{"04", "06"},
				{"04", "01"},
				{"04", "02", "02"},
				{"04", "02"},
				{"04", "04"},
				{"04"},
				{"01"},
			},
			expOpcodes: []string{
				opMUL, opMUL, opEQUALVERIFY, opEQUALVERIFY, opEQUALVERIFY,
				opADD, opADD, opEQUAL, opEQUAL, opEQUAL,
			},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{"", ""},
				{hashB472, ""},
				{hashB472, "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"},
				{hashB472, hashB472},
				{hashB472},
				{"01"},
			},
			expOpcodes: []string{opHASH160, opSHA256, opRIPEMD160, opEQUAL, opEQUAL, opEQUAL},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04", "07", "02", "03"},
				{"04", "07", "02"},
				{"04", "07", "06"},
				{"04", "07"},
				{"04", ""},
			},
			expOpcodes: []string{opMUL, opMUL, opEQUALVERIFY, opEQUALVERIFY, opEQUALVERIFY},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{
				dstack:  make([][]string, 0),
				astack:  make([][]string, 0),
				opcodes: make([]string, 0),
			}

			debugger := debug.NewDebugger()
			debugger.AttachBeforeStackPop(func(state *interpreter.State) {
				recordState(history, state)
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
		})
	}
}

func TestDebugger_AfterStackPop(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		lockingScriptHex   string
		unlockingScriptHex string
		expStackHistory    [][]string
		expOpcodes         []string
		expPopData         []string
	}{
		testSimpleScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5456",
			expStackHistory: [][]string{
				{"04", "06", "02"},
				{"04", "06"},
				{"04", "06"},
				{"04"},
				{"04"},
				{"04", "02"},
				{"04"},
				{"04"},
				{},
				{},
			},
			expOpcodes: []string{
				opMUL, opMUL, opEQUALVERIFY, opEQUALVERIFY, opEQUALVERIFY,
				opADD, opADD, opEQUAL, opEQUAL, opEQUAL,
			},
			expPopData: []string{"03", "02", "06", "06", "01", "02", "02", "04", "04", "01"},
		},
		testComplexScript: {
			lockingScriptHex:   scriptHexComplex,
			unlockingScriptHex: "00",
			expStackHistory: [][]string{
				{""},
				{hashB472},
				{hashB472},
				{hashB472},
				{},
				{},
			},
			expOpcodes: []string{opHASH160, opSHA256, opRIPEMD160, opEQUAL, opEQUAL, opEQUAL},
			expPopData: []string{"", "", "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", hashB472, hashB472, "01"},
		},
		testErrorScript: {
			lockingScriptHex:   "5253958852529387",
			unlockingScriptHex: "5457",
			expStackHistory: [][]string{
				{"04", "07", "02"},
				{"04", "07"},
				{"04", "07"},
				{"04"},
				{"04"},
			},
			expOpcodes: []string{opMUL, opMUL, opEQUALVERIFY, opEQUALVERIFY, opEQUALVERIFY},
			expPopData: []string{"03", "02", "06", "07", ""},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			lscript, uscript := parseScripts(t, test.lockingScriptHex, test.unlockingScriptHex)

			history := &stateHistory{}

			debugger := debug.NewDebugger()
			debugger.AttachAfterStackPop(func(state *interpreter.State, data []byte) {
				recordState(history, state)
				history.entries = append(history.entries, hex.EncodeToString(data))
			})

			_ = runEngine(t, lscript, uscript, debugger)

			assert.Equal(t, test.expStackHistory, history.dstack)
			assert.Equal(t, test.expOpcodes, history.opcodes)
			assert.Equal(t, test.expPopData, history.entries)
		})
	}
}
