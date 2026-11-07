package mir_test

import (
	"crypto/sha256"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildidentity"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

const closureMIRFactories = `module main;
struct Box{fn i32(i32) action;}
fn fn i32(i32) make(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){let value=offset+x;return value;};return action;}
fn fn i32(i32) subtract(i32 offset){let fn i32(i32) action=fn [i32 offset] i32(i32 x){return offset-x;};return action;}
fn i32 apply(fn i32(i32) action,i32 x){return action(x);}
fn Box boxed(i32 offset){let fn i32(i32) action=make(offset);return Box{.action=action};}
fn i32 main(){let first=make(40);let second=subtract(44);let same=first;let Box stored=boxed(40);let Box copied=stored;let fn i32(i32) extracted=copied.action;return apply(first,2)+apply(second,2)+same(2)+extracted(2);}`

func closureMIRFunction(t *testing.T, p *mir.Program, name string) *mir.Function {
	t.Helper()
	for i := range p.Functions {
		if p.Functions[i].Name == "main::"+name {
			return &p.Functions[i]
		}
	}
	t.Fatal("missing source function", name)
	return nil
}

func TestClosureValueMIRSourceFactoriesApplyAndStructCopies(t *testing.T) {
	identity, err := buildidentity.Current()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("test executable=%s", identity.String())
	p := lowerIgnoredCleanupSource(t, closureMIRFactories)
	t.Logf("source sha256=%x", sha256.Sum256([]byte(closureMIRFactories)))
	if closureMIRFunction(t, &p, "make").ResultType != hir.TypeClosureI32 || closureMIRFunction(t, &p, "subtract").ResultType != hir.TypeClosureI32 || !slices.Equal(closureMIRFunction(t, &p, "apply").ParameterTypes, []hir.Type{hir.TypeClosureI32, hir.TypeI32}) {
		t.Fatal("factory/apply signature lost first-class closure leaf")
	}
	if len(p.StructTypes) != 1 || len(p.StructTypes[0].Fields) != 1 || p.StructTypes[0].Fields[0].Type != hir.TypeClosureI32 {
		t.Fatal("ordinary struct lost closure field")
	}
	targets := map[hir.SymbolID]*mir.Function{}
	for i := range p.Functions {
		fn := &p.Functions[i]
		if fn.ClosureTarget {
			targets[fn.Symbol] = fn
			if fn.ClosureCaptures != 1 || !slices.Equal(fn.ParameterTypes, []hir.Type{hir.TypeI32, hir.TypeI32}) || fn.ResultType != hir.TypeI32 || len(fn.Effects) != 0 {
				t.Fatal("lifted helper lost capture/value signature", fn.Name)
			}
		}
	}
	creates, calls, constructs, reads, copies := 0, 0, 0, 0, 0
	for _, fn := range p.Functions {
		for _, block := range fn.Blocks {
			for _, op := range block.Operations {
				switch op.Kind {
				case mir.OperationClosureCreate:
					creates++
					if targets[op.Symbol] == nil || len(op.Operands) != 1 || op.Type != hir.TypeClosureI32 {
						t.Fatal("construction lost actual target/capture", op)
					}
				case mir.OperationStructConstruct:
					constructs++
				case mir.OperationFieldRead:
					if op.Type == hir.TypeClosureI32 && op.Field == "action" {
						reads++
					}
				case mir.OperationAssign:
					if len(op.Operands) == 1 && op.Operands[0].Kind == mir.OperandLocal {
						copies++
					}
				}
			}
			if term := block.Terminator; term.Kind == mir.TerminatorClosureCall {
				calls++
				if len(term.Arguments) != 2 || len(term.Successors) != 1 || term.UnwindSuccessor != mir.NoBlock || term.ForeignABI != "" {
					t.Fatal("value invocation lost its private, non-unwinding shape", term)
				}
			}
		}
	}
	if len(targets) != 2 || creates != 2 || calls != 3 || constructs != 1 || reads != 1 || copies < 2 {
		t.Fatalf("explicit MIR targets/create/call/struct/read/copy = %d/%d/%d/%d/%d/%d", len(targets), creates, calls, constructs, reads, copies)
	}
}

func TestClosureValueMIRSourceCaptureArities(t *testing.T) {
	for count := 0; count <= 4; count++ {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var declarations, captures strings.Builder
			for i := 0; i < count; i++ {
				fmt.Fprintf(&declarations, "let i32 c%d=%d;", i, i+11)
				if i > 0 {
					captures.WriteByte(',')
				}
				fmt.Fprintf(&captures, "i32 c%d", i)
			}
			text := "module main;fn i32 main(){" + declarations.String() + "let fn i32(i32) action=fn [" + captures.String() + "] i32(i32 x){return x;};return action(42);}"
			p := lowerIgnoredCleanupSource(t, text)
			creates, targets := 0, 0
			for _, fn := range p.Functions {
				if fn.ClosureTarget {
					targets++
					if int(fn.ClosureCaptures) != count || len(fn.Parameters) != count+1 {
						t.Fatal("capture arity lost", fn)
					}
				}
				for _, block := range fn.Blocks {
					for _, op := range block.Operations {
						if op.Kind == mir.OperationClosureCreate {
							creates++
							if len(op.Operands) != count {
								t.Fatal("constructor arity lost", op)
							}
						}
					}
				}
			}
			if creates != 1 || targets != 1 {
				t.Fatal("literal was erased or duplicated", creates, targets)
			}
		})
	}
}

