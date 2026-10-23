package cdriver

import (
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestRuntimeObjectCacheIdentity(t *testing.T) {
	base := runtimeObjectIdentity{CompilerPath: "/compiler", CompilerVersion: "1", Target: "target",
		Mode: C11ConsumerMode, Profile: LowProfile, Flags: []string{"-O0"},
		Environment: []string{"B=2", "A=1"}, Source: sha256.Sum256([]byte("source")), Preprocessed: sha256.Sum256([]byte("header"))}
	for name, mutate := range map[string]func(*runtimeObjectIdentity){
		"source":      func(v *runtimeObjectIdentity) { v.Source[0]++ },
		"header":      func(v *runtimeObjectIdentity) { v.Preprocessed[0]++ },
		"path":        func(v *runtimeObjectIdentity) { v.CompilerPath += "2" },
		"version":     func(v *runtimeObjectIdentity) { v.CompilerVersion += "2" },
		"binary":      func(v *runtimeObjectIdentity) { v.CompilerDigest[0]++ },
		"target":      func(v *runtimeObjectIdentity) { v.Target += "2" },
		"libc":        func(v *runtimeObjectIdentity) { v.Libc = "2" },
		"mode":        func(v *runtimeObjectIdentity) { v.Mode = C17ConsumerMode },
		"profile":     func(v *runtimeObjectIdentity) { v.Profile = OptimizedProfile },
		"flags":       func(v *runtimeObjectIdentity) { v.Flags = []string{"-O2"} },
		"environment": func(v *runtimeObjectIdentity) { v.Environment = []string{"A=2"} },
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			mutate(&changed)
			if changed.key() == base.key() || changed.unitKey() == base.unitKey() {
				t.Fatal("identity did not invalidate")
			}
		})
	}
	if base.key() == base.unitKey() {
		t.Fatal("runtime and unit key domains overlap")
	}
	reordered := base
	reordered.Environment = []string{"A=1", "B=2"}
	if base.key() != reordered.key() {
		t.Fatal("environment order changed identity")
	}
	left, right := base, base
	left.Environment = []string{"A=1", "A=2"}
	right.Environment = []string{"A=2", "A=1"}
	if left.key() == right.key() {
		t.Fatal("conflicting environment order lost last-value semantics")
	}
}

func runtimeTestStore(t *testing.T, root string) *packagecache.DiskStore {
	t.Helper()
	store, err := packagecache.NewDiskStore(root, runtimeObjectLimits())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestRuntimeObjectCacheHitCorruptionAndRetry(t *testing.T) {
	root := t.TempDir()
	store := runtimeTestStore(t, root)
	key := packagecache.Key(sha256.Sum256([]byte("runtime")))
	compiled, linked := 0, 0
	compile := func() ([]byte, error) { compiled++; return []byte("valid object"), nil }
	validate := func(value []byte) error {
		if string(value) != "valid object" {
			return errors.New("bad object")
		}
		return nil
	}
	link := func(value []byte) error { linked++; return validate(value) }
	run := func(want RuntimeObjectCacheStatus) {
		t.Helper()
		status, err := useRuntimeObject(store, key, compile, validate, link)
		if err != nil || status != want {
			t.Fatalf("status %q err %v, want %q", status, err, want)
		}
	}
	run(RuntimeCacheMiss)
	// Reopen to prove disk reuse across store instances, without a process L1.
	store = runtimeTestStore(t, root)
	run(RuntimeCacheHit)
	if compiled != 1 || linked != 2 {
		t.Fatalf("compile/link counts %d/%d", compiled, linked)
	}
	files, err := filepath.Glob(filepath.Join(root, "v1", "*", "*.json"))
	if err != nil || len(files) != 1 {
		t.Fatalf("generations %v: %v", files, err)
	}
	if err := os.WriteFile(files[0], []byte("truncated"), 0o600); err != nil {
		t.Fatal(err)
	}
	run(RuntimeCacheMiss)
	// A structurally accepted but link-incompatible object triggers one retry.
	attempts := 0
	link = func(value []byte) error {
		attempts++
		if attempts == 1 {
			return errors.New("link incompatible")
		}
		return validate(value)
	}
	run(RuntimeCacheRecovered)
	if attempts != 2 || compiled != 3 {
		t.Fatalf("retry attempts %d, compiles %d", attempts, compiled)
	}
	link = func([]byte) error { return errors.New("persistent link failure") }
	before := compiled
	if status, err := useRuntimeObject(store, key, compile, validate, link); status != RuntimeCacheRecovered || err == nil || compiled != before+1 {
		t.Fatalf("bounded retry %q %v compiles %d", status, err, compiled)
	}
	if _, found := store.Get(key); found {
		t.Fatal("failed hit was not quarantined")
	}
}

func TestRuntimeObjectCacheRejectsPayloadAndStoreFailures(t *testing.T) {
	store := runtimeTestStore(t, t.TempDir())
	key := packagecache.Key(sha256.Sum256([]byte("runtime")))
	bad := packagecache.Record{Key: key, PublicInterface: moduleinterface.Hash(key), Artifacts: []packagecache.Artifact{{Kind: packagecache.ArtifactObject, Data: []byte("invalid")}}}
	if err := store.Put(bad); err != nil {
		t.Fatal(err)
	}
	compiled := 0
	compile := func() ([]byte, error) { compiled++; return []byte("valid"), nil }
	validate := func(b []byte) error {
		if string(b) != "valid" {
			return errors.New("invalid")
		}
		return nil
	}
	link := func(b []byte) error { return validate(b) }
	if status, err := useRuntimeObject(store, key, compile, validate, link); status != RuntimeCacheMiss || err != nil || compiled != 1 {
		t.Fatalf("bad payload %q %v", status, err)
	}
	// A failed write does not turn a successful compile/link into a failure.
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := useRuntimeObject(store, key, compile, validate, link); err != nil {
		t.Fatal(err)
	}
	if status, err := useRuntimeObject(nil, key, compile, validate, link); status != RuntimeCacheBypass || err != nil {
		t.Fatalf("bypass %q %v", status, err)
	}
}
