package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
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
	// Source preparation alone cannot publish an executable before the matching
	// MIR operation, verifier and backend have been implemented.
	compiled, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err == nil && compiled.Verified {
		t.Fatal("source-only borrow gained executable admission")
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
