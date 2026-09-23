package cli

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestLinuxVectorSupportSource(t *testing.T) {
	ordinary, err := productSupportSource(nil)
	if err != nil {
		t.Fatal(err)
	}
	empty, err := linuxVectorSupportSource(nil)
	if err != nil || empty != ordinary {
		t.Fatalf("ordinary Linux support changed: %v", err)
	}
	selected, err := linuxVectorSupportSource([]string{
		"fy_private_vector_initialize_v0", "fy_private_vector_release_empty_v0", "fy_std_stdout_write_i32_v0",
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, declaration := range []string{
		"uint32_t fy_std_system_heap_allocate_v0(",
		"uint32_t fy_private_vector_initialize_v0(",
		"struct fy_std_console_write_result_v0 fy_std_stdout_write_i32_v0(",
	} {
		if strings.Count(selected, declaration) != 1 {
			t.Fatalf("%s was not selected exactly once", declaration)
		}
	}
	if strings.Contains(selected, "#define FY_STD_SYSTEM_HEAP_MAX_ALIGNMENT_V0 ((size_t)16)") {
		t.Fatal("Windows heap adapter entered Linux support")
	}
	if _, err := linuxVectorSupportSource([]string{"fy_private_vector_unknown_v0"}); err == nil {
		t.Fatal("unknown private vector symbol was admitted")
	}
}

func TestVectorWorkspaceEmitTargets(t *testing.T) {
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.vector-emit"
name = "client"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	for index, targetID := range []target.ID{target.X8664LinuxGNU, target.X8664WindowsMSVC} {
		t.Run(string(targetID), func(t *testing.T) {
			arguments := []string{"experimental", experimentalWorkspaceCCommand, "--target", string(targetID)}
			if index != 0 {
				arguments = append(arguments, "--locked")
			}
			code, emitted, diagnostics := runForTest(arguments, root)
			if code != ExitSuccess || diagnostics != "" ||
				!strings.Contains(emitted, "fy_private_vector_initialize_v0") ||
				!strings.Contains(emitted, "int main(void)") || !strings.HasSuffix(emitted, "\n") {
				t.Fatalf("vector workspace emit: code=%d bytes=%d diagnostics=%q", code, len(emitted), diagnostics)
			}
		})
	}
	if _, err := os.Stat(filepath.Join(root, "build")); !os.IsNotExist(err) {
		t.Fatalf("workspace C inspection created a build directory: %v", err)
	}
}

func TestLinuxVectorCandidateNativeBuild(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux amd64 glibc and GCC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.linux-vector"
name = "client"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	workspace, err := workspaceloader.LoadVectorCandidate(root)
	if err != nil {
		t.Fatal(err)
	}
	var diagnostics bytes.Buffer
	lowered, code := lowerLoadedWorkspaceExecutable(workspace, "missing", workspaceCompilePolicy{}, "Linux vector candidate", &diagnostics)
	if code != ExitSuccess || !lowered.sdkStorage {
		t.Fatalf("lower: %d %s", code, &diagnostics)
	}
	unit, err := cbackend.EmitStorageTransfers(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil || unit.Target != string(target.X8664LinuxGNU) {
		t.Fatalf("emit: %v target=%s", err, unit.Target)
	}
	support, err := linuxVectorSupportSource(unit.ForeignSymbols)
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, "gcc")
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.Build(toolchain, profile, unit.Source, support)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "vector-client")
			if err := os.WriteFile(path, artifact.Bytes, 0o700); err != nil {
				t.Fatal(err)
			}
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			output, err := exec.CommandContext(ctx, path).CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 41 || string(output) != "41" || ctx.Err() != nil {
				t.Fatalf("native: %v output=%q", err, output)
			}
		})
	}
}
