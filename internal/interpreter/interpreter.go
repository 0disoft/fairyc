// Package interpreter executes verified Stage 0 MIR as the first semantic oracle.
package interpreter

import (
	"fmt"
	"math"
	"slices"

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
	DropTrace    []DropEvent
}

// DropEvent is one resource drop-glue invocation in execution order.
type DropEvent struct {
	ResourceType hir.Type
	Binding      hir.Span
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
	completion, err := runtime.call(function, nil, 0)
	if err != nil {
		return Outcome{}, err
	}
	if completion.trapCategory != "" {
		return Outcome{Kind: OutcomeTrap, TrapCategory: completion.trapCategory, DropTrace: slices.Clone(runtime.dropTrace)}, nil
	}
	value, err := completion.value.asI32()
	if err != nil {
		return Outcome{}, fmt.Errorf("entry function %q: %w", entry, err)
	}
	return Outcome{Kind: OutcomeReturn, Value: value, DropTrace: slices.Clone(runtime.dropTrace)}, nil
}

type runtime struct {
	byName    map[string]*mir.Function
	bySymbol  map[hir.SymbolID]*mir.Function
	steps     int
	dropTrace []DropEvent
}

type valueKind uint8

const (
	valueI32 valueKind = iota + 1
	valueResult
	valueOption
	valueResource
)

type resourceState struct {
	typ        hir.Type
	fields     map[string]runtimeValue
	fieldOrder []string
	dropped    bool
}

type taggedState struct {
	variant string
	payload *runtimeValue
	dropped bool
}

type runtimeValue struct {
	kind     valueKind
	i32      int32
	tagged   *taggedState
	resource *resourceState
}

func i32Value(value int32) runtimeValue {
	return runtimeValue{kind: valueI32, i32: value}
}

func resultValue(variant string, payload runtimeValue) (runtimeValue, error) {
	if variant != "ok" && variant != "err" {
		return runtimeValue{}, fmt.Errorf("invalid Result variant %q", variant)
	}
	copy := payload
	return runtimeValue{kind: valueResult, tagged: &taggedState{variant: variant, payload: &copy}}, nil
}

func optionValue(variant string, payload *runtimeValue) (runtimeValue, error) {
	switch variant {
	case "some":
		if payload == nil {
			return runtimeValue{}, fmt.Errorf("Option::Some has no payload")
		}
		copy := *payload
		return runtimeValue{kind: valueOption, tagged: &taggedState{variant: variant, payload: &copy}}, nil
	case "none":
		if payload != nil {
			return runtimeValue{}, fmt.Errorf("Option::None has a payload")
		}
		return runtimeValue{kind: valueOption, tagged: &taggedState{variant: variant}}, nil
	default:
		return runtimeValue{}, fmt.Errorf("invalid Option variant %q", variant)
	}
}

func resourceValue(typ hir.Type, fields []string, values []runtimeValue) (runtimeValue, error) {
	if typ == hir.TypeInvalid || len(fields) == 0 || len(fields) != len(values) {
		return runtimeValue{}, fmt.Errorf("invalid resource construction for %q", typ)
	}
	state := &resourceState{typ: typ, fields: make(map[string]runtimeValue, len(fields)), fieldOrder: slices.Clone(fields)}
	for index, field := range fields {
		if field == "" {
			return runtimeValue{}, fmt.Errorf("resource %q has an empty field name", typ)
		}
		if _, duplicate := state.fields[field]; duplicate {
			return runtimeValue{}, fmt.Errorf("resource %q has duplicate field %q", typ, field)
		}
		state.fields[field] = values[index]
	}
	return runtimeValue{kind: valueResource, resource: state}, nil
}

func (v runtimeValue) asI32() (int32, error) {
	if v.kind != valueI32 {
		return 0, capabilityError("value is not an executable i32 scalar")
	}
	return v.i32, nil
}

