package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func vectorCandidateProgram() string {
	return strings.NewReplacer("import storage as v;", "import std::vec as v;", "v::Items", "v::Vec", "v::empty_items", "v::empty", "v::length", "v::len").Replace(vectorSDKProgram)
}

func TestPrivateVectorSDKCandidateOwnership(t *testing.T) {
	for _, name := range []string{"cleanup", "clear", "reserve", "borrow"} {
		t.Run(name, func(t *testing.T) {
			text := vectorCandidateProgram()
			text = strings.Replace(text, "assert(v::len<bytes::Buffer>(ref storage) == 0);", `assert(v::is_empty<bytes::Buffer>(ref storage));
    assert(v::capacity<bytes::Buffer>(ref storage) == 0);
    assert(switch (v::reserve<bytes::Buffer>(ref mut storage, 0)) { case .ok(count) : { count == 0 } case .err(_) : { false } });`, 1)
			probe := vectorCandidateProbe
			if name == "borrow" {
				text = strings.Replace(text, "    let bytes::Buffer last =", `    let usize capacity_before = v::capacity<bytes::Buffer>(ref storage);
    let ref bytes::Buffer first = v::at<bytes::Buffer>(ref storage, 0);
    let ref bytes::Buffer ninth = v::at<bytes::Buffer>(ref storage, 8);
    assert(valid(first));
    assert(valid(ninth));
    assert(v::len<bytes::Buffer>(ref storage) == 9);
    assert(v::capacity<bytes::Buffer>(ref storage) == capacity_before);
    let bytes::Buffer last =`, 1)
			}
			if name == "reserve" {
				text = strings.Replace(text, "    let bytes::Buffer last =", `    let usize before = v::capacity<bytes::Buffer>(ref storage);
    assert(unsafe { fail_next() } == 1);
    assert(switch (v::reserve<bytes::Buffer>(ref mut storage, 17)) { case .ok(_) : { false } case .err(status) : { status == 1 } });
    assert(v::capacity<bytes::Buffer>(ref storage) == before);
    assert(v::len<bytes::Buffer>(ref storage) == 9);
    assert(switch (v::reserve<bytes::Buffer>(ref mut storage, 17)) { case .ok(count) : { count >= 17 } case .err(_) : { false } });
    assert(v::len<bytes::Buffer>(ref storage) == 9);
    let bytes::Buffer last =`, 1)
				probe = strings.NewReplacer("== 11", "== 12", "vector_failures == 2", "vector_failures == 3").Replace(probe)
			}
			if name == "clear" {
				text = strings.Replace(text, "let v::Vec<bytes::Buffer> moved = storage;", `let usize capacity = v::capacity<bytes::Buffer>(ref storage);
    assert(v::clear<bytes::Buffer>(ref mut storage) == 8);
    assert(v::is_empty<bytes::Buffer>(ref storage));
    assert(v::capacity<bytes::Buffer>(ref storage) == capacity);
    assert(v::clear<bytes::Buffer>(ref mut storage) == 0);
    let v::Vec<bytes::Buffer> moved = storage;`, 1)
			}
			input, coordinates := vectorSDKInputWithCandidate(t, text, true)
			before, _ := json.Marshal(input)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err != nil || !result.Verified {
				t.Fatalf("candidate SDK vector: %v %+v", err, result.Semantic.Diagnostics)
			}
			after, _ := json.Marshal(input)
			if string(before) != string(after) {
				t.Fatal("candidate source mutated")
			}
			runVectorStorageMIRNativeLane(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", probe, true)
		})
	}
}

func TestPrivateVectorSDKCandidateOwnedStorage(t *testing.T) {
	client := strings.Replace(vectorCandidateProgram(), "    let bytes::Buffer last =", "    assert(valid(v::at<bytes::Buffer>(ref storage, 0)));\n    let bytes::Buffer last =", 1)
	input, coordinates := vectorSDKInputWithCandidate(t, client, true)
	analysis, err := compiler.AnalyzeOwnedStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
	if err != nil || !result.Verified {
		t.Fatalf("owner-restricted SDK: %v %+v", err, result.Semantic.Diagnostics)
	}
	runVectorStorageMIRNativeLane(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", vectorCandidateProbe, true)
	client = strings.Replace(client, "return 41;", "return unsafe { storage_take<i32>(null) };", 1)
	input, coordinates = vectorSDKInputWithCandidate(t, client, true)
	analysis, err = compiler.AnalyzeOwnedStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err == nil && len(analysis.Analysis.Semantic.Diagnostics) == 0 && len(analysis.Analysis.Semantic.Unsupported) == 0 {
		t.Fatal("SDK consumer gained private storage authority")
	}
}

func TestPrivateVectorSDKCandidateBorrowBounds(t *testing.T) {
	code, ok := cbackend.TrapCode("assertion-failed")
	if !ok {
		t.Fatal("assertion trap code missing")
	}
	for name, setup := range map[string]string{
		"empty":    "var v::Vec<bytes::Buffer> storage = v::empty<bytes::Buffer>();",
		"past-end": "var v::Vec<bytes::Buffer> storage = v::empty<bytes::Buffer>(); assert(retry(ref mut storage, make()));",
	} {
		t.Run(name, func(t *testing.T) {
			text := vectorCandidateProgram()
			start := strings.Index(text, "fn i32 exercise()")
			if start < 0 {
				t.Fatal("missing exercise")
			}
			text = text[:start] + `@link_name("fairyc_fixture_borrow_bad_index") unsafe extern C fn usize bad_index();
fn i32 exercise() { ` + setup + " let ref bytes::Buffer missing = v::at<bytes::Buffer>(ref storage, unsafe { bad_index() }); assert(valid(missing)); return 41; } fn i32 main() { return exercise(); }"
			input, coordinates := vectorSDKInputWithCandidate(t, text, true)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err != nil || !result.Verified {
				t.Fatalf("bounds fixture: %v %+v", err, result.Semantic.Diagnostics)
			}
			index := "0"
			if name == "past-end" {
				index = "1"
			}
			probe := vectorCandidateProbe + "\nsize_t fairyc_fixture_borrow_bad_index(void) { return " + index + "; }\n"
			runVectorStorageMIRNativeExit(t, result.MIR, hir.QualifiedModulePath(1, "main")+"::main", probe, true, int32(192+code))
		})
	}
}

func TestPrivateVectorSDKCandidateEncapsulation(t *testing.T) {
	for name, replacement := range map[string][2]string{
		"private container field": {"assert(valid(ref last));", "assert(storage.length == 8);"},
		"private raw operation":   {"assert(valid(ref last));", "let u32 status = unsafe { v::release_raw(null) };"},
		"foreign construction":    {"var v::Vec<bytes::Buffer> storage = v::empty<bytes::Buffer>();", "var v::Vec<bytes::Buffer> storage = v::Vec<bytes::Buffer> { .data = null, .bytes = 0, .alignment = 0, .length = 0, .element_size = 32, .element_alignment = 8 };"},
	} {
		t.Run(name, func(t *testing.T) {
			original := vectorCandidateProgram()
			text := strings.Replace(original, replacement[0], replacement[1], 1)
			if text == original {
				t.Fatal("rejection fixture did not change")
			}
			input, coordinates := vectorSDKInputWithCandidate(t, text, true)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("candidate encapsulation violation admitted")
			}
		})
	}
}

func TestPrivateVectorSDKCandidateBorrowInvalidation(t *testing.T) {
	for name, mutation := range map[string]string{
		"clear":   "let usize removed = v::clear<bytes::Buffer>(ref mut storage);",
		"reserve": "let Result<usize, u32> grown = v::reserve<bytes::Buffer>(ref mut storage, 17);",
		"pop":     "let Option<bytes::Buffer> removed = v::pop<bytes::Buffer>(ref mut storage);",
		"move":    "var v::Vec<bytes::Buffer> moved_early = storage;",
	} {
		t.Run(name, func(t *testing.T) {
			text := strings.Replace(vectorCandidateProgram(), "    let bytes::Buffer last =",
				"    let ref bytes::Buffer borrowed = v::at<bytes::Buffer>(ref storage, 0);\n    "+mutation+"\n    assert(valid(borrowed));\n    let bytes::Buffer last =", 1)
			if name == "move" {
				before, after, found := strings.Cut(text, "assert(valid(borrowed));")
				if !found {
					t.Fatal("missing borrowed use")
				}
				text = before + "assert(valid(borrowed));" + strings.ReplaceAll(after, "storage", "moved_early")
			}
			input, coordinates := vectorSDKInputWithCandidate(t, text, true)
			result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
			if err == nil && result.Verified {
				t.Fatal("vector invalidation with live element borrow admitted")
			}
			found := false
			want := "E2101"
			if name == "move" {
				want = "E2105"
			}
			for _, diagnostic := range result.Semantic.Diagnostics {
				if diagnostic.Code == want {
					found = true
				}
			}
			if !found {
				t.Fatalf("missing borrow-conflict diagnostic: %v %+v", err, result.Semantic.Diagnostics)
			}
		})
	}
}

const vectorCandidateProbe = `
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    return vector_live == 0 && vector_allocations == 11 && vector_frees == 11 && vector_failures == 2 ? 1 : 0;
}
`
