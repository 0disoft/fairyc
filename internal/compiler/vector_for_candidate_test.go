package compiler_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateVectorSDKCandidateLoopSourceGate(t *testing.T) {
	for _, sample := range []struct {
		name   string
		source string
	}{
		{name: "explicit owner borrow", source: `module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    for (let ref i32 item in ref values) { assert(*item == 0); }
    return 0;
}`},
		{name: "named immutable view", source: `module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 visit(ref v::Vec<i32> values) {
    for (let ref i32 item in values) { assert(*item == 0); }
    return 0;
}
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    return visit(ref values);
}`},
	} {
		t.Run(sample.name, func(t *testing.T) {
			input, coordinates := vectorSDKInputWithCandidate(t, sample.source, true)
			owners := []hir.PackageID{3}
			stable, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			if err != nil || len(stable.Analysis.Semantic.Unsupported) == 0 {
				t.Fatalf("stable source gate changed: %v %+v", err, stable.Analysis.Semantic)
			}
			candidate, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates, genericplan.DefaultPlanLimits(), owners)
			if err != nil || len(candidate.Analysis.Semantic.Diagnostics) != 0 ||
				len(candidate.Analysis.Semantic.Unsupported) != 0 ||
				candidate.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow {
				t.Fatalf("candidate source and Flow: %v %+v", err, candidate.Analysis.Semantic)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(candidate.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("candidate verified MIR: %v %+v", err, result)
			}
		})
	}
}

func TestPrivateVectorSDKCandidateLoopRejectsUnsafeSource(t *testing.T) {
	for _, sample := range []struct {
		name            string
		loop            string
		wantCode        string
		wantUnsupported bool
	}{
		{name: "wrong element", loop: "for (let ref u8 item in ref values) { assert(*item == 0); }", wantCode: "E1006"},
		{name: "mutable item", loop: "for (let ref mut i32 item in ref values) { assert(*item == 0); }", wantUnsupported: true},
		{name: "owner mutation", loop: "for (let ref i32 item in ref values) { values = v::empty<i32>(); assert(*item == 0); }", wantCode: "E2101"},
		{name: "owner move", loop: "for (let ref i32 item in ref values) { let v::Vec<i32> moved = values; assert(*item == 0); }", wantCode: "E2105"},
		{name: "scalar owner", loop: "var i32 scalar = 0; for (let ref i32 item in ref scalar) { assert(*item == 0); }", wantCode: "E1006"},
		{name: "computed iterable", loop: "for (let ref i32 item in v::empty<i32>()) { assert(*item == 0); }", wantUnsupported: true},
	} {
		t.Run(sample.name, func(t *testing.T) {
			client := `module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    ` + sample.loop + `
    return 0;
}`
			input, coordinates := vectorSDKInputWithCandidate(t, client, true)
			result, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates,
				genericplan.DefaultPlanLimits(), []hir.PackageID{3})
			if err != nil {
				t.Fatalf("source rejection escaped semantic analysis: %v", err)
			}
			semanticResult := result.Analysis.Semantic
			if sample.wantUnsupported {
				if len(semanticResult.Unsupported) == 0 {
					t.Fatalf("unsafe vector source was accepted: %+v", semanticResult)
				}
				return
			}
			for _, diagnostic := range semanticResult.Diagnostics {
				if diagnostic.Code == sample.wantCode {
					return
				}
			}
			t.Fatalf("missing %s rejection: diagnostics=%+v unsupported=%+v", sample.wantCode,
				semanticResult.Diagnostics, semanticResult.Unsupported)
		})
	}
}

func TestPrivateVectorSDKCandidateLoopRejectsLocalItemEscape(t *testing.T) {
	const source = `module main;
import std::vec as v;
import std::bytes as bytes;
fn ref i32 leak(ref i32 fallback) {
    var v::Vec<i32> values = v::empty<i32>();
    for (let ref i32 item in ref values) { return item; }
    return fallback;
}`
	input, coordinates := vectorSDKInputWithCandidate(t, source, true)
	result, err := compiler.AnalyzeSDKVectorCandidatePackages(input, coordinates,
		genericplan.DefaultPlanLimits(), []hir.PackageID{3})
	if err != nil {
		t.Fatal(err)
	}
	for _, diagnostic := range result.Analysis.Semantic.Diagnostics {
		if diagnostic.Code == "E2102" && len(diagnostic.Related) != 0 {
			return
		}
	}
	t.Fatalf("local vector item escaped without a borrow error: %+v %+v",
		result.Analysis.Semantic.Diagnostics, result.Analysis.Semantic.Unsupported)
}

