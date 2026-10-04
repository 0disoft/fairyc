package compiler_test

import (
	"bytes"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"testing"
)

func TestStructArrayRoundTrip(t *testing.T) {
	src, e := os.ReadFile("testdata/struct_arrays.fy")
	if e != nil {
		t.Fatal(e)
	}
	files := []semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(src))}
	checked, e := compiler.Check(files)
	if e != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", e, checked.Semantic)
	}
	again, e := compiler.Check(files)
	if e != nil || !again.Verified {
		t.Fatalf("repeated source check: %v %+v %+v", e, again.Semantic.Diagnostics, again.Semantic.Unsupported)
	}
	data, e := mircodec.Encode(checked.MIR, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	againData, e := mircodec.Encode(again.MIR, 1<<20)
	if e != nil || !bytes.Equal(data, againData) {
		t.Fatalf("repeated analysis changed MIR: %v", e)
	}
	p, e := mircodec.Decode(data, 1<<20)
	if e != nil {
		t.Fatal(e)
	}
	out, e := interpreter.Execute(p, "main::main")
	if e != nil || out.Kind != interpreter.OutcomeReturn || out.Value != 0 {
		t.Fatalf("execute: %+v %v", out, e)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		_, e := cbackend.Emit(p, "main::main", profile)
		if e != nil {
			t.Fatal(e)
		}

	}

	for _, name := range []string{"missing-struct", "borrowed-field", "scalar-construction", "scalar-store"} {
		t.Run(name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
			}
			if name == "missing-struct" {
				bad.StructTypes = nil
			} else if name == "borrowed-field" {
				bad.StructTypes[0].Fields[0].Type = hir.SharedBorrow(hir.TypeI32)
			} else {
				found := false
				for fi := range bad.Functions {
					for bi := range bad.Functions[fi].Blocks {
						for oi := range bad.Functions[fi].Blocks[bi].Operations {
							op := &bad.Functions[fi].Blocks[bi].Operations[oi]
							if !found && name == "scalar-construction" && op.Kind == mir.OperationArrayConstruct {
								op.Operands[0] = mir.Operand{Kind: mir.OperandI32, I32: 1}
								found = true
							}
							if !found && name == "scalar-store" && op.Kind == mir.OperationArrayIndexStore {
								op.Operands[2] = mir.Operand{Kind: mir.OperandI32, I32: 1}
								found = true
							}
						}
					}
				}
				if !found {
					t.Fatal("missing array mutation target")
				}
			}
			if len(mir.Verify(bad)) == 0 {
				t.Fatal("invalid struct array verified")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid struct array encoded")
			}
		})
	}
	for _, tc := range []struct {
		name   string
		kind   mir.OperationKind
		mutate func(*mir.Operation)
	}{
		{"wrong-array", mir.OperationArrayBorrowView, func(op *mir.Operation) { op.OwnerType = hir.FixedArray(hir.TypeI32, 3) }},
		{"numeric-load", mir.OperationViewIndex, func(op *mir.Operation) { op.Type = hir.TypeI32 }},
		{"numeric-store", mir.OperationViewIndexStore, func(op *mir.Operation) { op.Type = hir.TypeI32 }},
		{"numeric-reference", mir.OperationViewElementBorrow, func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeI32) }},
		{"slice-laundering", mir.OperationViewSlice, func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeI32Slice) }},
		{"no-bounds", mir.OperationViewIndexCheck, func(op *mir.Operation) { op.Kind = mir.OperationUse; op.Symbol = hir.SymbolID(op.Operands[0].Local) }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			bad, e := mircodec.Decode(data, 1<<20)
			if e != nil {
				t.Fatal(e)
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
				t.Fatal("invalid struct array MIR accepted or missing")
			}
			if _, e := mircodec.Encode(bad, 1<<20); e == nil {
				t.Fatal("invalid struct array MIR crossed codec")
			}
			for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
				if _, e := cbackend.Emit(bad, "main::main", profile); e == nil {
					t.Fatal("invalid struct array MIR emitted")
				}
			}
		})
	}
}
