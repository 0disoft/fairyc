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

const successSource = `module main;

fn main() -> i32 {
    let empty: &str = "";
    let text: &str = "Fairy λ";
    if unsafe { text as *u8 } != null { 42 } else { 1 }
}
`

const negativeSource = `module negative;

fn main() -> i32 {
    -1
}
`

const trapSource = `module trap;

fn main() -> i32 {
    42 / 0
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
	for name, source := range map[string]string{
		"main.fy":     successSource,
		"negative.fy": negativeSource,
		"trap.fy":     trapSource,
	} {
		if err := os.WriteFile(filepath.Join(root, name), []byte(source), 0o600); err != nil {
			return fmt.Errorf("write %s smoke source: %w", name, err)
		}
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
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return err
	}

	for _, testCase := range []struct {
		name         string
		source       string
		profile      string
		code         int
		stdoutPrefix string
		stderr       string
	}{
		{name: "success low", source: "main.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "success optimized", source: "main.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "negative low", source: "negative.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "negative optimized", source: "negative.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "trap low", source: "trap.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "trap optimized", source: "trap.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes and 6 run-host outcomes covered 42, -1, and division-by-zero across low/optimized\n", len(artifact))
	return nil
}

func runHostCase(root, name, source, profile string, wantCode int, wantStdoutPrefix, wantStderr string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{
		"experimental", "run-host", source, "--profile", profile,
	}, root, &stdout, &stderr)
	stdoutMatches := stdout.Len() == 0
	if wantStdoutPrefix != "" {
		stdoutMatches = strings.HasPrefix(stdout.String(), wantStdoutPrefix)
	}
	if code != wantCode || stderr.String() != wantStderr || !stdoutMatches {
		return fmt.Errorf("%s run-host returned code %d, stdout %q, stderr %q", name, code, stdout.String(), stderr.String())
	}
	return nil
}

func executeExpectResult(path string, expected int32) error {
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
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return fmt.Errorf("build-host artifact execution failed: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	actual := int32(uint32(exitCode))
	if actual != expected || output.Len() != 0 {
		exitCode := -1
		if command.ProcessState != nil {
			exitCode = command.ProcessState.ExitCode()
		}
		return fmt.Errorf("build-host artifact returned error %v, exit %d (i32 %d), output %q", err, exitCode, actual, output.String())
	}
	return nil
}
