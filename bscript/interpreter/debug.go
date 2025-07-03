package interpreter

// Debugger implements to enable debugging.
// If enabled, copies of state are provided to each of the functions on
// call.
//
// Each function is called during its stage of a thread lifecycle.
// A high level overview of this lifecycle is:
//
//	BeforeExecute
//	for step
//	   BeforeStep
//	   BeforeExecuteOpcode
//	   for each stack push
//	     BeforeStackPush
//	     AfterStackPush
//	   end for
//	   for each stack pop
//	     BeforeStackPop
//	     AfterStackPop
//	   end for
//	   AfterExecuteOpcode
//	   if end of script
//	     BeforeScriptChange
//	     AfterScriptChange
//	   end if
//	   if bip16 and end of final script
//	     BeforeStackPush
//	     AfterStackPush
//	   end if
//	   AfterStep
//	end for
//	AfterExecute
//	if success
//	  AfterSuccess
//	end if
//	if error
//	  AfterError
//	end if
type Debugger interface {
	AfterError(*State, error)
	AfterExecute(*State)
	AfterExecuteOpcode(*State)
	AfterScriptChange(*State)
	AfterStep(*State)
	AfterSuccess(*State)
	BeforeExecute(*State)
	BeforeExecuteOpcode(*State)
	BeforeScriptChange(*State)
	BeforeStep(*State)

	AfterStackPop(*State, []byte)
	AfterStackPush(*State, []byte)
	BeforeStackPop(*State)
	BeforeStackPush(*State, []byte)
}

type nopDebugger struct{}

// BeforeExecute is a no-op implementation of Debugger.BeforeExecute.
func (n *nopDebugger) BeforeExecute(*State) {}

// AfterExecute is a no-op implementation of Debugger.AfterExecute.
func (n *nopDebugger) AfterExecute(*State) {}

// BeforeStep is a no-op implementation of Debugger.BeforeStep.
func (n *nopDebugger) BeforeStep(*State) {}

// AfterStep is a no-op implementation of Debugger.AfterStep.
func (n *nopDebugger) AfterStep(*State) {}

// BeforeExecuteOpcode is a no-op implementation of Debugger.BeforeExecuteOpcode.
func (n *nopDebugger) BeforeExecuteOpcode(*State) {}

// AfterExecuteOpcode is a no-op implementation of Debugger.AfterExecuteOpcode.
func (n *nopDebugger) AfterExecuteOpcode(*State) {}

// BeforeScriptChange is a no-op implementation of Debugger.BeforeScriptChange.
func (n *nopDebugger) BeforeScriptChange(*State) {}

// AfterScriptChange is a no-op implementation of Debugger.AfterScriptChange.
func (n *nopDebugger) AfterScriptChange(*State) {}

// BeforeStackPush is a no-op implementation of Debugger.BeforeStackPush.
func (n *nopDebugger) BeforeStackPush(*State, []byte) {}

// AfterStackPush is a no-op implementation of Debugger.AfterStackPush.
func (n *nopDebugger) AfterStackPush(*State, []byte) {}

// BeforeStackPop is a no-op implementation of Debugger.BeforeStackPop.
func (n *nopDebugger) BeforeStackPop(*State) {}

// AfterStackPop is a no-op implementation of Debugger.AfterStackPop.
func (n *nopDebugger) AfterStackPop(*State, []byte) {}

// AfterSuccess is a no-op implementation of Debugger.AfterSuccess.
func (n *nopDebugger) AfterSuccess(*State) {}

// AfterError is a no-op implementation of Debugger.AfterError.
func (n *nopDebugger) AfterError(*State, error) {}