func TestClosureValueMIRSourceSnapshotAndOnceOrdering(t *testing.T) {
	const text = `module main;
fn fn i32(i32) make(i32 seed){var i32 a=seed;let i32 b=3;let fn i32(i32) action=fn [i32 a,i32 b] i32(i32 x){let unused=1/0;return a-b*x;};a=99;return action;}
fn i32 argument(){return 2;}
fn i32 main(){let original=make(20);let copied=original;return copied(argument());}`
	p := lowerIgnoredCleanupSource(t, text)
	factory := closureMIRFunction(t, &p, "make")
	var create *mir.Operation
	var before []mir.Operation
	mutations := 0
	for _, block := range factory.Blocks {
		for i := range block.Operations {
			op := &block.Operations[i]
			if op.Kind == mir.OperationClosureCreate {
				create = op
				before = slices.Clone(block.Operations[:i])
			}
			if create != nil && op.Kind == mir.OperationAssign && len(op.Operands) == 1 && op.Operands[0].Kind == mir.OperandI32 && op.Operands[0].I32 == 99 {
				mutations++
				for _, captured := range create.Operands {
					if captured.Kind != mir.OperandLocal || captured.Local == op.DestinationLocal {
						t.Fatal("captured value aliases mutated creator local")
					}
				}
			}
		}
	}
	if create == nil || len(create.Operands) != 2 || mutations != 1 || len(before) < 2 {
		t.Fatal("snapshot construction/mutation ordering missing")
	}
	for i, captured := range create.Operands {
		freeze := before[len(before)-2+i]
		if freeze.Kind != mir.OperationAssign || freeze.DestinationLocal != captured.Local || len(freeze.Operands) != 1 || freeze.Operands[0].Kind != mir.OperandLocal || freeze.Operands[0].Local == captured.Local {
			t.Fatalf("capture %d was not frozen once in written order: %+v", i, freeze)
		}
	}
	main, argument := closureMIRFunction(t, &p, "main"), closureMIRFunction(t, &p, "argument")
	argumentCalls, valueCalls := 0, 0
	for _, block := range main.Blocks {
		term := block.Terminator
		if term.Kind != mir.TerminatorCall || term.Callee != argument.Symbol {
			continue
		}
		argumentCalls++
		if len(block.Operations) == 0 || len(term.Successors) != 1 {
			t.Fatal("callable was not materialized before argument call")
		}
		freeze := block.Operations[len(block.Operations)-1]
		next := main.Blocks[term.Successors[0]].Terminator
		if freeze.Kind != mir.OperationAssign || len(freeze.Operands) != 1 || freeze.Operands[0].Kind != mir.OperandLocal || next.Kind != mir.TerminatorClosureCall || len(next.Arguments) != 2 || next.Arguments[0].Local != freeze.DestinationLocal || next.Arguments[1].Local != term.DestinationLocal {
			t.Fatal("callable snapshot -> once-only argument -> value invocation order lost")
		}
	}
	for _, block := range main.Blocks {
		if block.Terminator.Kind == mir.TerminatorClosureCall {
			valueCalls++
		}
	}
	if argumentCalls != 1 || valueCalls != 1 {
		t.Fatal("argument/value call duplicated", argumentCalls, valueCalls)
	}
	// Checked traps belong to the separate target; creating/copying a value
	// must never insert its arithmetic/trap operations into the factory.
	traps := 0
	for _, fn := range p.Functions {
		for _, block := range fn.Blocks {
			if block.Terminator.Kind == mir.TerminatorTrap {
				if !fn.ClosureTarget {
					t.Fatal("closure body trap ran during construction")
				}
				traps++
			}
		}
	}
	if traps == 0 {
		t.Fatal("unused checked body local lost its trap")
	}
}

