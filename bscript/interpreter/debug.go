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
func (n *nopDebugger) BeforeExecute(*State) {
	// Intentionally left blank to disable debugging while satisfying the
	// Debugger interface.
}

// AfterExecute is a no-op implementation of Debugger.AfterExecute.
func (n *nopDebugger) AfterExecute(*State) {
	// Intentionally left blank; nopDebugger does not perform post-execution
	// actions.
}

// BeforeStep is a no-op implementation of Debugger.BeforeStep.
func (n *nopDebugger) BeforeStep(*State) {
	// Intentionally left blank to avoid step processing when debugging is
	// disabled.
}

// AfterStep is a no-op implementation of Debugger.AfterStep.
func (n *nopDebugger) AfterStep(*State) {
	// Intentionally left blank as no post-step behavior is required.
}

// BeforeExecuteOpcode is a no-op implementation of Debugger.BeforeExecuteOpcode.
func (n *nopDebugger) BeforeExecuteOpcode(*State) {
	// Intentionally left blank; opcode execution hooks are disabled.
}

// AfterExecuteOpcode is a no-op implementation of Debugger.AfterExecuteOpcode.
func (n *nopDebugger) AfterExecuteOpcode(*State) {
	// Intentionally left blank since opcode execution tracing is disabled.
}

// BeforeScriptChange is a no-op implementation of Debugger.BeforeScriptChange.
func (n *nopDebugger) BeforeScriptChange(*State) {
	// Intentionally left blank to skip script change notifications.
}

// AfterScriptChange is a no-op implementation of Debugger.AfterScriptChange.
func (n *nopDebugger) AfterScriptChange(*State) {
	// Intentionally left blank because no action is needed after a script
	// change when debugging is disabled.
}

// BeforeStackPush is a no-op implementation of Debugger.BeforeStackPush.
func (n *nopDebugger) BeforeStackPush(*State, []byte) {
	// Intentionally left blank; stack operations are not traced.
}

// AfterStackPush is a no-op implementation of Debugger.AfterStackPush.
func (n *nopDebugger) AfterStackPush(*State, []byte) {
	// Intentionally left blank since no post-push behavior is required.
}

// BeforeStackPop is a no-op implementation of Debugger.BeforeStackPop.
func (n *nopDebugger) BeforeStackPop(*State) {
	// Intentionally left blank; nopDebugger ignores stack pops.
}

// AfterStackPop is a no-op implementation of Debugger.AfterStackPop.
func (n *nopDebugger) AfterStackPop(*State, []byte) {
	// Intentionally left blank as stack pop events are not observed.
}

// AfterSuccess is a no-op implementation of Debugger.AfterSuccess.
func (n *nopDebugger) AfterSuccess(*State) {
	// Intentionally left blank because success events are not handled.
}

// AfterError is a no-op implementation of Debugger.AfterError.
func (n *nopDebugger) AfterError(*State, error) {
	// Intentionally left blank since error events are ignored.
}
