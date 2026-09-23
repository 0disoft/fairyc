package cli

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/target"
)

func TestExperimentalLinuxVectorWorkspaceBuild(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("requires Linux amd64 glibc and GCC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.linux-workspace-build"
name = "client"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	for index, profile := range []string{"low", "optimized"} {
		t.Run(profile, func(t *testing.T) {
			outputPath := "build/client-" + profile
			arguments := []string{
				"experimental", "build-workspace", "--target", string(target.X8664LinuxGNU),
				"--cc", "gcc", "--profile", profile, "--output", outputPath,
			}
			if index != 0 {
				arguments = append(arguments, "--locked")
			}
			code, stdout, stderr := runForTest(arguments, root)
			if code != ExitSuccess || stderr != "" || !strings.Contains(stdout, "built workspace: "+outputPath) {
				t.Fatalf("build: code=%d stdout=%q stderr=%q", code, stdout, stderr)
			}
			path := filepath.Join(root, filepath.FromSlash(outputPath))
			artifact, err := os.ReadFile(path)
			if err != nil || len(artifact) < 4 || string(artifact[:4]) != "\x7fELF" {
				t.Fatalf("Linux build did not write an ELF executable: %v", err)
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

func TestExperimentalLinuxWorkspaceBuildFailsClosedOutsideLinux(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("Linux host executes the native build test")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", `schema = "fairyc.manifest/v1"
[package]
uid = "test.cli.linux-workspace-host-boundary"
name = "client"
version = "1.0.0"
language = "fairyc-language/17"
source = "src"
`)
	writeTestSource(t, root, "src/main.fy", nativeVectorClient)
	code, stdout, stderr := runForTest([]string{
		"experimental", "build-workspace", "--target", string(target.X8664LinuxGNU),
		"--cc", "gcc", "--profile", "low", "--output", "build/client",
	}, root)
	if code != ExitBuildFailure || stdout != "" || !strings.Contains(stderr, "requires linux/amd64") {
		t.Fatalf("non-Linux build: code=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if _, err := os.Stat(filepath.Join(root, "build", "client")); !os.IsNotExist(err) {
		t.Fatalf("unsupported host wrote an executable: %v", err)
	}
}
