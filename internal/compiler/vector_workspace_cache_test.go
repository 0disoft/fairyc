package compiler_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

// Reuse the ownership/OOM source and independent native counters, but acquire
// package identities and source digests from real manifests instead of fixture
// keys. This is the source-to-cache-to-execution lane missing from codec tests.
func TestPrivateVectorWorkspaceCacheCurrentSource(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "disk"
		}
		t.Run(name, func(t *testing.T) { testVectorWorkspaceCacheCurrentSource(t, persistent) })
	}
}

func testVectorWorkspaceCacheCurrentSource(t *testing.T, persistent bool) {
	t.Helper()
	client, provider := vectorPackageSources(t, nil)
	root := writeVectorCacheWorkspace(t, client, provider)
	configuration := vectorCacheConfiguration(8)
	if persistent {
		configuration = vectorPersistentConfiguration(t, t.TempDir())
	}
	session, err := workspacecompiler.NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	load := func() workspaceloader.Workspace {
		t.Helper()
		workspace, err := workspaceloader.Load(root)
		if err != nil {
			t.Fatal(err)
		}
		return workspace
	}
	defer session.Close()
	stats := func() any {
		t.Helper()
		if !persistent {
			return session.Stats()
		}
		result, err := session.PersistentStats()
		if err != nil {
			t.Fatal(err)
		}
		return result
	}
	workspace := load()
	cold, err := session.Check(workspace)
	if err != nil || !cold.Compiler.Verified || cold.Cache.Stored != 2 || cold.Cache.Misses != 2 || !cold.Cache.Lowered {
		t.Fatalf("cold source cache: %v %+v", err, cold)
	}
	warm, err := session.Check(workspace)
	if err != nil || !warm.Compiler.Verified || warm.Cache.Outcome != workspacecompiler.OutcomeAllHit || warm.Cache.Hits != 2 || warm.Cache.Lowered {
		t.Fatalf("warm source cache: %v %+v", err, warm.Cache)
	}
	if !reflect.DeepEqual(cold.Compiler.GenericProvenance.Declarations(), warm.Compiler.GenericProvenance.Declarations()) ||
		!reflect.DeepEqual(cold.Compiler.GenericProvenance.Specializations(), warm.Compiler.GenericProvenance.Specializations()) {
		t.Fatal("warm publication lost current generic provenance")
	}
	rootID, err := workspace.RootPackageID()
	if err != nil {
		t.Fatal(err)
	}
	t.Run("warm-native", func(t *testing.T) {
		runVectorStorageMIRNativeLane(t, warm.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", vectorResourceProbe, true)
	})

	// Ordinary stable selection does not acquire the private session capability.
	ordinary, err := workspacecompiler.NewSourceSession(vectorCacheConfiguration(8))
	if err != nil {
		t.Fatal(err)
	}
	defer ordinary.Close()
	denied, err := ordinary.Check(workspace)
	if err == nil && denied.Compiler.Verified {
		t.Fatal("stable workspace accepted private vector source")
	}

	// A public generic body edit changes provider closure and dependent keys.
	changedProvider := strings.Replace(provider, "return storage.length;", "let usize observed = storage.length; return observed;", 1)
	if changedProvider == provider {
		t.Fatal("provider mutation did not activate")
	}
	writeVectorCacheFile(t, root, "storage/src/storage.fy", changedProvider)
	changed, err := session.Check(load())
	if err != nil || !changed.Compiler.Verified || !changed.Cache.Lowered || changed.Cache.Misses != 2 {
		t.Fatalf("provider edit reused stale keys: %v %+v", err, changed.Cache)
	}

	// Nominal client definitions participate in provider specializations even
	// though the provider never imports the client package.
	changedClient := strings.Replace(client, "cleanup { let u32 status", "cleanup { let i32 marker = 1; let u32 status", 1)
	if changedClient == client {
		t.Fatal("nominal mutation did not activate")
	}
	writeVectorCacheFile(t, root, "src/main.fy", changedClient)
	changed, err = session.Check(load())
	if err != nil || !changed.Compiler.Verified || !changed.Cache.Lowered || changed.Cache.Misses != 2 {
		t.Fatalf("nominal edit reused stale keys: %v %+v", err, changed.Cache)
	}
	updatedWarm, err := session.Check(load())
	if err != nil || updatedWarm.Cache.Outcome != workspacecompiler.OutcomeAllHit || updatedWarm.Cache.Lowered {
		t.Fatalf("updated warm: %v %+v", err, updatedWarm.Cache)
	}
	t.Run("updated-native", func(t *testing.T) {
		runVectorStorageMIRNativeLane(t, updatedWarm.Compiler.MIR, hir.QualifiedModulePath(rootID, "main")+"::main", vectorResourceProbe, true)
	})

	// Current invalid source must be rejected before looking up retained code.
	before := stats()
	writeVectorCacheFile(t, root, "storage/src/storage.fy", strings.Replace(changedProvider, "return observed;", "return false;", 1))
	invalid, err := session.Check(load())
	if invalid.Compiler.Verified || invalid.Cache.Hits != 0 || invalid.Cache.Stored != 0 || !reflect.DeepEqual(before, stats()) {
		t.Fatalf("invalid source touched cache: %v %+v", err, invalid.Cache)
	}
}

func TestPrivateVectorWorkspaceCachePartialHits(t *testing.T) {
	for _, persistent := range []bool{false, true} {
		name := "memory"
		if persistent {
			name = "disk"
		}
		t.Run(name, func(t *testing.T) { testVectorWorkspaceCachePartialHits(t, persistent) })
	}
}

func testVectorWorkspaceCachePartialHits(t *testing.T, persistent bool) {
	t.Helper()
	client, provider := vectorPackageSources(t, nil)
	root := writeVectorCacheWorkspace(t, client, provider)
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	// One retained package cannot authorize publishing half of a program.
	configuration := vectorCacheConfiguration(1)
	if persistent {
		configuration = vectorPersistentConfiguration(t, t.TempDir())
		configuration.Persistent.Limits.MaxEntries = 1
	}
	session, err := workspacecompiler.NewStorageTransferSession(configuration, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	if _, err := session.Check(workspace); err != nil {
		t.Fatal(err)
	}
	partial, err := session.Check(workspace)
	if err != nil || !partial.Compiler.Verified || partial.Cache.Hits != 1 || partial.Cache.Misses != 1 ||
		!partial.Cache.Lowered || partial.Cache.Outcome != workspacecompiler.OutcomeFallback || partial.Cache.Stored != 2 {
		t.Fatalf("partial hit did not fall back: %v %+v", err, partial.Cache)
	}
}

func vectorCacheConfiguration(entries int) workspacecompiler.Configuration {
	return workspacecompiler.Configuration{Target: target.X8664WindowsMSVC,
		Cache: packagecache.Limits{MaxEntries: entries, MaxBytes: 32 << 20}, MaxArtifactBytes: 8 << 20}
}

func writeVectorCacheWorkspace(t *testing.T, client, provider string) string {
	t.Helper()
	root := t.TempDir()
	writeVectorCacheFile(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.vector-cache.client"
name = "client"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
[workspace]
members = ["storage"]
[dependencies.storage]
path = "storage"
version = "1.0.0"
features = []
`)
	writeVectorCacheFile(t, root, "storage/fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.vector-cache.storage"
name = "storage"
version = "1.0.0"
language = "fairyc-language/20"
source = "src"
`)
	writeVectorCacheFile(t, root, "src/main.fy", client)
	writeVectorCacheFile(t, root, "storage/src/storage.fy", provider)
	return root
}

func writeVectorCacheFile(t *testing.T, root, path, text string) {
	t.Helper()
	full := filepath.Join(root, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}