func (v runtimeValue) resultPayload(expected string, move bool) (runtimeValue, error) {
	if v.kind != valueResult || v.tagged == nil || v.tagged.payload == nil || v.tagged.dropped {
		return runtimeValue{}, capabilityError("value is not an executable Result")
	}
	if v.tagged.variant != expected {
		return runtimeValue{}, fmt.Errorf("Result payload expects %q, got %q", expected, v.tagged.variant)
	}
	payload := *v.tagged.payload
	if move {
		v.tagged.payload = nil
	}
	return payload, nil
}

func (v runtimeValue) optionPayload(move bool) (runtimeValue, error) {
	if v.kind != valueOption || v.tagged == nil || v.tagged.variant != "some" || v.tagged.payload == nil || v.tagged.dropped {
		return runtimeValue{}, capabilityError("value is not an executable Option::Some")
	}
	payload := *v.tagged.payload
	if move {
		v.tagged.payload = nil
	}
	return payload, nil
}

func (v runtimeValue) field(name string) (runtimeValue, error) {
	if v.kind != valueResource || v.resource == nil {
		return runtimeValue{}, capabilityError("field read received a non-resource value")
	}
	if v.resource.dropped {
		return runtimeValue{}, fmt.Errorf("resource %q was read after drop", v.resource.typ)
	}
	value, found := v.resource.fields[name]
	if !found {
		return runtimeValue{}, fmt.Errorf("resource %q has no field %q", v.resource.typ, name)
	}
	return value, nil
}

type callCompletion struct {
	value        runtimeValue
	trapCategory string
}

