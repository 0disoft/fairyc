package packageartifact

import (
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableCacheRoundTripsAcrossIndependentAdapters(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
	stateBody := stableCacheStateBody(t, artifact)
	limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	root := t.TempDir()
	writer, err := NewStableCache(root, limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := NewStableCache(root, limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if lookup := reader.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupMiss {
		t.Fatalf("cold lookup = %+v", lookup)
	}
	if err := writer.Put(key, outputInterface, artifact, stateBody); err != nil {
		t.Fatal(err)
	}
	lookup := reader.Get(key, outputInterface, artifact.Package, stateBody.Hash)
	if lookup.Status != LookupHit {
		t.Fatalf("warm lookup = %+v", lookup)
	}
	if err := ValidateStableCompilationArtifact(lookup.Compilation); err != nil {
		t.Fatalf("returned stable compilation: %v", err)
	}
	if len(lookup.Compilation.StateBody.Transactions) != 1 || len(lookup.Compilation.StateBody.Transactions[0].Operations) != 1 {
		t.Fatalf("returned compilation lost nonempty state operations: %#v", lookup.Compilation.StateBody)
	}
	lookup.Compilation.MIR.Functions[0].Function.Entry = 999
	lookup.Compilation.StateBody.Hash[0] ^= 0xff
	again := writer.Get(key, outputInterface, artifact.Package, stateBody.Hash)
	if again.Status != LookupHit || again.Compilation.MIR.Functions[0].Function.Entry == 999 || again.Compilation.StateBody.Hash != stateBody.Hash {
		t.Fatalf("stable cache payload mutated through caller: %+v", again)
	}
	stats, err := reader.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.VerifiedHits != 1 || stats.Misses != 1 || stats.Rejected != 0 || stats.Store.Entries != 1 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestStableCacheQuarantinesInterfacePackageAndPayloadContradictions(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
	stateBody := stableCacheStateBody(t, artifact)
	limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	cache, err := NewStableCache(t.TempDir(), limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(key, outputInterface, artifact, stateBody); err != nil {
		t.Fatal(err)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, contractstatemir.StateBodyHash{}); lookup.Status != LookupRejected {
		t.Fatalf("missing expected body hash = %+v", lookup)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupHit {
		t.Fatalf("missing expected hash disturbed valid record = %+v", lookup)
	}
	otherInterface := outputInterface
	otherInterface[len(otherInterface)-1] ^= 1
	if lookup := cache.Get(key, otherInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupRejected {
		t.Fatalf("interface mismatch = %+v", lookup)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupMiss {
		t.Fatalf("interface mismatch was not quarantined: %+v", lookup)
	}

	if err := cache.Put(key, outputInterface, artifact, stateBody); err != nil {
		t.Fatal(err)
	}
	otherPackage := artifact.Package
	otherPackage[len(otherPackage)-1] ^= 1
	if lookup := cache.Get(key, outputInterface, otherPackage, stateBody.Hash); lookup.Status != LookupRejected {
		t.Fatalf("package mismatch = %+v", lookup)
	}

	if err := cache.Put(key, outputInterface, artifact, stateBody); err != nil {
		t.Fatal(err)
	}
	otherBody := stateBody.Hash
	otherBody[0] ^= 1
	if lookup := cache.Get(key, outputInterface, artifact.Package, otherBody); lookup.Status != LookupRejected {
		t.Fatalf("body mismatch = %+v", lookup)
	}

	stableMIR, err := DescribeStableArtifact(artifact, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	oldMIRBytes, err := EncodeStableArtifact(stableMIR, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: outputInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: oldMIRBytes}},
	}); err != nil {
		t.Fatal(err)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupRejected {
		t.Fatalf("legacy MIR-only record = %+v", lookup)
	}

	compilation, err := DescribeStableCompilationArtifact(artifact, stateBody, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableCompilationArtifact(compilation, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	encoded[0] ^= 1
	if err := cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: outputInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactCompilation, Data: encoded}},
	}); err != nil {
		t.Fatal(err)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupRejected {
		t.Fatalf("malformed payload = %+v", lookup)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package, stateBody.Hash); lookup.Status != LookupMiss {
		t.Fatalf("malformed payload was not quarantined: %+v", lookup)
	}
	stats, err := cache.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rejected != 6 || stats.Misses != 2 || stats.VerifiedHits != 1 || stats.Store.Entries != 0 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestStableCacheRejectsInconsistentBounds(t *testing.T) {
	t.Parallel()
	limits := packagecache.DiskLimits{MaxEntries: 1, MaxBytes: 4096, MaxRecordBytes: 1024, TempMaxAge: time.Minute}
	if _, err := NewStableCache(t.TempDir(), limits, 0); err == nil {
		t.Fatal("zero artifact limit accepted")
	}
	if _, err := NewStableCache(t.TempDir(), limits, 2048); err == nil {
		t.Fatal("artifact limit larger than record limit accepted")
	}
}

func stableCacheStateBody(t *testing.T, artifact Artifact) contractstatemir.StableStateBodyArtifact {
	t.Helper()
	encoded, err := contractstate.U64.Encode(42)
	if err != nil {
		t.Fatal(err)
	}
	const symbol hir.SymbolID = 700
	stateHIR := hir.Package{
		Modules: []hir.Module{{Package: artifact.PackageID, Path: "cache_state", Functions: []hir.Function{{
			Symbol: symbol, Name: "write", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageWrite},
		}}}},
		StateLayouts: []hir.StateLayout{{Package: artifact.PackageID, ContractUID: "example.cache", Fields: []hir.StateField{{
			Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID(),
		}}}},
		StateTransactions: []hir.StateTransaction{{
			Package: artifact.PackageID, Function: symbol, Exit: hir.StateTransactionCommit,
			Accesses: []hir.StateAccess{{
				Kind: hir.StateAccessCellWrite, Field: "value",
				Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded},
			}},
		}},
	}
	bodies, err := contractstatemir.DescribeStableHIRStateBodies(
		stateHIR,
		[]hir.PackageID{artifact.PackageID},
		map[hir.PackageID]packagegraph.CoordinateID{artifact.PackageID: artifact.PackageCoordinate},
	)
	if err != nil {
		t.Fatal(err)
	}
	return bodies[artifact.PackageID]
}
