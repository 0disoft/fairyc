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

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/windowslayoutprobe"
)

const negativeSource = `module negative;

fn main() -> i32 {
    -1
}
`

const divisionByZeroSource = `module division_by_zero;

fn main() -> i32 {
    42 / 0
}
`

const remainderByZeroSource = `module remainder_by_zero;

fn main() -> i32 {
    42 % 0
}
`

const signedDivisionOverflowSource = `module signed_division_overflow;

fn main() -> i32 {
    (-2147483647 - 1) / -1
}
`

const integerOverflowSource = `module integer_overflow;

fn main() -> i32 {
    2147483647 + 1
}
`

const invalidShiftSource = `module invalid_shift;

fn main() -> i32 {
    1 >> -1
}
`

const checkedInSuccessPath = "examples/windows_smoke/main.fy"
const windowsLayoutManifestPath = "spec/fixtures/layout/manifest-windows-msvc-v0.json"

const maxSmokeArtifactBytes = 64 * 1024 * 1024

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fairyc Windows host smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	successSource, err := os.ReadFile(filepath.FromSlash(checkedInSuccessPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows smoke source: %w", err)
	}
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	for name, source := range map[string]string{
		checkedInSuccessPath:          string(successSource),
		"negative.fy":                 negativeSource,
		"division_by_zero.fy":         divisionByZeroSource,
		"remainder_by_zero.fy":        remainderByZeroSource,
		"signed_division_overflow.fy": signedDivisionOverflowSource,
		"integer_overflow.fy":         integerOverflowSource,
		"invalid_shift.fy":            invalidShiftSource,
	} {
		destination := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return fmt.Errorf("create %s smoke source directory: %w", name, err)
		}
		if err := os.WriteFile(destination, []byte(source), 0o600); err != nil {
			return fmt.Errorf("write %s smoke source: %w", name, err)
		}
	}

	var buildStdout bytes.Buffer
	var buildStderr bytes.Buffer
	buildCode := cli.Run([]string{
		"experimental", "build-host", checkedInSuccessPath,
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
		{name: "success low", source: checkedInSuccessPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "success optimized", source: checkedInSuccessPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "negative low", source: "negative.fy", profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "negative optimized", source: "negative.fy", profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: -1 (windows/amd64 host-smoke, msvc "},
		{name: "division by zero low", source: "division_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "division by zero optimized", source: "division_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero low", source: "remainder_by_zero.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "remainder by zero optimized", source: "remainder_by_zero.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: division-by-zero\n"},
		{name: "signed division overflow low", source: "signed_division_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "signed division overflow optimized", source: "signed_division_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: signed-division-overflow\n"},
		{name: "integer overflow low", source: "integer_overflow.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "integer overflow optimized", source: "integer_overflow.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: integer-overflow\n"},
		{name: "invalid shift low", source: "invalid_shift.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
		{name: "invalid shift optimized", source: "invalid_shift.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: invalid-shift\n"},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	toolchainVersion, layoutExecutions, layoutCases, err := runWindowsLayoutProbe()
	if err != nil {
		return err
	}
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes, 14 run-host outcomes covered signed results and four arithmetic trap categories, and %d LLP64 layout cases passed %d MSVC probe executions across low/optimized with msvc %s\n", len(artifact), layoutCases, layoutExecutions, toolchainVersion)
	return nil
}

func runWindowsLayoutProbe() (string, int, int, error) {
	unit, err := windowslayoutprobe.Generate(filepath.FromSlash(windowsLayoutManifestPath))
	if err != nil {
		return "", 0, 0, fmt.Errorf("generate Windows MSVC layout probe: %w", err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return "", 0, 0, fmt.Errorf("resolve Windows MSVC layout probe toolchain: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, unit.Source)
		if err != nil {
			return "", executions, len(unit.Cases), fmt.Errorf("build Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return "", executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Output != "" {
			return "", executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned %d and output %q", profile.Name, outcome.ExitCode, outcome.Output)
		}
		executions++
	}
	return toolchain.Version, executions, len(unit.Cases), nil
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