func (r *runtime) call(function *mir.Function, arguments []runtimeValue, depth int) (callCompletion, error) {
	if depth >= maxCallDepth {
		return callCompletion{}, capabilityErrorKind("call-depth-limit", "interpreter call-depth limit exceeded")
	}
	if len(arguments) != len(function.Parameters) {
		return callCompletion{}, fmt.Errorf("call %q received %d argument(s), want %d", function.Name, len(arguments), len(function.Parameters))
	}
	locals := make(map[mir.LocalID]runtimeValue)
	for index, parameter := range function.Parameters {
		locals[parameter] = arguments[index]
	}
	current := function.Entry
	for {
		if r.steps <= 0 {
			return callCompletion{}, capabilityErrorKind(LimitStep, "interpreter step limit exceeded")
		}
		r.steps--
		if int(current) < 0 || int(current) >= len(function.Blocks) {
			return callCompletion{}, fmt.Errorf("function %q jumps to missing block %d", function.Name, current)
		}
		block := function.Blocks[current]
		for _, operation := range block.Operations {
			if r.steps <= 0 {
				return callCompletion{}, capabilityErrorKind(LimitStep, "interpreter step limit exceeded")
			}
			r.steps--
			switch operation.Kind {
			case mir.OperationInitialize:
				if len(operation.Operands) != 1 {
					return callCompletion{}, capabilityError("resource initialization in %q has no executable value", function.Name)
				}
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("resource initialization in %q: %w", function.Name, err)
				}
				locals[mir.LocalID(operation.Symbol)] = value
			case mir.OperationAssign:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("assign in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationUnary:
				value, err := evaluateOneI32(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("unary in %q: %w", function.Name, err)
				}
				result, trap, err := unary(operation.Operator, value)
				if err != nil {
					return callCompletion{}, err
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = i32Value(result)
			case mir.OperationBinary:
				left, right, err := evaluateTwoI32(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("binary in %q: %w", function.Name, err)
				}
				result, trap, err := binary(operation.Operator, left, right)
				if err != nil {
					return callCompletion{}, err
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = i32Value(result)
			case mir.OperationResultConstruct:
				payload, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result construction in %q: %w", function.Name, err)
				}
				constructed, err := resultValue(operation.Variant, payload)
				if err != nil {
					return callCompletion{}, err
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationResultPayload:
				result, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result payload in %q: %w", function.Name, err)
				}
				payload, err := result.resultPayload(operation.Variant, operation.MovePayload)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Result payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationOptionConstruct:
				var payload *runtimeValue
				if operation.Variant == "some" {
					value, err := evaluateOne(locals, operation.Operands)
					if err != nil {
						return callCompletion{}, fmt.Errorf("Option construction in %q: %w", function.Name, err)
					}
					payload = &value
				} else if len(operation.Operands) != 0 {
					return callCompletion{}, fmt.Errorf("Option::None construction in %q has a payload", function.Name)
				}
				constructed, err := optionValue(operation.Variant, payload)
				if err != nil {
					return callCompletion{}, err
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationOptionPayload:
				option, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Option payload in %q: %w", function.Name, err)
				}
				payload, err := option.optionPayload(operation.MovePayload)
				if err != nil {
					return callCompletion{}, fmt.Errorf("Option payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationResourceConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := resourceValue(operation.Type, operation.Fields, values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationFieldRead:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				value, err := receiver.field(operation.Field)
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationMove:
				delete(locals, mir.LocalID(operation.Symbol))
			case mir.OperationDrop:
				local := mir.LocalID(operation.Symbol)
				value, found := locals[local]
				if !found {
					return callCompletion{}, fmt.Errorf("drop in %q names missing local %d", function.Name, local)
				}
				if err := r.dropValue(value, operation.Span); err != nil {
					return callCompletion{}, fmt.Errorf("drop in %q: %w", function.Name, err)
				}
				delete(locals, local)
			}
		}

		terminator := block.Terminator
		switch terminator.Kind {
		case mir.TerminatorGoto:
			next, err := oneSuccessor(terminator)
			if err != nil {
				return callCompletion{}, err
			}
			current = next
		case mir.TerminatorCall:
			callee := r.bySymbol[terminator.Callee]
			if callee == nil {
				return callCompletion{}, fmt.Errorf("call in %q names missing symbol %d", function.Name, terminator.Callee)
			}
			values := make([]runtimeValue, len(terminator.Arguments))
			for index, operand := range terminator.Arguments {
				value, err := evaluate(locals, operand)
				if err != nil {
					return callCompletion{}, fmt.Errorf("call argument in %q: %w", function.Name, err)
				}
				values[index] = value
			}
			completion, err := r.call(callee, values, depth+1)
			if err != nil || completion.trapCategory != "" {
				return completion, err
			}
			locals[terminator.DestinationLocal] = completion.value
			next, err := oneSuccessor(terminator)
			if err != nil {
				return callCompletion{}, err
			}
			current = next
		case mir.TerminatorSwitch:
			next, trap, err := evaluateSwitch(locals, terminator)
			if err != nil {
				return callCompletion{}, err
			}
			if trap != "" {
				return callCompletion{trapCategory: trap}, nil
			}
			current = next
		case mir.TerminatorReturn:
			value, err := evaluate(locals, terminator.ReturnValue)
			if err != nil {
				return callCompletion{}, fmt.Errorf("return from %q: %w", function.Name, err)
			}
			return callCompletion{value: value}, nil
		case mir.TerminatorTrap:
			if len(terminator.TrapCategories) == 0 {
				return callCompletion{}, fmt.Errorf("trap in %q has no category", function.Name)
			}
			return callCompletion{trapCategory: terminator.TrapCategories[0]}, nil
		case mir.TerminatorUnreachable:
			return callCompletion{}, fmt.Errorf("function %q reached an unreachable MIR block", function.Name)
		default:
			return callCompletion{}, fmt.Errorf("function %q has unsupported terminator %q", function.Name, terminator.Kind)
		}
	}
}

func (r *runtime) dropValue(value runtimeValue, binding hir.Span) error {
	switch value.kind {
	case valueI32:
		return nil
	case valueResult, valueOption:
		if value.tagged == nil {
			return fmt.Errorf("tagged value has no state")
		}
		if value.tagged.dropped {
			return fmt.Errorf("%s::%s was dropped twice", taggedFamily(value.kind), value.tagged.variant)
		}
		value.tagged.dropped = true
		if value.tagged.payload == nil {
			return nil
		}
		return r.dropValue(*value.tagged.payload, binding)
	case valueResource:
		if value.resource == nil {
			return fmt.Errorf("resource value has no state")
		}
		if value.resource.dropped {
			return fmt.Errorf("resource %q was dropped twice", value.resource.typ)
		}
		value.resource.dropped = true
		r.dropTrace = append(r.dropTrace, DropEvent{ResourceType: value.resource.typ, Binding: binding})
		for index := len(value.resource.fieldOrder) - 1; index >= 0; index-- {
			field := value.resource.fieldOrder[index]
			if err := r.dropValue(value.resource.fields[field], binding); err != nil {
				return err
			}
		}
		return nil
	default:
		return capabilityError("value has no executable drop behavior")
	}
}

func taggedFamily(kind valueKind) string {
	if kind == valueResult {
		return "Result"
	}
	return "Option"
}

func evaluate(locals map[mir.LocalID]runtimeValue, operand mir.Operand) (runtimeValue, error) {
	switch operand.Kind {
	case mir.OperandI32:
		return i32Value(operand.I32), nil
	case mir.OperandLocal:
		value, found := locals[operand.Local]
		if !found {
			return runtimeValue{}, capabilityError("local %d has no executable value", operand.Local)
		}
		return value, nil
	default:
		return runtimeValue{}, capabilityError("operand has no executable value")
	}
}

func evaluateOne(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, error) {
	if len(operands) != 1 {
		return runtimeValue{}, fmt.Errorf("operation has %d operands, want 1", len(operands))
	}
	return evaluate(locals, operands[0])
}

func evaluateOneI32(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (int32, error) {
	value, err := evaluateOne(locals, operands)
	if err != nil {
		return 0, err
	}
	return value.asI32()
}

func evaluateTwoI32(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (int32, int32, error) {
	if len(operands) != 2 {
		return 0, 0, fmt.Errorf("operation has %d operands, want 2", len(operands))
	}
	leftValue, err := evaluate(locals, operands[0])
	if err != nil {
		return 0, 0, err
	}
	left, err := leftValue.asI32()
	if err != nil {
		return 0, 0, err
	}
	rightValue, err := evaluate(locals, operands[1])
	if err != nil {
		return 0, 0, err
	}
	right, err := rightValue.asI32()
	return left, right, err
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("terminator %q has %d successors, want 1", terminator.Kind, len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}

func evaluateSwitch(locals map[mir.LocalID]runtimeValue, terminator mir.Terminator) (mir.BlockID, string, error) {
	switch terminator.SwitchKind {
	case "bool":
		value, err := evaluateOneI32(locals, terminator.SwitchOperands)
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
	case "result":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueResult || value.tagged == nil || value.tagged.payload == nil || value.tagged.dropped {
			return mir.NoBlock, "", capabilityError("Result switch received a non-Result value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("Result switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.tagged.variant == "ok" {
			return terminator.Successors[0], "", nil
		}
		if value.tagged.variant == "err" {
			return terminator.Successors[1], "", nil
		}
		return mir.NoBlock, "", fmt.Errorf("Result switch received invalid variant %q", value.tagged.variant)
	case "option":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueOption || value.tagged == nil || value.tagged.dropped {
			return mir.NoBlock, "", capabilityError("Option switch received a non-Option value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("Option switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.tagged.variant == "some" && value.tagged.payload != nil {
			return terminator.Successors[0], "", nil
		}
		if value.tagged.variant == "none" && value.tagged.payload == nil {
			return terminator.Successors[1], "", nil
		}
		return mir.NoBlock, "", fmt.Errorf("Option switch received invalid variant %q", value.tagged.variant)
	case "checked-division":
		left, right, err := evaluateTwoI32(locals, terminator.SwitchOperands)
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
