package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
)

const i64StorageViewSource = `module main;
resource Owner { ptr mut void data; usize length; cleanup {} }
fn ref i64[] view(ref Owner owner) { return unsafe { storage_borrow<i64[]>(owner, owner.data, owner.length) }; }
fn void mutate(ref mut Owner owner) {}
fn i32 main() {
    var Owner owner = Owner { .data = null, .length = 0 };
    let ref i64[] values = view(ref owner);
    assert(values[..].is_empty);
    return 0;
}
`

func TestI64StorageViewMIR(t *testing.T) {
	checkStorageViewMIR(t, i64StorageViewSource, false, hir.TypeI64Slice)
}

func mutableI64StorageViewSource() string {
	return strings.NewReplacer("fn ref i64[] view", "fn ref mut i64[] view", "ref Owner owner", "ref mut Owner owner", "storage_borrow<i64[]>", "storage_borrow_mut<i64[]>", "view(ref owner)", "view(ref mut owner)", "let ref i64[] values", "let ref mut i64[] values").Replace(i64StorageViewSource)
}

func TestI64MutableStorageViewMIR(t *testing.T) {
	checkStorageViewMIR(t, mutableI64StorageViewSource(), true, hir.TypeI64Slice)
}

func checkStorageViewMIR(t *testing.T, source string, mutable bool, slice hir.Type) {
	t.Helper()
	kind, viewType, cDescriptor := mir.OperationStorageBorrow, hir.SharedBorrow(slice), "(fy_i64_slice){ .data = (const int64_t *)"
	if mutable {
		kind, viewType, cDescriptor = mir.OperationStorageBorrowMut, hir.MutableBorrow(slice), "(fy_mut_i64_slice){ .data = (int64_t *)"
	}
	otherSlice := hir.TypeByteSlice
	if slice == hir.TypeByteSlice {
		otherSlice = hir.TypeI64Slice
		cDescriptor = "(fy_byte_slice){ .data = (const uint8_t *)"
		if mutable {
			cDescriptor = "(fy_mut_byte_slice){ .data = (uint8_t *)"
		}
	}
	analysis, err := analyzeStorageBorrow(t, source)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !compiled.Verified {
		t.Fatalf("lower: %v %+v %+v", err, compiled.Semantic.Diagnostics, compiled.Semantic.Unsupported)
	}
	encoded, err := mircodec.EncodeStorageTransfers(compiled.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("private view escaped its transfer admission")
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.EmitStorageTransfers(program, hir.QualifiedModulePath(1, "main")+"::main", profile)
		if err != nil || !strings.Contains(unit.Source, cDescriptor) {
			t.Fatalf("typed view emission: %v", err)
		}
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"missing-length":           func(op *mir.Operation) { op.Operands = op.Operands[:2] },
		"wrong-length":             func(op *mir.Operation) { op.Operands[2] = mir.Operand{Kind: mir.OperandI64, I64: 0} },
		"length-local-laundering":  func(op *mir.Operation) { op.Operands[2] = op.Operands[1]; op.Operands[2].Type = hir.TypeUSize },
		"owner-local-laundering":   func(op *mir.Operation) { op.Operands[0] = op.Operands[1]; op.Operands[0].Type = op.OwnerType },
		"address-local-laundering": func(op *mir.Operation) { op.Operands[1] = op.Operands[0]; op.Operands[1].Type = op.IndexType },
		"wrong-element-result": func(op *mir.Operation) {
			op.Type = hir.SharedBorrow(otherSlice)
			if mutable {
				op.Type = hir.MutableBorrow(otherSlice)
			}
		},
		"wrong-result-mutability": func(op *mir.Operation) {
			op.Type = hir.MutableBorrow(slice)
			if mutable {
				op.Type = hir.SharedBorrow(slice)
			}
		},
		"wrong-operation-kind": func(op *mir.Operation) {
			op.Kind = mir.OperationStorageBorrowMut
			if mutable {
				op.Kind = mir.OperationStorageBorrow
			}
		},
		"mutable-owner": func(op *mir.Operation) {
			owner, _, _ := hir.BorrowElement(op.OwnerType)
			op.OwnerType = hir.MutableBorrow(owner)
			if mutable {
				op.OwnerType = hir.SharedBorrow(owner)
			}
		},
		"typed-address":     func(op *mir.Operation) { op.IndexType = hir.RawPointer(hir.TypeI64, true) },
		"missing-unsafe":    func(op *mir.Operation) { op.UnsafeScope = hir.Span{} },
		"destination-alias": func(op *mir.Operation) { op.DestinationLocal = op.Operands[2].Local },
	} {
		t.Run(name, func(t *testing.T) {
			changed, err := mircodec.DecodeStorageTransfers(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			found := false
			for fi := range changed.Functions {
				for bi := range changed.Functions[fi].Blocks {
					for oi := range changed.Functions[fi].Blocks[bi].Operations {
						op := &changed.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == kind && op.Type == viewType {
							mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.VerifyStorageTransfers(changed)) == 0 {
				t.Fatal("malformed storage view accepted")
			}
		})
	}
}

func TestI64StorageViewSourceRejections(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"unsafe-missing":        {"unsafe { storage_borrow<i64[]>(owner, owner.data, owner.length) }", "storage_borrow<i64[]>(owner, owner.data, owner.length)"},
		"wrong-count":           {"owner.data, owner.length", "owner.data, true"},
		"missing-count":         {"owner.data, owner.length", "owner.data"},
		"extra-count":           {"owner.data, owner.length", "owner.data, owner.length, 0"},
		"mutable-view":          {"storage_borrow<i64[]>", "storage_borrow_mut<i64[]>"},
		"wrong-element":         {"storage_borrow<i64[]>", "storage_borrow<u8[]>"},
		"length-mutates-owner":  {"view(ref owner)", "unsafe { storage_borrow<i64[]>(ref owner, owner.data, { mutate(ref mut owner); 0 }) }"},
		"address-mutates-owner": {"view(ref owner)", "unsafe { storage_borrow<i64[]>(ref owner, { mutate(ref mut owner); owner.data }, owner.length) }"},
	} {
		t.Run(name, func(t *testing.T) {
			analysis, err := analyzeStorageBorrow(t, strings.Replace(i64StorageViewSource, replacement[0], replacement[1], 1))
			if err == nil && len(analysis.Analysis.Semantic.Diagnostics) == 0 && len(analysis.Analysis.Semantic.Unsupported) == 0 {
				t.Fatal("invalid private view source accepted")
			}
			if name == "length-mutates-owner" || name == "address-mutates-owner" {
				found := false
				for _, diagnostic := range analysis.Analysis.Semantic.Diagnostics {
					found = found || diagnostic.Code == "E2101"
				}
				if !found {
					t.Fatalf("owner hold missing: %v %+v %+v", err, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
				}
			}
		})
	}
}

func TestI64MutableStorageViewSourceRejections(t *testing.T) {
	source := mutableI64StorageViewSource()
	for name, replacement := range map[string][2]string{
		"unsafe missing":   {"unsafe { storage_borrow_mut<i64[]>(owner, owner.data, owner.length) }", "storage_borrow_mut<i64[]>(owner, owner.data, owner.length)"},
		"missing count":    {"owner.data, owner.length", "owner.data"},
		"wrong count":      {"owner.data, owner.length", "owner.data, true"},
		"wrong element":    {"storage_borrow_mut<i64[]>", "storage_borrow_mut<i32[]>"},
		"shared owner":     {"ref mut Owner owner", "ref Owner owner"},
		"shared address":   {"ptr mut void data", "ptr void data"},
		"address mutation": {"view(ref mut owner)", "unsafe { storage_borrow_mut<i64[]>(ref mut owner, { mutate(ref mut owner); owner.data }, owner.length) }"},
		"count mutation":   {"view(ref mut owner)", "unsafe { storage_borrow_mut<i64[]>(ref mut owner, owner.data, { mutate(ref mut owner); 0 }) }"},
	} {
		t.Run(name, func(t *testing.T) {
			analysis, err := analyzeStorageBorrow(t, strings.Replace(source, replacement[0], replacement[1], 1))
			if err == nil && len(analysis.Analysis.Semantic.Diagnostics) == 0 && len(analysis.Analysis.Semantic.Unsupported) == 0 {
				t.Fatal("invalid mutable private view accepted")
			}
			if name == "address mutation" || name == "count mutation" {
				found := false
				for _, d := range analysis.Analysis.Semantic.Diagnostics {
					found = found || d.Code == "E2101"
				}
				if !found {
					t.Fatalf("owner hold missing: %v %+v", err, analysis.Analysis.Semantic)
				}
			}
		})
	}
}

func TestU8StorageViewMIR(t *testing.T) {
	checkStorageViewMIR(t, strings.ReplaceAll(i64StorageViewSource, "i64", "u8"), false, hir.TypeByteSlice)
}
func TestU8MutableStorageViewMIR(t *testing.T) {
	checkStorageViewMIR(t, strings.ReplaceAll(mutableI64StorageViewSource(), "i64", "u8"), true, hir.TypeByteSlice)
}
func TestU8StorageViewSourceRejections(t *testing.T) {
	for _, mutable := range []bool{false, true} {
		source := strings.ReplaceAll(i64StorageViewSource, "i64", "u8")
		selector := "storage_borrow<u8[]>"
		invocation, ownerArg := "view(ref owner)", "ref owner"
		if mutable {
			source = strings.ReplaceAll(mutableI64StorageViewSource(), "i64", "u8")
			selector = "storage_borrow_mut<u8[]>"
			invocation, ownerArg = "view(ref mut owner)", "ref mut owner"
		}
		for name, replacement := range map[string][2]string{
			"unsafe-missing":      {"unsafe { " + selector + "(owner, owner.data, owner.length) }", selector + "(owner, owner.data, owner.length)"},
			"count-missing":       {"owner.data, owner.length", "owner.data"},
			"count-wrong":         {"owner.data, owner.length", "owner.data, true"},
			"unsupported-element": {selector, strings.Replace(selector, "u8", "i32", 1)},
			"address-effect":      {invocation, "unsafe { " + selector + "(" + ownerArg + ", { mutate(ref mut owner); owner.data }, owner.length) }"},
			"count-effect":        {invocation, "unsafe { " + selector + "(" + ownerArg + ", owner.data, { mutate(ref mut owner); 0 }) }"},
		} {
			t.Run(selector+"/"+name, func(t *testing.T) {
				analysis, err := analyzeStorageBorrow(t, strings.Replace(source, replacement[0], replacement[1], 1))
				if err == nil && len(analysis.Analysis.Semantic.Diagnostics) == 0 && len(analysis.Analysis.Semantic.Unsupported) == 0 {
					t.Fatal("invalid u8 storage view accepted")
				}
				if name == "address-effect" || name == "count-effect" {
					found := false
					for _, d := range analysis.Analysis.Semantic.Diagnostics {
						found = found || d.Code == "E2101"
					}
					if !found {
						t.Fatalf("missing owner hold: %v %+v %+v", err, analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
					}
				}
			})
		}
	}
}
