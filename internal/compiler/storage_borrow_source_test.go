package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

func analyzeStorageBorrow(t *testing.T, text string) (compiler.GenericAnalysis, error) {
	t.Helper()
	input := semantic.PackageInput{Files: []semantic.File{parseCompilerFile(t, 1, "main.fy", text)}}
	before, _ := json.Marshal(input)
	analysis, err := compiler.AnalyzeStorageTransferPackages(input, map[hir.PackageID]packagegraph.CoordinateID{1: genericCoordinate(t)}, genericplan.DefaultPlanLimits())
	after, _ := json.Marshal(input)
	if string(before) != string(after) {
		t.Fatal("storage borrow mutated source")
	}
	return analysis, err
}

func TestPrivateVectorStorageBorrowSourceOrigin(t *testing.T) {
	t.Run("scalar", func(t *testing.T) { checkStorageBorrowOrigin(t, storageBorrowSource) })
	t.Run("resource", func(t *testing.T) {
		text := strings.NewReplacer("resource Owner", "resource Item { i32 value; cleanup {} } resource Owner", "ref i32", "ref Item", "borrow_element<i32>", "borrow_element<Item>").Replace(storageBorrowSource)
		checkStorageBorrowOrigin(t, text)
	})
}

func checkStorageBorrowOrigin(t *testing.T, text string) {
	t.Helper()
	analysis, err := analyzeStorageBorrow(t, text)
	result := analysis.Analysis.Semantic
	if err != nil || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("storage borrow source: %v %+v %+v", err, result.Diagnostics, result.Unsupported)
	}
	escaping := 0
	for _, fact := range result.Borrows {
		if fact.Escapes {
			if fact.Origin != "Param(0)" {
				t.Fatalf("borrow escaped with wrong origin: %+v", fact)
			}
			escaping++
		}
	}
	if escaping != 2 {
		t.Fatalf("missing direct/relay owner origin: %+v", result.Borrows)
	}
	encoded, _ := json.Marshal(result.HIR)
	if !strings.Contains(string(encoded), "storage-borrow") {
		t.Fatal("typed storage borrow missing")
	}
	compiled, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !compiled.Verified {
		t.Fatalf("storage borrow MIR: %v", err)
	}
	program, err := mir.Lower(result.HIR)
	if err != nil || len(mir.VerifyStorageTransfers(program)) != 0 {
		t.Fatalf("storage borrow lowering: %v", err)
	}
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("storage borrow escaped private admission")
	}
}

func TestPrivateVectorStorageBorrowSourceMIRMutations(t *testing.T) {
	for name, mutate := range map[string]func(*mir.Operation){
		"owner omitted":        func(op *mir.Operation) { op.Operands = op.Operands[1:] },
		"mutable result":       func(op *mir.Operation) { op.Type = hir.MutableBorrow(hir.TypeI32) },
		"unknown element":      func(op *mir.Operation) { op.Type = hir.SharedBorrow("main::Missing") },
		"mutable owner":        func(op *mir.Operation) { op.OwnerType = hir.MutableBorrow("main::Owner") },
		"raw owner":            func(op *mir.Operation) { op.OwnerType = hir.RawPointer(hir.TypeVoid, false) },
		"typed address":        func(op *mir.Operation) { op.IndexType = hir.RawPointer(hir.TypeI32, false) },
		"unknown owner local":  func(op *mir.Operation) { op.Operands[0].Local = 900000 },
		"result aliases owner": func(op *mir.Operation) { op.DestinationLocal = op.Operands[0].Local },
		"unsafe scope":         func(op *mir.Operation) { op.UnsafeScope = hir.Span{} },
		"source identity":      func(op *mir.Operation) { op.Entity = "forged" },
		"ownership transfer":   func(op *mir.Operation) { op.MovePayload = true },
	} {
		t.Run(name, func(t *testing.T) {
			analysis, err := analyzeStorageBorrow(t, storageBorrowSource)
			if err != nil {
				t.Fatal(err)
			}
			program, err := mir.Lower(analysis.Analysis.Semantic.HIR)
			if err != nil {
				t.Fatal(err)
			}
			changed := 0
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationStorageBorrow {
							mutate(op)
							changed++
						}
					}
				}
			}
			if changed != 1 || len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatalf("mutation not rejected: %d operations", changed)
			}
		})
	}
}

