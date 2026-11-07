package cbackend_test

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/target"
)

// Ordinary checked arithmetic is lowered first; only the new verified MIR
// construction/value-call edges are synthetic. No verifier is bypassed.
func closureRuntimeProgram(t *testing.T, captures int, store, trap, call bool) mir.Program {
	t.Helper()
	params, body := "i32 x", "x"
	for i := captures - 1; i >= 0; i-- {
		name := []string{"a", "b", "c", "d"}[i]
		params = "i32 " + name + "," + params
		body = name + "+(" + body + ")"
	}
	if trap {
		body = "x/" + []string{"x", "a"}[min(captures, 1)]
	}
	p := lowerSource(t, "module main;struct Box{i32 action;}fn i32 target("+params+"){return "+body+";}fn i32 identity(i32 x){return x;}fn i32 factory(i32 offset){return offset;}fn i32 apply(i32 action,i32 x){return x;}fn i32 main(){return 0;}")
	var factory, apply, main, lifted, identity *mir.Function
	for i := range p.Functions {
		f := &p.Functions[i]
		switch f.Name {
		case "main::target":
			lifted = f
		case "main::factory":
			factory = f
		case "main::apply":
			apply = f
		case "main::main":
			main = f
		case "main::identity":
			identity = f
		}
	}
	lifted.ClosureTarget, lifted.ClosureCaptures = true, uint8(captures)
	identity.ClosureTarget = true
	local := func(id mir.LocalID, typ hir.Type) mir.Operand {
		return mir.Operand{Kind: mir.OperandLocal, Local: id, Type: typ}
	}
	i32 := func(v int32) mir.Operand { return mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: v} }
	ret := func(v mir.Operand) mir.Terminator {
		return mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: v, DestinationLocal: mir.NoLocal, UnwindSuccessor: mir.NoBlock}
	}
	values := []mir.Operand{}
	for i := 0; i < captures; i++ {
		if i == 0 {
			values = append(values, local(factory.Parameters[0], hir.TypeI32))
		} else {
			values = append(values, i32(int32(i+1)))
		}
	}
	factory.ResultType = hir.TypeClosureI32
	factory.Blocks = []mir.Block{{ID: 0, Operations: []mir.Operation{{Kind: mir.OperationClosureCreate, Type: hir.TypeClosureI32, Symbol: lifted.Symbol, Operands: values, DestinationLocal: 100, Entity: "closure-create"}}, Terminator: ret(local(100, hir.TypeClosureI32))}}
	factory.Blocks[0].Operations = append(factory.Blocks[0].Operations, mir.Operation{Kind: mir.OperationAssign, Type: hir.TypeI32, DestinationLocal: factory.Parameters[0], Operands: []mir.Operand{i32(99)}})
	apply.ParameterTypes[0] = hir.TypeClosureI32
	apply.Blocks = []mir.Block{
		{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorClosureCall, Arguments: []mir.Operand{local(apply.Parameters[0], hir.TypeClosureI32), local(apply.Parameters[1], hir.TypeI32)}, DestinationLocal: 100, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock, Entity: "closure-invoke"}},
		{ID: 1, Terminator: ret(local(100, hir.TypeI32))},
	}
	offset, argument := int32(10), int32(23)
	if captures == 0 {
		argument = 42
	}
	if captures == 1 {
		argument = 32
	}
	if trap {
		offset = 0
		argument = 0
	}
	main.Blocks = []mir.Block{
		{ID: 0, Terminator: mir.Terminator{Kind: mir.TerminatorCall, Callee: factory.Symbol, Arguments: []mir.Operand{i32(offset)}, DestinationLocal: 100, Successors: []mir.BlockID{1}, UnwindSuccessor: mir.NoBlock}},
		{ID: 1, Operations: []mir.Operation{{Kind: mir.OperationAssign, Type: hir.TypeClosureI32, DestinationLocal: 101, Operands: []mir.Operand{local(100, hir.TypeClosureI32)}}}},
		{ID: 2, Terminator: ret(local(102, hir.TypeI32))},
	}
	callable := local(101, hir.TypeClosureI32)
	if store {
		box := &p.StructTypes[0]
		box.Fields[0].Type = hir.TypeClosureI32
		main.Blocks[1].Operations = append(main.Blocks[1].Operations,
			mir.Operation{Kind: mir.OperationStructConstruct, Type: box.Type, Fields: []string{"action"}, Operands: []mir.Operand{callable}, DestinationLocal: 103},
			mir.Operation{Kind: mir.OperationAssign, Type: box.Type, Operands: []mir.Operand{local(103, box.Type)}, DestinationLocal: 104},
			mir.Operation{Kind: mir.OperationFieldRead, Type: hir.TypeClosureI32, OwnerType: box.Type, Field: "action", Operands: []mir.Operand{local(104, box.Type)}, DestinationLocal: 105})
		callable = local(105, hir.TypeClosureI32)
	}
	main.Blocks[1].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: apply.Symbol, Arguments: []mir.Operand{callable, i32(argument)}, DestinationLocal: 102, Successors: []mir.BlockID{2}, UnwindSuccessor: mir.NoBlock}
	if call {
		second := main.Blocks[1].Terminator
		main.Blocks[1].Operations = append(main.Blocks[1].Operations, mir.Operation{Kind: mir.OperationClosureCreate, Type: hir.TypeClosureI32, Symbol: identity.Symbol, DestinationLocal: 106})
		main.Blocks[1].Terminator = mir.Terminator{Kind: mir.TerminatorCall, Callee: apply.Symbol, Arguments: []mir.Operand{local(106, hir.TypeClosureI32), i32(42)}, DestinationLocal: 107, Successors: []mir.BlockID{3}, UnwindSuccessor: mir.NoBlock}
		main.Blocks = append(main.Blocks, mir.Block{ID: 3, Terminator: second})
	}
	if !call {
		main.Blocks[1].Terminator = ret(i32(42))
		main.Blocks = main.Blocks[:2]
	}
	return p
}

