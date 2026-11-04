package cdriver

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestWindowsProjectResponseAtUnitLimit(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows x64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Skipf("MSVC unavailable: %v", err)
	}
	directory := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.Mkdir(directory, 0o700); err != nil {
		t.Fatal(err)
	}
	// Compile just two real objects; copies give the linker the full admitted
	// object list without paying for 4,096 redundant compiler invocations.
	for _, input := range []struct{ name, source string }{
		{"fy_literals", "int main(void){return 23;}\n"},
		{"fy_unit_0", "typedef int fy_empty_unit;\n"},
	} {
		if err := os.WriteFile(filepath.Join(directory, input.name+".c"), []byte(input.source), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := runWindowsHostCommandWithLimitAndTimeout(toolchain.Path, toolchain.Environment, directory, maxProcessOutput, time.Minute,
			"/nologo", "/TC", "/MT", "/Brepro", "/c", input.name+".c", "/Fo:"+input.name+".obj"); err != nil {
			t.Fatal(err)
		}
	}
	objects := []string{"fy_literals.obj", "fy_unit_0.obj"}
	for index := 1; index < 4096; index++ {
		objects = append(objects, fmt.Sprintf("fy_unit_%d.obj", index))
	}
	objects = append(objects, "fy_runtime.obj")
	empty, err := os.ReadFile(filepath.Join(directory, "fy_unit_0.obj"))
	if err != nil {
		t.Fatal(err)
	}
	for _, object := range objects[2:] {
		if err := os.WriteFile(filepath.Join(directory, object), empty, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if len(strings.Join(objects, " ")) <= 32767 {
		t.Fatal("test object list does not exceed the Windows command-line limit")
	}
	if err := linkWindowsMSVCProject(toolchain, directory, objects, time.Minute, 0); err != nil {
		t.Fatal(err)
	}
	response, err := os.ReadFile(filepath.Join(directory, "fy_objects.rsp"))
	if err != nil || string(response) != strings.Join(objects, "\n")+"\n" {
		t.Fatalf("response lost object order/content: %v", err)
	}
	image, err := readArtifact(filepath.Join(directory, "program.exe"))
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := ExecuteWindowsMSVCTarget(WindowsHostArtifact{Bytes: image, Toolchain: toolchain, Profile: LowProfile, ConsumerMode: C11ConsumerMode})
	if err != nil || outcome.ExitCode != 23 || outcome.Stdout != "" || outcome.Stderr != "" {
		t.Fatalf("maximum project link result: %+v, %v", outcome, err)
	}
	t.Logf("linked %d objects from %d response bytes; exit=%d", len(objects), len(response), outcome.ExitCode)
}