func TestPrivateVectorStorageBorrowSourceRejections(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"unsafe missing":              {"unsafe { storage_borrow<T>(owner, owner.data) }", "storage_borrow<T>(owner, owner.data)"},
		"owner missing":               {"storage_borrow<T>(owner, owner.data)", "storage_borrow<T>(owner.data, owner.data)"},
		"pointer missing":             {"storage_borrow<T>(owner, owner.data)", "storage_borrow<T>(owner, 0)"},
		"wrong arity":                 {"storage_borrow<T>(owner, owner.data)", "storage_borrow<T>(owner)"},
		"owner moved before use":      {"observe(view); consume(owner);", "consume(owner); observe(view);"},
		"exclusive access before use": {"observe(view); consume(owner);", "mutate(ref mut owner); observe(view); consume(owner);"},
		"local return escape":         {"fn i32 main()", "fn ref i32 escape() { let Owner owner = Owner { .data = null }; return relay(ref owner); } fn i32 main()"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(storageBorrowSource, replacement[0], replacement[1], 1)
			if text == storageBorrowSource {
				t.Fatal("borrow rejection fixture unchanged")
			}
			analysis, err := analyzeStorageBorrow(t, text)
			result := analysis.Analysis.Semantic
			if err == nil && len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("invalid storage borrow admitted")
			}
		})
	}
}

func TestPrivateVectorStorageBorrowSourceMutableOrigin(t *testing.T) {
	analysis, err := analyzeStorageBorrow(t, storageBorrowMutSource)
	result := analysis.Analysis.Semantic
	if err != nil || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("mutable storage borrow source: %v %+v %+v", err, result.Diagnostics, result.Unsupported)
	}
	encoded, _ := json.Marshal(result.HIR)
	if !strings.Contains(string(encoded), "storage-borrow-mut") {
		t.Fatal("typed mutable storage borrow missing")
	}
	var escaped bool
	for _, fact := range result.Borrows {
		if fact.Escapes && fact.Origin == "Param(0)" {
			escaped = true
		}
	}
	if !escaped {
		t.Fatalf("mutable borrow lost its owner origin: %+v", result.Borrows)
	}
}

func TestPrivateVectorStorageBorrowSourceElementAddressPreparation(t *testing.T) {
	const original = "storage_borrow_mut<T>(owner, owner.data)"
	const address = "storage_borrow_mut<T>(owner, storage_element_address<T>(owner.data, 0))"
	text := strings.Replace(storageBorrowMutSource, original, address, 1)
	if text == storageBorrowMutSource {
		t.Fatal("element address fixture unchanged")
	}
	analysis, err := analyzeStorageBorrow(t, text)
	result := analysis.Analysis.Semantic
	if err != nil || len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
		t.Fatalf("private element address source: %v %+v %+v", err, result.Diagnostics, result.Unsupported)
	}
	encoded, err := json.Marshal(result.HIR)
	if err != nil || !strings.Contains(string(encoded), "storage-element-address") {
		t.Fatalf("typed element address missing: %v %s", err, encoded)
	}
	program, err := mir.Lower(result.HIR)
	if err != nil || len(mir.VerifyStorageTransfers(program)) != 0 {
		t.Fatalf("private element address MIR: %v %+v", err, mir.VerifyStorageTransfers(program))
	}
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("element address escaped private MIR admission")
	}
	addresses := 0
	for _, function := range program.Functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind == mir.OperationStorageElementAddress {
					addresses++
				}
			}
		}
	}
	if addresses != 1 {
		t.Fatalf("element address operations = %d, want one", addresses)
	}
	for _, invalid := range []string{
		"storage_element_address<T>(owner.data)",
		"storage_element_address<T>(owner.data, 0, 1)",
		"storage_element_address<T>(0, 0)",
		"storage_element_address<T>(owner.data, true)",
	} {
		t.Run(invalid, func(t *testing.T) {
			candidate := strings.Replace(text, "storage_element_address<T>(owner.data, 0)", invalid, 1)
			analysis, err := analyzeStorageBorrow(t, candidate)
			if err == nil && len(analysis.Analysis.Semantic.Diagnostics) == 0 && len(analysis.Analysis.Semantic.Unsupported) == 0 {
				t.Fatal("invalid private element address was admitted")
			}
		})
	}
}

