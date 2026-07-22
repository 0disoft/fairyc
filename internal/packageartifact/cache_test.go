package packageartifact

import (
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestCacheStoresAndReturnsVerifiedArtifact(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
	cache, err := NewCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if lookup := cache.Get(key, outputInterface); lookup.Status != LookupMiss {
		t.Fatalf("cold lookup = %+v", lookup)
	}
	if err := cache.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	lookup := cache.Get(key, outputInterface)
	if lookup.Status != LookupHit {
		t.Fatalf("warm lookup = %+v", lookup)
	}
	if err := Verify(lookup.Artifact); err != nil {
		t.Fatalf("returned artifact: %v", err)
	}
	lookup.Artifact.Functions[0].Body.Entry = 999
	again := cache.Get(key, outputInterface)
	if again.Status != LookupHit || again.Artifact.Functions[0].Body.Entry == 999 {
		t.Fatalf("cache payload mutated through caller: %+v", again)
	}
	stats := cache.Stats()
	if stats.VerifiedHits != 2 || stats.Misses != 1 || stats.Rejected != 0 || stats.Store.Entries != 1 || stats.Store.Hits != 2 || stats.Store.Misses != 1 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestCacheRejectsAndQuarantinesContradictoryOrForeignPayload(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
	cache, err := NewCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	foreign := fixedProcessCodec(t, 9)
	encoded, err := foreign.Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.store.Put(packagecache.Record{
		Key: key, PublicInterface: outputInterface,
		Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactMIR, Data: encoded}},
	}); err != nil {
		t.Fatal(err)
	}
	lookup := cache.Get(key, outputInterface)
	if lookup.Status != LookupRejected || lookup.Reason == "" {
		t.Fatalf("foreign lookup = %+v", lookup)
	}
	if next := cache.Get(key, outputInterface); next.Status != LookupMiss {
		t.Fatalf("quarantined lookup = %+v", next)
	}
	if err := cache.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	otherInterface := outputInterface
	otherInterface[len(otherInterface)-1] ^= 1
	lookup = cache.Get(key, otherInterface)
	if lookup.Status != LookupRejected {
		t.Fatalf("interface mismatch lookup = %+v", lookup)
	}
	stats := cache.Stats()
	if stats.Rejected != 2 || stats.Misses != 1 || stats.VerifiedHits != 0 || stats.Store.Entries != 0 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestCachePreservesExistingEntryWhenOversizePutFails(t *testing.T) {
	t.Parallel()
	artifact, key, outputInterface := cacheFixture(t)
	codec := fixedProcessCodec(t, 4)
	encoded, err := codec.Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	cache, err := NewCache(packagecache.Limits{MaxEntries: 1, MaxBytes: uint64(len(encoded)) + 256}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(key, outputInterface, artifact); err != nil {
		t.Fatal(err)
	}
	tooSmall, err := newProcessCodec(cache.codec.identity, uint64(len(encoded)-1))
	if err != nil {
		t.Fatal(err)
	}
	cache.codec = tooSmall
	if err := cache.Put(key, outputInterface, artifact); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("oversize Put() error = %v", err)
	}
	cache.codec = codecForIdentity(t, cache.codec.identity, processArtifactLimit)
	if lookup := cache.Get(key, outputInterface); lookup.Status != LookupHit {
		t.Fatalf("existing entry after oversize rejection = %+v", lookup)
	}
}

func cacheFixture(t *testing.T) (Artifact, packagecache.Key, moduleinterface.Hash) {
	t.Helper()
	_, program, identities := checkedWorkspaceProgram(t)
	artifacts, err := ExtractAll(program, identities)
	if err != nil {
		t.Fatal(err)
	}
	key := packagecache.Key{1}
	outputInterface := moduleinterface.Hash{1}
	return artifacts[0], key, outputInterface
}

func codecForIdentity(t *testing.T, identity [32]byte, maxBytes uint64) *ProcessCodec {
	t.Helper()
	codec, err := newProcessCodec(identity, maxBytes)
	if err != nil {
		t.Fatal(err)
	}
	return codec
}
