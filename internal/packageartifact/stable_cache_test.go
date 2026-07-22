package packageartifact

import (
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestStableCacheRoundTripsAcrossIndependentAdapters(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
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
	if lookup := reader.Get(key, outputInterface, artifact.Package); lookup.Status != LookupMiss {
		t.Fatalf("cold lookup = %+v", lookup)
	}
	if err := writer.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	lookup := reader.Get(key, outputInterface, artifact.Package)
	if lookup.Status != LookupHit {
		t.Fatalf("warm lookup = %+v", lookup)
	}
	if err := ValidateStableArtifact(lookup.Artifact); err != nil {
		t.Fatalf("returned stable artifact: %v", err)
	}
	lookup.Artifact.Functions[0].Function.Entry = 999
	again := writer.Get(key, outputInterface, artifact.Package)
	if again.Status != LookupHit || again.Artifact.Functions[0].Function.Entry == 999 {
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
	limits := packagecache.DiskLimits{MaxEntries: 4, MaxBytes: 16 << 20, MaxRecordBytes: 8 << 20, TempMaxAge: time.Minute}
	cache, err := NewStableCache(t.TempDir(), limits, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	otherInterface := outputInterface
	otherInterface[len(otherInterface)-1] ^= 1
	if lookup := cache.Get(key, otherInterface, artifact.Package); lookup.Status != LookupRejected {
		t.Fatalf("interface mismatch = %+v", lookup)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package); lookup.Status != LookupMiss {
		t.Fatalf("interface mismatch was not quarantined: %+v", lookup)
	}

	if err := cache.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	otherPackage := artifact.Package
	otherPackage[len(otherPackage)-1] ^= 1
	if lookup := cache.Get(key, outputInterface, otherPackage); lookup.Status != LookupRejected {
		t.Fatalf("package mismatch = %+v", lookup)
	}

	stable, err := DescribeStableArtifact(artifact, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeStableArtifact(stable, 8<<20)
	if err != nil {
		t.Fatal(err)
	}
	encoded[0] ^= 1
	if err := cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: outputInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: encoded}},
	}); err != nil {
		t.Fatal(err)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package); lookup.Status != LookupRejected {
		t.Fatalf("malformed payload = %+v", lookup)
	}
	if lookup := cache.Get(key, outputInterface, artifact.Package); lookup.Status != LookupMiss {
		t.Fatalf("malformed payload was not quarantined: %+v", lookup)
	}
	stats, err := cache.Stats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.Rejected != 3 || stats.Misses != 2 || stats.VerifiedHits != 0 || stats.Store.Entries != 0 {
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
