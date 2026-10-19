// Package interpreter executes verified Stage 0 MIR as the first semantic oracle.
package interpreter

import (
	"context"
	"errors"
	"fmt"

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

	maxConfiguredTestOutputBytes = 1 << 20
)

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

// ExecuteWithOptions runs verified MIR with caller budgets, cancellation, and optional test intrinsics.
func ExecuteWithOptions(program mir.Program, entry string, options Options) (Outcome, error) {
	return executeEntry(program, entry, hir.TypeI32, options)
}

// ExecuteVoid verifies program and invokes one zero-argument void entry function in a fresh,
// deterministic engine. It is the semantic test-runner entry and exposes no fixture intrinsics.
func ExecuteVoid(program mir.Program, entry string) (Outcome, error) {
	return ExecuteVoidWithOptions(program, entry, Options{})
}

// ExecuteVoidWithOptions runs a fresh semantic-test entry with caller budgets and cancellation.
func ExecuteVoidWithOptions(program mir.Program, entry string, options Options) (Outcome, error) {
	return executeEntry(program, entry, hir.TypeVoid, options)
}

func executeEntry(program mir.Program, entry string, resultType hir.Type, options Options) (Outcome, error) {
	var err error
	options, err = options.normalized()
	if err != nil {
		return Outcome{}, err
	}
	if err := options.Context.Err(); err != nil {
		return Outcome{Kind: OutcomeCanceled}, err
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		return Outcome{}, fmt.Errorf("verify MIR before interpretation: %v", failures)
	}
	engine := executionState{
		byName:         make(map[string]*mir.Function, len(program.Functions)),
		bySymbol:       make(map[hir.SymbolID]*mir.Function, len(program.Functions)),
		structTypes:    make(map[hir.Type]*mir.StructType, len(program.StructTypes)),
		enumTypes:      make(map[hir.Type]*mir.EnumType, len(program.EnumTypes)),
		resourceTypes:  make(map[hir.Type]*mir.ResourceType, len(program.ResourceTypes)),
		testIntrinsics: make(map[hir.SymbolID]TestIntrinsicKind, len(options.TestIntrinsics)),
		outputLimit:    options.OutputLimit,
		steps:          options.MaxSteps,
		callDepthLimit: options.MaxCallDepth,
		context:        options.Context,
	}
	for index := range program.StructTypes {
		definition := &program.StructTypes[index]
		engine.structTypes[definition.Type] = definition
	}
	for index := range program.EnumTypes {
		definition := &program.EnumTypes[index]
		engine.enumTypes[definition.Type] = definition
	}
	for index := range program.ResourceTypes {
		definition := &program.ResourceTypes[index]
		engine.resourceTypes[definition.Type] = definition
	}
	for index := range program.Functions {
		function := &program.Functions[index]
		if _, duplicate := engine.byName[function.Name]; duplicate {
			return Outcome{}, fmt.Errorf("duplicate MIR function name %q", function.Name)
		}
		if _, duplicate := engine.bySymbol[function.Symbol]; duplicate {
			return Outcome{}, fmt.Errorf("duplicate MIR function symbol %d", function.Symbol)
		}
		engine.byName[function.Name] = function
		engine.bySymbol[function.Symbol] = function
	}
	function := engine.byName[entry]
	if function == nil {
		return Outcome{}, capabilityError("entry function %q is missing", entry)
	}
	if len(function.Parameters) != 0 || function.ResultType != resultType {
		return Outcome{}, capabilityError("entry function %q must have type fn() -> %s", entry, resultType)
	}
	if err := engine.configureTestIntrinsics(entry, options); err != nil {
		return Outcome{}, err
	}
	if err := engine.configureAbstractArenas(options); err != nil {
		return Outcome{}, err
	}
	completion, err := engine.call(function, nil, 0)
	if err != nil {
		var capability CapabilityError
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return engine.outcome(OutcomeCanceled, 0, ""), err
		}
		if errors.As(err, &capability) && (capability.Kind == LimitStep || capability.Kind == LimitCallDepth || capability.Kind == LimitOutput) {
			return engine.outcome(OutcomeLimit, 0, ""), err
		}
		return Outcome{}, err
	}
	if completion.trapCategory != "" {
		return engine.outcome(OutcomeTrap, 0, completion.trapCategory), nil
	}
	if resultType == hir.TypeVoid {
		return engine.outcome(OutcomeReturn, 0, ""), nil
	}
	value, err := completion.value.asI32()
	if err != nil {
		return Outcome{}, fmt.Errorf("entry function %q: %w", entry, err)
	}
	return engine.outcome(OutcomeReturn, value, ""), nil
}
