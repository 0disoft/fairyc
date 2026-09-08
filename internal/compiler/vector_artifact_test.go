package compiler_test

import (
	"bytes"
	"encoding/json"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packageartifact"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestPrivateVectorStableArtifactRoundTrip(t *testing.T) {
	input, coordinates := vectorPackageInput(t)
	result, _, err := compiler.CheckStorageTransferPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("stable input: %v %+v", err, result.Semantic)
	}
	identities := map[hir.PackageID]packagegraph.InstanceID{1: {1}, 2: {2}}
	artifacts, err := packageartifact.ExtractAllStorageTransfers(result.MIR, identities, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	const limit = 8 << 20
	encodedByPackage := map[packagegraph.InstanceID][]byte{}
	var decoded []packageartifact.StableArtifact
	guarded := 0
	for _, artifact := range artifacts {
		stable, err := packageartifact.DescribeStorageTransferStableArtifact(artifact, limit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := packageartifact.EncodeStorageTransferStableArtifact(stable, limit)
		if err != nil {
			t.Fatal(err)
		}
		encodedByPackage[stable.Package] = encoded
		if _, err := packageartifact.DecodeStableArtifact(encoded, limit); err == nil {
			t.Fatal("ordinary reader accepted private stable envelope")
		}
		for name, data := range map[string][]byte{
			"whitespace": append(bytes.Clone(encoded), '\n'),
			"trailing":   append(bytes.Clone(encoded), []byte("{}")...),
			"unknown":    bytes.Replace(encoded, []byte(`{"schema":`), []byte(`{"unknown":true,"schema":`), 1),
			"schema":     bytes.Replace(encoded, []byte(packageartifact.StorageStableArtifactSchema), []byte(packageartifact.StableArtifactSchema), 1),
		} {
			if _, err := packageartifact.DecodeStorageTransferStableArtifact(data, limit); err == nil {
				t.Fatalf("private stable reader accepted %s bytes", name)
			}
		}
		if _, err := packageartifact.DecodeStorageTransferStableArtifact(encoded, uint64(len(encoded)-1)); err == nil {
			t.Fatal("private stable reader ignored byte bound")
		}
		stable, err = packageartifact.DecodeStorageTransferStableArtifact(encoded, limit)
		if err != nil {
			t.Fatal(err)
		}
		decoded = append(decoded, stable)
		for _, body := range stable.Functions {
			for _, block := range body.Function.Blocks {
				for _, operation := range block.Operations {
					if operation.Kind != mir.OperationStoragePut && operation.Kind != mir.OperationStorageTake && operation.Kind != mir.OperationStorageLayout {
						continue
					}
					guarded++
					if _, err := packageartifact.EncodeStableArtifact(stable, limit); err == nil {
						t.Fatal("ordinary writer accepted private stable operations")
					}
					forged := bytes.Replace(encoded, []byte(packageartifact.StorageStableArtifactSchema), []byte(packageartifact.StableArtifactSchema), 1)
					forged = bytes.Replace(forged, []byte(packageartifact.StorageStableArtifactVersion), []byte(packageartifact.StableArtifactVersion), 1)
					if _, err := packageartifact.DecodeStableArtifact(forged, limit); err == nil {
						t.Fatal("schema substitution admitted private operations")
					}
				}
			}
		}
	}
	if guarded == 0 {
		t.Fatal("fixture did not exercise private stable operations")
	}
	freshCoordinates := map[hir.PackageID]packagegraph.CoordinateID{101: coordinates[1], 205: coordinates[2]}
	freshIdentities := map[hir.PackageID]packagegraph.InstanceID{101: identities[1], 205: identities[2]}
	before, _ := json.Marshal(decoded)
	rebound, err := packageartifact.RebindStorageTransferStableArtifacts(decoded, freshCoordinates, 1000, 10000, 500)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(decoded)
	if !bytes.Equal(before, after) {
		t.Fatal("stable rebind modified input")
	}
	if _, err := packageartifact.RebindStableArtifacts(decoded, freshCoordinates, 1000, 10000, 500); err == nil {
		t.Fatal("ordinary rebind accepted storage operations")
	}
	if _, err := packageartifact.RebindStorageTransferStableArtifacts(decoded[:1], freshCoordinates, 1000, 10000, 500); err == nil {
		t.Fatal("stable rebind accepted incomplete package closure")
	}
	slices.Reverse(decoded)
	reordered, err := packageartifact.RebindStorageTransferStableArtifacts(decoded, freshCoordinates, 1000, 10000, 500)
	if err != nil || !reflect.DeepEqual(rebound, reordered) {
		t.Fatalf("package order changed stable rebind: %v", err)
	}
	reextracted, err := packageartifact.ExtractAllStorageTransfers(rebound, freshIdentities, freshCoordinates)
	if err != nil {
		t.Fatal(err)
	}
	for _, artifact := range reextracted {
		stable, err := packageartifact.DescribeStorageTransferStableArtifact(artifact, limit)
		if err != nil {
			t.Fatal(err)
		}
		encoded, err := packageartifact.EncodeStorageTransferStableArtifact(stable, limit)
		if err != nil || !bytes.Equal(encoded, encodedByPackage[stable.Package]) {
			t.Fatalf("fresh numbering changed stable bytes: %v", err)
		}
	}
	runVectorStorageMIRNativeLane(t, rebound, hir.QualifiedModulePath(101, "main")+"::main", vectorResourceProbe, true)
	// Structural decoding is not executable validation. A damaged ownership
	// operation must fail the final transfer verifier without publishing MIR.
	mutated := false
	for artifactIndex := range decoded {
		for bodyIndex := range decoded[artifactIndex].Functions {
			for blockIndex := range decoded[artifactIndex].Functions[bodyIndex].Function.Blocks {
				block := &decoded[artifactIndex].Functions[bodyIndex].Function.Blocks[blockIndex]
				for operationIndex := range block.Operations {
					operation := &block.Operations[operationIndex]
					if !mutated && operation.Kind == mir.OperationStoragePut {
						operation.UnsafeScope = hir.Span{}
						mutated = true
					}
				}
			}
		}
	}
	if !mutated {
		t.Fatal("ownership mutation did not activate")
	}
	invalid, err := packageartifact.RebindStorageTransferStableArtifacts(decoded, freshCoordinates, 1000, 10000, 500)
	if err == nil || len(invalid.Functions) != 0 {
		t.Fatalf("invalid stable transfer published executable MIR: %v", err)
	}
}

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
	cache, err := packageartifact.NewStorageTransferCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	interfaces := result.GenericProvenance.PackageInterfaces()
	for i, artifact := range artifacts {
		key := packagecache.GenericKey{byte(artifact.PackageID)} // Closed fixture keys, not production source keys.
		hash := interfaces[i].Generic
		if interfaces[i].Package != artifact.PackageID {
			t.Fatal("fixture interface order mismatch")
		}
		if cache.Get(key, hash).Status != packageartifact.LookupMiss {
			t.Fatal("cold transfer cache did not miss")
		}
		if err := cache.Put(key, hash, artifact); err != nil {
			t.Fatal(err)
		}
		loaded := cache.Get(key, hash)
		if loaded.Status != packageartifact.LookupHit {
			t.Fatal(loaded.Reason)
		}
		artifacts[i] = loaded.Artifact
	}
	if cache.Stats().VerifiedHits != 2 {
		t.Fatal("missing verified transfer cache hit")
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