const vectorForNativeProbe = `
int32_t fairyc_fixture_vector_clean(void) {
    return vector_live == 0 && vector_allocations == vector_frees ? 1 : 0;
}
`

func TestPrivateVectorSDKCandidateLoopNativeParity(t *testing.T) {
	const header = `module main;
import std::vec as v;
import std::bytes as bytes;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn i32 exercise() {
    var v::Vec<i32> values = v::empty<i32>();
`
	const footer = `
}
fn i32 main() {
    let i32 result = exercise();
    assert(unsafe { clean() } == 1);
    return result;
}`
	const pushes = `
    assert(switch (v::push<i32>(ref mut values, 1)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (v::push<i32>(ref mut values, 2)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (v::push<i32>(ref mut values, 3)) { case .ok(_) : { true } case .err(_) : { false } });
    assert(switch (v::push<i32>(ref mut values, 4)) { case .ok(_) : { true } case .err(_) : { false } });
    var i32 sum = 0;
`
	const body = `
        if (*item == 2) { continue; }
        sum += *item;
        if (*item == 3) { break; }
`
	for _, sample := range []struct {
		name     string
		setup    string
		body     string
		returned string
		expected int32
	}{
		{name: "empty", body: "        assert(false);", returned: "    return 42;", expected: 42},
		{name: "ordered control", setup: pushes, body: body, returned: "    return 40 + sum;", expected: 44},
		{name: "early return cleanup", setup: pushes, body: "        if (*item == 3) { return 43; }", returned: "    return 9;", expected: 43},
	} {
		t.Run(sample.name, func(t *testing.T) {
			for _, form := range []struct {
				name      string
				candidate bool
				loop      string
			}{
				{name: "vector for", candidate: true,
					loop: "    for (let ref i32 item in ref values) {\n" + sample.body + "\n    }"},
				{name: "explicit index", loop: `    let usize bound = v::len<i32>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref i32 item = v::at<i32>(ref values, index);
` + sample.body + `
    }`},
			} {
				t.Run(form.name, func(t *testing.T) {
					client := header + sample.setup + form.loop + "\n" + sample.returned + footer
					input, coordinates := vectorSDKInputWithCandidate(t, client, true)
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
					if err != nil || analysis.Analysis.Semantic.CompletedThrough != semantic.PhaseFlow {
						t.Fatalf("native source analysis: %v diagnostics=%+v unsupported=%+v", err,
							analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
					}
					result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
					if err != nil || !result.Verified {
						t.Fatalf("native MIR: %v %+v", err, result.Semantic.Diagnostics)
					}
					runVectorStorageMIRNativeExit(t, result.MIR,
						hir.QualifiedModulePath(1, "main")+"::main", vectorForNativeProbe, true, sample.expected)
				})
			}
		})
	}
}

func TestPrivateVectorSDKCandidateLoopPropagationCleanup(t *testing.T) {
	const ownedProbe = `
int32_t fairyc_fixture_vector_clean(void) {
    return vector_live == 0 && vector_allocations >= 3 && vector_allocations == vector_frees ? 1 : 0;
}
`
	const header = `module main;
import std::vec as v;
import std::bytes as bytes;
@link_name("fairyc_fixture_vector_clean") unsafe extern C fn i32 clean();
fn bytes::Buffer make(ref u8[] text) {
    return switch (bytes::copy(text)) {
        case .ok(value): { value }
        case .err(_): { assert(false); bytes::empty() }
    };
}
fn Result<i32, i32> step(u8 byte) {
    if (byte == 66) { return .err(5); }
    return .ok(1);
}
fn Result<i32, i32> route() {
    var v::Vec<bytes::Buffer> values = v::empty<bytes::Buffer>();
    assert(switch (v::push<bytes::Buffer>(ref mut values, make(b"A"[..]))) { case .ok(_): { true } case .err(_): { false } });
    assert(switch (v::push<bytes::Buffer>(ref mut values, make(b"B"[..]))) { case .ok(_): { true } case .err(_): { false } });
`
	const footer = `
    return .ok(0);
}
fn i32 main() {
    let Result<i32, i32> result = route();
    assert(switch (result) { case .ok(_): { false } case .err(code): { code == 5 } });
    assert(unsafe { clean() } == 1);
    return 42;
}`
	for _, form := range []struct {
		name      string
		candidate bool
		loop      string
	}{
		{name: "vector for", candidate: true, loop: `    for (let ref bytes::Buffer item in ref values) {
        let ref u8[] view = bytes::view(item);
        let i32 value = step(view[0])?;
    }`},
		{name: "explicit index", loop: `    let usize bound = v::len<bytes::Buffer>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref bytes::Buffer item = v::at<bytes::Buffer>(ref values, index);
        let ref u8[] view = bytes::view(item);
        let i32 value = step(view[0])?;
    }`},
	} {
		t.Run(form.name, func(t *testing.T) {
			input, coordinates := vectorSDKInputWithCandidate(t, header+form.loop+footer, true)
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
				t.Fatalf("propagation source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("propagation verified MIR: %v diagnostics=%+v", err, result.Semantic.Diagnostics)
			}
			runVectorStorageMIRNativeExit(t, result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", ownedProbe, true, 42)
		})
	}
}

func TestPrivateVectorSDKCandidateLoopRegionFunctionExits(t *testing.T) {
	const probe = `
static const int32_t expected_drops[] = {2, 1, 2, 1};
static size_t drop_position = 0;
static int drop_mismatch = 0;
int32_t fairyc_fixture_region_record(int32_t id) {
    if (drop_position >= 4 || expected_drops[drop_position] != id) drop_mismatch = 1;
    drop_position++;
    return id;
}
int32_t fairyc_fixture_vector_clean(void) {
    return !drop_mismatch && drop_position == 4 && vector_live == 0 &&
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
fn Result<i32, i32> step(i32 value) { return .err(value - 61); }
fn Result<i32, i32> route(alloc::Arena owned, bool propagate) {
    var alloc::Arena arena = owned;
    var v::Vec<i32> values = v::empty<i32>();
    assert(switch (v::push<i32>(ref mut values, 66)) { case .ok(_): { true } case .err(_): { false } });
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
	const body = `
        let Mark body = Mark { .id = 1 };
        region scratch in ref mut arena {
            let Mark inner = Mark { .id = 2 };
            assert(switch (scratch.place(*item)) {
                case .ok(stored): { *stored == 66 }
                case .err(_): { false }
            });
            if (propagate) { let i32 value = step(*item)?; }
            return .ok(42);
        }
`
	for _, form := range []struct {
		name      string
		candidate bool
		loop      string
	}{
		{name: "vector for", candidate: true,
			loop: "    for (let ref i32 item in ref values) {\n" + body + "\n    }"},
		{name: "explicit index", loop: `    let usize bound = v::len<i32>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref i32 item = v::at<i32>(ref values, index);
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
				t.Fatalf("Region exit source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("Region exit verified MIR: %v diagnostics=%+v", err, result.Semantic.Diagnostics)
			}
			runVectorStorageMIRNativeExit(t, result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", probe, true, 42)
		})
	}
}