// Handwritten, separately verified control: no lowering algorithm constructs
// the forged cases or supplies their expected acceptance/rejection verdicts.
func closureMIRControl(count int) mir.Program {
	target := mir.Function{Symbol: 1, Name: "target", ClosureTarget: true, ClosureCaptures: uint8(count), ResultType: hir.TypeI32}
	for i := 0; i <= count; i++ {
		target.Parameters = append(target.Parameters, mir.LocalID(i))
		target.ParameterTypes = append(target.ParameterTypes, hir.TypeI32)
	}
	target.Blocks = []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, DestinationLocal: mir.NoLocal, UnwindSuccessor: mir.NoBlock, ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: mir.LocalID(count)}}}}
	create := mir.Operation{Kind: mir.OperationClosureCreate, Symbol: 1, Type: hir.TypeClosureI32, DestinationLocal: 10}
	for i := 0; i < count; i++ {
		create.Operands = append(create.Operands, mir.Operand{Kind: mir.OperandI32, I32: int32(11 + i)})
	}
	caller := mir.Function{Symbol: 2, Name: "caller", ResultType: hir.TypeI32, Blocks: []mir.Block{
		{ID: 0, Operations: []mir.Operation{create}, Terminator: mir.Terminator{Kind: mir.TerminatorClosureCall, DestinationLocal: 11, UnwindSuccessor: mir.NoBlock, Arguments: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}, {Kind: mir.OperandI32, I32: 42}}, Successors: []mir.BlockID{1}}},
		{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, DestinationLocal: mir.NoLocal, UnwindSuccessor: mir.NoBlock, ReturnValue: mir.Operand{Kind: mir.OperandLocal, Local: 11}}},
	}}
	return mir.Program{Functions: []mir.Function{target, caller}}
}

func closureMIRMustReject(t *testing.T, p mir.Program) {
	t.Helper()
	failures := mir.Verify(p)
	if len(failures) == 0 {
		t.Fatal("forged closure MIR was accepted")
	}
	t.Logf("rejected: %+v", failures)
}

func TestClosureValueMIRForgedCaptureCounts(t *testing.T) {
	for count := 0; count <= 4; count++ {
		p := closureMIRControl(count)
		if failures := mir.Verify(p); len(failures) != 0 {
			t.Fatalf("independent valid %d-capture control: %+v", count, failures)
		}
		t.Run(fmt.Sprintf("mismatch-%d", count), func(t *testing.T) {
			p.Functions[0].ClosureCaptures = uint8((count + 1) % 5)
			closureMIRMustReject(t, p)
		})
	}
	t.Run("five-captures", func(t *testing.T) { closureMIRMustReject(t, closureMIRControl(5)) })
}

