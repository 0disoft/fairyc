package compiler_test

import (
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
