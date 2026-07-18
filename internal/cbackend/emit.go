// Package cbackend emits the constrained Stage 0 C reference transport.
package cbackend

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

const (
	// TargetX8664LinuxGNU is the first planned Stage 0 C transport target.
	TargetX8664LinuxGNU = "x86_64-linux-gnu"
	// DialectC11 identifies the extension-free ISO C11 emission subset.
	DialectC11 = "c11"
)

// Profile names the target and C dialect accepted by one emission run.
type Profile struct {
	Target  string
	Dialect string
}

// Stage0ScalarProfile returns the only currently implemented C emission profile.
func Stage0ScalarProfile() Profile {
	return Profile{Target: TargetX8664LinuxGNU, Dialect: DialectC11}
}

// Unit is one deterministic generated-C translation unit plus bounded stage facts.
type Unit struct {
	Target    string
	Dialect   string
	Entry     string
	Functions []string
	Features  []string
	Source    string
}

// CapabilityError reports verified MIR outside the implemented C transport subset.
type CapabilityError struct {
	Function string
	Feature  string
}

func (e CapabilityError) Error() string {
	if e.Function == "" {
		return fmt.Sprintf("C backend does not support %s", e.Feature)
	}
	return fmt.Sprintf("C backend does not support %s in %s", e.Feature, e.Function)
}

type emitter struct {
	functions          []mir.Function
	bySymbol           map[hir.SymbolID]int
	entryIndex         int
	features           map[string]struct{}
	trapCategories     map[string]struct{}
	dynamicTrapTargets map[int]map[mir.BlockID]struct{}
	needsBitcast       bool
	needsShift         bool
}

