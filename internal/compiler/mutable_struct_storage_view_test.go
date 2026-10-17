package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

func mutableStructStorageSource() string {
	source := strings.NewReplacer("fn ref Token[] view", "fn ref mut Token[] view", "ref Owner owner", "ref mut Owner owner", "storage_borrow<Token[]>", "storage_borrow_mut<Token[]>", "view(ref owner)", "view(ref mut owner)", "let ref Token[] values", "let ref mut Token[] values").Replace(structStorageViewSource)
	return strings.Replace(source, "if(values.len > 0) {let Token copy = values[0]; assert(copy.active);}", `if(values.len > 0) {let Other other=Other{.kind=1,.offset=1,.value=1,.active=false};values[0]=Token{.kind=2,.offset=3,.value=4000000000,.active=true};let Token copy=values[0];assert(copy.active);}`, 1)
}
func TestMutableStructStorageViewMIR(t *testing.T) {
	slice := hir.ParseType("[" + hir.QualifiedModulePath(1, "main") + "::Token]")
	checkStorageViewMIR(t, mutableStructStorageSource(), true, slice)
	analysis, err := analyzeStorageBorrow(t, mutableStructStorageSource())
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("lower: %v", err)
	}
	encoded, err := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, mutation := range []string{"shared-store", "wrong-value-nominal", "value-annotation", "owner-annotation", "missing-check", "missing-definition", "nested-pointer"} {
		t.Run(mutation, func(t *testing.T) {
			program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			changed := false
			for fi := range program.Functions {
				f := &program.Functions[fi]
				for bi := range f.Blocks {
					b := &f.Blocks[bi]
					for oi := range b.Operations {
						op := &b.Operations[oi]
						if mutation == "missing-check" && op.Kind == mir.OperationViewIndexCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							changed = true
							break
						}
						if op.Kind != mir.OperationViewIndexStore {
							continue
						}
						switch mutation {
						case "shared-store":
							op.OwnerType = hir.SharedBorrow(slice)
							changed = true
						case "wrong-value-nominal":
							op.Type = hir.ParseType(hir.QualifiedModulePath(1, "main") + "::Other")
							changed = true
						case "value-annotation":
							op.Operands[2] = op.Operands[0]
							op.Operands[2].Type = op.Type
							changed = true
						case "owner-annotation":
							op.Operands[0] = op.Operands[2]
							op.Operands[0].Type = op.OwnerType
							changed = true
						}
					}
				}
			}
			if mutation == "missing-definition" || mutation == "nested-pointer" {
				for i := range program.StructTypes {
					s := &program.StructTypes[i]
					if strings.HasSuffix(s.Type.String(), "::Token") {
						if mutation == "missing-definition" {
							s.Type = hir.ParseType("main::Missing")
						} else {
							s.Fields[0].Type = hir.RawPointer(hir.TypeI64, true)
						}
						changed = true
					}
				}
			}
			if !changed {
				t.Fatal("mutation did not reach its target")
			}
			if len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatal("invalid nominal store accepted")
			}
			if _, err := mircodec.EncodeStorageTransfers(program, 1<<20); err == nil {
				t.Fatal("invalid nominal store crossed codec")
			}
		})
	}
}
