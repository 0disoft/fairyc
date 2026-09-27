package compiler_test

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/target"
)

// Cross-target emission does not replace the still-missing Linux native run.
// It checks that the private source form needs no extra Linux runtime symbol
// or generated function compared with the equivalent explicit-index loop.
func TestPrivateVectorSDKCandidateLinuxVectorLoopEmission(t *testing.T) {
	const header = `module main;
import std::vec as v;
import std::bytes as bytes;
fn i32 main() {
    var v::Vec<i32> values = v::empty<i32>();
    assert(switch (v::push<i32>(ref mut values, 7)) {
        case .ok(_): { true }
        case .err(_): { false }
    });
    var i32 sum = 0;
`
	const body = `        sum += *item;`
	const footer = `    return sum;
}`
	var units [2]cbackend.Unit
	for index, form := range []struct {
		name, loop string
		candidate  bool
	}{
		{"explicit index", `    let usize bound = v::len<i32>(ref values);
    for (var usize index = 0; index < bound; index += 1) {
        let ref i32 item = v::at<i32>(ref values, index);
` + body + `
    }
`, false},
		{"vector for", `    for (let ref i32 item in ref values) {
` + body + `
    }
`, true},
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
				t.Fatalf("Linux vector source and Flow: %v diagnostics=%+v unsupported=%+v", err,
					analysis.Analysis.Semantic.Diagnostics, analysis.Analysis.Semantic.Unsupported)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("Linux vector verified MIR: %v diagnostics=%+v", err, result.Semantic.Diagnostics)
			}
			unit, err := cbackend.EmitStorageTransfers(result.MIR,
				hir.QualifiedModulePath(1, "main")+"::main", cbackend.Stage0Profile())
			if err != nil || unit.Target != string(target.X8664LinuxGNU) || len(unit.Source) == 0 {
				t.Fatalf("Linux vector C emission: %v target=%q bytes=%d", err, unit.Target, len(unit.Source))
			}
			units[index] = unit
		})
	}
	if len(units[0].Functions) != len(units[1].Functions) ||
		!slices.Equal(units[0].ForeignSymbols, units[1].ForeignSymbols) {
		t.Fatalf("Linux vector source added generated functions or runtime symbols: indexed=%d/%v candidate=%d/%v",
			len(units[0].Functions), units[0].ForeignSymbols,
			len(units[1].Functions), units[1].ForeignSymbols)
	}
}