func TestClosureValueRuntimeMIRAndEmission(t *testing.T) {
	for _, tc := range []struct {
		name              string
		captures          int
		store, trap, call bool
	}{
		{"zero", 0, false, false, true}, {"one", 1, false, false, true}, {"four", 4, false, false, true}, {"struct-copy", 4, true, false, true}, {"body-trap", 1, false, true, true}, {"unused-trapping-body", 1, false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := closureRuntimeProgram(t, tc.captures, tc.store, tc.trap, tc.call)
			if failures := mir.Verify(p); len(failures) != 0 {
				for _, fn := range p.Functions {
					if fn.ClosureTarget {
						dump, err := json.MarshalIndent(fn, "", "  ")
						if err != nil {
							t.Fatal(err)
						}
						t.Logf("synthetic closure target %s:\n%s", fn.Name, dump)
					}
				}
				t.Fatalf("parent MIR gate: %+v", failures)
			}
			before, _ := json.Marshal(p)
			out, err := interpreter.Execute(p, "main::main")
			wantTrap := ""
			if tc.trap && tc.call {
				wantTrap = "division-by-zero"
			}
			if err != nil || out.TrapCategory != wantTrap || (wantTrap == "" && (out.Kind != interpreter.OutcomeReturn || out.Value != 42)) {
				t.Fatalf("interpreter %+v %v", out, err)
			}
			u, err := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Contains(u.Functions, "main::target") || !slices.Contains(u.Features, "copied-closure-value") || !strings.Contains(u.Source, "int32_t captures[4]") || strings.Contains(u.Source, "malloc(") {
				t.Fatalf("closure transport/reachability missing: %+v", u)
			}
			if tc.captures == 0 && !strings.Contains(u.Source, "{ INT32_C(0), INT32_C(0), INT32_C(0), INT32_C(0) }") {
				t.Fatal("unused slots not initialized")
			}
			u2, err := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil || u.Source != u2.Source {
				t.Fatal("nondeterministic closure emission", err)
			}
			parts, err := cbackend.EmitBuildContext(context.Background(), p, "main::main", cbackend.Stage0WindowsMSVCProfile(), false, 1)
			if err != nil || parts.Header == "" || !strings.Contains(parts.Header, "fy_closure_") {
				t.Fatalf("partitioned closure prototypes: %+v %v", parts, err)
			}
			after, _ := json.Marshal(p)
			if string(before) != string(after) {
				t.Fatal("execution/emission mutated MIR")
			}
		})
	}
}

