package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cli"
)

const smokeSource = `module main;

fn main() -> i32 {
    let empty: &str = "";
    let text: &str = "Fairy λ";
    if unsafe { text as *u8 } != null { 42 } else { 1 }
}
`

const maxSmokeArtifactBytes = 64 * 1024 * 1024

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fairyc Windows host smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(smokeSource), 0o600); err != nil {
		return fmt.Errorf("write smoke source: %w", err)
	}

	var buildStdout bytes.Buffer
	var buildStderr bytes.Buffer
	buildCode := cli.Run([]string{
		"experimental", "build-host", "main.fy",
		"--profile", "optimized", "--output", "build/main.exe",
	}, root, &buildStdout, &buildStderr)
	if buildCode != cli.ExitSuccess || buildStderr.Len() != 0 || !strings.HasPrefix(buildStdout.String(), "built: build/main.exe (windows/amd64 host-smoke, msvc ") {
		return fmt.Errorf("build-host returned code %d, stdout %q, stderr %q", buildCode, buildStdout.String(), buildStderr.String())
	}
	artifactPath := filepath.Join(root, "build", "main.exe")
	artifact, err := os.ReadFile(artifactPath)
	if err != nil {
		return fmt.Errorf("read build-host artifact: %w", err)
	}
	if len(artifact) < 2 || len(artifact) > maxSmokeArtifactBytes || artifact[0] != 'M' || artifact[1] != 'Z' {
		return fmt.Errorf("build-host artifact is not a bounded PE image")
	}
	if err := executeExpect42(artifactPath); err != nil {
		return err
	}

	var runStdout bytes.Buffer
	var runStderr bytes.Buffer
	runCode := cli.Run([]string{
		"experimental", "run-host", "main.fy", "--profile", "optimized",
	}, root, &runStdout, &runStderr)
	if runCode != cli.ExitSuccess || runStderr.Len() != 0 || !strings.HasPrefix(runStdout.String(), "native result: 42 (windows/amd64 host-smoke, msvc ") {
		return fmt.Errorf("run-host returned code %d, stdout %q, stderr %q", runCode, runStdout.String(), runStderr.String())
	}
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes and run-host both returned 42\n", len(artifact))
	return nil
}

func executeExpect42(path string) error {
	contextValue, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, path)
	command.Stdin = nil
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("build-host artifact exceeded execution timeout")
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) || exitError.ExitCode() != 42 || output.Len() != 0 {
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		return fmt.Errorf("build-host artifact returned error %v, exit %d, output %q", err, exitCode, output.String())
	}
	return nil
}
