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
)

func TestIntegerViewArtifactExecution(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "integer_views", "src", "main.fy"))
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
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		unit, err := cbackend.Emit(decoded, "main::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(unit.Source, "const int32_t *data;") || !strings.Contains(unit.Source, "fy_i32_slice") {
			t.Fatal("missing typed non-owning integer view")
		}
		if strings.Contains(unit.Source, "volatile int64_t") {
			t.Fatal("i64 workaround leaked into other element types")
		}

	}
	for _, element := range []hir.Type{hir.TypeI32, hir.TypeU32, hir.TypeUSize} {
		for name, test := range map[string]struct {
			kind   mir.OperationKind
			mutate func(*mir.Operation)
		}{
			"array element mismatch":   {mir.OperationArrayBorrowView, func(op *mir.Operation) { op.OwnerType = hir.FixedArray(hir.TypeU8, 4) }},
			"mismatched borrow kind":   {mir.OperationArrayBorrowView, func(op *mir.Operation) { op.Operator = "&mut" }},
			"wrong load type":          {mir.OperationViewIndex, func(op *mir.Operation) { op.Type = hir.TypeU8 }},
			"byte receiver laundering": {mir.OperationViewIndex, func(op *mir.Operation) { op.Type = hir.TypeU8; op.OwnerType = hir.SharedBorrow(hir.TypeByteSlice) }},
			"byte slice laundering":    {mir.OperationViewSlice, func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeByteSlice); op.OwnerType = op.Type }},
			"wrong bound type":         {mir.OperationViewIndex, func(op *mir.Operation) { op.Operands[1] = mir.Operand{Kind: mir.OperandI64, I64: 0} }},
			"missing slice guard":      {mir.OperationViewSliceCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
		} {
			t.Run(string(element)+"/"+name, func(t *testing.T) {
				program, err := mircodec.Decode(encoded, 1<<20)
				if err != nil {
					t.Fatal(err)
				}
				found := false
				for fi := range program.Functions {
					for bi := range program.Functions[fi].Blocks {
						for oi := range program.Functions[fi].Blocks[bi].Operations {
							op := &program.Functions[fi].Blocks[bi].Operations[oi]
							ownerElement, _ := hir.IndexedViewElement(op.OwnerType)
							if arrayElement, _, array := hir.FixedArrayParts(op.OwnerType); array {
								ownerElement = arrayElement
							}
							if !found && op.Kind == test.kind && ownerElement == element {
								test.mutate(op)
								found = true
							}
						}
					}
				}
				if !found || len(mir.Verify(program)) == 0 {
					t.Fatal("invalid view MIR accepted or absent")
				}
			})
		}
	}
}
