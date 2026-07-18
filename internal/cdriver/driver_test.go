package cdriver

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestProfilesAreClosedAndDistinct(t *testing.T) {
	t.Parallel()
	if LowProfile.Name != "low" || LowProfile.optimizationFlag != "-O0" || OptimizedProfile.Name != "optimized" || OptimizedProfile.optimizationFlag != "-O2" || LowProfile == OptimizedProfile {
		t.Fatalf("profiles = %+v, %+v", LowProfile, OptimizedProfile)
	}
	if sanitizerProfile.Name != AddressUndefinedSanitizerMode || sanitizerProfile.optimizationFlag != "-O1" || sanitizerProfile == LowProfile || sanitizerProfile == OptimizedProfile {
		t.Fatalf("sanitizer profile = %+v", sanitizerProfile)
	}
}

func TestBuildArgumentsKeepSanitizersOutOfProductProfiles(t *testing.T) {
	t.Parallel()
	product := buildArguments(LowProfile, nil, "unit.c", "runtime.c", "program")
	sanitized := buildArguments(sanitizerProfile, sanitizerInstrumentationFlags(), "unit.c", "runtime.c", "program")
	productText := strings.Join(product, " ")
	sanitizedText := strings.Join(sanitized, " ")
	if strings.Contains(productText, "sanitize") || !strings.Contains(sanitizedText, "-O1 -fsanitize=address,undefined -fno-omit-frame-pointer -fno-sanitize-recover=all") {
		t.Fatalf("product args = %q; sanitized args = %q", productText, sanitizedText)
	}
	if product[len(product)-5] != "-O0" || sanitized[len(sanitized)-8] != "-O1" {
		t.Fatalf("unexpected argument ordering: product=%v sanitized=%v", product, sanitized)
	}
}

func TestCrossCompilerArgumentsStayClosed(t *testing.T) {
	t.Parallel()
	arguments := objectArguments(OptimizedProfile, "caller.c", "caller.o")
	if strings.Join(arguments, " ") != "-std=c11 -pedantic-errors -Wall -Wextra -Werror -fno-common -O2 -c caller.c -o caller.o" {
		t.Fatalf("object arguments = %v", arguments)
	}
}

func TestReadArtifactEnforcesRegularBoundedNonEmptyImage(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	validPath := filepath.Join(root, "valid")
	if err := os.WriteFile(validPath, []byte("image"), 0o600); err != nil {
		t.Fatal(err)
	}
	contents, err := readArtifact(validPath)
	if err != nil || string(contents) != "image" {
		t.Fatalf("readArtifact(valid) = %q, %v", contents, err)
	}
	emptyPath := filepath.Join(root, "empty")
	if err := os.WriteFile(emptyPath, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := readArtifact(emptyPath); err == nil || !strings.Contains(err.Error(), "invalid size 0") {
		t.Fatalf("readArtifact(empty) error = %v", err)
	}
	largePath := filepath.Join(root, "large")
	large, err := os.Create(largePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := large.Truncate(maxArtifactBytes + 1); err != nil {
		_ = large.Close()
		t.Fatal(err)
	}
	if err := large.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := readArtifact(largePath); err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("readArtifact(large) error = %v", err)
	}
	if _, err := readArtifact(root); err == nil || !strings.Contains(err.Error(), "invalid size") {
		t.Fatalf("readArtifact(directory) error = %v", err)
	}
}

func TestResolveRejectsUnknownCompilerBeforeLookup(t *testing.T) {
	if _, err := Resolve(target.X8664LinuxGNU, "arbitrary-compiler"); err == nil || !strings.Contains(err.Error(), "unsupported C compiler") {
		t.Fatalf("Resolve(arbitrary) error = %v", err)
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		if _, err := Resolve(target.X8664LinuxGNU, "clang"); err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
			t.Fatalf("Resolve(clang) error = %v", err)
		}
	}
	if _, err := Resolve("unknown-target", "clang"); err == nil || !strings.Contains(err.Error(), "unsupported C target") {
		t.Fatalf("Resolve(unknown target) error = %v", err)
	}
}

func TestBuildRejectsIneligibleHost(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("eligible host build execution belongs to the native differential command")
	}
	_, err := Build(Toolchain{Target: target.X8664LinuxGNU, Name: "clang", Path: "clang"}, LowProfile, "unit", "runtime")
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("Build() error = %v", err)
	}
	_, err = BuildSanitized(Toolchain{Target: target.X8664LinuxGNU, Name: "clang", Path: "clang"}, "unit", "runtime")
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("BuildSanitized() error = %v", err)
	}
	_, err = BuildCrossCompiler(
		Toolchain{Target: target.X8664LinuxGNU, Name: "clang", Path: "clang", Libc: "glibc test"},
		Toolchain{Target: target.X8664LinuxGNU, Name: "gcc", Path: "gcc", Libc: "glibc test"},
		LowProfile, "caller", "callee",
	)
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("BuildCrossCompiler() error = %v", err)
	}
}
