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
	analysis, err := analyzeStorageBorrow(t, i64StorageViewSource)
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
		if err != nil || !strings.Contains(unit.Source, "(fy_i64_slice){ .data = (const int64_t *)") {
			t.Fatalf("typed view emission: %v", err)
		}
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"missing-length":           func(op *mir.Operation) { op.Operands = op.Operands[:2] },
		"wrong-length":             func(op *mir.Operation) { op.Operands[2] = mir.Operand{Kind: mir.OperandI64, I64: 0} },
		"length-local-laundering":  func(op *mir.Operation) { op.Operands[2] = op.Operands[1]; op.Operands[2].Type = hir.TypeUSize },
		"owner-local-laundering":   func(op *mir.Operation) { op.Operands[0] = op.Operands[1]; op.Operands[0].Type = op.OwnerType },
		"address-local-laundering": func(op *mir.Operation) { op.Operands[1] = op.Operands[0]; op.Operands[1].Type = op.IndexType },
		"byte-result":              func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeByteSlice) },
		"mutable-result":           func(op *mir.Operation) { op.Type = hir.MutableBorrow(hir.TypeI64Slice) },
		"mutable-owner": func(op *mir.Operation) {
			owner, _, _ := hir.BorrowElement(op.OwnerType)
			op.OwnerType = hir.MutableBorrow(owner)
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
						if op.Kind == mir.OperationStorageBorrow && op.Type == hir.SharedBorrow(hir.TypeI64Slice) {
							mutate(op)
							found = true
						}
					}
				}
			}
			if !found || len(mir.VerifyStorageTransfers(changed)) == 0 {
				t.Fatal("malformed i64 view accepted")
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
