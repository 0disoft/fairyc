package compiler_test

import (
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func storageQueryHIR(t *testing.T) hir.Package {
	t.Helper()
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
struct Padded { u8 tag; u32 value; }
resource Owner { Padded payload; ptr mut void handle; cleanup {} }
fn usize bool_size() { return 0; }
fn usize bool_alignment() { return 0; }
fn usize padded_size() { return 0; }
fn usize padded_alignment() { return 0; }
fn usize owner_size() { return 0; }
fn usize owner_alignment() { return 0; }
fn i32 main() {
    return if (bool_size() == 4 && bool_alignment() == 4 && padded_size() == 8 && padded_alignment() == 4 && owner_size() == 16 && owner_alignment() == 8) { 42 } else { 0 };
}`)
	result := semantic.Analyze([]semantic.File{file})
	if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("fixture semantics: %+v %+v", result.Diagnostics, result.Unsupported)
	}
	queries := map[string]struct {
		typ   hir.Type
		field string
	}{
		"bool_size": {hir.TypeBool, "size"}, "bool_alignment": {hir.TypeBool, "alignment"},
		"padded_size": {"main::Padded", "size"}, "padded_alignment": {"main::Padded", "alignment"},
		"owner_size": {"main::Owner", "size"}, "owner_alignment": {"main::Owner", "alignment"},
	}
	count := 0
	for moduleIndex := range result.HIR.Modules {
		for functionIndex := range result.HIR.Modules[moduleIndex].Functions {
			function := &result.HIR.Modules[moduleIndex].Functions[functionIndex]
			query, found := queries[function.Name]
			if !found {
				continue
			}
			if len(function.Body.Statements) != 1 || function.Body.Statements[0].Expression == nil {
				t.Fatal("query fixture shape changed")
			}
			expression := function.Body.Statements[0].Expression
			*expression = hir.Expression{Kind: "storage-layout", Type: hir.TypeUSize, TargetType: query.typ,
				Operator: query.field, Symbol: hir.NoSymbol, Span: expression.Span}
			count++
		}
	}
	if count != len(queries) {
		t.Fatalf("inserted %d HIR queries, want %d", count, len(queries))
	}
	return result.HIR
}

func storageQueryMIR(t *testing.T) mir.Program {
	t.Helper()
	program, err := mir.Lower(storageQueryHIR(t))
	if err != nil {
		t.Fatal(err)
	}
	if failures := mir.VerifyStorageQueries(program); len(failures) != 0 {
		t.Fatal(failures)
	}
	return program
}

func TestPrivateStorageQueryHIRAndConsumerBoundaries(t *testing.T) {
	program := storageQueryMIR(t)
	count := 0
	for _, function := range program.Functions {
		for _, block := range function.Blocks {
			for _, op := range block.Operations {
				if op.Kind == mir.OperationStorageLayout {
					count++
					if op.OwnerType == "" || len(op.Operands) != 0 || op.Type != hir.TypeUSize {
						t.Fatal("query lost symbolic type or acquired an input value")
					}
				}
			}
		}
	}
	if count != 6 || len(mir.Verify(program)) == 0 {
		t.Fatal("private query escaped its verifier gate")
	}
	if _, err := interpreter.Execute(program, "main::main"); err == nil {
		t.Fatal("targetless interpreter accepted a storage query")
	}
	if _, err := mircodec.Encode(program, 1<<20); err == nil {
		t.Fatal("default MIR artifact encoded a private query")
	}
	encoded, err := json.Marshal(map[string]any{"schema": mircodec.Schema, "mirVersion": mircodec.MIRVersion, "program": program})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := mircodec.Decode(encoded, 1<<20); err == nil || !strings.Contains(err.Error(), "private storage query") {
		t.Fatalf("default MIR artifact decoder: %v", err)
	}
}

func TestPrivateStorageQueryRejectsMalformedMIR(t *testing.T) {
	for name, mutate := range map[string]func(*mir.Operation){
		"result":      func(op *mir.Operation) { op.Type = hir.TypeI32 },
		"field":       func(op *mir.Operation) { op.Field = "offset" },
		"type":        func(op *mir.Operation) { op.OwnerType = "main::Missing" },
		"destination": func(op *mir.Operation) { op.DestinationLocal = mir.NoLocal },
		"operand":     func(op *mir.Operation) { op.Operands = []mir.Operand{{Kind: mir.OperandI32, I32: 1}} },
		"ownership":   func(op *mir.Operation) { op.MovePayload = true },
		"trap":        func(op *mir.Operation) { op.TrapCategories = []string{"bounds"} },
	} {
		t.Run(name, func(t *testing.T) {
			program := storageQueryMIR(t)
			changed := false
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if !changed && op.Kind == mir.OperationStorageLayout {
							mutate(op)
							changed = true
						}
					}
				}
			}
			if !changed || len(mir.VerifyStorageQueries(program)) == 0 {
				t.Fatal("malformed query passed private verification")
			}
		})
	}
}

func TestPrivateStorageQueryGateIncludesUnreachableBlocks(t *testing.T) {
	program := storageQueryMIR(t)
	var query mir.Operation
	for fi := range program.Functions {
		for bi := range program.Functions[fi].Blocks {
			for oi := range program.Functions[fi].Blocks[bi].Operations {
				op := &program.Functions[fi].Blocks[bi].Operations[oi]
				if op.Kind == mir.OperationStorageLayout {
					query = *op
					*op = mir.Operation{Kind: mir.OperationAssign, DestinationLocal: op.DestinationLocal,
						Type: hir.TypeUSize, Operands: []mir.Operand{{Kind: mir.OperandUSize}}}
				}
			}
		}
	}
	if failures := mir.Verify(program); len(failures) != 0 {
		t.Fatalf("ordinary control: %v", failures)
	}
	function := &program.Functions[0]
	function.Blocks = append(function.Blocks, mir.Block{ID: mir.BlockID(len(function.Blocks)),
		Operations: []mir.Operation{query}, Terminator: mir.Terminator{Kind: mir.TerminatorUnreachable}})
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) != 0 {
		t.Fatal("unreachable valid query did not retain its private admission gate")
	}
	function.Blocks[len(function.Blocks)-1].Operations[0].OwnerType = "main::Missing"
	if len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("unreachable malformed query escaped shape/type verification")
	}
}

func TestPrivateStorageQueryMSVC(t *testing.T) {
	checkStorageQueryNative(t, storageQueryMIR(t))
}

func checkStorageQueryNative(t *testing.T, program mir.Program) {
	t.Helper()
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native storage-query proof requires Windows amd64")
	}
	unit, err := cbackend.Emit(program, "main::main", cbackend.Stage0WindowsMSVCProfile())
	if err != nil || !slices.Contains(unit.Features, "private-storage-layout") {
		t.Fatalf("emit: %v", err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("storage query execution: %+v %v", outcome, err)
			}
		})
	}
}
