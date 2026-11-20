package atlasmap

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type closureTrace struct {
	functions map[hir.SymbolID]*mir.Function
	callers   map[hir.SymbolID][]closureCall
	bodies    map[hir.SymbolID]closureBody
	work      int
}

type closureCall struct {
	caller hir.SymbolID
	term   *mir.Terminator
}

// A return trace binds the callee's parameters to this exact call site.
type closureFrame struct {
	call   closureCall
	parent *closureFrame
}

type closureDefinition struct {
	op        *mir.Operation
	call      *mir.Terminator
	parameter int // One-based position, zero for other definitions.
}

type closureBody struct {
	locals  map[mir.LocalID]closureDefinition
	returns []mir.Operand
}

func newClosureTrace(program mir.Program, pkg hir.PackageID) closureTrace {
	trace := closureTrace{
		functions: make(map[hir.SymbolID]*mir.Function),
		callers:   make(map[hir.SymbolID][]closureCall),
		bodies:    make(map[hir.SymbolID]closureBody),
		work:      maxTestPlanClosureWork,
	}
	for i := range program.Functions {
		fn := &program.Functions[i]
		if fn.Span.Package == pkg {
			trace.functions[fn.Symbol] = fn
		}
	}
	for i := range program.Functions {
		fn := &program.Functions[i]
		for j := range fn.Blocks {
			term := &fn.Blocks[j].Terminator
			if term.Kind == mir.TerminatorCall && term.ForeignABI == "" && trace.functions[term.Callee] != nil {
				trace.callers[term.Callee] = append(trace.callers[term.Callee], closureCall{caller: fn.Symbol, term: term})
			}
		}
	}
	return trace
}

func (trace *closureTrace) body(symbol hir.SymbolID) closureBody {
	if body, ok := trace.bodies[symbol]; ok {
		return body
	}
	fn := trace.functions[symbol]
	if fn == nil {
		return closureBody{}
	}
	body := closureBody{locals: make(map[mir.LocalID]closureDefinition)}
	// The verifier requires one definition for each closure carrier. Ignore
	// multi-write ordinary locals even if their operations resemble carriers.
	define := func(local mir.LocalID, def closureDefinition) {
		if _, found := body.locals[local]; found {
			def = closureDefinition{}
		}
		body.locals[local] = def
	}
	for i, local := range fn.Parameters {
		define(local, closureDefinition{parameter: i + 1})
	}
	for j := range fn.Blocks {
		block := &fn.Blocks[j]
		for i := range block.Operations {
			op := &block.Operations[i]
			switch op.Kind {
			case mir.OperationClosureCreate, mir.OperationAssign, mir.OperationFieldRead, mir.OperationStructConstruct:
				define(op.DestinationLocal, closureDefinition{op: op})
			}
		}
		term := &block.Terminator
		switch term.Kind {
		case mir.TerminatorCall:
			define(term.DestinationLocal, closureDefinition{call: term})
		case mir.TerminatorClosureCall:
			define(term.DestinationLocal, closureDefinition{})
		case mir.TerminatorReturn:
			body.returns = append(body.returns, term.ReturnValue)
		}
	}
	trace.bodies[symbol] = body
	return body
}

func (trace *closureTrace) resolve(function hir.SymbolID, value mir.Operand, field string, frame *closureFrame, depth int) hir.SymbolID {
	if depth == maxTestPlanDepth || trace.work < 0 {
		return hir.NoSymbol
	}
	body := trace.body(function)
	for value.Kind == mir.OperandLocal {
		trace.work--
		if trace.work < 0 {
			break
		}
		def := body.locals[value.Local]
		switch {
		case def.op != nil:
			op := def.op
			switch op.Kind {
			case mir.OperationClosureCreate:
				if field == "" {
					return op.Symbol
				}
			case mir.OperationAssign:
				value = op.Operands[0]
				continue
			case mir.OperationFieldRead:
				if field == "" && op.Type == hir.TypeClosureI32 {
					field, value = op.Field, op.Operands[0]
					continue
				}
			case mir.OperationStructConstruct:
				trace.work -= len(op.Fields)
				if field != "" && trace.work >= 0 {
					if position := slices.Index(op.Fields, field); position >= 0 {
						field, value = "", op.Operands[position]
						continue
					}
				}
			}
		case def.call != nil:
			return trace.result(closureCall{caller: function, term: def.call}, field, frame, depth+1)
		case def.parameter != 0:
			return trace.parameter(function, def.parameter-1, field, frame, depth+1)
		}
		break
	}
	return hir.NoSymbol
}

func (trace *closureTrace) result(call closureCall, field string, parent *closureFrame, depth int) hir.SymbolID {
	if call.term.ForeignABI != "" {
		return hir.NoSymbol
	}
	callee := call.term.Callee
	frame := &closureFrame{call: call, parent: parent}
	target := hir.NoSymbol
	for _, value := range trace.body(callee).returns {
		resolved := trace.resolve(callee, value, field, frame, depth)
		if resolved == hir.NoSymbol || target != hir.NoSymbol && target != resolved {
			return hir.NoSymbol
		}
		target = resolved
	}
	return target
}

func (trace *closureTrace) parameter(function hir.SymbolID, position int, field string, frame *closureFrame, depth int) hir.SymbolID {
	if frame != nil {
		return trace.resolve(frame.call.caller, frame.call.term.Arguments[position], field, frame.parent, depth)
	}
	// An unqualified graph edge must hold under every observed caller. If
	// callers supply different targets, attaching either edge would attribute
	// one caller's test to the other caller's closure.
	target := hir.NoSymbol
	for _, call := range trace.callers[function] {
		resolved := trace.resolve(call.caller, call.term.Arguments[position], field, nil, depth)
		if resolved == hir.NoSymbol || target != hir.NoSymbol && target != resolved {
			return hir.NoSymbol
		}
		target = resolved
	}
	return target
}

// Split a disagreeing invocation by its verified callers. Extend the active
// stack at its outer end so argument lookup follows exactly that call chain.
func (trace *closureTrace) invocation(function hir.SymbolID, value mir.Operand, sources map[hir.SymbolID]string, record func(hir.SymbolID, *closureFrame) bool) bool {
	var frame *closureFrame
	var visit func(hir.SymbolID, **closureFrame, int) bool
	visit = func(outer hir.SymbolID, next **closureFrame, depth int) bool {
		trace.work--
		if trace.work < 0 || sources[outer] == "" {
			return false
		}
		target := trace.resolve(function, value, "", frame, 0)
		if target != hir.NoSymbol {
			trace.work -= depth
			return trace.work >= 0 && record(target, frame)
		}
		if trace.work < 0 || depth == maxTestPlanDepth-1 {
			return false
		}
		callers := trace.callers[outer]
		complete := len(callers) != 0
		for _, call := range callers {
			caller := closureFrame{call: call}
			*next = &caller
			if !visit(call.caller, &caller.parent, depth+1) {
				complete = false
			}
			if trace.work < 0 {
				break
			}
		}
		*next = nil
		return complete
	}
	return visit(function, &frame, 0)
}
