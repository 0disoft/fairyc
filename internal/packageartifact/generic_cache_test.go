package packageartifact

import (
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestGenericCacheVerifiedIsolationAndLRU(t *testing.T) {
	artifact, _, _ := cacheFixture(t)
	cache, err := NewGenericCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	hash := moduleinterface.GenericHash{1}
	key := packagecache.GenericKey{1}
	if cache.Get(key, hash).Status != LookupMiss {
		t.Fatal("cold cache did not miss")
	}
	for _, id := range []byte{1, 2} {
		if err := cache.Put(packagecache.GenericKey{id}, hash, artifact); err != nil {
			t.Fatal(err)
		}
	}
	first := cache.Get(key, hash)
	if first.Status != LookupHit {
		t.Fatal(first.Reason)
	}
	first.Artifact.Functions[0].Body.Entry = 999
	again := cache.Get(key, hash)
	if again.Status != LookupHit || again.Artifact.Functions[0].Body.Entry == 999 {
		t.Fatal("caller mutation reached retained payload")
	}
	if err := cache.Put(packagecache.GenericKey{3}, hash, artifact); err != nil {
		t.Fatal(err)
	}
	if cache.Get(packagecache.GenericKey{2}, hash).Status != LookupMiss || cache.Get(key, hash).Status != LookupHit {
		t.Fatal("incorrect LRU victim")
	}
	stats := cache.Stats()
	if stats.Store.Entries != 2 || stats.Store.Evictions != 1 || stats.Store.Bytes == 0 || stats.Store.Bytes > 16<<20 || stats.VerifiedHits != 3 {
		t.Fatalf("stats: %+v", stats)
	}
	cache.limits.MaxBytes = 1
	if err := cache.Put(key, hash, artifact); !errors.Is(err, packagecache.ErrRecordTooLarge) {
		t.Fatalf("oversize: %v", err)
	}
	if cache.Get(key, hash).Status != LookupHit {
		t.Fatal("oversize replacement removed prior entry")
	}
}

func TestGenericCacheQuarantinesRejectedPayloadsAndPreservesReplacements(t *testing.T) {
	artifact, _, _ := cacheFixture(t)
	cache, err := NewGenericCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	key, hash := packagecache.GenericKey{1}, moduleinterface.GenericHash{1}
	for _, mode := range []string{"interface", "foreign", "corrupt"} {
		if err := cache.Put(key, hash, artifact); err != nil {
			t.Fatal(err)
		}
		record := cache.entries[key].Value.(*genericRecord)
		expected := hash
		switch mode {
		case "interface":
			expected[0]++
		case "foreign":
			record.payload, err = fixedProcessCodec(t, 9).Encode(artifact)
		case "corrupt":
			record.payload = []byte("broken")
		}
		if err != nil {
			t.Fatal(err)
		}
		if result := cache.Get(key, expected); result.Status != LookupRejected || result.Reason == "" {
			t.Fatalf("%s: %+v", mode, result)
		}
		if cache.Get(key, hash).Status != LookupMiss {
			t.Fatal("rejected record not quarantined")
		}
	}
	if err := cache.Put(key, hash, artifact); err != nil {
		t.Fatal(err)
	}
	old := cache.entries[key].Value.(*genericRecord)
	if err := cache.Put(key, hash, artifact); err != nil {
		t.Fatal(err)
	}
	cache.rejectGeneric(old, "rejected old generation")
	if cache.Get(key, hash).Status != LookupHit {
		t.Fatal("quarantine removed concurrent replacement")
	}
	if cache.Stats().Rejected != 4 {
		t.Fatal("rejection counter drift")
	}
}

func TestGenericCacheRejectsMissingBoundsAndInvalidWrites(t *testing.T) {
	for _, limits := range []packagecache.Limits{{}, {MaxEntries: 1}, {MaxBytes: 1}} {
		if _, err := NewGenericCache(limits, processArtifactLimit); err == nil {
			t.Fatal("missing bounds admitted")
		}
	}
	if _, err := NewGenericCache(packagecache.Limits{MaxEntries: 1, MaxBytes: 1}, 0); err == nil {
		t.Fatal("missing artifact bound admitted")
	}
	artifact, _, _ := cacheFixture(t)
	cache, err := NewGenericCache(packagecache.Limits{MaxEntries: 1, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if err := cache.Put(packagecache.GenericKey{}, moduleinterface.GenericHash{1}, artifact); err == nil {
		t.Fatal("missing key admitted")
	}
	if err := cache.Put(packagecache.GenericKey{1}, moduleinterface.GenericHash{}, artifact); err == nil {
		t.Fatal("missing interface admitted")
	}
	artifact.Functions[0].Body.Entry = 999
	if err := cache.Put(packagecache.GenericKey{1}, moduleinterface.GenericHash{1}, artifact); err == nil {
		t.Fatal("invalid MIR admitted")
	}
	if cache.Stats().Store.Entries != 0 {
		t.Fatal("failed write retained data")
	}
}