func TestClosureValueMIRForgedContracts(t *testing.T) {
	base := closureMIRControl(1)
	if failures := mir.Verify(base); len(failures) != 0 {
		t.Fatal("independent valid control", failures)
	}
	for _, tc := range []struct {
		name string
		edit func(*mir.Program)
	}{
		{"target-missing", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Symbol = 99 }},
		{"target-unmarked", func(p *mir.Program) { p.Functions[0].ClosureTarget = false; p.Functions[0].ClosureCaptures = 0 }},
		{"orphan-capture-metadata", func(p *mir.Program) { p.Functions[0].ClosureTarget = false }},
		{"target-parameter-count", func(p *mir.Program) { p.Functions[0].ParameterTypes = p.Functions[0].ParameterTypes[:1] }},
		{"target-parameter-type", func(p *mir.Program) { p.Functions[0].ParameterTypes[0] = hir.TypeBool }},
		{"target-result-signature", func(p *mir.Program) { p.Functions[0].ResultType = hir.TypeBool }},
		{"target-effects", func(p *mir.Program) { p.Functions[0].Effects = []hir.Effect{"console"} }},
		{"target-body-operation", func(p *mir.Program) {
			p.Functions[0].Blocks[0].Operations = []mir.Operation{{Kind: mir.OperationClosureCreate, Type: hir.TypeClosureI32, Symbol: 1, DestinationLocal: 8}}
		}},
		{"target-dead-body-operation", func(p *mir.Program) {
			p.Functions[0].Blocks = append(p.Functions[0].Blocks, mir.Block{ID: 1, Operations: []mir.Operation{{Kind: mir.OperationUse}}, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}})
		}},
		{"target-parameter-write", func(p *mir.Program) {
			p.Functions[0].Blocks[0].Operations = []mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 0, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 9}}}}
		}},
		{"target-duplicate-parameter", func(p *mir.Program) { p.Functions[0].Parameters[1] = p.Functions[0].Parameters[0] }},
		{"target-local-rewrite", func(p *mir.Program) {
			op := mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 8, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 9}}}
			p.Functions[0].Blocks[0].Operations = []mir.Operation{op, op}
		}},
		{"target-cycle", func(p *mir.Program) {
			p.Functions[0].Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{0}}
		}},
		{"target-dead-cycle", func(p *mir.Program) {
			p.Functions[0].Blocks = append(p.Functions[0].Blocks, mir.Block{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorGoto, Successors: []mir.BlockID{1}}})
		}},
		{"target-return-bool", func(p *mir.Program) {
			p.Functions[0].Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandI32, I32: 1, Type: hir.TypeBool}
		}},
		{"target-return-empty", func(p *mir.Program) { p.Functions[0].Blocks[0].Terminator.ReturnValue = mir.Operand{} }},
		{"capture-kind-spoof", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations[0].Operands[0] = mir.Operand{Kind: mir.OperandU32, U64: 7, Type: hir.TypeI32}
		}},
		{"capture-annotation-spoof", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Operands[0].Type = hir.TypeBool }},
		{"capture-local-type-spoof", func(p *mir.Program) {
			block := &p.Functions[1].Blocks[0]
			block.Operations[0].Operands[0] = mir.Operand{Kind: mir.OperandLocal, Local: 20, Type: hir.TypeI32}
			block.Operations = append([]mir.Operation{{Kind: mir.OperationAssign, DestinationLocal: 20, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 1, Type: hir.TypeBool}}}}, block.Operations...)
		}},
		{"capture-dirty-bits", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Operands[0].U64 = 1 }},
		{"capture-uninitialized", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations[0].Operands[0] = mir.Operand{Kind: mir.OperandLocal, Local: 99, Type: hir.TypeI32}
		}},
		{"construction-type-spoof", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Type = hir.TypeI32 }},
		{"construction-shape", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Operator = "+" }},
		{"callable-uninitialized", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Arguments[0].Local = 99 }},
		{"callable-typed-uninitialized", func(p *mir.Program) {
			caller := &p.Functions[1]
			caller.Blocks = append(caller.Blocks, mir.Block{ID: 2, Operations: slices.Clone(caller.Blocks[0].Operations), Terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: mir.Operand{Kind: mir.OperandI32, I32: 0}}})
			caller.Blocks[0].Operations = nil
		}},
		{"callable-local-type-spoof", func(p *mir.Program) {
			block := &p.Functions[1].Blocks[0]
			block.Operations = append(block.Operations, mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 20, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 42}}})
			block.Terminator.Arguments[0] = mir.Operand{Kind: mir.OperandLocal, Local: 20, Type: hir.TypeClosureI32}
		}},
		{"callable-annotation-spoof", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Arguments[0].Type = hir.TypeI32 }},
		{"callable-scalar-spoof", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Terminator.Arguments[0] = mir.Operand{Kind: mir.OperandI32, I32: 42, Type: hir.TypeClosureI32}
		}},
		{"callable-callback-spoof", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].Type = hir.TypeCCallbackI32UserData }},
		{"call-argument-type", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Arguments[1].Type = hir.TypeBool }},
		{"call-ABI", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.ForeignABI = "C" }},
		{"call-callee", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Callee = 1 }},
		{"call-unwind", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.UnwindSuccessor = 1 }},
		{"call-no-successor", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Successors = nil }},
		{"call-two-successors", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Successors = []mir.BlockID{1, 1} }},
		{"call-missing-successor", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.Successors = []mir.BlockID{99} }},
		{"call-no-result", func(p *mir.Program) { p.Functions[1].Blocks[0].Terminator.DestinationLocal = mir.NoLocal }},
		{"call-missing-argument", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Terminator.Arguments = p.Functions[1].Blocks[0].Terminator.Arguments[:1]
		}},
		{"return-closure-as-i32", func(p *mir.Program) {
			p.Functions[1].Blocks[1].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandLocal, Local: 10}
		}},
		{"return-i32-as-closure", func(p *mir.Program) { p.Functions[1].ResultType = hir.TypeClosureI32 }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := mir.Clone(base)
			tc.edit(&p)
			closureMIRMustReject(t, p)
		})
	}
}

