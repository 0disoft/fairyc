package cdriver

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/packagecache"
	"github.com/0disoft/fairyc/internal/target"
)

func TestRuntimeObjectCacheNative(t *testing.T) {
	if runtime.GOOS == "linux" {
		for _, name := range []string{"gcc", "clang"} {
			t.Run(name, func(t *testing.T) { testRuntimeObjectCacheNative(t, name) })
		}
		return
	}
	testRuntimeObjectCacheNative(t, "msvc")
}

func testRuntimeObjectCacheNative(t *testing.T, compilerName string) {
	if runtime.GOARCH != "amd64" || (runtime.GOOS != "windows" && runtime.GOOS != "linux") {
		t.Skip("native compiler needs Windows or Linux amd64")
	}
	var windows WindowsHostToolchain
	var linux Toolchain
	var err error
	if runtime.GOOS == "windows" {
		windows, err = ResolveWindowsHostMSVC()
	} else {
		linux, err = Resolve(target.X8664LinuxGNU, compilerName)
	}
	if err != nil {
		t.Skipf("native compiler unavailable: %v", err)
	}
	cacheRoot := t.TempDir()
	open := func() (*packagecache.DiskStore, error) {
		return packagecache.NewDiskStore(cacheRoot, runtimeObjectLimits())
	}
	header := filepath.Join(t.TempDir(), "value.h")
	writeHeader := func(n int) {
		t.Helper()
		if err := os.WriteFile(header, []byte(fmt.Sprintf("#define ANSWER %d\n", n)), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	writeHeader(7)
	support := fmt.Sprintf("#include <stdio.h>\n#include \"%s\"\nint answer(void) { return ANSWER; }\n", filepath.ToSlash(header))
	unit := "int answer(void); int main(void) { return answer(); }\n"
	build := func(profile Profile, unit, support string) (RuntimeObjectCacheStatus, int32) {
		t.Helper()
		if runtime.GOOS == "windows" {
			artifact, err := buildWindowsRuntimeCached(windows, profile, unit, support, open)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := ExecuteWindowsMSVCTarget(artifact)
			if err != nil {
				t.Fatal(err)
			}
			return artifact.RuntimeCache, outcome.ExitCode
		}
		artifact, err := buildLinuxRuntimeCached(linux, profile, unit, support, open)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := ExecuteLinuxTarget(artifact)
		if err != nil {
			t.Fatal(err)
		}
		return artifact.RuntimeCache, outcome.ExitCode
	}
	check := func(profile Profile, source, support string, wantStatus RuntimeObjectCacheStatus, wantExit int32) {
		t.Helper()
		status, exit := build(profile, source, support)
		if status != wantStatus || exit != wantExit {
			t.Fatalf("got %s exit %d; want %s exit %d", status, exit, wantStatus, wantExit)
		}
	}
	check(LowProfile, unit, support, RuntimeCacheMiss, 7)
	check(LowProfile, unit, support, RuntimeCacheHit, 7)
	check(LowProfile, strings.Replace(unit, "answer();", "answer() + 1;", 1), support, RuntimeCacheHit, 8)
	writeHeader(11)
	check(LowProfile, unit, support, RuntimeCacheMiss, 11)
	check(OptimizedProfile, unit, support, RuntimeCacheMiss, 11)
	check(OptimizedProfile, unit, support, RuntimeCacheHit, 11)
	check(LowProfile, unit, support+"\n/* raw source change */\n", RuntimeCacheMiss, 11)
	if runtime.GOOS == "windows" {
		windows.Version += "-changed"
	} else {
		linux.Version += "-changed"
	}
	check(LowProfile, unit, support, RuntimeCacheMiss, 11)
	// Corrupt all retained files, then prove the executable still comes from source.
	files, err := filepath.Glob(filepath.Join(cacheRoot, "v1", "*", "*.json"))
	if err != nil || len(files) == 0 {
		t.Fatalf("cache files %v: %v", files, err)
	}
	for _, file := range files {
		if err := os.WriteFile(file, []byte("broken"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	check(LowProfile, unit, support, RuntimeCacheMiss, 11)
	// A checksum-valid, structurally valid object from another translation unit
	// must be discarded after the real linker rejects its missing answer symbol.
	var wrongObject []byte
	if runtime.GOOS == "windows" {
		wrong, err := BuildWindowsMSVCProbe(windows, LowProfile, "int main(void) { return 0; }\n")
		if err != nil {
			t.Fatal(err)
		}
		wrongObject = wrong.Objects[0].Bytes
	} else {
		dir := t.TempDir()
		path, output := filepath.Join(dir, "wrong.c"), filepath.Join(dir, "wrong.o")
		if err := os.WriteFile(path, []byte("int wrong(void) { return 0; }\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := runBounded(linux.Path, objectArguments(LowProfile, path, output)...); err != nil {
			t.Fatal(err)
		}
		wrongObject, err = readArtifact(output)
		if err != nil {
			t.Fatal(err)
		}
	}
	files, err = filepath.Glob(filepath.Join(cacheRoot, "v1", "*", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	store, err := open()
	if err != nil {
		t.Fatal(err)
	}
	poisoned := 0
	for _, file := range files {
		encoded, err := hex.DecodeString(filepath.Base(filepath.Dir(file)))
		if err != nil || len(encoded) != len(packagecache.Key{}) {
			t.Fatalf("invalid key: %v", err)
		}
		var key packagecache.Key
		copy(key[:], encoded)
		record, found := store.Get(key)
		if !found {
			continue
		} // Other keys still contain deliberately corrupt generations.
		if !store.DeleteIfMatch(record) {
			t.Fatal("cannot quarantine original generation")
		}
		record.Artifacts[0].Data = wrongObject
		if err := store.Put(record); err != nil {
			t.Fatal(err)
		}
		poisoned++
	}
	if poisoned == 0 {
		t.Fatal("no valid generation to poison")
	}
	store.Close()
	check(LowProfile, unit, support, RuntimeCacheRecovered, 11)
	check(LowProfile, unit, support, RuntimeCacheHit, 11)
	open = func() (*packagecache.DiskStore, error) { return nil, fmt.Errorf("test cache unavailable") }
	check(LowProfile, unit, support, RuntimeCacheBypass, 11)
}
