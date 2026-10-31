//go:build windows && amd64

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
)

func buildDependencyTracer(t *testing.T, tc cdriver.WindowsHostToolchain) string {
	t.Helper()
	artifact, err := cdriver.BuildWindowsMSVCProbe(tc, cdriver.LowProfile, windowsDependencyTracerC)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "trace.exe")
	if err := os.WriteFile(path, artifact.Bytes, 0700); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestWindowsDLLLoadTrace(t *testing.T) {
	tc, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	tracer := buildDependencyTracer(t, tc)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(tc, profile, "#include <stdlib.h>\nint main(void){void *p=malloc(17);if(!p)return 1;free(p);return 42;}\n")
		if err != nil {
			t.Fatal(err)
		}
		image := filepath.Join(t.TempDir(), "probe.exe")
		if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		receipt, out, stderr, err := traceDependencyProcess(tracer, image, nil)
		if err != nil || receipt.Exit != 42 || len(out) != 0 || len(stderr) != 0 {
			t.Fatalf("trace %+v %s %s %v", receipt, out, stderr, err)
		}
		if err := validateDependencyResolutions(&receipt); err != nil {
			t.Fatal(err)
		}
		t.Logf("%s DLL imports=%d load events=%d exit=%d", profile.Name, len(receipt.Imports.Libraries), len(receipt.Modules), receipt.Exit)
	}
}
