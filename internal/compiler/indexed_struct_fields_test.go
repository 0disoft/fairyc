package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"strings"
	"testing"
)

func TestIndexedStructFieldsMIR(t *testing.T) {
	source := `module main;struct Span{usize start;usize end;}struct Token{i64 value;Span position;}struct Other{i64 value;Span position;}
 fn ref mut i64 select(ref mut Token[] view,usize index){return ref mut view[index].value;}
 fn ref usize read(ref Token[] view){return ref view[0].position.start;}
 fn void edit(ref mut Token[] view){view[0].value+=3;view[0].position=Span{.start=2,.end=4};}
 resource Owner{ptr mut void data;usize length;cleanup{}}
 fn i32 main(){var Owner owner=Owner{.data=null,.length=0};let ref mut Token[] view=unsafe{storage_borrow_mut<Token[]>(ref mut owner,owner.data,owner.length)};if(view.len>0){edit(view);let ref mut i64 value=select(view,0);assert(*value>=0);let ref usize start=read(view);assert(*start==2);}return 0;}`
	analysis, err := analyzeStorageBorrow(t, source)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v", err, analysis.Analysis)
	}
	encoded, err := mircodec.EncodeStorageTransfers(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
		result, err := cbackend.EmitStorageTransfers(checked.MIR, hir.QualifiedModulePath(1, "main")+"::main", profile)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(result.Source, "&((") {
			t.Fatal("missing indexed address")
		}
	}
	for _, mutation := range []string{"shared-exclusive", "wrong-nominal", "wrong-owner-local", "wrong-index-local", "missing-check", "nonplain", "wrong-field"} {
		t.Run(mutation, func(t *testing.T) {
			program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			changed := false
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					b := &program.Functions[fi].Blocks[bi]
					for oi := range b.Operations {
						op := &b.Operations[oi]
						if mutation == "missing-check" && op.Kind == mir.OperationViewIndexCheck {
							b.Operations = append(b.Operations[:oi], b.Operations[oi+1:]...)
							changed = true
							break
						}
						if mutation == "wrong-field" && op.Kind == mir.OperationFieldBorrow {
							op.Field = "missing"
							changed = true
							break
						}
						if op.Kind != mir.OperationViewElementBorrow {
							continue
						}
						switch mutation {
						case "shared-exclusive":
							if hir.MutableIndexedView(op.OwnerType) {
								slice, _, _ := hir.BorrowElement(op.OwnerType)
								op.OwnerType = hir.SharedBorrow(slice)
								changed = true
							}
						case "wrong-nominal":
							element, mut, _ := hir.BorrowElement(op.Type)
							element = hir.ParseType(strings.Replace(element.String(), "::Token", "::Other", 1))
							op.Type = hir.SharedBorrow(element)
							if mut {
								op.Type = hir.MutableBorrow(element)
							}
							changed = true
						case "wrong-owner-local":
							op.Operands[0] = op.Operands[1]
							op.Operands[0].Type = op.OwnerType
							changed = true
						case "wrong-index-local":
							op.Operands[1] = op.Operands[0]
							op.Operands[1].Type = hir.TypeUSize
							changed = true
						}
						if changed {
							break
						}
					}
					if changed {
						break
					}
				}
				if changed {
					break
				}
			}
			if mutation == "nonplain" {
				for si := range program.StructTypes {
					if strings.HasSuffix((program.StructTypes[si].Type).String(), "::Token") {
						program.StructTypes[si].Fields[0].Type = hir.RawPointer(hir.TypeI64, true)
						changed = true
					}
				}
			}
			if !changed {
				t.Fatal("mutation missed target")
			}
			if len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatal("invalid indexed borrow accepted")
			}
			if _, err := mircodec.EncodeStorageTransfers(program, 1<<20); err == nil {
				t.Fatal("invalid indexed borrow crossed codec")
			}
		})
	}
}
