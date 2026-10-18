package interpreter

import (
	"context"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

type executionState struct {
	byName         map[string]*mir.Function
	bySymbol       map[hir.SymbolID]*mir.Function
	structTypes    map[hir.Type]*mir.StructType
	enumTypes      map[hir.Type]*mir.EnumType
	resourceTypes  map[hir.Type]*mir.ResourceType
	testIntrinsics map[hir.SymbolID]TestIntrinsicKind
	outputLimit    int
	steps          int
	callDepthLimit int
	context        context.Context
	dropTrace      []DropEvent
	regionTrace    []RegionEvent
	stdout         []byte
	stderr         []byte
	ioTrace        []IOEvent
}

func (r *executionState) configureTestIntrinsics(entry string, options Options) error {
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

func (r *executionState) outcome(kind string, value int32, trapCategory string) Outcome {
	return Outcome{
		Kind: kind, Value: value, TrapCategory: trapCategory,
		DropTrace: slices.Clone(r.dropTrace), RegionTrace: slices.Clone(r.regionTrace),
		Stdout: slices.Clone(r.stdout), Stderr: slices.Clone(r.stderr), IOTrace: slices.Clone(r.ioTrace),
	}
}

func (r *executionState) callTestIntrinsic(kind TestIntrinsicKind, arguments []runtimeValue) (runtimeValue, error) {
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