// Emit verifies MIR, rejects unsupported semantic families, and emits one deterministic C unit.
func Emit(program mir.Program, entry string, profile Profile) (Unit, error) {
	if profile != Stage0ScalarProfile() {
		return Unit{}, fmt.Errorf("unsupported C emission profile target=%q dialect=%q", profile.Target, profile.Dialect)
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return Unit{}, fmt.Errorf("C emission requires verified MIR: %s at %s", failures[0].Code, failures[0].Entity)
	}
	functions := slices.Clone(program.Functions)
	slices.SortFunc(functions, func(left, right mir.Function) int {
		return strings.Compare(left.Name, right.Name)
	})
	value := &emitter{
		functions:          functions,
		bySymbol:           make(map[hir.SymbolID]int, len(functions)),
		entryIndex:         -1,
		features:           make(map[string]struct{}),
		trapCategories:     make(map[string]struct{}),
		dynamicTrapTargets: make(map[int]map[mir.BlockID]struct{}),
	}
	for index := range functions {
		function := &functions[index]
		if _, duplicate := value.bySymbol[function.Symbol]; duplicate {
			return Unit{}, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		value.bySymbol[function.Symbol] = index
		if function.Name == entry {
			if value.entryIndex >= 0 {
				return Unit{}, fmt.Errorf("duplicate C entry function %q", entry)
			}
			value.entryIndex = index
		}
	}
	if value.entryIndex < 0 {
		return Unit{}, fmt.Errorf("C entry function %q is missing", entry)
	}
	entryFunction := &functions[value.entryIndex]
	if len(entryFunction.Parameters) != 0 || entryFunction.ResultType != hir.TypeI32 {
		return Unit{}, fmt.Errorf("C entry %q must have type fn() -> i32", entry)
	}
	for index := range functions {
		if err := value.inspectFunction(index); err != nil {
			return Unit{}, err
		}
	}

	var output strings.Builder
	value.emitPreamble(&output)
	for index := range functions {
		value.emitPrototype(&output, index)
	}
	output.WriteByte('\n')
	for index := range functions {
		if err := value.emitFunction(&output, index); err != nil {
			return Unit{}, err
		}
	}
	fmt.Fprintf(&output, "int main(void) {\n    return (int)fy_fn_%d();\n}\n", value.entryIndex)

	names := make([]string, len(functions))
	for index := range functions {
		names[index] = functions[index].Name
	}
	features := make([]string, 0, len(value.features))
	for feature := range value.features {
		features = append(features, feature)
	}
	slices.Sort(features)
	return Unit{
		Target: profile.Target, Dialect: profile.Dialect, Entry: entry,
		Functions: names, Features: features, Source: output.String(),
	}, nil
}

func (e *emitter) inspectFunction(functionIndex int) error {
	function := &e.functions[functionIndex]
	if function.ResultType != hir.TypeI32 && function.ResultType != hir.TypeBool && function.ResultType != hir.TypeVoid {
		return CapabilityError{Function: function.Name, Feature: "result type " + strconv.Quote(string(function.ResultType))}
	}
	if len(function.ParameterTypes) != len(function.Parameters) {
		return fmt.Errorf("MIR function %s has %d parameter types for %d parameters", function.Name, len(function.ParameterTypes), len(function.Parameters))
	}
	for _, parameterType := range function.ParameterTypes {
		if parameterType != hir.TypeI32 && parameterType != hir.TypeBool {
			return CapabilityError{Function: function.Name, Feature: "parameter type " + strconv.Quote(string(parameterType))}
		}
	}
	if len(function.Resources) != 0 || len(function.DropFlags) != 0 {
		return CapabilityError{Function: function.Name, Feature: "resource values"}
	}
	if len(function.Loans) != 0 || len(function.ReturnOrigins) != 0 {
		return CapabilityError{Function: function.Name, Feature: "safe-borrow values"}
	}
	if len(function.RegionValues) != 0 {
		return CapabilityError{Function: function.Name, Feature: "Region values"}
	}
	if len(function.UnsafeOperations) != 0 {
		return CapabilityError{Function: function.Name, Feature: "unsafe operations"}
	}
	for blockIndex := range function.Blocks {
		block := &function.Blocks[blockIndex]
		if block.ID != mir.BlockID(blockIndex) {
			return fmt.Errorf("C emission requires canonical block ids in %s: block index %d has id %d", function.Name, blockIndex, block.ID)
		}
		for operationIndex := range block.Operations {
			operation := &block.Operations[operationIndex]
			if err := validateOperationShape(function.Name, *operation); err != nil {
				return err
			}
			switch operation.Kind {
			case mir.OperationAssign, mir.OperationUse:
			case mir.OperationArithmeticCheck:
				e.features["checked-arithmetic"] = struct{}{}
			case mir.OperationCheckedDivision:
				e.features["checked-division"] = struct{}{}
			case mir.OperationUnary:
				if !supportedUnary(operation.Operator) {
					return CapabilityError{Function: function.Name, Feature: "unary operator " + strconv.Quote(operation.Operator)}
				}
				if operation.Operator == "~" {
					e.features["scalar-bitwise"] = struct{}{}
					e.needsBitcast = true
				}
			case mir.OperationBinary:
				if !supportedBinary(operation.Operator) {
					return CapabilityError{Function: function.Name, Feature: "binary operator " + strconv.Quote(operation.Operator)}
				}
				switch operation.Operator {
				case "==", "!=", "<", "<=", ">", ">=":
					e.features["scalar-comparison"] = struct{}{}
				case "&", "^", "|":
					e.features["scalar-bitwise"] = struct{}{}
					e.needsBitcast = true
				case ">>":
					e.features["checked-shift"] = struct{}{}
					e.needsBitcast = true
					e.needsShift = true
				}
			default:
				return CapabilityError{Function: function.Name, Feature: "MIR operation " + strconv.Quote(string(operation.Kind))}
			}
		}
		terminator := block.Terminator
		for _, successor := range terminator.Successors {
			if successor < 0 || int(successor) >= len(function.Blocks) {
				return fmt.Errorf("C emission successor %d is outside %s", successor, function.Name)
			}
		}
		switch terminator.Kind {
		case mir.TerminatorGoto:
			e.features["scalar-control-flow"] = struct{}{}
		case mir.TerminatorCall:
			e.features["scalar-call"] = struct{}{}
			calleeIndex, found := e.bySymbol[terminator.Callee]
			if !found {
				return fmt.Errorf("C call in %s names missing function symbol %d", function.Name, terminator.Callee)
			}
			callee := &e.functions[calleeIndex]
			if callee.ResultType == hir.TypeVoid && terminator.DestinationLocal != mir.NoLocal {
				return fmt.Errorf("void C call in %s has destination local %d", function.Name, terminator.DestinationLocal)
			}
			if callee.ResultType != hir.TypeVoid && terminator.DestinationLocal < 0 {
				return fmt.Errorf("value C call in %s has no destination local", function.Name)
			}
		case mir.TerminatorSwitch:
			e.features["scalar-control-flow"] = struct{}{}
			if err := e.inspectSwitch(functionIndex, terminator); err != nil {
				return err
			}
		case mir.TerminatorReturn:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("return in %s has successors", function.Name)
			}
			if function.ResultType == hir.TypeVoid && terminator.ReturnValue.Kind != mir.OperandInvalid {
				return fmt.Errorf("void return in %s carries a value", function.Name)
			}
			if function.ResultType != hir.TypeVoid {
				if _, err := cOperand(terminator.ReturnValue); err != nil {
					return fmt.Errorf("return in %s: %w", function.Name, err)
				}
			}
		case mir.TerminatorTrap:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("trap in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			if err := e.recordTrapCategories(function.Name, terminator.TrapCategories); err != nil {
				return err
			}
		case mir.TerminatorUnreachable:
			if len(terminator.Successors) != 0 {
				return fmt.Errorf("unreachable terminator in %s has successors", function.Name)
			}
			e.features["trap"] = struct{}{}
			e.trapCategories["unreachable"] = struct{}{}
		default:
			return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
		}
	}
	return nil
}

