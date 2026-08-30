package packagecache

import (
	"errors"
	"testing"
)

func TestMemoryStoreBoundsAndLRU(t *testing.T) {
	t.Parallel()
	baseKey, err := Compute(testInputs(t))
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	recordA := testRecord(t, baseKey, "a")
	size, err := validateRecord(recordA)
	if err != nil {
		t.Fatalf("validateRecord() error = %v", err)
	}
	store, err := NewMemoryStore(Limits{MaxEntries: 2, MaxBytes: size * 2})
	if err != nil {
		t.Fatalf("NewMemoryStore() error = %v", err)
	}
	recordB := testRecord(t, keyWithLastByte(baseKey, 1), "b")
	recordC := testRecord(t, keyWithLastByte(baseKey, 2), "c")
	if err := store.Put(recordA); err != nil {
		t.Fatalf("Put(A) error = %v", err)
	}
	if err := store.Put(recordB); err != nil {
		t.Fatalf("Put(B) error = %v", err)
	}
	if _, found := store.Get(recordA.Key); !found {
		t.Fatal("Get(A) missed")
	}
	if err := store.Put(recordC); err != nil {
		t.Fatalf("Put(C) error = %v", err)
	}
	if _, found := store.Get(recordB.Key); found {
		t.Fatal("least recently used B survived eviction")
	}
	if _, found := store.Get(recordA.Key); !found {
		t.Fatal("recently used A was evicted")
	}
	stats := store.Stats()
	if stats.Entries != 2 || stats.Bytes != size*2 || stats.Evictions != 1 || stats.Hits != 2 || stats.Misses != 1 {
		t.Fatalf("Stats() = %+v", stats)
	}
}

func TestMemoryStoreCopiesAndRejectsOversize(t *testing.T) {
	t.Parallel()
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	record := testRecord(t, key, "abc")
	size, err := validateRecord(record)
	if err != nil {
		t.Fatalf("validateRecord() error = %v", err)
	}
	store, err := NewMemoryStore(Limits{MaxEntries: 1, MaxBytes: size})
	if err != nil {
		t.Fatalf("NewMemoryStore() error = %v", err)
	}
	if err := store.Put(record); err != nil {
		t.Fatalf("Put() error = %v", err)
	}
	record.Artifacts[0].Data[0] = 'z'
	got, found := store.Get(key)
	if !found || string(got.Artifacts[0].Data) != "abc" {
		t.Fatalf("Get() = %+v, %v", got, found)
	}
	got.Artifacts[0].Data[0] = 'y'
	again, _ := store.Get(key)
	if string(again.Artifacts[0].Data) != "abc" {
		t.Fatalf("stored data mutated through returned record: %q", again.Artifacts[0].Data)
	}

	tooLarge := testRecord(t, keyWithLastByte(key, 3), "abcd")
	if err := store.Put(tooLarge); !errors.Is(err, ErrRecordTooLarge) {
		t.Fatalf("Put(tooLarge) error = %v", err)
	}
	if _, found := store.Get(key); !found {
		t.Fatal("oversize rejection disturbed existing record")
	}
}

func TestMemoryStoreRejectsInvalidPolicyAndRecords(t *testing.T) {
	t.Parallel()
	if _, err := NewMemoryStore(Limits{}); err == nil {
		t.Fatal("zero limits accepted")
	}
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatalf("Compute() error = %v", err)
	}
	store, err := NewMemoryStore(Limits{MaxEntries: 1, MaxBytes: 1024})
	if err != nil {
		t.Fatalf("NewMemoryStore() error = %v", err)
	}
	invalid := testRecord(t, key, "x")
	invalid.Artifacts = append(invalid.Artifacts, invalid.Artifacts[0])
	if err := store.Put(invalid); err == nil {
		t.Fatal("duplicate artifact kind accepted")
	}
	missingKey := testRecord(t, Key{}, "x")
	if err := store.Put(missingKey); err == nil {
		t.Fatal("missing record key accepted")
	}
}

func TestMemoryStoreDeleteIfMatchPreservesConcurrentReplacement(t *testing.T) {
	t.Parallel()
	key, err := Compute(testInputs(t))
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(Limits{MaxEntries: 1, MaxBytes: 4096})
	if err != nil {
		t.Fatal(err)
	}
	old := testRecord(t, key, "old")
	if err := store.Put(old); err != nil {
		t.Fatal(err)
	}
	observed, found := store.Get(key)
	if !found {
		t.Fatal("Get() missed old record")
	}
	replacement := testRecord(t, key, "replacement")
	if err := store.Put(replacement); err != nil {
		t.Fatal(err)
	}
	if store.DeleteIfMatch(observed) {
		t.Fatal("DeleteIfMatch removed a newer replacement")
	}
	current, found := store.Get(key)
	if !found || string(current.Artifacts[0].Data) != "replacement" {
		t.Fatalf("current record = %+v, %v", current, found)
	}
	if !store.DeleteIfMatch(current) {
		t.Fatal("DeleteIfMatch did not remove the matching record")
	}
	if _, found := store.Get(key); found {
		t.Fatal("matching record survived DeleteIfMatch")
	}
}

func testRecord(t *testing.T, key Key, data string) Record {
	t.Helper()
	return Record{
		Key:             key,
		PublicInterface: mustInterface(t, "fyif5-sha256-a5e79db9b105ffac6d57d0ff054b75813344fdf4e9f9a5c1873da8907ae281b9"),
		Artifacts:       []Artifact{{Kind: ArtifactMIR, Data: []byte(data)}},
	}
}

func keyWithLastByte(key Key, delta byte) Key {
	key[len(key)-1] += delta
	return key
}
