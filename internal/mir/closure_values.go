package mir

import (
	"reflect"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyClosureConstruction(op Operation, functions map[hir.SymbolID]*Function, types map[LocalID]hir.Type) string {
	canonical := Operation{Kind: OperationClosureCreate, Entity: op.Entity, Span: op.Span, Symbol: op.Symbol, Type: hir.TypeClosureI32, DestinationLocal: op.DestinationLocal, Operands: op.Operands}
	if op.DestinationLocal < 0 || len(op.Operands) > 4 || !reflect.DeepEqual(op, canonical) {
		return "copied closure construction has invalid shape"
	}
	target := functions[op.Symbol]
	if target == nil || !target.ClosureTarget || int(target.ClosureCaptures) != len(op.Operands) || len(target.ParameterTypes) != len(op.Operands)+1 || target.ResultType != hir.TypeI32 {
		return "copied closure construction has no matching lifted target"
	}
	for _, value := range op.Operands {
		if !closureOperandMatches(value, hir.TypeI32, types) {
			return "copied closure capture is not an exact i32 value"
		}
	}
	return ""
}

func verifyClosureInvocation(term Terminator, types map[LocalID]hir.Type) string {
	canonical := Terminator{Kind: TerminatorClosureCall, Entity: term.Entity, Span: term.Span, DestinationLocal: term.DestinationLocal, Arguments: term.Arguments, Successors: term.Successors, UnwindSuccessor: NoBlock}
	if term.DestinationLocal < 0 || len(term.Arguments) != 2 || len(term.Successors) != 1 || !reflect.DeepEqual(term, canonical) {
		return "copied closure invocation requires one normal successor and two value operands"
	}
	if !closureOperandMatches(term.Arguments[0], hir.TypeClosureI32, types) || !closureOperandMatches(term.Arguments[1], hir.TypeI32, types) {
		return "copied closure invocation has a wrong callable or argument type"
	}
	return ""
}

func closureOperandMatches(value Operand, expected hir.Type, types map[LocalID]hir.Type) bool {
	if fieldStoreOperandType(value, types) != expected || value.Type != (hir.Type{}) && value.Type != expected {
		return false
	}
	switch value.Kind {
	case OperandLocal:
		return value.Local >= 0 && value.I32 == 0 && value.I64 == 0 && value.U64 == 0 && value.StaticData == ""
	case OperandI32:
		return expected == hir.TypeI32 && value.I64 == 0 && value.U64 == 0 && value.StaticData == ""
	default:
		return false
	}
}

// The target marker grants no authority by itself. Validate the whole pure
// integer body, including dead blocks, before a backend can bind its address.
func verifyClosureTarget(fn *Function) string {
	if !fn.ClosureTarget {
		if fn.ClosureCaptures != 0 {
			return "capture metadata is attached to a non-closure target"
		}
		return ""
	}
	if fn.ClosureCaptures > 4 || len(fn.ParameterTypes) != int(fn.ClosureCaptures)+1 || len(fn.Parameters) != len(fn.ParameterTypes) || fn.ResultType != hir.TypeI32 || len(fn.Effects) != 0 || len(fn.Resources) != 0 || len(fn.UnsafeOperations) != 0 || fn.DropGlueFor != (hir.Type{}) || len(fn.Calls) != 0 || len(fn.Loans) != 0 || len(fn.DropFlags) != 0 || len(fn.RegionValues) != 0 || len(fn.Reassignments) != 0 || len(fn.Moves) != 0 || len(fn.ReturnOrigins) != 0 || len(fn.RegionScopes) != 0 {
		return "copied closure target has an invalid signature or lifetime/effect facts"
	}
	for _, cleanup := range fn.CleanupPaths {
		if cleanup.ExitKind != "return" || len(cleanup.Actions) != 0 {
			return "copied closure target has executable cleanup"
		}
	}
	for _, typ := range fn.ParameterTypes {
		if typ != hir.TypeI32 {
			return "copied closure target parameters must be i32"
		}
	}
	writes := make(map[LocalID]bool, len(fn.Parameters))
	for _, local := range fn.Parameters {
		if local < 0 || writes[local] {
			return "copied closure target has invalid parameter locals"
		}
		writes[local] = true
	}
	types, err := fieldStoreLocalTypesMode(fn, nil, nil, true)
	if err != nil {
		return "copied closure target has conflicting local types"
	}
	for _, block := range fn.Blocks {
		for _, op := range block.Operations {
			if op.Type != (hir.Type{}) && op.Type != hir.TypeI32 || op.OwnerType != (hir.Type{}) && op.OwnerType != hir.TypeI32 {
				return "copied closure target contains a non-i32 operation"
			}
			for _, value := range op.Operands {
				if !closureOperandMatches(value, hir.TypeI32, types) {
					return "copied closure target contains a non-i32 operand"
				}
			}
			switch op.Kind {
			case OperationAssign, OperationUnary, OperationBinary:
				if op.DestinationLocal < 0 || writes[op.DestinationLocal] {
					return "copied closure target mutates an immutable local"
				}
				writes[op.DestinationLocal] = true
			case OperationArithmeticCheck, OperationCheckedDivision:
			default:
				return "copied closure target contains a non-arithmetic operation"
			}
		}
		switch term := block.Terminator; term.Kind {
		case TerminatorReturn:
			if !closureOperandMatches(term.ReturnValue, hir.TypeI32, types) {
				return "copied closure target returns a non-i32 value"
			}
		case TerminatorGoto, TerminatorTrap:
		case TerminatorSwitch:
			if term.SwitchType != hir.TypeI32 || term.SwitchKind != "checked-unary" && term.SwitchKind != "checked-arithmetic" && term.SwitchKind != "checked-division" && term.SwitchKind != "checked-shift" {
				return "copied closure target contains non-arithmetic control flow"
			}
			for _, value := range term.SwitchOperands {
				if !closureOperandMatches(value, hir.TypeI32, types) {
					return "copied closure target checks a non-i32 operand"
				}
			}
		default:
			return "copied closure target contains an unsupported control transfer"
		}
	}
	if !closureTargetAcyclic(fn) {
		return "copied closure target contains cyclic or invalid control flow"
	}
	return ""
}

func closureTargetAcyclic(fn *Function) bool {
	incoming := make([]int, len(fn.Blocks))
	for i, block := range fn.Blocks {
		if block.ID != BlockID(i) {
			return false
		}
		for _, successor := range block.Terminator.Successors {
			if successor < 0 || int(successor) >= len(incoming) {
				return false
			}
			incoming[successor]++
		}
	}
	var ready []int
	for i, count := range incoming {
		if count == 0 {
			ready = append(ready, i)
		}
	}
	for i := 0; i < len(ready); i++ {
		for _, successor := range fn.Blocks[ready[i]].Terminator.Successors {
			incoming[successor]--
			if incoming[successor] == 0 {
				ready = append(ready, int(successor))
			}
		}
	}
	return len(ready) == len(fn.Blocks)
}