func TestPrivateVectorStorageBorrowSourceElementAddressMIRMutations(t *testing.T) {
	text := strings.Replace(storageBorrowMutSource,
		"storage_borrow_mut<T>(owner, owner.data)",
		"storage_borrow_mut<T>(owner, storage_element_address<T>(owner.data, 0))", 1)
	analysis, err := analyzeStorageBorrow(t, text)
	if err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"shared result":  func(op *mir.Operation) { op.Type = hir.RawPointer(hir.TypeVoid, false) },
		"wrong element":  func(op *mir.Operation) { op.OwnerType = hir.TypeInvalid },
		"wrong index":    func(op *mir.Operation) { op.IndexType = hir.TypeU32 },
		"missing base":   func(op *mir.Operation) { op.Operands = op.Operands[1:] },
		"aliased result": func(op *mir.Operation) { op.DestinationLocal = op.Operands[0].Local },
		"forged scope":   func(op *mir.Operation) { op.UnsafeScope = hir.Span{} },
		"forged origin":  func(op *mir.Operation) { op.Entity = "forged" },
	} {
		t.Run(name, func(t *testing.T) {
			program, err := mir.Lower(analysis.Analysis.Semantic.HIR)
			if err != nil {
				t.Fatal(err)
			}
			changed := 0
			for fi := range program.Functions {
				for bi := range program.Functions[fi].Blocks {
					for oi := range program.Functions[fi].Blocks[bi].Operations {
						op := &program.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationStorageElementAddress {
							mutate(op)
							changed++
						}
					}
				}
			}
			if changed != 1 || len(mir.VerifyStorageTransfers(program)) == 0 {
				t.Fatalf("malformed element address admitted: changed=%d", changed)
			}
		})
	}
}

