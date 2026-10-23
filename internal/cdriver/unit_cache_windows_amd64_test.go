package cdriver

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
)

func TestWindowsGeneratedUnitCache(t *testing.T) {
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeRoot, unitRoot := t.TempDir(), t.TempDir()
	openRuntime := func() (*packagecache.DiskStore, error) {
		return packagecache.NewDiskStore(runtimeRoot, runtimeObjectLimits())
	}
	openUnit := func() (*packagecache.DiskStore, error) {
		return packagecache.NewDiskStore(unitRoot, unitObjectLimits())
	}
	header := filepath.Join(t.TempDir(), "unit.h")
	writeHeader := func(value int) {
		t.Helper()
		if err := os.WriteFile(header, []byte(fmt.Sprintf("#define OFFSET %d\n", value)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeHeader(3)
	source := fmt.Sprintf("#include \"%s\"\nint answer(void); int main(void){return answer()+OFFSET;}\n", filepath.ToSlash(header))
	support := "int answer(void){return 5;}\n"
	build := func(profile Profile, unit, runtime string, runtimeStatus, unitStatus RuntimeObjectCacheStatus, want int32) WindowsHostArtifact {
		t.Helper()
		artifact, err := buildWindowsObjectsCached(toolchain, profile, unit, runtime, openRuntime, openUnit)
		if err != nil {
			t.Fatal(err)
		}
		if artifact.RuntimeCache != runtimeStatus || artifact.UnitCache != unitStatus {
			t.Fatalf("cache: runtime=%s unit=%s, want %s/%s", artifact.RuntimeCache, artifact.UnitCache, runtimeStatus, unitStatus)
		}
		outcome, err := ExecuteWindowsMSVCTarget(artifact)
		if err != nil || outcome.ExitCode != want || outcome.Stdout != "" || outcome.Stderr != "" {
			t.Fatalf("execution: %+v %v, want %d", outcome, err, want)
		}
		return artifact
	}
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		cold, warm := RuntimeCacheMiss, RuntimeCacheHit
		if profile == LowProfile {
			cold, warm = RuntimeCacheBypass, RuntimeCacheBypass
		}
		build(profile, source, support, RuntimeCacheMiss, cold, 8)
		build(profile, source, support, RuntimeCacheHit, warm, 8)
	}
	writeHeader(4)
	build(OptimizedProfile, source, support, RuntimeCacheHit, RuntimeCacheMiss, 9)
	source += "\n/* edited generated source */\n"
	build(OptimizedProfile, source, support, RuntimeCacheHit, RuntimeCacheMiss, 9)
	support = "int answer(void){return 9;}\n"
	build(OptimizedProfile, source, support, RuntimeCacheMiss, RuntimeCacheHit, 13)

	// A hit still links current library inputs: changing only the runtime changes output.
	// Corrupt files must miss without blocking source-based reconstruction.
	files, err := filepath.Glob(filepath.Join(unitRoot, "v1", "*", "*.json"))
	if err != nil || len(files) != 2 {
		t.Fatalf("retained unit entries: %v %v", files, err)
	}
	for _, path := range files {
		if err := os.WriteFile(path, []byte("truncated"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	valid := build(OptimizedProfile, source, support, RuntimeCacheHit, RuntimeCacheMiss, 13)
	// Replace the valid unit with a checksum-valid runtime object, which lacks main.
	// The failed real link must trigger a fresh unit compile and recover.
	store, err := openUnit()
	if err != nil {
		t.Fatal(err)
	}
	files, err = filepath.Glob(filepath.Join(unitRoot, "v1", "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	poisoned := 0
	for _, path := range files {
		decoded, err := hex.DecodeString(filepath.Base(filepath.Dir(path)))
		if err != nil || len(decoded) != 32 {
			t.Fatal("invalid cache key path")
		}
		var key packagecache.Key
		copy(key[:], decoded)
		if record, ok := store.Get(key); ok {
			if !store.DeleteIfMatch(record) {
				t.Fatal("cannot replace original unit")
			}
			record.Artifacts[0].Data = valid.Objects[1].Bytes
			if err := store.Put(record); err != nil {
				t.Fatal(err)
			}
			poisoned++
		}
	}
	store.Close()
	if poisoned != 1 {
		t.Fatalf("poisoned %d entries, want 1", poisoned)
	}
	build(OptimizedProfile, source, support, RuntimeCacheHit, RuntimeCacheRecovered, 13)
	build(OptimizedProfile, source, support, RuntimeCacheHit, RuntimeCacheHit, 13)
	for index := range 8 {
		build(OptimizedProfile, source+fmt.Sprintf("/* edit %d */\n", index), support, RuntimeCacheHit, RuntimeCacheMiss, 13)
	}
	store, err = openUnit()
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.Stats()
	if err != nil || before.Entries != 2 || before.Bytes > unitObjectLimits().MaxBytes {
		t.Fatalf("unbounded unit retention: %+v %v", before, err)
	}
	if _, err := buildWindowsObjectsCached(toolchain, OptimizedProfile,
		"int missing(void); int main(void){return missing();}\n", support, openRuntime, openUnit); err == nil {
		t.Fatal("missing symbol link falsely succeeded")
	}
	after, err := store.Stats()
	store.Close()
	if err != nil || after.Entries != before.Entries || after.Bytes != before.Bytes {
		t.Fatalf("failed link published a unit: before=%+v after=%+v err=%v", before, after, err)
	}
	openUnit = func() (*packagecache.DiskStore, error) { return nil, fmt.Errorf("unavailable") }
	build(OptimizedProfile, source, support, RuntimeCacheMiss, RuntimeCacheBypass, 13)
	toolchain.Environment = append(toolchain.Environment, "CL=/DCACHE_BYPASS_PROBE=1")
	build(OptimizedProfile, source, support, RuntimeCacheBypass, RuntimeCacheBypass, 13)
}