func TestClosureValueRuntimeNative(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	toolchain, err := cdriver.ResolveNativeToolchain(id, "")
	if err != nil {
		t.Fatal(err)
	}
	profile := cbackend.Stage0Profile()
	render := cruntime.LinuxRunSource
	if id == target.X8664WindowsMSVC {
		profile = cbackend.Stage0WindowsMSVCProfile()
		render = cruntime.WindowsRunSource
	}
	support, err := render(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name              string
		captures          int
		store, trap, call bool
	}{
		{"returned-struct-copy", 4, true, false, true}, {"zero", 0, false, false, true}, {"unused-trapping-body", 1, false, true, false}, {"invoke-trap", 1, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := closureRuntimeProgram(t, tc.captures, tc.store, tc.trap, tc.call)
			u, err := cbackend.Emit(p, "main::main", profile)
			if err != nil {
				t.Fatal(err)
			}
			for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := toolchain.Build(cdriver.NativeBuildOptions{Profile: optimization, UnitSource: u.Source, RuntimeSource: support, TrapControl: true})
				if err != nil {
					t.Fatalf("%v\n%s", err, u.Source)
				}
				out, err := toolchain.Execute(artifact, cdriver.NativeExecutionOptions{RejectOutput: true})
				if err != nil || (tc.trap && tc.call && out.TrapCategory != "division-by-zero") || (!(tc.trap && tc.call) && out.ExitCode != 42) {
					t.Fatalf("%s: %+v %v", optimization.Name, out, err)
				}
			}
		})
	}
}

func TestClosureValueRuntimeWindowsC17(t *testing.T) {
	id, err := cdriver.HostNativeTarget()
	if err != nil || id != target.X8664WindowsMSVC {
		t.Skip("Windows C17 consumer probe")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	support, err := cruntime.WindowsRunSource(cdriver.NativeTrapExitBase)
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name              string
		captures          int
		store, trap, call bool
	}{
		{"returned-struct-copy", 4, true, false, true}, {"zero", 0, false, false, true}, {"unused-trapping-body", 1, false, true, false}, {"invoke-trap", 1, false, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := closureRuntimeProgram(t, tc.captures, tc.store, tc.trap, tc.call)
			u, err := cbackend.Emit(p, "main::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			for _, optimization := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsMSVCABIProbeWithConsumerMode(toolchain, optimization, cdriver.C17ConsumerMode, u.Source, support)
				if err != nil {
					t.Fatal(err)
				}
				artifact.TrapControl = true
				raw, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil {
					t.Fatal(err)
				}
				out, err := cdriver.ClassifyWindowsNativeOutcome(raw)
				if err != nil || (tc.trap && tc.call && out.TrapCategory != "division-by-zero") || (!(tc.trap && tc.call) && out.ExitCode != 42) {
					t.Fatalf("%s %+v %v", optimization.Name, out, err)
				}
			}
		})
	}
}

func TestClosureValueRuntimePartitionedNative(t *testing.T) {
	p := closureRuntimeProgram(t, 4, true, false, true)
	id, err := cdriver.HostNativeTarget()
	if err != nil {
		t.Skip(err)
	}
	profile := cbackend.Stage0Profile()
	if id == target.X8664WindowsMSVC {
		profile = cbackend.Stage0WindowsMSVCProfile()
	}
	parts, err := cbackend.EmitBuildContext(t.Context(), p, "main::main", profile, false, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, optimization := range []string{"low", "optimized"} {
		t.Run(optimization, func(t *testing.T) { compileSplitForTest(t, parts, optimization, "", 42) })
	}
}
