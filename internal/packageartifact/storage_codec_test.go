package packageartifact

import (
	"bytes"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestStorageProcessCodecIsolationAndBounds(t *testing.T) {
	artifact, _, _ := cacheFixture(t)
	ordinary := fixedProcessCodec(t, 1)
	transfer, err := newProcessCodec(ordinary.identity, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	transfer.storageTransfers = true
	encoded, err := transfer.Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if transfer.Identity() == ordinary.Identity() || !bytes.Contains(encoded, []byte(StorageProcessCodecSchema)) {
		t.Fatal("codec domains collapsed")
	}
	if _, err := ordinary.Decode(encoded); err == nil {
		t.Fatal("ordinary reader admitted transfer envelope")
	}
	base, err := ordinary.Encode(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := transfer.Decode(base); err == nil {
		t.Fatal("transfer reader admitted ordinary envelope")
	}
	other, err := NewStorageTransferProcessCodec(processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := other.Decode(encoded); !errors.Is(err, ErrProcessArtifactForeign) {
		t.Fatalf("foreign transfer codec: %v", err)
	}
	if _, err := transfer.Decode(append(bytes.Clone(encoded), '\n')); !errors.Is(err, ErrProcessArtifactEncoding) {
		t.Fatalf("noncanonical transfer: %v", err)
	}
	small := *transfer
	small.maxBytes = uint64(len(encoded) - 1)
	if _, err := small.Decode(encoded); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("decode bound: %v", err)
	}
	if _, err := small.Encode(artifact); !errors.Is(err, ErrProcessArtifactTooLarge) {
		t.Fatalf("encode bound: %v", err)
	}
	if _, err := NewStorageTransferProcessCodec(0); err == nil {
		t.Fatal("zero codec bound accepted")
	}
}

func TestStorageCacheRetainsLRUAndQuarantine(t *testing.T) {
	artifact, _, _ := cacheFixture(t)
	cache, err := NewStorageTransferCache(packagecache.Limits{MaxEntries: 2, MaxBytes: 16 << 20}, processArtifactLimit)
	if err != nil {
		t.Fatal(err)
	}
	hash := moduleinterface.GenericHash{1}
	for _, id := range []byte{1, 2} {
		if err := cache.Put(packagecache.GenericKey{id}, hash, artifact); err != nil {
			t.Fatal(err)
		}
	}
	first := cache.Get(packagecache.GenericKey{1}, hash)
	if first.Status != LookupHit {
		t.Fatal(first.Reason)
	}
	first.Artifact.Functions[0].Body.Entry = 999
	if again := cache.Get(packagecache.GenericKey{1}, hash); again.Status != LookupHit || again.Artifact.Functions[0].Body.Entry == 999 {
		t.Fatal("cache payload aliases caller")
	}
	if err := cache.Put(packagecache.GenericKey{3}, hash, artifact); err != nil {
		t.Fatal(err)
	}
	if cache.Get(packagecache.GenericKey{2}, hash).Status != LookupMiss {
		t.Fatal("wrong LRU victim")
	}
	if cache.Get(packagecache.GenericKey{1}, moduleinterface.GenericHash{2}).Status != LookupRejected {
		t.Fatal("wrong interface accepted")
	}
	if cache.Get(packagecache.GenericKey{1}, hash).Status != LookupMiss {
		t.Fatal("rejected entry not quarantined")
	}
	if cache.Stats().Store.Evictions != 1 {
		t.Fatal("LRU eviction not accounted")
	}
}
