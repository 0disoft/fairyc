package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagecontent"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func genericCachedPackageResult(t *testing.T) compiler.Result {
	t.Helper()
	input, coordinates := genericPackageInput(t)
	analysis, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	compiled, err := compiler.LowerAnalyzed(analysis.Analysis)
	if err != nil || !compiled.Verified {
		t.Fatalf("compile: %v", err)
	}
	// Fixed source-instance identities belong to this closed three-source fixture.
	identities := map[hir.PackageID]packagegraph.InstanceID{1: {1}, 2: {2}, 3: {3}}
	artifacts, err := packageartifact.ExtractAll(compiled.MIR, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := packageartifact.NewGenericCache(packagecache.Limits{MaxEntries: 3, MaxBytes: 16 << 20}, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	keys := map[hir.PackageID]packagecache.GenericKey{}
	for _, artifact := range artifacts {
		id := artifact.PackageID
		base := packagecache.Inputs{
			CompilerBuild: "test-process:" + cache.Identity(), LanguageRevision: "test-language", MIRVersion: "test-mir",
			TargetSchema: "test-target", ABIProfile: "test-abi", ToolchainFingerprint: "test-toolchain", Backend: "test-backend", RuntimeABI: "test-runtime",
			StandardLibrary: packagegraph.InstanceID{9}, Package: identities[id], Source: packagecontent.Digest{byte(id)},
			CurrentBody: contractstatemir.StateBodyHash{1}, CurrentInterface: analysis.OrdinaryInterfaces[id],
		}
		var dependencies []packagecache.GenericDependency
		for _, imported := range input.Imports {
			if imported.FromPackage != id {
				continue
			}
			base.Dependencies = append(base.Dependencies, packagecache.Dependency{Coordinate: coordinates[imported.ToPackage], Interface: analysis.OrdinaryInterfaces[imported.ToPackage]})
			dependencies = append(dependencies, packagecache.GenericDependency{Coordinate: coordinates[imported.ToPackage], Interface: analysis.Interfaces[imported.ToPackage]})
		}
		key, err := packagecache.ComputeGeneric(packagecache.GenericInputs{Base: base, CurrentInterface: analysis.Interfaces[id], Dependencies: dependencies, Specializations: analysis.Specializations[id]})
		if err != nil {
			t.Fatal(err)
		}
		keys[id] = key
		if cache.Get(key, analysis.Interfaces[id]).Status != packageartifact.LookupMiss {
			t.Fatal("cold generic cache did not miss")
		}
		if err := cache.Put(key, analysis.Interfaces[id], artifact); err != nil {
			t.Fatal(err)
		}
	}
	var retained []packageartifact.Artifact
	for _, artifact := range artifacts {
		lookup := cache.Get(keys[artifact.PackageID], analysis.Interfaces[artifact.PackageID])
		if lookup.Status != packageartifact.LookupHit {
			t.Fatalf("cached artifact: %s", lookup.Reason)
		}
		retained = append(retained, lookup.Artifact)
	}
	if cache.Stats().VerifiedHits != 3 {
		t.Fatal("not every source package was loaded from cache")
	}
	compiled.MIR, err = packageartifact.Reassemble(retained)
	if err != nil {
		t.Fatal(err)
	}
	return compiled
}

func TestGenericCachedMIRReassemblesAndExecutes(t *testing.T) {
	result := genericCachedPackageResult(t)
	outcome, err := interpreter.Execute(result.MIR, hir.QualifiedModulePath(1, "main")+"::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("cached execution: %+v %v", outcome, err)
	}
}