func validateOperationShape(function string, operation mir.Operation) error {
	switch operation.Kind {
	case mir.OperationAssign, mir.OperationUnary, mir.OperationBinary:
		if operation.DestinationLocal < 0 {
			return fmt.Errorf("%s operation in %s has no destination local", operation.Kind, function)
		}
	}
	return nil
}

func (e *emitter) inspectSwitch(functionIndex int, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	var categories []string
	switch terminator.SwitchKind {
	case "bool":
		return nil
	case "checked-division":
		e.features["checked-division"] = struct{}{}
		categories = []string{"division-by-zero", "signed-division-overflow"}
	case "checked-unary", "checked-arithmetic":
		e.features["checked-arithmetic"] = struct{}{}
		categories = []string{"integer-overflow"}
	case "checked-shift":
		e.features["checked-shift"] = struct{}{}
		categories = []string{"invalid-shift"}
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("checked switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	if err := e.recordTrapCategories(function.Name, categories); err != nil {
		return err
	}
	targets := e.dynamicTrapTargets[functionIndex]
	if targets == nil {
		targets = make(map[mir.BlockID]struct{})
		e.dynamicTrapTargets[functionIndex] = targets
	}
	targets[terminator.Successors[1]] = struct{}{}
	e.trapCategories["unreachable"] = struct{}{}
	return nil
}

func (e *emitter) recordTrapCategories(function string, categories []string) error {
	if len(categories) == 0 {
		return fmt.Errorf("trap in %s has no category", function)
	}
	for _, category := range categories {
		if _, _, ok := cTrapSpec(category); !ok {
			return CapabilityError{Function: function, Feature: "trap category " + strconv.Quote(category)}
		}
		e.trapCategories[category] = struct{}{}
	}
	return nil
}

func (e *emitter) emitPreamble(output *strings.Builder) {
	output.WriteString("/* fairyc constrained C transport: x86_64-linux-gnu, ISO C11 */\n")
	output.WriteString("#include <stdint.h>\n")
	if e.needsBitcast {
		output.WriteString("#include <string.h>\n")
	}
	output.WriteByte('\n')
	if len(e.trapCategories) != 0 {
		categories := make([]string, 0, len(e.trapCategories))
		for category := range e.trapCategories {
			categories = append(categories, category)
		}
		slices.Sort(categories)
		for _, category := range categories {
			name, code, _ := cTrapSpec(category)
			fmt.Fprintf(output, "#define %s UINT32_C(%d)\n", name, code)
		}
		output.WriteString("_Noreturn void fy_runtime_trap(uint32_t category);\n\n")
	}
	if e.needsBitcast {
		output.WriteString("static int32_t fy_bits_to_i32(uint32_t bits) {\n")
		output.WriteString("    int32_t value;\n    memcpy(&value, &bits, sizeof value);\n    return value;\n}\n\n")
	}
	if e.needsShift {
		output.WriteString("static int32_t fy_sar_i32(int32_t value, int32_t amount) {\n")
		output.WriteString("    uint32_t bits = (uint32_t)value;\n")
		output.WriteString("    if (amount == 0 || value >= 0) {\n        return fy_bits_to_i32(bits >> (uint32_t)amount);\n    }\n")
		output.WriteString("    return fy_bits_to_i32((bits >> (uint32_t)amount) | (UINT32_MAX << (32u - (uint32_t)amount)));\n}\n\n")
	}
}

func (e *emitter) emitPrototype(output *strings.Builder, functionIndex int) {
	function := &e.functions[functionIndex]
	fmt.Fprintf(output, "static %s fy_fn_%d(", cResultType(function.ResultType), functionIndex)
	emitParameters(output, function.Parameters)
	output.WriteString(");\n")
}

func (e *emitter) emitFunction(output *strings.Builder, functionIndex int) error {
	function := &e.functions[functionIndex]
	fmt.Fprintf(output, "static %s fy_fn_%d(", cResultType(function.ResultType), functionIndex)
	emitParameters(output, function.Parameters)
	output.WriteString(") {\n")
	parameters := make(map[mir.LocalID]struct{}, len(function.Parameters))
	for _, parameter := range function.Parameters {
		parameters[parameter] = struct{}{}
		fmt.Fprintf(output, "    (void)fy_l_%d;\n", parameter)
	}
	for _, local := range collectLocals(*function) {
		if _, parameter := parameters[local]; parameter {
			continue
		}
		fmt.Fprintf(output, "    int32_t fy_l_%d;\n", local)
	}
	dynamicTargets := e.dynamicTrapTargets[functionIndex]
	if len(dynamicTargets) != 0 {
		output.WriteString("    uint32_t fy_pending_trap = FY_TRAP_UNREACHABLE;\n")
	}
	fmt.Fprintf(output, "    goto fy_b_%d;\n", function.Entry)
	blocks := slices.Clone(function.Blocks)
	slices.SortFunc(blocks, func(left, right mir.Block) int { return int(left.ID - right.ID) })
	for blockIndex := range blocks {
		block := &blocks[blockIndex]
		fmt.Fprintf(output, "fy_b_%d:\n", block.ID)
		for operationIndex := range block.Operations {
			if err := e.emitOperation(output, function.Name, block.Operations[operationIndex]); err != nil {
				return err
			}
		}
		if err := e.emitTerminator(output, functionIndex, block.ID, block.Terminator); err != nil {
			return err
		}
	}
	output.WriteString("}\n\n")
	return nil
}

func (e *emitter) emitOperation(output *strings.Builder, function string, operation mir.Operation) error {
	switch operation.Kind {
	case mir.OperationAssign:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("assign in %s: %w", function, err)
		}
		fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
	case mir.OperationUnary:
		operand, err := oneOperand(operation.Operands)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("unary in %s: %w", function, err)
		}
		switch operation.Operator {
		case "+":
			fmt.Fprintf(output, "    fy_l_%d = %s;\n", operation.DestinationLocal, value)
		case "-":
			fmt.Fprintf(output, "    fy_l_%d = -(%s);\n", operation.DestinationLocal, value)
		case "!":
			fmt.Fprintf(output, "    fy_l_%d = (%s == INT32_C(0)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, value)
		case "~":
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32(~(uint32_t)(%s));\n", operation.DestinationLocal, value)
		}
	case mir.OperationBinary:
		left, right, err := twoOperands(operation.Operands)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("binary in %s: %w", function, err)
		}
		switch operation.Operator {
		case "+", "-", "*", "/", "%":
			fmt.Fprintf(output, "    fy_l_%d = (%s) %s (%s);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		case ">>":
			fmt.Fprintf(output, "    fy_l_%d = fy_sar_i32(%s, %s);\n", operation.DestinationLocal, leftValue, rightValue)
		case "&", "^", "|":
			fmt.Fprintf(output, "    fy_l_%d = fy_bits_to_i32((uint32_t)(%s) %s (uint32_t)(%s));\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		case "==", "!=", "<", "<=", ">", ">=":
			fmt.Fprintf(output, "    fy_l_%d = ((%s) %s (%s)) ? INT32_C(1) : INT32_C(0);\n", operation.DestinationLocal, leftValue, operation.Operator, rightValue)
		}
	case mir.OperationUse, mir.OperationArithmeticCheck, mir.OperationCheckedDivision:
	default:
		return CapabilityError{Function: function, Feature: "MIR operation " + strconv.Quote(string(operation.Kind))}
	}
	return nil
}

func (e *emitter) emitTerminator(output *strings.Builder, functionIndex int, blockID mir.BlockID, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	switch terminator.Kind {
	case mir.TerminatorGoto:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("goto in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
	case mir.TerminatorCall:
		target, err := oneSuccessor(terminator)
		if err != nil {
			return fmt.Errorf("call in %s: %w", function.Name, err)
		}
		calleeIndex, found := e.bySymbol[terminator.Callee]
		if !found {
			return fmt.Errorf("call in %s names missing function symbol %d", function.Name, terminator.Callee)
		}
		callee := &e.functions[calleeIndex]
		if len(terminator.Arguments) != len(callee.Parameters) {
			return fmt.Errorf("call in %s passes %d arguments to %s, want %d", function.Name, len(terminator.Arguments), callee.Name, len(callee.Parameters))
		}
		output.WriteString("    ")
		if callee.ResultType != hir.TypeVoid {
			fmt.Fprintf(output, "fy_l_%d = ", terminator.DestinationLocal)
		}
		fmt.Fprintf(output, "fy_fn_%d(", calleeIndex)
		for index, argument := range terminator.Arguments {
			if index != 0 {
				output.WriteString(", ")
			}
			value, err := cOperand(argument)
			if err != nil {
				return fmt.Errorf("call argument in %s: %w", function.Name, err)
			}
			output.WriteString(value)
		}
		output.WriteString(");\n")
		fmt.Fprintf(output, "    goto fy_b_%d;\n", target)
	case mir.TerminatorSwitch:
		return e.emitSwitch(output, functionIndex, terminator)
	case mir.TerminatorReturn:
		if function.ResultType == hir.TypeVoid {
			output.WriteString("    return;\n")
			return nil
		}
		value, err := cOperand(terminator.ReturnValue)
		if err != nil {
			return fmt.Errorf("return in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    return %s;\n", value)
	case mir.TerminatorTrap:
		if _, dynamic := e.dynamicTrapTargets[functionIndex][blockID]; dynamic {
			output.WriteString("    fy_runtime_trap(fy_pending_trap);\n")
			return nil
		}
		if len(terminator.TrapCategories) == 0 {
			return fmt.Errorf("trap in %s has no category", function.Name)
		}
		name, _, ok := cTrapSpec(terminator.TrapCategories[0])
		if !ok {
			return CapabilityError{Function: function.Name, Feature: "trap category " + strconv.Quote(terminator.TrapCategories[0])}
		}
		fmt.Fprintf(output, "    fy_runtime_trap(%s);\n", name)
	case mir.TerminatorUnreachable:
		output.WriteString("    fy_runtime_trap(FY_TRAP_UNREACHABLE);\n")
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR terminator " + strconv.Quote(string(terminator.Kind))}
	}
	return nil
}

func (e *emitter) emitSwitch(output *strings.Builder, functionIndex int, terminator mir.Terminator) error {
	function := &e.functions[functionIndex]
	if len(terminator.Successors) != 2 {
		return fmt.Errorf("switch in %s has %d successors", function.Name, len(terminator.Successors))
	}
	success, failure := terminator.Successors[0], terminator.Successors[1]
	switch terminator.SwitchKind {
	case "bool":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("bool switch in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s != INT32_C(0)) goto fy_b_%d;\n", value, success)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", failure)
	case "checked-unary":
		operand, err := oneOperand(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		if terminator.Operator != "-" {
			return CapabilityError{Function: function.Name, Feature: "checked unary operator " + strconv.Quote(terminator.Operator)}
		}
		value, err := cOperand(operand)
		if err != nil {
			return fmt.Errorf("checked unary in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_MIN) { fy_pending_trap = FY_TRAP_INTEGER_OVERFLOW; goto fy_b_%d; }\n", value, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-arithmetic":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		if terminator.Operator != "+" && terminator.Operator != "-" && terminator.Operator != "*" {
			return CapabilityError{Function: function.Name, Feature: "checked arithmetic operator " + strconv.Quote(terminator.Operator)}
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked arithmetic in %s: %w", function.Name, err)
		}
		expression := fmt.Sprintf("((int64_t)(%s) %s (int64_t)(%s))", leftValue, terminator.Operator, rightValue)
		fmt.Fprintf(output, "    if (%s < (int64_t)INT32_MIN || %s > (int64_t)INT32_MAX) { fy_pending_trap = FY_TRAP_INTEGER_OVERFLOW; goto fy_b_%d; }\n", expression, expression, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-division":
		left, right, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		leftValue, err := cOperand(left)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		rightValue, err := cOperand(right)
		if err != nil {
			return fmt.Errorf("checked division in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s == INT32_C(0)) { fy_pending_trap = FY_TRAP_DIVISION_BY_ZERO; goto fy_b_%d; }\n", rightValue, failure)
		fmt.Fprintf(output, "    if (%s == INT32_MIN && %s == -INT32_C(1)) { fy_pending_trap = FY_TRAP_SIGNED_DIVISION_OVERFLOW; goto fy_b_%d; }\n", leftValue, rightValue, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	case "checked-shift":
		_, amount, err := twoOperands(terminator.SwitchOperands)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		amountValue, err := cOperand(amount)
		if err != nil {
			return fmt.Errorf("checked shift in %s: %w", function.Name, err)
		}
		fmt.Fprintf(output, "    if (%s < INT32_C(0) || %s >= INT32_C(32)) { fy_pending_trap = FY_TRAP_INVALID_SHIFT; goto fy_b_%d; }\n", amountValue, amountValue, failure)
		fmt.Fprintf(output, "    goto fy_b_%d;\n", success)
	default:
		return CapabilityError{Function: function.Name, Feature: "MIR switch " + strconv.Quote(terminator.SwitchKind)}
	}
	return nil
}

func emitParameters(output *strings.Builder, parameters []mir.LocalID) {
	if len(parameters) == 0 {
		output.WriteString("void")
		return
	}
	for index, parameter := range parameters {
		if index != 0 {
			output.WriteString(", ")
		}
		fmt.Fprintf(output, "int32_t fy_l_%d", parameter)
	}
}

func collectLocals(function mir.Function) []mir.LocalID {
	values := make(map[mir.LocalID]struct{})
	for _, parameter := range function.Parameters {
		values[parameter] = struct{}{}
	}
	addOperand := func(operand mir.Operand) {
		if operand.Kind == mir.OperandLocal {
			values[operand.Local] = struct{}{}
		}
	}
	for _, block := range function.Blocks {
		for _, operation := range block.Operations {
			if operation.DestinationLocal != mir.NoLocal {
				values[operation.DestinationLocal] = struct{}{}
			}
			for _, operand := range operation.Operands {
				addOperand(operand)
			}
		}
		terminator := block.Terminator
		if terminator.DestinationLocal != mir.NoLocal {
			values[terminator.DestinationLocal] = struct{}{}
		}
		for _, operand := range terminator.Arguments {
			addOperand(operand)
		}
		for _, operand := range terminator.SwitchOperands {
			addOperand(operand)
		}
		addOperand(terminator.ReturnValue)
	}
	result := make([]mir.LocalID, 0, len(values))
	for value := range values {
		if value >= 0 {
			result = append(result, value)
		}
	}
	slices.Sort(result)
	return result
}

func cResultType(value hir.Type) string {
	if value == hir.TypeVoid {
		return "void"
	}
	return "int32_t"
}

func cOperand(operand mir.Operand) (string, error) {
	switch operand.Kind {
	case mir.OperandLocal:
		if operand.Local < 0 {
			return "", fmt.Errorf("local operand has invalid id %d", operand.Local)
		}
		return fmt.Sprintf("fy_l_%d", operand.Local), nil
	case mir.OperandI32:
		return cI32(operand.I32), nil
	default:
		return "", fmt.Errorf("unsupported scalar operand kind %q", operand.Kind)
	}
}

func cI32(value int32) string {
	if value == -2147483648 {
		return "INT32_MIN"
	}
	if value < 0 {
		return fmt.Sprintf("(-INT32_C(%d))", -int64(value))
	}
	return fmt.Sprintf("INT32_C(%d)", value)
}

// TrapCode returns the fixed Stage 0 runtime code for one canonical trap category.
func TrapCode(category string) (uint32, bool) {
	_, code, ok := cTrapSpec(category)
	return code, ok
}

func cTrapSpec(category string) (string, uint32, bool) {
	switch category {
	case "bounds":
		return "FY_TRAP_BOUNDS", 1, true
	case "integer-overflow":
		return "FY_TRAP_INTEGER_OVERFLOW", 2, true
	case "division-by-zero":
		return "FY_TRAP_DIVISION_BY_ZERO", 3, true
	case "signed-division-overflow":
		return "FY_TRAP_SIGNED_DIVISION_OVERFLOW", 4, true
	case "invalid-shift":
		return "FY_TRAP_INVALID_SHIFT", 5, true
	case "assertion-failed":
		return "FY_TRAP_ASSERTION_FAILED", 6, true
	case "unreachable":
		return "FY_TRAP_UNREACHABLE", 7, true
	default:
		return "", 0, false
	}
}

func supportedUnary(operator string) bool {
	return operator == "+" || operator == "-" || operator == "!" || operator == "~"
}

func supportedBinary(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", ">>", "&", "^", "|", "==", "!=", "<", "<=", ">", ">=":
		return true
	default:
		return false
	}
}

func oneOperand(operands []mir.Operand) (mir.Operand, error) {
	if len(operands) != 1 || operands[0].Kind == mir.OperandInvalid {
		return mir.Operand{}, fmt.Errorf("expected one scalar operand")
	}
	return operands[0], nil
}

func twoOperands(operands []mir.Operand) (mir.Operand, mir.Operand, error) {
	if len(operands) != 2 || operands[0].Kind == mir.OperandInvalid || operands[1].Kind == mir.OperandInvalid {
		return mir.Operand{}, mir.Operand{}, fmt.Errorf("expected two scalar operands")
	}
	return operands[0], operands[1], nil
}

func oneSuccessor(terminator mir.Terminator) (mir.BlockID, error) {
	if len(terminator.Successors) != 1 {
		return mir.NoBlock, fmt.Errorf("expected one successor, got %d", len(terminator.Successors))
	}
	return terminator.Successors[0], nil
}
