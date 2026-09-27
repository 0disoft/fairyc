package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorSDKCandidateResourceLoopRegionExits(t *testing.T) {
	const probe = `
static const int32_t expected_drops[] = {3, 2, 1, 3, 2, 1};
static size_t drop_position = 0;
static int drop_mismatch = 0;
int32_t fairyc_fixture_region_record(int32_t id) {
    if (drop_position >= 6 || expected_drops[drop_position] != id) drop_mismatch = 1;
    drop_position++;
    return id;
}
int32_t fairyc_fixture_vector_clean(void) {
    return !drop_mismatch && drop_position == 6 && vector_live == 0 &&
        vector_allocations >= 4 && vector_allocations == vector_frees ? 1 : 0;
}
`
	const header = `module main;
import std::vec as v;
import std::bytes as bytes;
import std::alloc as alloc;
@link_name("fairyc_fixture_region_record") unsafe extern C fn i32 record(i32 id);
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn i32 observe(i32 id) { return unsafe { record(id) }; }
resource Mark { i32 id; cleanup { let i32 logged = observe(this.id); } }
fn i32 number(ref Mark item) { return item.id; }
fn Result<i32, i32> step(i32 value) { return .err(value - 61); }
fn Result<i32, i32> route(alloc::Arena owned, bool propagate) {
    var alloc::Arena arena = owned;
    var v::Vec<Mark> values = v::empty<Mark>();
    assert(switch (v::push<Mark>(ref mut values, Mark { .id = 1 })) { case .ok(_): { true } case .err(_): { false } });
    assert(switch (v::push<Mark>(ref mut values, Mark { .id = 2 })) { case .ok(_): { true } case .err(_): { false } });
`
	const body = `
        let i32 id = number(item);
        if (id == 1) { continue; }
        assert(id == 2);
        region scratch in ref mut arena {
            let Mark inner = Mark { .id = 3 };
            discard scratch.place(7);
            if (propagate) { let i32 value = step(66)?; }
            return .ok(42);
        }
`
	const footer = `
    return .ok(0);
}
fn i32 run(bool propagate) {
    return switch (alloc::create()) {
        case .ok(arena): {
            let Result<i32, i32> result = route(arena, propagate);
            switch (result) {
                case .ok(value): { value }
                case .err(code): { code + 37 }
            }
        }
        case .err(_): { 1 }
    };
}
fn i32 main() {
    assert(run(false) == 42);
    assert(run(true) == 42);
    assert(unsafe { clean() } == 1);
    return 42;
}`
	for _, form := range []struct {
		name      string
		candidate bool
		loop      string
	}{
		{name: "vector for", candidate: true,
			loop: "    for (let ref Mark item in ref values) {\n" + body + "\n    }"},
		{name: "explicit index", loop: `    let usize bound = v::len<Mark>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref Mark item = v::at<Mark>(ref values, index);
` + body + `
    }`},
	} {
		t.Run(form.name, func(t *testing.T) {
			input, coordinates := vectorSDKInputWithCandidate(t, header+form.loop+footer, true)
			input.Imports = append(input.Imports, semantic.PackageImport{
				FromPackage: 1, ImportPath: "std::alloc", ToPackage: 3,
			})
			owners := []hir.PackageID{3}
			var analysis compiler.GenericAnalysis
			var err error
			if form.candidate {
				analysis, err = compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates,
					genericplan.DefaultPlanLimits(), owners)
			} else {
				analysis, err = compiler.AnalyzeSDKStoragePackages(input, coordinates,
					genericplan.DefaultPlanLimits(), owners)
			}
			if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow ||
				len(analysis.Analysis.Semantic.Diagnostics) != 0 || len(analysis.Analysis.Semantic.Unsupported) != 0 {
				t.Fatalf("resource Region source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("resource Region verified MIR: %v diagnostics=%+v", err, result.Semantic.Diagnostics)
			}
			runVectorStorageMIRNativeExit(t, result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", probe, true, 42)
		})
	}
}