func TestPrivateVectorSDKCandidateBufferLoopNativeParity(t *testing.T) {
	const anchor = "    let bytes::Buffer last ="
	for _, form := range []struct {
		name      string
		candidate bool
		loop      string
	}{
		{name: "vector for", candidate: true, loop: `    var usize total = 0;
    for (let ref bytes::Buffer item in ref storage) {
        let ref u8[] view = bytes::view(item);
        total += view.len;
    }
    assert(total == 45);
`},
		{name: "explicit index", loop: `    var usize total = 0;
    let usize bound = v::len<bytes::Buffer>(ref storage);
    for (var usize index = 0; index < bound; index += 1) {
        let ref bytes::Buffer item = v::at<bytes::Buffer>(ref storage, index);
        let ref u8[] view = bytes::view(item);
        total += view.len;
    }
    assert(total == 45);
`},
	} {
		t.Run(form.name, func(t *testing.T) {
			client := strings.Replace(vectorCandidateProgram(), anchor, form.loop+anchor, 1)
			if client == vectorCandidateProgram() {
				t.Fatal("vector candidate fixture anchor missing")
			}
			input, coordinates := vectorSDKInputWithCandidate(t, client, true)
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
				t.Fatalf("buffer loop source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("buffer loop verified MIR: %v %+v", err, result.Semantic.Diagnostics)
			}
			runVectorStorageMIRNativeLane(t, result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", vectorCandidateProbe, true)
		})
	}
}