func TestPrivateVectorStorageBorrowSourceMutableMIR(t *testing.T) {
	analysis, err := analyzeStorageBorrow(t, storageBorrowMutSource)
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !compiled.Verified {
		t.Fatalf("mutable storage borrow MIR: %v", err)
	}
	program, err := mir.Lower(analysis.Analysis.Semantic.HIR)
	if err != nil || len(mir.VerifyStorageTransfers(program)) != 0 {
		t.Fatalf("mutable storage borrow lowering: %v %+v", err, mir.VerifyStorageTransfers(program))
	}
	if len(mir.Verify(program)) == 0 || len(mir.VerifyStorageQueries(program)) == 0 {
		t.Fatal("mutable storage borrow escaped private MIR admission")
	}
	count := 0
	for _, function := range program.Functions {
		for _, block := range function.Blocks {
			for _, operation := range block.Operations {
				if operation.Kind != mir.OperationStorageBorrowMut {
					continue
				}
				ownerElement, ownerMutable, ownerBorrowed := hir.BorrowElement(operation.OwnerType)
				if operation.Type != hir.MutableBorrow(hir.TypeI32) ||
					!ownerBorrowed || !ownerMutable || !strings.HasSuffix(string(ownerElement), "::main::Owner") ||
					operation.IndexType != hir.RawPointer(hir.TypeVoid, true) {
					t.Fatalf("mutable storage borrow lost operand types: %+v", operation)
				}
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("mutable storage borrow operation count = %d", count)
	}
	for name, mutate := range map[string]func(*mir.Operation){
		"shared result": func(op *mir.Operation) { op.Type = hir.SharedBorrow(hir.TypeI32) },
		"shared owner": func(op *mir.Operation) {
			owner, _, _ := hir.BorrowElement(op.OwnerType)
			op.OwnerType = hir.SharedBorrow(owner)
		},
		"shared address": func(op *mir.Operation) { op.IndexType = hir.RawPointer(hir.TypeVoid, false) },
		"forged origin":  func(op *mir.Operation) { op.Entity = "forged" },
	} {
		t.Run(name, func(t *testing.T) {
			copy, err := mir.Lower(analysis.Analysis.Semantic.HIR)
			if err != nil {
				t.Fatal(err)
			}
			changed := 0
			for fi := range copy.Functions {
				for bi := range copy.Functions[fi].Blocks {
					for oi := range copy.Functions[fi].Blocks[bi].Operations {
						op := &copy.Functions[fi].Blocks[bi].Operations[oi]
						if op.Kind == mir.OperationStorageBorrowMut {
							mutate(op)
							changed++
						}
					}
				}
			}
			if changed != 1 || len(mir.VerifyStorageTransfers(copy)) == 0 {
				t.Fatalf("mutable MIR mutation was accepted: %d operations", changed)
			}
		})
	}
}

func TestPrivateVectorStorageBorrowSourceMutableRejections(t *testing.T) {
	for name, test := range map[string]struct {
		from, to, code, unsupported string
	}{
		"shared owner":           {from: "fn ref mut T borrow_element<T>(ref mut Owner owner)", to: "fn ref mut T borrow_element<T>(ref Owner owner)", unsupported: "matching owner and void pointer mutability"},
		"shared address":         {from: "ptr mut void data", to: "ptr void data", unsupported: "matching owner and void pointer mutability"},
		"unsafe missing":         {from: "unsafe { storage_borrow_mut<T>(owner, owner.data) }", to: "storage_borrow_mut<T>(owner, owner.data)", code: "E5101"},
		"owner moved before use": {from: "observe(view); consume(owner);", to: "consume(owner); observe(view);", code: "E2105"},
		"exclusive alias":        {from: "observe(view); consume(owner);", to: "mutate(ref mut owner); observe(view); consume(owner);", code: "E2101"},
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(storageBorrowMutSource, test.from, test.to, 1)
			if text == storageBorrowMutSource {
				t.Fatal("mutable borrow rejection fixture unchanged")
			}
			analysis, err := analyzeStorageBorrow(t, text)
			result := analysis.Analysis.Semantic
			if err == nil && len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("invalid mutable storage borrow admitted")
			}
			if test.code != "" {
				for _, diagnostic := range result.Diagnostics {
					if diagnostic.Code == test.code {
						return
					}
				}
				t.Fatalf("missing %s: %v %+v %+v", test.code, err, result.Diagnostics, result.Unsupported)
			}
			for _, unsupported := range result.Unsupported {
				if strings.Contains(unsupported.Feature, test.unsupported) {
					return
				}
			}
			t.Fatalf("missing rejection %q: %v %+v %+v", test.unsupported, err, result.Diagnostics, result.Unsupported)
		})
	}
}

const storageBorrowMutSource = `module main;
resource Owner { ptr mut void data; cleanup {} }
fn ref mut T borrow_element<T>(ref mut Owner owner) { return unsafe { storage_borrow_mut<T>(owner, owner.data) }; }
fn void observe(ref mut i32 value) {}
fn void consume(Owner value) {}
fn void mutate(ref mut Owner value) {}
fn i32 main() {
    var Owner owner = Owner { .data = null };
    let ref mut i32 view = borrow_element<i32>(ref mut owner);
    observe(view); consume(owner);
    return 42;
}
`

const storageBorrowSource = `module main;
resource Owner { ptr mut void data; cleanup {} }
fn ref T borrow_element<T>(ref Owner owner) { return unsafe { storage_borrow<T>(owner, owner.data) }; }
fn ref i32 relay(ref Owner owner) { return borrow_element<i32>(owner); }
fn void observe(ref i32 value) {}
fn void consume(Owner value) {}
fn void mutate(ref mut Owner value) {}
fn i32 main() {
    var Owner owner = Owner { .data = null };
    let ref i32 view = relay(ref owner);
    observe(view); consume(owner);
    return 42;
}
`
