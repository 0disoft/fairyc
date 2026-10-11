package cdriver

import (
	"bytes"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cruntime"
)

func TestWindowsProjectSeparateCompilationAndFailures(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows x64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	scalar, err := cruntime.ScalarSource(100)
	if err != nil {
		t.Fatal(err)
	}
	header := "#ifndef FY_PROJECT_TEST\n#define FY_PROJECT_TEST\n#include <stdint.h>\nextern const uint8_t payload[];\nint32_t other(void);\n#endif\n"
	literals := "#include \"fy_project.h\"\nconst uint8_t payload[]={42};\n"
	units := []string{"#include \"fy_project.h\"\nint main(void){return (int)other()-42;}\n", "#include \"fy_project.h\"\nint32_t other(void){return payload[0];}\n"}
	for _, profile := range []Profile{LowProfile, OptimizedProfile} {
		artifact, err := BuildWindowsMSVCTargetParts(toolchain, profile, header, literals, units, scalar)
		if err != nil {
			t.Fatal(err)
		}
		if len(artifact.Objects) != 4 {
			t.Fatalf("expected four separately inspected objects, got %d", len(artifact.Objects))
		}
		rebuilt, err := BuildWindowsMSVCTargetParts(toolchain, profile, header, literals, units, scalar)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(artifact.Bytes, rebuilt.Bytes) {
			t.Fatalf("private project images differ across fresh build directories for profile %s", profile.Name)
		}
		outcome, err := ExecuteWindowsMSVCTarget(artifact)
		if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			t.Fatalf("separate C result: %+v, %v", outcome, err)
		}
	}
	broken := []string{units[0], "#include \"fy_project.h\"\n#error compile_failure\n"}
	artifact, err := BuildWindowsMSVCTargetParts(toolchain, LowProfile, header, literals, broken, scalar)
	if err == nil || !strings.Contains(err.Error(), "compile project source fy_unit_1.c") || len(artifact.Bytes) != 0 || len(artifact.Objects) != 0 {
		t.Fatalf("compile failure: image=%d bytes, objects=%d, error=%v", len(artifact.Bytes), len(artifact.Objects), err)
	}
	artifact, err = BuildWindowsMSVCTargetParts(toolchain, LowProfile, header, literals, units[:1], scalar)
	if err == nil || !strings.Contains(err.Error(), "link project C") || len(artifact.Bytes) != 0 || len(artifact.Objects) != 0 {
		t.Fatalf("link failure: image=%d bytes, objects=%d, error=%v", len(artifact.Bytes), len(artifact.Objects), err)
	}
}