func TestClosureValueMIRForgedTransportBoundaries(t *testing.T) {
	box := hir.ParseType("test::Box")
	base := closureMIRControl(1)
	base.StructTypes = []mir.StructType{{Type: box, Name: "Box", Fields: []mir.StructField{{Name: "action", Type: hir.TypeClosureI32}}}}
	ordinary := func(param, result hir.Type) mir.Function {
		fn := mir.Function{Symbol: 3, Name: "transport", Parameters: []mir.LocalID{0}, ParameterTypes: []hir.Type{param}, ResultType: result,
			Blocks: []mir.Block{{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, DestinationLocal: mir.NoLocal, UnwindSuccessor: mir.NoBlock}}}}
		if result != hir.TypeVoid {
			fn.Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandLocal, Local: 0}
		}
		return fn
	}
	// These controls exercise actual ordinary carrier transport independently
	// of a target marker, including a complete struct copy/return path.
	for _, typ := range []hir.Type{hir.TypeClosureI32, box} {
		p := mir.Clone(base)
		p.Functions = append(p.Functions, ordinary(typ, typ))
		if failures := mir.Verify(p); len(failures) != 0 {
			t.Fatalf("valid ordinary %s carrier: %+v", typ, failures)
		}
	}
	reject := func(t *testing.T, p mir.Program) {
		t.Helper()
		failures := mir.Verify(p)
		for _, failure := range failures {
			if failure.Code == mir.CodeOperationType && strings.HasPrefix(failure.Detail, "copied closure transport:") {
				t.Logf("transport rejection: %+v", failures)
				return
			}
		}
		t.Fatalf("forged transport did not reach the closure transport rejection: %+v", failures)
	}
	for _, wrapper := range []struct {
		name string
		typ  hir.Type
	}{
		{"option", hir.Option(hir.TypeClosureI32)},
		{"result-success", hir.Result(hir.TypeClosureI32, hir.TypeU32)},
		{"result-error", hir.Result(hir.TypeI32, hir.TypeClosureI32)},
		{"array", hir.FixedArray(hir.TypeClosureI32, 2)},
		{"shared-pointer", hir.SharedBorrow(hir.TypeClosureI32)},
		{"mutable-pointer", hir.MutableBorrow(hir.TypeClosureI32)},
		{"raw-pointer", hir.RawPointer(hir.TypeClosureI32, false)},
		{"carrier-option", hir.Option(box)},
		{"carrier-pointer", hir.RawPointer(box, true)},
	} {
		for _, result := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s-result-%t", wrapper.name, result), func(t *testing.T) {
				p := mir.Clone(base)
				fn := ordinary(wrapper.typ, hir.TypeVoid)
				if result {
					fn.ResultType = wrapper.typ
					fn.Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandLocal, Local: 0}
				}
				p.Functions = append(p.Functions, fn)
				reject(t, p)
			})
		}
	}
	for _, typ := range []hir.Type{hir.TypeClosureI32, box} {
		for _, result := range []bool{false, true} {
			t.Run(fmt.Sprintf("FFI-%s-result-%t", typ, result), func(t *testing.T) {
				p := mir.Clone(base)
				foreign := mir.ForeignFunction{Symbol: 4, Name: "foreign", LinkName: "foreign", ABI: "C", ResultType: hir.TypeVoid}
				if result {
					foreign.ResultType = typ
				} else {
					foreign.ParameterTypes = []hir.Type{typ}
				}
				p.ForeignFunctions = []mir.ForeignFunction{foreign}
				reject(t, p)
			})
		}
	}
	for _, tc := range []struct {
		name string
		edit func(*mir.Program)
	}{
		{"repr-C", func(p *mir.Program) { p.StructTypes[0].ReprC = true }},
		{"nested-struct", func(p *mir.Program) {
			p.StructTypes = append(p.StructTypes, mir.StructType{Type: hir.ParseType("test::Outer"), Name: "Outer", Fields: []mir.StructField{{Name: "inner", Type: box}}})
		}},
		{"wrapped-field", func(p *mir.Program) {
			p.StructTypes = append(p.StructTypes, mir.StructType{Type: hir.ParseType("test::Outer"), Name: "Outer", Fields: []mir.StructField{{Name: "inner", Type: hir.Option(hir.TypeClosureI32)}}})
		}},
		{"enum-leaf", func(p *mir.Program) {
			p.EnumTypes = []mir.EnumType{{Type: hir.ParseType("test::Choice"), Name: "Choice", Variants: []mir.EnumVariant{{Name: "Action", Fields: []mir.EnumField{{Name: "action", Type: hir.TypeClosureI32}}}}}}
		}},
		{"enum-carrier", func(p *mir.Program) {
			p.EnumTypes = []mir.EnumType{{Type: hir.ParseType("test::Choice"), Name: "Choice", Variants: []mir.EnumVariant{{Name: "Action", Fields: []mir.EnumField{{Name: "action", Type: box}}}}}}
		}},
		{"resource-leaf", func(p *mir.Program) {
			p.ResourceTypes = []mir.ResourceType{{Type: hir.ParseType("test::Owner"), Fields: []mir.ResourceField{{Name: "action", Type: hir.TypeClosureI32}}, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}}
		}},
		{"resource-carrier", func(p *mir.Program) {
			p.ResourceTypes = []mir.ResourceType{{Type: hir.ParseType("test::Owner"), Fields: []mir.ResourceField{{Name: "action", Type: box}}, DropKind: hir.ResourceDropEmpty, DropSymbol: hir.NoSymbol}}
		}},
		{"unary-consumption", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, mir.Operation{Kind: mir.OperationUnary, DestinationLocal: 20, Type: hir.TypeI32, Operator: "-", Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}})
		}},
		{"cast-production", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, mir.Operation{Kind: mir.OperationScalarCast, DestinationLocal: 20, Type: hir.TypeClosureI32, Operands: []mir.Operand{{Kind: mir.OperandI32, I32: 42}}})
		}},
		{"owner-operation", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].OwnerType = box }},
		{"index-type", func(p *mir.Program) { p.Functions[1].Blocks[0].Operations[0].IndexType = hir.TypeClosureI32 }},
		{"switch-consumption", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Terminator.SwitchOperands = []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}
		}},
		{"duplicate-construction", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, p.Functions[1].Blocks[0].Operations[0])
		}},
		{"assign-overwrite", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, mir.Operation{Kind: mir.OperationAssign, DestinationLocal: 10, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}})
		}},
		{"parameter-overwrite", func(p *mir.Program) {
			p.Functions[1].Parameters = []mir.LocalID{10}
			p.Functions[1].ParameterTypes = []hir.Type{hir.TypeClosureI32}
		}},
		{"struct-overwrite", func(p *mir.Program) {
			op := mir.Operation{Kind: mir.OperationStructConstruct, DestinationLocal: 20, Type: box, Fields: []string{"action"}, Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}}
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, op, op)
		}},
		{"ordinary-scalar-parameter", func(p *mir.Program) {
			p.Functions = append(p.Functions, ordinary(hir.TypeI32, hir.TypeI32))
			p.Functions[1].Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 3, DestinationLocal: 11, Arguments: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		}},
		{"ordinary-closure-parameter", func(p *mir.Program) {
			fn := ordinary(hir.TypeClosureI32, hir.TypeI32)
			fn.Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandI32, I32: 42}
			p.Functions = append(p.Functions, fn)
			p.Functions[1].Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 3, DestinationLocal: 11, Arguments: []mir.Operand{{Kind: mir.OperandI32, I32: 42}}, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		}},
		{"ordinary-carrier-parameter", func(p *mir.Program) {
			fn := ordinary(box, hir.TypeI32)
			fn.Blocks[0].Terminator.ReturnValue = mir.Operand{Kind: mir.OperandI32, I32: 42}
			p.Functions = append(p.Functions, fn)
			p.Functions[1].Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 3, DestinationLocal: 11, Arguments: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		}},
		{"ordinary-return-mismatch", func(p *mir.Program) { p.Functions = append(p.Functions, ordinary(box, hir.TypeClosureI32)) }},
		{"call-result-overwrite", func(p *mir.Program) {
			p.Functions = append(p.Functions, ordinary(hir.TypeClosureI32, hir.TypeClosureI32))
			p.Functions[1].Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: 3, DestinationLocal: 10, Arguments: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}
		}},
	} {
		t.Run(tc.name, func(t *testing.T) { p := mir.Clone(base); tc.edit(&p); reject(t, p) })
	}
}
