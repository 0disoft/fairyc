package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

const structStorageViewSource = `module main;
struct Token {u32 kind; usize offset; i64 value; bool active;}
struct Other {u32 kind; usize offset; i64 value; bool active;}
resource Owner {ptr mut void data; usize length; cleanup {}}
fn ref Token[] view(ref Owner owner) { return unsafe {storage_borrow<Token[]>(owner,owner.data,owner.length)}; }
fn i32 main() {
 var Owner owner = Owner {.data = null, .length = 0};
 let ref Token[] values = view(ref owner);
 assert(values[..].is_empty);
 if(values.len > 0) {let Token copy = values[0]; assert(copy.active);}
 return 0;
}`

func TestStructStorageViewMIR(t *testing.T) {
	slice := hir.Type("[" + hir.QualifiedModulePath(1, "main") + "::Token]")
	checkStorageViewMIR(t, structStorageViewSource, false, slice)
	analysis, err := analyzeStorageBorrow(t, structStorageViewSource)
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
	for _, mutation := range []string{"same-layout-nominal", "mutable-result", "empty-struct", "pointer-field", "resource-field", "missing-definition", "load-element", "receiver-annotation"} {
		t.Run(mutation, func(t *testing.T) {
			program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			for si := range program.StructTypes {
				s := &program.StructTypes[si]
				if strings.HasSuffix(string(s.Type), "::Token") {
					switch mutation {
					case "empty-struct":
						s.Fields = nil
					case "pointer-field":
						s.Fields[0].Type = hir.RawPointer(hir.TypeI32, true)
					case "resource-field":
						s.Fields[0].Type = program.ResourceTypes[0].Type
					case "missing-definition":
						s.Type = hir.Type("main::Missing")
					}
				}
			}
			changed := false
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if changed {
							continue
						}
						if op.Kind == mir.OperationStorageBorrow {
							switch mutation {
							case "same-layout-nominal":
								op.Type = hir.SharedBorrow(hir.Type("[" + hir.QualifiedModulePath(1, "main") + "::Other]"))
								changed = true
							case "mutable-result":
								op.Type = hir.MutableBorrow(slice)
								changed = true
							}
						}
						if op.Kind == mir.OperationViewIndex {
							switch mutation {
							case "load-element":
								op.Type = hir.Type(hir.QualifiedModulePath(1, "main") + "::Other")
								changed = true
							case "receiver-annotation":
								op.OwnerType = hir.SharedBorrow(hir.TypeByteSlice)
								op.Type = hir.TypeU8
								op.Operands[0].Type = op.OwnerType
								changed = true
							}
						}
					}
				}
			}
			if len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatal("invalid struct descriptor accepted")
			}
			if _, err := mircodec.EncodeStorageTransfers(program, 1<<20); err == nil {
				t.Fatal("invalid descriptor crossed private codec")
			}
		})
	}
}
