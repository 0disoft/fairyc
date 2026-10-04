package compiler_test

import (
	"os"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestStructArrayFieldRoundTrip(t *testing.T) {
	source, err := os.ReadFile("testdata/struct_array_fields.fy")
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	data, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(data, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	out, err := interpreter.Execute(program, "main::main")
	if err != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		if _, err := cbackend.Emit(program, "main::main", profile); err != nil {
			t.Fatal(err)
		}
	}
	for _, tc := range []struct {
		name   string
		kind   mir.OperationKind
		mutate func(*mir.Operation)
	}{
		{"missing-bounds", mir.OperationViewIndexCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
		{"wrong-owner", mir.OperationArrayBorrowView, func(op *mir.Operation) { op.OwnerType = hir.FixedArray(hir.TypeI32, 2) }},
		{"shared-store", mir.OperationFieldStore, func(op *mir.Operation) {
			element, _, _ := hir.BorrowElement(op.OwnerType)
			op.OwnerType = hir.SharedBorrow(element)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad, err := mircodec.Decode(data, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range bad.Functions {
				for bi := range bad.Functions[fi].Blocks {
					for oi := range bad.Functions[fi].Blocks[bi].Operations {
						op := &bad.Functions[fi].Blocks[bi].Operations[oi]
						if !found && op.Kind == tc.kind {
							tc.mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid indexed store accepted or missing mutation target")
			}
			if _, err := mircodec.Encode(bad, 1<<20); err == nil {
				t.Fatal("invalid indexed store encoded")
			}
		})
	}
}
