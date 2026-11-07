package interpreter

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// Records are immutable and execution-owned. A scalar handle preserves compact
// runtimeValue copies without any pointer into the factory's locals or frame.
type closureValueState struct {
	target   hir.SymbolID
	captures [4]int32
	count    uint8
}

func (r *executionState) createClosure(locals map[mir.LocalID]runtimeValue, op mir.Operation) (runtimeValue, error) {
	target := r.bySymbol[op.Symbol]
	if op.Type != hir.TypeClosureI32 || len(op.Operands) > 4 || target == nil || !target.ClosureTarget || int(target.ClosureCaptures) != len(op.Operands) || target.ResultType != hir.TypeI32 || len(target.ParameterTypes) != len(op.Operands)+1 {
		return runtimeValue{}, fmt.Errorf("invalid closure construction target %d", op.Symbol)
	}
	for _, typ := range target.ParameterTypes {
		if typ != hir.TypeI32 {
			return runtimeValue{}, fmt.Errorf("invalid closure target parameter")
		}
	}
	state := closureValueState{target: op.Symbol, count: target.ClosureCaptures}
	for i, operand := range op.Operands {
		value, err := evaluate(locals, operand)
		if err != nil {
			return runtimeValue{}, err
		}
		if value.kind != valueI32 || value.typ == hir.TypeBool {
			return runtimeValue{}, fmt.Errorf("invalid closure capture %d", i)
		}
		state.captures[i] = value.i32()
	}
	r.closures = append(r.closures, state)
	return runtimeValue{kind: valueClosure, typ: hir.TypeClosureI32, scalar: uint64(len(r.closures))}, nil
}

func (r *executionState) callClosure(locals map[mir.LocalID]runtimeValue, term mir.Terminator, depth int) (callCompletion, error) {
	if len(term.Arguments) != 2 || term.Callee != 0 || term.ForeignABI != "" {
		return callCompletion{}, fmt.Errorf("invalid closure call")
	}
	callable, err := evaluate(locals, term.Arguments[0])
	if err != nil {
		return callCompletion{}, err
	}
	if callable.kind != valueClosure || callable.typ != hir.TypeClosureI32 || callable.scalar == 0 || callable.scalar > uint64(len(r.closures)) {
		return callCompletion{}, fmt.Errorf("invalid closure callable")
	}
	// Copy before evaluating the argument or entering any other frame.
	state := r.closures[callable.scalar-1]
	argument, err := evaluate(locals, term.Arguments[1])
	if err != nil {
		return callCompletion{}, err
	}
	if argument.kind != valueI32 || argument.typ == hir.TypeBool {
		return callCompletion{}, fmt.Errorf("invalid closure value argument")
	}
	target := r.bySymbol[state.target]
	if target == nil || !target.ClosureTarget || target.ClosureCaptures != state.count || state.count > 4 {
		return callCompletion{}, fmt.Errorf("invalid closure invocation target")
	}
	var values [5]runtimeValue
	for i := uint8(0); i < state.count; i++ {
		values[i] = i32Value(state.captures[i])
	}
	values[state.count] = argument
	return r.call(target, values[:int(state.count)+1], depth+1)
}
