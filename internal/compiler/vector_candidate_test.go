package compiler_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
)

func vectorCandidateProgram() string {
	return strings.NewReplacer("import storage as v;", "import std::vec as v;", "v::Items", "v::Vec", "v::empty_items", "v::empty", "v::length", "v::len").Replace(vectorSDKProgram)
}

func TestPrivateVectorSDKCandidateOwnership(t *testing.T) {
	for _, name := range []string{"cleanup", "clear", "reserve"} {
		t.Run(name, func(t *testing.T) {
			text := vectorCandidateProgram()
			text = strings.Replace(text, "assert(v::len<bytes::Buffer>(ref storage) == 0);", `assert(v::is_empty<bytes::Buffer>(ref storage));
    assert(v::capacity<bytes::Buffer>(ref storage) == 0);
    assert(switch (v::reserve<bytes::Buffer>(ref mut storage, 0)) { case .ok(count) : { count == 0 } case .err(_) : { false } });`, 1)
			probe := vectorCandidateProbe
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

const vectorCandidateProbe = `
int32_t fairyc_fixture_resource_fail(void) { vector_fail = 1; return 1; }
int32_t fairyc_fixture_resource_finished(void) {
    return vector_live == 0 && vector_allocations == 11 && vector_frees == 11 && vector_failures == 2 ? 1 : 0;
}
`
