package compiler_test

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestPrivateVectorPackageArtifactReassembly(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("artifact input: %v %+v", err, result.Semantic)
	}
	// Exact fixture source-instance identities; production callers use the loader.
	identities := map[hir.PackageID]packagegraph.InstanceID{1: {1}, 2: {2}}
	artifacts, err := packageartifact.ExtractAllStorageTransfers(result.MIR, identities, coordinates)
	if err != nil || len(artifacts) != 2 {
		t.Fatalf("extract transfers: %v", err)
	}
	before, _ := json.Marshal(artifacts)
	for _, artifact := range artifacts {
		if err := packageartifact.VerifyStorageTransfers(artifact); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := packageartifact.ExtractAll(result.MIR, identities, coordinates); err == nil {
		t.Fatal("ordinary extraction admitted transfers")
	}
	if _, err := packageartifact.Reassemble(artifacts); err == nil {
		t.Fatal("ordinary assembly admitted transfers")
	}
	rebuilt, err := packageartifact.ReassembleStorageTransfers(artifacts)
	if err != nil || len(mir.VerifyStorageTransfers(rebuilt)) != 0 {
		t.Fatalf("reassemble: %v", err)
	}
	after, _ := json.Marshal(artifacts)
	if string(before) != string(after) {
		t.Fatal("assembly modified package inputs")
	}
	oldSymbols := map[hir.SymbolID]bool{}
	for _, function := range result.MIR.Functions {
		oldSymbols[function.Symbol] = true
	}
	for _, function := range rebuilt.Functions {
		if oldSymbols[function.Symbol] {
			t.Fatal("assembly did not allocate fresh function symbols")
		}
	}
	if _, err := packageartifact.ReassembleStorageTransfers(artifacts[:1]); err == nil {
		t.Fatal("missing package admitted")
	}
	if _, err := packageartifact.ReassembleStorageTransfers(append(slices.Clone(artifacts), artifacts[0])); err == nil {
		t.Fatal("duplicate package admitted")
	}
	bad := artifacts[0]
	bad.Package = packagegraph.InstanceID{99}
	if err := packageartifact.VerifyStorageTransfers(bad); err == nil {
		t.Fatal("inconsistent package owner admitted")
	}
	slices.Reverse(artifacts)
	reordered, err := packageartifact.ReassembleStorageTransfers(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	first, _ := json.Marshal(rebuilt)
	second, _ := json.Marshal(reordered)
	if string(first) != string(second) {
		t.Fatal("artifact input order changed reassembly")
	}
	runVectorStorageMIRNativeLane(t, rebuilt, hir.QualifiedModulePath(1, "main")+"::main", vectorResourceProbe, true)
}
