package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

func TestI64MutableViewArtifactExecution(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "i64_mut_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil || !strings.Contains(unit.Source, "typedef struct fy_mut_i64_slice") || !strings.Contains(unit.Source, "    int64_t *data;") {
			t.Fatalf("emit: %v", err)
		}
		workaround := strings.Contains(unit.Source, "const volatile int64_t *") && strings.Contains(unit.Source, "volatile int64_t *")
		if workaround != (profile.Target == target.X8664WindowsMSVC) {
			t.Fatal("mutable i64 read/store workaround crossed its target boundary")
		}

	}
	for name, test := range map[string]struct {
		kind   mir.OperationKind
		mutate func(*mir.Operation)
	}{
		"shared store":            {mir.OperationViewIndexStore, func(op *mir.Operation) { op.OwnerType = hir.SharedBorrow(hir.TypeI64Slice) }},
		"byte store laundering":   {mir.OperationViewIndexStore, func(op *mir.Operation) { op.OwnerType = hir.MutableBorrow(hir.TypeByteSlice); op.Type = hir.TypeU8 }},
		"wrong store value":       {mir.OperationViewIndexStore, func(op *mir.Operation) { op.Operands[2] = mir.Operand{Kind: mir.OperandU8, U64: 1} }},
		"wrong index":             {mir.OperationViewIndexStore, func(op *mir.Operation) { op.Operands[1] = mir.Operand{Kind: mir.OperandI64, I64: 0} }},
		"shared slice laundering": {mir.OperationViewSlice, func(op *mir.Operation) { op.OwnerType = hir.SharedBorrow(hir.TypeI64Slice); op.Type = op.OwnerType }},
		"missing store guard":     {mir.OperationViewIndexCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
	} {
		t.Run(name, func(t *testing.T) {
			mutated, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range mutated.Functions {
				for bi := range mutated.Functions[fi].Blocks {
					for oi := range mutated.Functions[fi].Blocks[bi].Operations {
						op := &mutated.Functions[fi].Blocks[bi].Operations[oi]
						if !found && op.Kind == test.kind {
							test.mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(mutated)) == 0 {
				t.Fatal("invalid mutable view MIR accepted or absent")
			}
		})
	}
}

func TestI64MutableViewTraps(t *testing.T) {
	for name, item := range map[string]struct{ action, trap string }{
		"index-end":  {"view[2] = 7;", "bounds"},
		"index-huge": {"view[18446744073709551615] = 7;", "bounds"},
		"empty":      {"view[2..][0] = 7;", "bounds"},
		"reversed":   {"let ref mut i64[] child = view[2..1];", "bounds"},
		"past-end":   {"let ref mut i64[] child = view[..3];", "bounds"},
		"overflow":   {"view[0] += 1;", "integer-overflow"},
		"underflow":  {"view[1] -= 1;", "integer-overflow"},
	} {
		t.Run(name, func(t *testing.T) {
			source := "module main; fn i32 main() { var i64[2] data = [9223372036854775807,-9223372036854775808]; let ref mut i64[] view = ref mut data; " + item.action + " return 0; }"
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked.Semantic)
			}
			outcome, err := interpreter.Execute(checked.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != item.trap {
				t.Fatalf("trap: %v %+v", err, outcome)
			}
		})
	}
}
