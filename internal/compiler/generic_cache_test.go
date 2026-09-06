package compiler_test

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestGenericCacheKeyChangesWhenConsumerAddsTypeArguments(t *testing.T) {
	input, coordinates := genericPackageInput(t)
	first, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	input.Files[0] = parseCompilerFile(t, 1, "main.fy", "module main; import util as u; fn i32 main() { let bool extra = u::wrap<bool>(true); return u::wrap<i32>(42); }")
	second, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Specializations[1]) != 0 || len(first.Specializations[2]) != 1 || len(first.Specializations[3]) != 1 || len(second.Specializations[2]) != 2 || len(second.Specializations[3]) != 2 {
		t.Fatal("specializations not attributed to their defining package")
	}
	if first.Interfaces[2] != second.Interfaces[2] || first.OrdinaryInterfaces[2] != second.OrdinaryInterfaces[2] {
		t.Fatal("consumer requests changed producer source interface")
	}
	if !slices.IsSorted(second.Specializations[2]) {
		t.Fatal("noncanonical emitted specialization set")
	}
	key := func(analysis compiler.GenericAnalysis) packagecache.GenericKey {
		t.Helper()
		// Hold the producer source and complete environment fixture fixed: only
		// the consumer's requested generated functions changed in this test.
		base := packagecache.Inputs{
			CompilerBuild: "test-compiler", LanguageRevision: "test-language", MIRVersion: "test-mir",
			TargetSchema: "test-target", ABIProfile: "test-abi", ToolchainFingerprint: "test-toolchain",
			Backend: "test-backend", RuntimeABI: "test-runtime", StandardLibrary: packagegraph.InstanceID{1},
			Package: packagegraph.InstanceID{2}, Source: packagecontent.Digest{3}, CurrentBody: contractstatemir.StateBodyHash{4},
			CurrentInterface: analysis.OrdinaryInterfaces[2],
			Dependencies:     []packagecache.Dependency{{Coordinate: coordinates[3], Interface: analysis.OrdinaryInterfaces[3]}},
		}
		result, err := packagecache.ComputeGeneric(packagecache.GenericInputs{Base: base, CurrentInterface: analysis.Interfaces[2], Specializations: analysis.Specializations[2], Dependencies: []packagecache.GenericDependency{{Coordinate: coordinates[3], Interface: analysis.Interfaces[3]}}})
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	if key(first) == key(second) {
		t.Fatal("consumer's new type specialization reused producer cache key")
	}
}
