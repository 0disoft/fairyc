package cbackend

import (
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"math"
)

// constantCheckedFailure proves only checked integer failures. Copy facts are
// local to this straight-line block; unknown operations discard all facts so
// indirect writes, calls and predecessor state cannot silently preserve them.
// MIR verification remains mandatory before this C-only reachability decision.
func constantCheckedFailure(block mir.Block) (mir.BlockID, bool) {
	term := block.Terminator
	if term.Kind != mir.TerminatorSwitch || len(term.Successors) != 2 ||
		(term.SwitchKind != "checked-arithmetic" && term.SwitchKind != "checked-unary") {
		return mir.NoBlock, false
	}
	known := map[mir.LocalID]mir.Operand{}
	resolve := func(op mir.Operand) mir.Operand {
		if op.Kind == mir.OperandLocal {
			if value, ok := known[op.Local]; ok {
				return value
			}
		}
		return op
	}
	for _, op := range block.Operations {
		switch op.Kind {
		case mir.OperationAssign, mir.OperationInitialize:
			if len(op.Operands) != 1 {
				clear(known)
				continue
			}
			value := resolve(op.Operands[0])
			delete(known, op.DestinationLocal)
			switch value.Kind {
			case mir.OperandI32, mir.OperandI64, mir.OperandU8, mir.OperandU32, mir.OperandUSize:
				known[op.DestinationLocal] = value
			}
		case mir.OperationArithmeticCheck:
			// A verified marker performs no write; its actual guard is the terminator.
		default:
			clear(known)
		}
	}
	operands := make([]mir.Operand, len(term.SwitchOperands))
	for i, op := range term.SwitchOperands {
		operands[i] = resolve(op)
	}
	if literalCheckedOverflow(term.SwitchType, term.Operator, term.SwitchKind, operands) {
		return term.Successors[1], true
	}
	return mir.NoBlock, false
}

func literalCheckedOverflow(typ hir.Type, operator, kind string, operands []mir.Operand) bool {
	if typ == hir.TypeI32 || typ == hir.TypeI64 {
		minimum, maximum := int64(math.MinInt64), int64(math.MaxInt64)
		if typ == hir.TypeI32 {
			minimum, maximum = math.MinInt32, math.MaxInt32
		}
		signed := func(op mir.Operand) (int64, bool) {
			if typ == hir.TypeI32 && op.Kind == mir.OperandI32 && op.Type != hir.TypeBool {
				return int64(op.I32), true
			}
			if typ == hir.TypeI64 && op.Kind == mir.OperandI64 {
				return op.I64, true
			}
			return 0, false
		}
		if kind == "checked-unary" && operator == "-" && len(operands) == 1 {
			n, ok := signed(operands[0])
			return ok && n == minimum
		}
		if kind != "checked-arithmetic" || len(operands) != 2 {
			return false
		}
		l, ok := signed(operands[0])
		if !ok {
			return false
		}
		r, ok := signed(operands[1])
		if !ok {
			return false
		}
		switch operator {
		case "+":
			return r > 0 && l > maximum-r || r < 0 && l < minimum-r
		case "-":
			return r < 0 && l > maximum+r || r > 0 && l < minimum+r
		case "*":
			if l > 0 {
				if r > 0 {
					return l > maximum/r
				}
				return r < minimum/l
			}
			if l < 0 {
				if r > 0 {
					return l < minimum/r
				}
				return r < 0 && l < maximum/r
			}
		}
		return false
	}
	if kind != "checked-arithmetic" || len(operands) != 2 {
		return false
	}
	var operandKind mir.OperandKind
	var maximum uint64
	switch typ {
	case hir.TypeU8:
		operandKind, maximum = mir.OperandU8, math.MaxUint8
	case hir.TypeU32:
		operandKind, maximum = mir.OperandU32, math.MaxUint32
	case hir.TypeUSize:
		// Both admitted C profiles are x86-64, with a 64-bit uintptr_t transport.
		operandKind, maximum = mir.OperandUSize, math.MaxUint64
	default:
		return false
	}
	l, r := operands[0], operands[1]
	if l.Kind != operandKind || r.Kind != operandKind || l.U64 > maximum || r.U64 > maximum {
		return false
	}
	switch operator {
	case "+":
		return l.U64 > maximum-r.U64
	case "-":
		return l.U64 < r.U64
	case "*":
		return r.U64 != 0 && l.U64 > maximum/r.U64
	}
	return false
}
