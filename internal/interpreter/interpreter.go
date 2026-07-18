// Package interpreter executes verified Stage 0 MIR as the first semantic oracle.
package interpreter

import (
	"fmt"
	"math"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

const (
	OutcomeReturn = "return"
	OutcomeTrap   = "trap"
	OutcomeLimit  = "limit"
	LimitStep     = "step-limit"
	maxCallDepth  = 256
	maxSteps      = 1_000_000
)

// Outcome is one bounded interpreter completion.
type Outcome struct {
	Kind         string
	Value        int32
	TrapCategory string
}

// CapabilityError reports a valid verified-MIR program outside the implemented interpreter slice.
type CapabilityError struct {
	Kind   string
	Detail string
}

func (e CapabilityError) Error() string {
	return e.Detail
}

func capabilityError(format string, arguments ...any) error {
	return CapabilityError{Detail: fmt.Sprintf(format, arguments...)}
}

func capabilityErrorKind(kind, format string, arguments ...any) error {
	return CapabilityError{Kind: kind, Detail: fmt.Sprintf(format, arguments...)}
}

// Execute verifies program and invokes one zero-argument i32 entry function.
func Execute(program mir.Program, entry string) (Outcome, error) {
	if failures := mir.Verify(program); len(failures) != 0 {
		return Outcome{}, fmt.Errorf("verify MIR before interpretation: %v", failures)
	}
	runtime := runtime{
		byName:   make(map[string]*mir.Function, len(program.Functions)),
		bySymbol: make(map[hir.SymbolID]*mir.Function, len(program.Functions)),
		steps:    maxSteps,
	}
	for index := range program.Functions {
		function := &program.Functions[index]
		if _, duplicate := runtime.byName[function.Name]; duplicate {
			return Outcome{}, fmt.Errorf("duplicate MIR function name %q", function.Name)
		}
		if _, duplicate := runtime.bySymbol[function.Symbol]; duplicate {
			return Outcome{}, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		runtime.byName[function.Name] = function
		runtime.bySymbol[function.Symbol] = function
	}
	function := runtime.byName[entry]
	if function == nil {
		return Outcome{}, capabilityError("entry function %q is missing", entry)
	}
	if len(function.Parameters) != 0 || function.ResultType != hir.TypeI32 {
		return Outcome{}, capabilityError("entry function %q must have type fn() -> i32", entry)
	}
	return runtime.call(function, nil, 0)
}

type runtime struct {
	byName   map[string]*mir.Function
	bySymbol map[hir.SymbolID]*mir.Function
	steps    int
}

func (r *runtime) call(function *mir.Function, arguments []int32, depth int) (Outcome, error) {
	if depth >= maxCallDepth {
		return Outcome{}, capabilityErrorKind("call-depth-limit", "interpreter call-depth limit exceeded")
	}
	if len(arguments) != len(function.Parameters) {
		return Outcome{}, fmt.Errorf("call %q received %d argument(s), want %d", function.Name, len(arguments), len(function.Parameters))
	}
	locals := make(map[mir.LocalID]int32)
	for index, parameter := range function.Parameters {
		locals[parameter] = arguments[index]
	}
	current := function.Entry
	for {
		if r.steps <= 0 {
			return Outcome{}, capabilityErrorKind(LimitStep, "interpreter step limit exceeded")
		}
		r.steps--
		if int(current) < 0 || int(current) >= len(function.Blocks) {
			return Outcome{}, fmt.Errorf("function %q jumps to missing block %d", function.Name, current)
		}
		block := function.Blocks[current]
		for _, operation := range block.Operations {
			if r.steps <= 0 {
				return Outcome{}, capabilityErrorKind(LimitStep, "interpreter step limit exceeded")
			}
			r.steps--
			switch operation.Kind {
			case mir.OperationAssign:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return Outcome{}, fmt.Errorf("assign in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationUnary:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return Outcome{}, fmt.Errorf("unary in %q: %w", function.Name, err)
				}
				result, trap, err := unary(operation.Operator, value)
				if err != nil {
					return Outcome{}, err
				}
				if trap != "" {
					return Outcome{Kind: OutcomeTrap, TrapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = result
			case mir.OperationBinary:
				left, right, err := evaluateTwo(locals, operation.Operands)
				if err != nil {
					return Outcome{}, fmt.Errorf("binary in %q: %w", function.Name, err)
				}
				result, trap, err := binary(operation.Operator, left, right)
				if err != nil {
					return Outcome{}, err
				}
				if trap != "" {
					return Outcome{Kind: OutcomeTrap, TrapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = result
			}
		}

		terminator := block.Terminator
		switch terminator.Kind {
		case mir.TerminatorGoto:
			next, err := oneSuccessor(terminator)
			if err != nil {
				return Outcome{}, err
			}
			current = next
		case mir.TerminatorCall:
			callee := r.bySymbol[terminator.Callee]
			if callee == nil {
				return Outcome{}, fmt.Errorf("call in %q names missing symbol %d", function.Name, terminator.Callee)
			}
			values := make([]int32, len(terminator.Arguments))
			for index, operand := range terminator.Arguments {
				value, err := evaluate(locals, operand)
				if err != nil {
					return Outcome{}, fmt.Errorf("call argument in %q: %w", function.Name, err)
				}
				values[index] = value
			}
			outcome, err := r.call(callee, values, depth+1)
			if err != nil || outcome.Kind == OutcomeTrap {
				return outcome, err
			}
			locals[terminator.DestinationLocal] = outcome.Value
			next, err := oneSuccessor(terminator)
			if err != nil {
				return Outcome{}, err
			}
			current = next
		case mir.TerminatorSwitch:
			next, trap, err := evaluateSwitch(locals, terminator)
			if err != nil {
				return Outcome{}, err
			}
			if trap != "" {
				return Outcome{Kind: OutcomeTrap, TrapCategory: trap}, nil
			}
			current = next
		case mir.TerminatorReturn:
			value, err := evaluate(locals, terminator.ReturnValue)
			if err != nil {
				return Outcome{}, fmt.Errorf("return from %q: %w", function.Name, err)
			}
			return Outcome{Kind: OutcomeReturn, Value: value}, nil
		case mir.TerminatorTrap:
			if len(terminator.TrapCategories) == 0 {
				return Outcome{}, fmt.Errorf("trap in %q has no category", function.Name)
			}
			return Outcome{Kind: OutcomeTrap, TrapCategory: terminator.TrapCategories[0]}, nil
		case mir.TerminatorUnreachable:
			return Outcome{}, fmt.Errorf("function %q reached an unreachable MIR block", function.Name)
		default:
			return Outcome{}, fmt.Errorf("function %q has unsupported terminator %q", function.Name, terminator.Kind)
		}
	}
}

func evaluate(locals map[mir.LocalID]int32, operand mir.Operand) (int32, error) {
	switch operand.Kind {
	case mir.OperandI32:
		return operand.I32, nil
	case mir.OperandLocal:
		value, found := locals[operand.Local]
		if !found {
			return 0, capabilityError("local %d has no scalar value", operand.Local)
		}
		return value, nil
	default:
		return 0, capabilityError("operand has no executable scalar value")
	}
}

func evaluateOne(locals map[mir.LocalID]int32, operands []mir.Operand) (int32, error) {
	if len(operands) != 1 {
		return 0, fmt.Errorf("operation has %d operands, want 1", len(operands))
	}
	return evaluate(locals, operands[0])
}

func evaluateTwo(locals map[mir.LocalID]int32, operands []mir.Operand) (int32, int32, error) {
	if len(operands) != 2 {
		return 0, 0, fmt.Errorf("operation has %d operands, want 2", len(operands))
	}
	left, err := evaluate(locals, operands[0])
	if err != nil {
		return 0, 0, err
	}
	right, err := evaluate(locals, operands[1])
	return left, right, err
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("terminator %q has %d successors, want 1", terminator.Kind, len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}

func evaluateSwitch(locals map[mir.LocalID]int32, terminator mir.Terminator) (mir.BlockID, string, error) {
	switch terminator.SwitchKind {
	case "bool":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("bool switch has %d successors, want 2", len(terminator.Successors))
		}
		if value != 0 && value != 1 {
			return mir.NoBlock, "", fmt.Errorf("bool switch received non-boolean value %d", value)
		}
		return terminator.Successors[value^1], "", nil
	case "checked-division":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if right == 0 {
			return mir.NoBlock, "division-by-zero", nil
		}
		if left == math.MinInt32 && right == -1 {
			return mir.NoBlock, "signed-division-overflow", nil
		}
		return terminator.Successors[0], "", nil
	default:
		return mir.NoBlock, "", capabilityError("unsupported MIR switch %q", terminator.SwitchKind)
	}
}

func unary(operator string, value int32) (int32, string, error) {
	switch operator {
	case "+":
		return value, "", nil
	case "-":
		if value == math.MinInt32 {
			return 0, "integer-overflow", nil
		}
		return -value, "", nil
	case "~":
		return ^value, "", nil
	case "!":
		if value != 0 && value != 1 {
			return 0, "", fmt.Errorf("logical not received non-boolean value %d", value)
		}
		return value ^ 1, "", nil
	default:
		return 0, "", capabilityError("unsupported unary operator %q", operator)
	}
}

func binary(operator string, left, right int32) (int32, string, error) {
	switch operator {
	case "+":
		return checkedInteger(int64(left) + int64(right))
	case "-":
		return checkedInteger(int64(left) - int64(right))
	case "*":
		return checkedInteger(int64(left) * int64(right))
	case "/":
		if right == 0 {
			return 0, "division-by-zero", nil
		}
		if left == math.MinInt32 && right == -1 {
			return 0, "signed-division-overflow", nil
		}
		return left / right, "", nil
	case "%":
		if right == 0 {
			return 0, "division-by-zero", nil
		}
		if left == math.MinInt32 && right == -1 {
			return 0, "signed-division-overflow", nil
		}
		return left % right, "", nil
	case "<<", ">>":
		if right < 0 || right >= 32 {
			return 0, "invalid-shift", nil
		}
		if operator == "<<" {
			return checkedInteger(int64(left) << uint32(right))
		}
		return left >> uint32(right), "", nil
	case "&":
		return left & right, "", nil
	case "^":
		return left ^ right, "", nil
	case "|":
		return left | right, "", nil
	case "==":
		return boolValue(left == right), "", nil
	case "!=":
		return boolValue(left != right), "", nil
	case "<":
		return boolValue(left < right), "", nil
	case "<=":
		return boolValue(left <= right), "", nil
	case ">":
		return boolValue(left > right), "", nil
	case ">=":
		return boolValue(left >= right), "", nil
	default:
		return 0, "", capabilityError("unsupported binary operator %q", operator)
	}
}

func checkedInteger(value int64) (int32, string, error) {
	if value < math.MinInt32 || value > math.MaxInt32 {
		return 0, "integer-overflow", nil
	}
	return int32(value), "", nil
}

func boolValue(value bool) int32 {
	if value {
		return 1
	}
	return 0
}
