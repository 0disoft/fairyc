// Package interpreter executes verified Stage 0 MIR as the first semantic oracle.
package interpreter

import (
	"errors"
	"fmt"
	"math"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

// TestIntrinsicKind identifies one closed, fixture-only interpreter hook.
type TestIntrinsicKind string

const (
	OutcomeReturn                                  = "return"
	OutcomeTrap                                    = "trap"
	OutcomeLimit                                   = "limit"
	LimitStep                                      = "step-limit"
	LimitOutput                                    = "output-limit"
	TestIntrinsicWriteStdoutByte TestIntrinsicKind = "write-stdout-byte"
	TestIntrinsicWriteStderrByte TestIntrinsicKind = "write-stderr-byte"
	maxCallDepth                                   = 256
	maxSteps                                       = 1_000_000
	maxConfiguredTestOutputBytes                   = 1 << 20
)

// Options enables closed interpreter-harness behavior that is not part of the Fairy C language.
type Options struct {
	TestIntrinsics map[string]TestIntrinsicKind
	OutputLimit    int
}

// Outcome is one bounded interpreter completion.
type Outcome struct {
	Kind         string
	Value        int32
	TrapCategory string
	DropTrace    []DropEvent
	RegionTrace  []RegionEvent
	Stdout       []byte
	Stderr       []byte
	IOTrace      []IOEvent
}

// IOEvent records one ordered byte written to a controlled in-memory test stream.
type IOEvent struct {
	Stream string
	Byte   byte
}

// DropEvent is one resource drop-glue invocation in execution order.
type DropEvent struct {
	ResourceType hir.Type
	Binding      hir.Span
}

// RegionEvent records one normal lexical reset and the work completed before it.
type RegionEvent struct {
	Region               hir.Span
	ExitKind             string
	Placements           int
	DropCountBeforeReset int
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
	return ExecuteWithOptions(program, entry, Options{})
}

// ExecuteWithOptions runs verified MIR with explicitly configured, in-memory test intrinsics.
func ExecuteWithOptions(program mir.Program, entry string, options Options) (Outcome, error) {
	if failures := mir.Verify(program); len(failures) != 0 {
		return Outcome{}, fmt.Errorf("verify MIR before interpretation: %v", failures)
	}
	runtime := runtime{
		byName:         make(map[string]*mir.Function, len(program.Functions)),
		bySymbol:       make(map[hir.SymbolID]*mir.Function, len(program.Functions)),
		structTypes:    make(map[hir.Type]*mir.StructType, len(program.StructTypes)),
		enumTypes:      make(map[hir.Type]*mir.EnumType, len(program.EnumTypes)),
		resourceTypes:  make(map[hir.Type]*mir.ResourceType, len(program.ResourceTypes)),
		testIntrinsics: make(map[hir.SymbolID]TestIntrinsicKind, len(options.TestIntrinsics)),
		outputLimit:    options.OutputLimit,
		steps:          maxSteps,
	}
	for index := range program.StructTypes {
		definition := &program.StructTypes[index]
		runtime.structTypes[definition.Type] = definition
	}
	for index := range program.EnumTypes {
		definition := &program.EnumTypes[index]
		runtime.enumTypes[definition.Type] = definition
	}
	for index := range program.ResourceTypes {
		definition := &program.ResourceTypes[index]
		runtime.resourceTypes[definition.Type] = definition
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
	if err := runtime.configureTestIntrinsics(entry, options); err != nil {
		return Outcome{}, err
	}
	completion, err := runtime.call(function, nil, 0)
	if err != nil {
		var capability CapabilityError
		if errors.As(err, &capability) && capability.Kind != "" {
			return runtime.outcome(OutcomeLimit, 0, ""), err
		}
		return Outcome{}, err
	}
	if completion.trapCategory != "" {
		return runtime.outcome(OutcomeTrap, 0, completion.trapCategory), nil
	}
	value, err := completion.value.asI32()
	if err != nil {
		return Outcome{}, fmt.Errorf("entry function %q: %w", entry, err)
	}
	return runtime.outcome(OutcomeReturn, value, ""), nil
}

type runtime struct {
	byName         map[string]*mir.Function
	bySymbol       map[hir.SymbolID]*mir.Function
	structTypes    map[hir.Type]*mir.StructType
	enumTypes      map[hir.Type]*mir.EnumType
	resourceTypes  map[hir.Type]*mir.ResourceType
	testIntrinsics map[hir.SymbolID]TestIntrinsicKind
	outputLimit    int
	steps          int
	dropTrace      []DropEvent
	regionTrace    []RegionEvent
	stdout         []byte
	stderr         []byte
	ioTrace        []IOEvent
}

func (r *runtime) configureTestIntrinsics(entry string, options Options) error {
	if len(options.TestIntrinsics) == 0 {
		if options.OutputLimit != 0 {
			return fmt.Errorf("test output limit requires at least one configured test intrinsic")
		}
		return nil
	}
	if options.OutputLimit <= 0 || options.OutputLimit > maxConfiguredTestOutputBytes {
		return fmt.Errorf("test output limit must be between 1 and %d bytes", maxConfiguredTestOutputBytes)
	}
	names := make([]string, 0, len(options.TestIntrinsics))
	for name := range options.TestIntrinsics {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		kind := options.TestIntrinsics[name]
		if name == entry {
			return fmt.Errorf("entry function %q cannot be a test intrinsic", name)
		}
		function := r.byName[name]
		if function == nil {
			return fmt.Errorf("test intrinsic function %q is missing", name)
		}
		if len(function.Parameters) != 1 || function.ResultType != hir.TypeI32 {
			return fmt.Errorf("test intrinsic function %q must have type fn(i32) -> i32", name)
		}
		switch kind {
		case TestIntrinsicWriteStdoutByte, TestIntrinsicWriteStderrByte:
		default:
			return fmt.Errorf("test intrinsic function %q has unsupported kind %q", name, kind)
		}
		if _, duplicate := r.testIntrinsics[function.Symbol]; duplicate {
			return fmt.Errorf("test intrinsic symbol %d is configured twice", function.Symbol)
		}
		r.testIntrinsics[function.Symbol] = kind
	}
	return nil
}

func (r *runtime) outcome(kind string, value int32, trapCategory string) Outcome {
	return Outcome{
		Kind: kind, Value: value, TrapCategory: trapCategory,
		DropTrace: slices.Clone(r.dropTrace), RegionTrace: slices.Clone(r.regionTrace),
		Stdout: slices.Clone(r.stdout), Stderr: slices.Clone(r.stderr), IOTrace: slices.Clone(r.ioTrace),
	}
}

func (r *runtime) callTestIntrinsic(kind TestIntrinsicKind, arguments []runtimeValue) (runtimeValue, error) {
	if len(arguments) != 1 {
		return runtimeValue{}, fmt.Errorf("test intrinsic %q received %d arguments, want 1", kind, len(arguments))
	}
	value, err := arguments[0].asI32()
	if err != nil {
		return runtimeValue{}, fmt.Errorf("test intrinsic %q byte: %w", kind, err)
	}
	if value < 0 || value > 255 {
		return runtimeValue{}, capabilityError("test intrinsic %q byte %d is outside 0..255", kind, value)
	}
	stream := "stdout"
	if kind == TestIntrinsicWriteStderrByte {
		stream = "stderr"
	}
	if len(r.ioTrace) >= r.outputLimit {
		return runtimeValue{}, capabilityErrorKind(LimitOutput, "interpreter test output limit of %d bytes exceeded", r.outputLimit)
	}
	byteValue := byte(value)
	if stream == "stdout" {
		r.stdout = append(r.stdout, byteValue)
	} else {
		r.stderr = append(r.stderr, byteValue)
	}
	r.ioTrace = append(r.ioTrace, IOEvent{Stream: stream, Byte: byteValue})
	return i32Value(1), nil
}

type valueKind uint8

const (
	valueI32 valueKind = iota + 1
	valueU8
	valueU32
	valueUSize
	valueView
	valueNull
	valueResult
	valueOption
	valueEnum
	valueStruct
	valueResource
)

type structState struct {
	definition *mir.StructType
	fields     map[string]runtimeValue
}

type enumState struct {
	definition *mir.EnumType
	variant    string
	fields     map[string]runtimeValue
}

type resourceState struct {
	definition *mir.ResourceType
	fields     map[string]runtimeValue
	dropped    bool
}

type taggedState struct {
	variant string
	payload *runtimeValue
	dropped bool
}

type runtimeValue struct {
	kind       valueKind
	i32        int32
	u64        uint64
	staticData string
	typ        hir.Type
	tagged     *taggedState
	enum       *enumState
	structure  *structState
	resource   *resourceState
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

func enumValue(definition *mir.EnumType, variantName string, fieldNames []string, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(fieldNames) != len(values) {
		return runtimeValue{}, fmt.Errorf("invalid enum construction definition")
	}
	var variant *mir.EnumVariant
	for index := range definition.Variants {
		if definition.Variants[index].Name == variantName {
			variant = &definition.Variants[index]
			break
		}
	}
	if variant == nil || len(variant.Fields) != len(fieldNames) {
		return runtimeValue{}, fmt.Errorf("enum %q has no matching variant %q", definition.Type, variantName)
	}
	declared := make(map[string]struct{}, len(variant.Fields))
	for _, field := range variant.Fields {
		declared[field.Name] = struct{}{}
	}
	state := &enumState{definition: definition, variant: variantName, fields: make(map[string]runtimeValue, len(fieldNames))}
	for index, fieldName := range fieldNames {
		if _, found := declared[fieldName]; !found {
			return runtimeValue{}, fmt.Errorf("enum %q variant %q has no field %q", definition.Type, variantName, fieldName)
		}
		if _, duplicate := state.fields[fieldName]; duplicate {
			return runtimeValue{}, fmt.Errorf("enum %q variant %q repeats field %q", definition.Type, variantName, fieldName)
		}
		state.fields[fieldName] = values[index]
	}
	return runtimeValue{kind: valueEnum, enum: state}, nil
}

func resourceValue(definition *mir.ResourceType, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(definition.Fields) == 0 || len(definition.Fields) != len(values) || definition.DropKind != hir.ResourceDropEmpty {
		return runtimeValue{}, fmt.Errorf("invalid resource construction definition")
	}
	state := &resourceState{definition: definition, fields: make(map[string]runtimeValue, len(definition.Fields))}
	for index, field := range definition.Fields {
		if field.Name == "" {
			return runtimeValue{}, fmt.Errorf("resource %q has an empty field name", definition.Type)
		}
		if _, duplicate := state.fields[field.Name]; duplicate {
			return runtimeValue{}, fmt.Errorf("resource %q has duplicate field %q", definition.Type, field.Name)
		}
		state.fields[field.Name] = values[index]
	}
	return runtimeValue{kind: valueResource, resource: state}, nil
}

func structValue(definition *mir.StructType, values []runtimeValue) (runtimeValue, error) {
	if definition == nil || definition.Type == hir.TypeInvalid || len(definition.Fields) == 0 || len(definition.Fields) != len(values) {
		return runtimeValue{}, fmt.Errorf("invalid struct construction definition")
	}
	state := &structState{definition: definition, fields: make(map[string]runtimeValue, len(definition.Fields))}
	for index, field := range definition.Fields {
		if field.Name == "" {
			return runtimeValue{}, fmt.Errorf("struct %q has an empty field name", definition.Type)
		}
		if _, duplicate := state.fields[field.Name]; duplicate {
			return runtimeValue{}, fmt.Errorf("struct %q has duplicate field %q", definition.Type, field.Name)
		}
		state.fields[field.Name] = values[index]
	}
	return runtimeValue{kind: valueStruct, structure: state}, nil
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

func (v runtimeValue) field(definition *mir.ResourceType, name string) (runtimeValue, error) {
	if v.kind != valueResource || v.resource == nil {
		return runtimeValue{}, capabilityError("field read received a non-resource value")
	}
	if definition == nil || v.resource.definition == nil || v.resource.definition.Type != definition.Type {
		return runtimeValue{}, fmt.Errorf("field read resource definition does not match the runtime value")
	}
	if v.resource.dropped {
		return runtimeValue{}, fmt.Errorf("resource %q was read after drop", v.resource.definition.Type)
	}
	value, found := v.resource.fields[name]
	if !found {
		return runtimeValue{}, fmt.Errorf("resource %q has no field %q", v.resource.definition.Type, name)
	}
	return value, nil
}

func (v runtimeValue) structField(definition *mir.StructType, name string) (runtimeValue, error) {
	if v.kind != valueStruct || v.structure == nil {
		return runtimeValue{}, capabilityError("field read received a non-struct value")
	}
	if definition == nil || v.structure.definition == nil || v.structure.definition.Type != definition.Type {
		return runtimeValue{}, fmt.Errorf("field read struct definition does not match the runtime value")
	}
	value, found := v.structure.fields[name]
	if !found {
		return runtimeValue{}, fmt.Errorf("struct %q has no field %q", v.structure.definition.Type, name)
	}
	return value, nil
}

func (v runtimeValue) enumPayload(definition *mir.EnumType, variant, field string) (runtimeValue, error) {
	if v.kind != valueEnum || v.enum == nil || v.enum.definition == nil {
		return runtimeValue{}, capabilityError("enum payload received a non-enum value")
	}
	if definition == nil || v.enum.definition.Type != definition.Type || v.enum.variant != variant {
		return runtimeValue{}, fmt.Errorf("enum payload definition or active variant does not match")
	}
	value, found := v.enum.fields[field]
	if !found {
		return runtimeValue{}, fmt.Errorf("enum %q variant %q has no active field %q", definition.Type, variant, field)
	}
	return value, nil
}

type callCompletion struct {
	value        runtimeValue
	trapCategory string
}

type runtimeRegion struct {
	span       hir.Span
	placements []runtimeValue
}

func (r *runtime) call(function *mir.Function, arguments []runtimeValue, depth int) (callCompletion, error) {
	if depth >= maxCallDepth {
		return callCompletion{}, capabilityErrorKind("call-depth-limit", "interpreter call-depth limit exceeded")
	}
	if len(arguments) != len(function.Parameters) {
		return callCompletion{}, fmt.Errorf("call %q received %d argument(s), want %d", function.Name, len(arguments), len(function.Parameters))
	}
	locals := make(map[mir.LocalID]runtimeValue)
	regions := make(map[hir.SymbolID]*runtimeRegion)
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
				value, trap, err := evaluateUnary(locals, operation)
				if err != nil {
					return callCompletion{}, fmt.Errorf("unary in %q: %w", function.Name, err)
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationBinary:
				result, trap, err := evaluateBinary(locals, operation)
				if err != nil {
					return callCompletion{}, fmt.Errorf("binary in %q: %w", function.Name, err)
				}
				if trap != "" {
					return callCompletion{trapCategory: trap}, nil
				}
				locals[operation.DestinationLocal] = result
			case mir.OperationScalarCast:
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("value-preserving integer cast in %q: %w", function.Name, err)
				}
				converted, err := valuePreservingIntegerCast(value, operation.OwnerType, operation.Type)
				if err != nil {
					return callCompletion{}, fmt.Errorf("value-preserving integer cast in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = converted
			case mir.OperationViewLength:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("borrowed-view length in %q: %w", function.Name, err)
				}
				if receiver.kind != valueView || receiver.typ != operation.OwnerType || operation.Type != hir.TypeUSize || operation.Field != "len" {
					return callCompletion{}, fmt.Errorf("borrowed-view length in %q has incompatible runtime value", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueUSize, u64: receiver.u64, typ: hir.TypeUSize}
			case mir.OperationViewEmpty:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("borrowed-view emptiness in %q: %w", function.Name, err)
				}
				if receiver.kind != valueView || receiver.typ != operation.OwnerType || operation.Type != hir.TypeBool || operation.Field != "is_empty" {
					return callCompletion{}, fmt.Errorf("borrowed-view emptiness in %q has incompatible runtime value", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueI32, i32: boolValue(receiver.u64 == 0), typ: hir.TypeBool}
			case mir.OperationViewIndex:
				receiver, index, err := evaluateViewIndexOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("immutable byte-view index in %q: %w", function.Name, err)
				}
				if receiver.typ != operation.OwnerType || operation.OwnerType != hir.SharedBorrow(hir.TypeByteSlice) ||
					operation.IndexType != hir.TypeUSize || operation.Type != hir.TypeU8 || operation.Operator != "[]" {
					return callCompletion{}, fmt.Errorf("immutable byte-view index in %q has incompatible runtime value", function.Name)
				}
				if index.u64 >= receiver.u64 || index.u64 >= uint64(len(receiver.staticData)) {
					return callCompletion{}, fmt.Errorf("immutable byte-view index in %q executed without a dominating bounds check", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{kind: valueU8, u64: uint64(receiver.staticData[index.u64]), typ: hir.TypeU8}
			case mir.OperationViewIndexCheck:
			case mir.OperationViewSlice:
				receiver, start, end, err := evaluateViewSliceOperands(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q: %w", function.Name, err)
				}
				wantView := hir.SharedBorrow(hir.TypeByteSlice)
				if receiver.typ != operation.OwnerType || operation.OwnerType != wantView ||
					operation.IndexType != hir.TypeUSize || operation.Type != wantView || operation.Operator != ".." {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q has incompatible runtime value", function.Name)
				}
				if start.u64 > end.u64 || end.u64 > receiver.u64 || end.u64 > uint64(len(receiver.staticData)) {
					return callCompletion{}, fmt.Errorf("immutable byte-view slice in %q executed without a dominating bounds check", function.Name)
				}
				locals[operation.DestinationLocal] = runtimeValue{
					kind: valueView, u64: end.u64 - start.u64, typ: wantView,
					staticData: receiver.staticData[start.u64:end.u64],
				}
			case mir.OperationViewSliceCheck:
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
			case mir.OperationEnumConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("enum construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := enumValue(r.enumTypes[operation.Type], operation.Variant, operation.Fields, values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationEnumPayload:
				container, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum payload in %q: %w", function.Name, err)
				}
				payload, err := container.enumPayload(r.enumTypes[operation.OwnerType], operation.Variant, operation.Field)
				if err != nil {
					return callCompletion{}, fmt.Errorf("enum payload in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = payload
			case mir.OperationRawAddress:
				return callCompletion{}, capabilityErrorKind("raw-address", "raw address in %q requires a native backend", function.Name)
			case mir.OperationStructConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("struct construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := structValue(r.structTypes[operation.Type], values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("struct construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationResourceConstruct:
				values := make([]runtimeValue, len(operation.Operands))
				for index, operand := range operation.Operands {
					value, err := evaluate(locals, operand)
					if err != nil {
						return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
					}
					values[index] = value
				}
				constructed, err := resourceValue(r.resourceTypes[operation.Type], values)
				if err != nil {
					return callCompletion{}, fmt.Errorf("resource construction in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = constructed
			case mir.OperationFieldRead:
				receiver, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				var value runtimeValue
				if definition := r.structTypes[operation.OwnerType]; definition != nil {
					value, err = receiver.structField(definition, operation.Field)
				} else {
					value, err = receiver.field(r.resourceTypes[operation.OwnerType], operation.Field)
				}
				if err != nil {
					return callCompletion{}, fmt.Errorf("field read in %q: %w", function.Name, err)
				}
				locals[operation.DestinationLocal] = value
			case mir.OperationRegionEnter:
				if _, active := regions[operation.Symbol]; active {
					return callCompletion{}, fmt.Errorf("region enter in %q reuses active region %d", function.Name, operation.Symbol)
				}
				regions[operation.Symbol] = &runtimeRegion{span: operation.Span}
			case mir.OperationRegionPlace:
				region := regions[operation.Symbol]
				if region == nil {
					return callCompletion{}, fmt.Errorf("region place in %q uses inactive region %d", function.Name, operation.Symbol)
				}
				value, err := evaluateOne(locals, operation.Operands)
				if err != nil {
					return callCompletion{}, fmt.Errorf("region place in %q: %w", function.Name, err)
				}
				region.placements = append(region.placements, value)
				locals[operation.DestinationLocal] = value
			case mir.OperationRegionReset:
				region := regions[operation.Symbol]
				if region == nil {
					return callCompletion{}, fmt.Errorf("region reset in %q uses inactive region %d", function.Name, operation.Symbol)
				}
				r.regionTrace = append(r.regionTrace, RegionEvent{
					Region: region.span, ExitKind: operation.ExitKind,
					Placements: len(region.placements), DropCountBeforeReset: len(r.dropTrace),
				})
				delete(regions, operation.Symbol)
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
			if terminator.ForeignABI != "" {
				return callCompletion{}, capabilityErrorKind("foreign-call", "interpreter cannot execute foreign ABI %q call symbol %d", terminator.ForeignABI, terminator.Callee)
			}
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
			completion := callCompletion{}
			var callErr error
			if intrinsic, configured := r.testIntrinsics[callee.Symbol]; configured {
				value, err := r.callTestIntrinsic(intrinsic, values)
				if err != nil {
					return callCompletion{}, err
				}
				completion.value = value
			} else {
				completion, callErr = r.call(callee, values, depth+1)
			}
			if callErr != nil || completion.trapCategory != "" {
				return completion, callErr
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
			if len(regions) != 0 {
				return callCompletion{}, fmt.Errorf("return from %q leaves %d active region(s)", function.Name, len(regions))
			}
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
		if value.resource.definition == nil || value.resource.definition.DropKind != hir.ResourceDropEmpty {
			return fmt.Errorf("resource value has no executable drop definition")
		}
		if value.resource.dropped {
			return fmt.Errorf("resource %q was dropped twice", value.resource.definition.Type)
		}
		value.resource.dropped = true
		r.dropTrace = append(r.dropTrace, DropEvent{ResourceType: value.resource.definition.Type, Binding: binding})
		for index := len(value.resource.definition.Fields) - 1; index >= 0; index-- {
			field := value.resource.definition.Fields[index]
			if err := r.dropValue(value.resource.fields[field.Name], binding); err != nil {
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
		if operand.Type == hir.TypeBool {
			if operand.I32 != 0 && operand.I32 != 1 {
				return runtimeValue{}, capabilityError("boolean immediate has value %d", operand.I32)
			}
			return runtimeValue{kind: valueI32, i32: operand.I32, typ: hir.TypeBool}, nil
		}
		return i32Value(operand.I32), nil
	case mir.OperandU8:
		return runtimeValue{kind: valueU8, u64: operand.U64, typ: hir.TypeU8}, nil
	case mir.OperandU32:
		return runtimeValue{kind: valueU32, u64: operand.U64, typ: hir.TypeU32}, nil
	case mir.OperandUSize:
		return runtimeValue{kind: valueUSize, u64: operand.U64, typ: hir.TypeUSize}, nil
	case mir.OperandString, mir.OperandByteString:
		return runtimeValue{kind: valueView, u64: uint64(len(operand.StaticData)), staticData: operand.StaticData, typ: operand.Type}, nil
	case mir.OperandNull:
		return runtimeValue{kind: valueNull, typ: operand.Type}, nil
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

func evaluateBinary(locals map[mir.LocalID]runtimeValue, operation mir.Operation) (runtimeValue, string, error) {
	if len(operation.Operands) != 2 {
		return runtimeValue{}, "", fmt.Errorf("binary operation has %d operands, want 2", len(operation.Operands))
	}
	leftValue, err := evaluate(locals, operation.Operands[0])
	if err != nil {
		return runtimeValue{}, "", err
	}
	rightValue, err := evaluate(locals, operation.Operands[1])
	if err != nil {
		return runtimeValue{}, "", err
	}
	switch operation.Operator {
	case "==", "!=":
		_, _, pointer := hir.RawPointerElement(operation.OwnerType)
		if (!hir.ScalarEqualityType(operation.OwnerType) && !pointer) || operation.Type != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("equality comparison has incompatible owner or result type")
		}
		equal, err := equalScalarValues(leftValue, rightValue, operation.OwnerType)
		if err != nil {
			return runtimeValue{}, "", err
		}
		if operation.Operator == "!=" {
			equal = !equal
		}
		return i32Value(boolValue(equal)), "", nil
	case "<", "<=", ">", ">=":
		if !hir.ScalarOrderedType(operation.OwnerType) || operation.Type != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("ordered comparison has incompatible owner or result type")
		}
		result, err := orderedScalarValues(operation.Operator, leftValue, rightValue, operation.OwnerType)
		return i32Value(boolValue(result)), "", err
	case "+", "-", "*", "/", "%":
		if !hir.CheckedDivisionType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("checked integer operation has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	case ">>", "<<":
		if !hir.CheckedShiftType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("checked shift has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	case "&", "^", "|":
		if !hir.BitwiseIntegerType(operation.OwnerType) || operation.Type != operation.OwnerType {
			return runtimeValue{}, "", capabilityError("bitwise operation has incompatible owner or result type")
		}
		if operation.OwnerType != hir.TypeI32 {
			return checkedUnsignedIntegerOperation(operation.Operator, leftValue, rightValue, operation.OwnerType)
		}
	}
	if operation.OwnerType != hir.TypeI32 || operation.Type != hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("arithmetic operation has incompatible owner or result type")
	}
	left, err := leftValue.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	right, err := rightValue.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	result, trap, err := binary(operation.Operator, left, right)
	return i32Value(result), trap, err
}

func evaluateUnary(locals map[mir.LocalID]runtimeValue, operation mir.Operation) (runtimeValue, string, error) {
	if len(operation.Operands) != 1 {
		return runtimeValue{}, "", fmt.Errorf("unary operation has %d operands, want 1", len(operation.Operands))
	}
	value, err := evaluate(locals, operation.Operands[0])
	if err != nil {
		return runtimeValue{}, "", err
	}
	if operation.Operator == "~" && operation.OwnerType != hir.TypeI32 {
		if !hir.BitwiseNotType(operation.OwnerType) || operation.Type != operation.OwnerType ||
			!comparisonRuntimeMatches(value, operation.OwnerType) {
			return runtimeValue{}, "", capabilityError("bitwise not has incompatible operand or result type")
		}
		limit, ok := unsignedArithmeticLimit(operation.OwnerType)
		if !ok {
			return runtimeValue{}, "", capabilityError("bitwise not has unsupported type %q", operation.OwnerType)
		}
		return unsignedRuntimeValue(operation.OwnerType, limit^value.u64), "", nil
	}
	if operation.Operator == "!" {
		if operation.OwnerType != hir.TypeBool || operation.Type != hir.TypeBool {
			return runtimeValue{}, "", capabilityError("logical not has incompatible operand or result type")
		}
	} else if operation.OwnerType != hir.TypeI32 || operation.Type != hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("unary operation has incompatible operand or result type")
	}
	i32, err := value.asI32()
	if err != nil {
		return runtimeValue{}, "", err
	}
	result, trap, err := unary(operation.Operator, i32)
	return i32Value(result), trap, err
}

func checkedUnsignedIntegerOperation(operator string, left, right runtimeValue, ownerType hir.Type) (runtimeValue, string, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) ||
		ownerType == hir.TypeBool || ownerType == hir.TypeI32 {
		return runtimeValue{}, "", capabilityError("unsigned integer operation received incompatible operands")
	}
	limit, ok := unsignedArithmeticLimit(ownerType)
	if !ok {
		return runtimeValue{}, "", capabilityError("unsigned integer operation has unsupported type %q", ownerType)
	}
	var result uint64
	switch operator {
	case "+":
		if left.u64 > limit-right.u64 {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64 + right.u64
	case "-":
		if left.u64 < right.u64 {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64 - right.u64
	case "*":
		if right.u64 != 0 && left.u64 > limit/right.u64 {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64 * right.u64
	case "/":
		if right.u64 == 0 {
			return runtimeValue{}, "division-by-zero", nil
		}
		result = left.u64 / right.u64
	case "%":
		if right.u64 == 0 {
			return runtimeValue{}, "division-by-zero", nil
		}
		result = left.u64 % right.u64
	case "&":
		result = left.u64 & right.u64
	case "^":
		result = left.u64 ^ right.u64
	case "|":
		result = left.u64 | right.u64
	case ">>":
		width := uint64(64)
		if ownerType == hir.TypeU8 {
			width = 8
		} else if ownerType == hir.TypeU32 {
			width = 32
		}
		if right.u64 >= width {
			return runtimeValue{}, "invalid-shift", nil
		}
		result = left.u64 >> uint32(right.u64)
	case "<<":
		width := uint64(64)
		if ownerType == hir.TypeU8 {
			width = 8
		} else if ownerType == hir.TypeU32 {
			width = 32
		}
		if right.u64 >= width {
			return runtimeValue{}, "invalid-shift", nil
		}
		if left.u64 > limit>>uint32(right.u64) {
			return runtimeValue{}, "integer-overflow", nil
		}
		result = left.u64 << uint32(right.u64)
	default:
		return runtimeValue{}, "", capabilityError("unsupported unsigned integer operator %q", operator)
	}
	return unsignedRuntimeValue(ownerType, result), "", nil
}

func unsignedArithmeticLimit(ownerType hir.Type) (uint64, bool) {
	switch ownerType {
	case hir.TypeU8:
		return math.MaxUint8, true
	case hir.TypeU32:
		return math.MaxUint32, true
	case hir.TypeUSize:
		return math.MaxUint64, true
	default:
		return 0, false
	}
}

func unsignedRuntimeValue(ownerType hir.Type, value uint64) runtimeValue {
	switch ownerType {
	case hir.TypeU8:
		return runtimeValue{kind: valueU8, u64: value, typ: hir.TypeU8}
	case hir.TypeU32:
		return runtimeValue{kind: valueU32, u64: value, typ: hir.TypeU32}
	case hir.TypeUSize:
		return runtimeValue{kind: valueUSize, u64: value, typ: hir.TypeUSize}
	default:
		return runtimeValue{}
	}
}

func equalScalarValues(left, right runtimeValue, ownerType hir.Type) (bool, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) {
		return false, capabilityError("comparison operands have different runtime scalar types")
	}
	switch ownerType {
	case hir.TypeBool, hir.TypeI32:
		return left.i32 == right.i32, nil
	case hir.TypeU8, hir.TypeU32, hir.TypeUSize:
		return left.u64 == right.u64, nil
	default:
		if _, _, pointer := hir.RawPointerElement(ownerType); pointer {
			return true, nil
		}
		return false, capabilityError("value is not an executable equality value")
	}
}

func orderedScalarValues(operator string, left, right runtimeValue, ownerType hir.Type) (bool, error) {
	if !comparisonRuntimeMatches(left, ownerType) || !comparisonRuntimeMatches(right, ownerType) {
		return false, capabilityError("ordered comparison operands have incompatible runtime scalar types")
	}
	if ownerType == hir.TypeI32 {
		switch operator {
		case "<":
			return left.i32 < right.i32, nil
		case "<=":
			return left.i32 <= right.i32, nil
		case ">":
			return left.i32 > right.i32, nil
		case ">=":
			return left.i32 >= right.i32, nil
		}
	}
	switch operator {
	case "<":
		return left.u64 < right.u64, nil
	case "<=":
		return left.u64 <= right.u64, nil
	case ">":
		return left.u64 > right.u64, nil
	case ">=":
		return left.u64 >= right.u64, nil
	default:
		return false, capabilityError("unsupported ordered comparison operator %q", operator)
	}
}

func comparisonRuntimeMatches(value runtimeValue, ownerType hir.Type) bool {
	switch ownerType {
	case hir.TypeBool, hir.TypeI32:
		return value.kind == valueI32
	case hir.TypeU8:
		return value.kind == valueU8 && value.typ == hir.TypeU8 && value.u64 <= math.MaxUint8
	case hir.TypeU32:
		return value.kind == valueU32 && value.typ == hir.TypeU32 && value.u64 <= math.MaxUint32
	case hir.TypeUSize:
		return value.kind == valueUSize && value.typ == hir.TypeUSize
	default:
		_, _, pointer := hir.RawPointerElement(ownerType)
		return pointer && value.kind == valueNull && value.typ == ownerType
	}
}

func valuePreservingIntegerCast(value runtimeValue, sourceType, targetType hir.Type) (runtimeValue, error) {
	if !hir.ValuePreservingIntegerCast(sourceType, targetType) || value.typ != sourceType {
		return runtimeValue{}, capabilityError("integer cast has incompatible source or target type")
	}
	switch sourceType {
	case hir.TypeU8:
		if value.kind != valueU8 || value.u64 > math.MaxUint8 {
			return runtimeValue{}, capabilityError("integer cast received an invalid u8 value")
		}
		switch targetType {
		case hir.TypeI32:
			return i32Value(int32(value.u64)), nil
		case hir.TypeU32:
			return runtimeValue{kind: valueU32, u64: value.u64, typ: hir.TypeU32}, nil
		case hir.TypeUSize:
			return runtimeValue{kind: valueUSize, u64: value.u64, typ: hir.TypeUSize}, nil
		}
	case hir.TypeU32:
		if value.kind == valueU32 && value.u64 <= math.MaxUint32 && targetType == hir.TypeUSize {
			return runtimeValue{kind: valueUSize, u64: value.u64, typ: hir.TypeUSize}, nil
		}
	}
	return runtimeValue{}, capabilityError("integer cast is outside the executable widening subset")
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
	leftValue, rightValue, err := evaluateTwo(locals, operands)
	if err != nil {
		return 0, 0, err
	}
	left, err := leftValue.asI32()
	if err != nil {
		return 0, 0, err
	}
	right, err := rightValue.asI32()
	return left, right, err
}

func evaluateTwo(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, error) {
	if len(operands) != 2 {
		return runtimeValue{}, runtimeValue{}, fmt.Errorf("operation has %d operands, want 2", len(operands))
	}
	left, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
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
	case "assert":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueI32 || value.typ != hir.TypeBool || (value.i32 != 0 && value.i32 != 1) {
			return mir.NoBlock, "", capabilityError("assert received a non-boolean value")
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("assert switch has %d successors, want 2", len(terminator.Successors))
		}
		if value.i32 == 0 {
			return mir.NoBlock, "assertion-failed", nil
		}
		return terminator.Successors[0], "", nil
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
	case "enum":
		value, err := evaluateOne(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if value.kind != valueEnum || value.enum == nil || value.enum.definition == nil || value.enum.definition.Type != terminator.SwitchType {
			return mir.NoBlock, "", capabilityError("enum switch received a non-matching enum value")
		}
		if len(terminator.Successors) != len(terminator.SwitchVariants) {
			return mir.NoBlock, "", fmt.Errorf("enum switch has %d successors for %d variants", len(terminator.Successors), len(terminator.SwitchVariants))
		}
		for index, variant := range terminator.SwitchVariants {
			if value.enum.variant == variant {
				return terminator.Successors[index], "", nil
			}
		}
		return mir.NoBlock, "", fmt.Errorf("enum switch received invalid variant %q", value.enum.variant)
	case "checked-division":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 division received incompatible operands")
			}
			if rightI32 == 0 {
				return mir.NoBlock, "division-by-zero", nil
			}
			if leftI32 == math.MinInt32 && rightI32 == -1 {
				return mir.NoBlock, "signed-division-overflow", nil
			}
		} else {
			_, trap, err := checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
			if err != nil {
				return mir.NoBlock, "", err
			}
			if trap != "" {
				return mir.NoBlock, trap, nil
			}
		}
		return terminator.Successors[0], "", nil
	case "checked-view-index":
		receiver, index, err := evaluateViewIndexOperands(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("checked byte-view index has %d successors, want 2", len(terminator.Successors))
		}
		if index.u64 >= receiver.u64 {
			return mir.NoBlock, "bounds", nil
		}
		return terminator.Successors[0], "", nil
	case "checked-view-slice":
		receiver, start, end, err := evaluateViewSliceOperands(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if len(terminator.Successors) != 2 {
			return mir.NoBlock, "", fmt.Errorf("checked byte-view slice has %d successors, want 2", len(terminator.Successors))
		}
		if start.u64 > end.u64 || end.u64 > receiver.u64 {
			return mir.NoBlock, "bounds", nil
		}
		return terminator.Successors[0], "", nil
	case "checked-unary":
		value, err := evaluateOneI32(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		_, trap, err := unary(terminator.Operator, value)
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	case "checked-arithmetic":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		var trap string
		if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 arithmetic received incompatible operands")
			}
			_, trap, err = binary(terminator.Operator, leftI32, rightI32)
		} else {
			_, trap, err = checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
		}
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	case "checked-shift":
		left, right, err := evaluateTwo(locals, terminator.SwitchOperands)
		if err != nil {
			return mir.NoBlock, "", err
		}
		var trap string
		if terminator.SwitchType == hir.TypeI32 {
			leftI32, leftErr := left.asI32()
			rightI32, rightErr := right.asI32()
			if leftErr != nil || rightErr != nil {
				return mir.NoBlock, "", capabilityError("checked i32 right shift received incompatible operands")
			}
			_, trap, err = binary(terminator.Operator, leftI32, rightI32)
		} else {
			_, trap, err = checkedUnsignedIntegerOperation(terminator.Operator, left, right, terminator.SwitchType)
		}
		if err != nil {
			return mir.NoBlock, "", err
		}
		if trap != "" {
			return mir.NoBlock, trap, nil
		}
		return terminator.Successors[0], "", nil
	default:
		return mir.NoBlock, "", capabilityError("unsupported MIR switch %q", terminator.SwitchKind)
	}
}

func evaluateViewIndexOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, error) {
	if len(operands) != 2 {
		return runtimeValue{}, runtimeValue{}, fmt.Errorf("byte-view index has %d operands, want 2", len(operands))
	}
	receiver, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	index, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, err
	}
	if receiver.kind != valueView || receiver.typ != hir.SharedBorrow(hir.TypeByteSlice) || index.kind != valueUSize || index.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, capabilityError("byte-view index received incompatible operands")
	}
	return receiver, index, nil
}

func evaluateViewSliceOperands(locals map[mir.LocalID]runtimeValue, operands []mir.Operand) (runtimeValue, runtimeValue, runtimeValue, error) {
	if len(operands) != 3 {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, fmt.Errorf("byte-view slice has %d operands, want 3", len(operands))
	}
	receiver, err := evaluate(locals, operands[0])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	start, err := evaluate(locals, operands[1])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	end, err := evaluate(locals, operands[2])
	if err != nil {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, err
	}
	wantView := hir.SharedBorrow(hir.TypeByteSlice)
	if receiver.kind != valueView || receiver.typ != wantView ||
		start.kind != valueUSize || start.typ != hir.TypeUSize || end.kind != valueUSize || end.typ != hir.TypeUSize {
		return runtimeValue{}, runtimeValue{}, runtimeValue{}, capabilityError("byte-view slice received incompatible operands")
	}
	return receiver, start, end, nil
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
