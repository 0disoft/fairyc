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
	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/windowsabiprobe"
	"github.com/0disoft/fairyc/internal/windowslayoutprobe"
	"github.com/0disoft/fairyc/internal/windowsunwindprobe"
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

const boundsSource = `module bounds;

fn main() -> i32 {
    let value: u8 = b"A"[1];
    42
}
`

const checkedInSuccessPath = "examples/windows_smoke/main.fy"
const checkedInModuleMainPath = "examples/run/modules/main.fy"
const checkedInModuleSupportPath = "examples/run/modules/math.fy"
const checkedInWindowsTargetPath = "examples/windows_target/main.fy"
const checkedInWindowsQuickstartPath = "examples/windows_quickstart/main.fy"
const checkedInWindowsOutputPath = "examples/windows_output/main.fy"
const checkedInWindowsOutputSupportPath = "examples/windows_output/windows_console.fy"
const windowsLayoutManifestPath = "spec/fixtures/layout/manifest-windows-msvc-v0.json"
const windowsABIManifestPath = "spec/fixtures/c-abi/manifest-windows-msvc-v0.json"
const subprocessOutputLimit = 64 * 1024

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "fairyc Windows host smoke failed: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	repositoryRoot, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("resolve repository root: %w", err)
	}
	successSource, err := os.ReadFile(filepath.FromSlash(checkedInSuccessPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows smoke source: %w", err)
	}
	moduleMainSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleMainPath))
	if err != nil {
		return fmt.Errorf("read checked-in multi-source entry: %w", err)
	}
	moduleSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInModuleSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in multi-source support: %w", err)
	}
	windowsTargetSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsTargetPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows target source: %w", err)
	}
	windowsQuickstartSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsQuickstartPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows quickstart source: %w", err)
	}
	windowsOutputSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows output source: %w", err)
	}
	windowsOutputSupportSource, err := os.ReadFile(filepath.FromSlash(checkedInWindowsOutputSupportPath))
	if err != nil {
		return fmt.Errorf("read checked-in Windows output support source: %w", err)
	}
	root, err := os.MkdirTemp("", "fairyc-windows-host-smoke-")
	if err != nil {
		return fmt.Errorf("create temporary source root: %w", err)
	}
	defer os.RemoveAll(root)
	for name, source := range map[string]string{
		checkedInSuccessPath:              string(successSource),
		checkedInModuleMainPath:           string(moduleMainSource),
		checkedInModuleSupportPath:        string(moduleSupportSource),
		checkedInWindowsTargetPath:        string(windowsTargetSource),
		checkedInWindowsQuickstartPath:    string(windowsQuickstartSource),
		checkedInWindowsOutputPath:        string(windowsOutputSource),
		checkedInWindowsOutputSupportPath: string(windowsOutputSupportSource),
		"negative.fy":                     negativeSource,
		"division_by_zero.fy":             divisionByZeroSource,
		"remainder_by_zero.fy":            remainderByZeroSource,
		"signed_division_overflow.fy":     signedDivisionOverflowSource,
		"integer_overflow.fy":             integerOverflowSource,
		"invalid_shift.fy":                invalidShiftSource,
		"bounds.fy":                       boundsSource,
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
		"experimental", "build-host", checkedInModuleMainPath,
		"--source", checkedInModuleSupportPath,
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
	peReport, err := pecoff.InspectExecutable(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host PE executable image: %w", err)
	}
	unwindReport, err := pecoff.InspectAMD64Unwind(artifact)
	if err != nil {
		return fmt.Errorf("inspect build-host Microsoft x64 unwind data: %w", err)
	}
	if err := executeExpectResult(artifactPath, 42); err != nil {
		return err
	}

	targetBuildExecutions := 0
	for _, profile := range []string{"low", "optimized"} {
		output := fmt.Sprintf("build/windows-target-%s.exe", profile)
		var targetBuildStdout bytes.Buffer
		var targetBuildStderr bytes.Buffer
		targetBuildCode := cli.Run([]string{
			"experimental", "build-target", checkedInWindowsTargetPath,
			"--target", "x86_64-windows-msvc", "--profile", profile,
			"--output", output,
		}, root, &targetBuildStdout, &targetBuildStderr)
		if targetBuildCode != cli.ExitSuccess || targetBuildStderr.Len() != 0 || !strings.HasPrefix(targetBuildStdout.String(), "built: "+output+" (x86_64-windows-msvc, msvc ") {
			return fmt.Errorf("build-target %s returned code %d, stdout %q, stderr %q", profile, targetBuildCode, targetBuildStdout.String(), targetBuildStderr.String())
		}
		targetArtifactPath := filepath.Join(root, filepath.FromSlash(output))
		targetArtifact, err := os.ReadFile(targetArtifactPath)
		if err != nil {
			return fmt.Errorf("read build-target %s artifact: %w", profile, err)
		}
		if _, err := pecoff.InspectExecutable(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s PE executable image: %w", profile, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(targetArtifact); err != nil {
			return fmt.Errorf("inspect build-target %s Microsoft x64 unwind data: %w", profile, err)
		}
		if err := executeExpectResult(targetArtifactPath, 42); err != nil {
			return fmt.Errorf("execute build-target %s artifact: %w", profile, err)
		}
		targetBuildExecutions++
	}
	targetRunExecutions := 0
	var targetRunStdout bytes.Buffer
	var targetRunStderr bytes.Buffer
	targetRunCode := cli.Run([]string{
		"experimental", "run-target", checkedInWindowsTargetPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetRunStdout, &targetRunStderr)
	if targetRunCode != cli.ExitSuccess || targetRunStderr.Len() != 0 || !strings.HasPrefix(targetRunStdout.String(), "native result: 42 (x86_64-windows-msvc, msvc ") {
		return fmt.Errorf("run-target returned code %d, stdout %q, stderr %q", targetRunCode, targetRunStdout.String(), targetRunStderr.String())
	}
	targetRunExecutions++
	targetExecExecutions := 0
	var targetExecStdout bytes.Buffer
	var targetExecStderr bytes.Buffer
	targetExecCode := cli.Run([]string{
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	}, root, &targetExecStdout, &targetExecStderr)
	if targetExecCode != 42 || targetExecStdout.String() != "Fairy C says hi\n" || targetExecStderr.Len() != 0 {
		return fmt.Errorf("exec-target returned code %d, stdout %q, stderr %q", targetExecCode, targetExecStdout.String(), targetExecStderr.String())
	}
	targetExecExecutions++
	standaloneCompilerBytes, err := runStandaloneCompiler(repositoryRoot, root)
	if err != nil {
		return err
	}
	targetRunExecutions++
	targetExecExecutions++

	for _, testCase := range []struct {
		name         string
		source       string
		support      string
		profile      string
		code         int
		stdoutPrefix string
		stderr       string
	}{
		{name: "success low", source: checkedInSuccessPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "multi-source success optimized", source: checkedInModuleMainPath, support: checkedInModuleSupportPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "value-preserving cast low", source: checkedInWindowsQuickstartPath, profile: "low", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
		{name: "value-preserving cast optimized", source: checkedInWindowsQuickstartPath, profile: "optimized", code: cli.ExitSuccess, stdoutPrefix: "native result: 42 (windows/amd64 host-smoke, msvc "},
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
		{name: "byte-view bounds low", source: "bounds.fy", profile: "low", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
		{name: "byte-view bounds optimized", source: "bounds.fy", profile: "optimized", code: cli.ExitProgramTrap, stderr: "trap: bounds\n"},
	} {
		if err := runHostCase(root, testCase.name, testCase.source, testCase.support, testCase.profile, testCase.code, testCase.stdoutPrefix, testCase.stderr); err != nil {
			return err
		}
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return fmt.Errorf("resolve Windows MSVC probe toolchain: %w", err)
	}
	layoutExecutions, layoutCases, err := runWindowsLayoutProbe(toolchain)
	if err != nil {
		return err
	}
	abiExecutions, abiCases, err := runWindowsABIProbe(toolchain)
	if err != nil {
		return err
	}
	unwindExecutions, unwindCases, err := runWindowsUnwindProbe(toolchain)
	if err != nil {
		return err
	}
	fmt.Printf("Windows host smoke passed: build-host PE %d bytes with %d validated sections, %d runtime-function records, and %d version-2 unwind records, %d source-level Windows target builds, %d direct run-target executions, and %d exec-target executions checked LLP64 c.long emission, closed SystemHeap, Buffer, Arena/Region, String/CString/CStr, UTF-8 crossings, safe borrowed-view internal calls, byte lengths, checked byte indexing, and value-preserving integer casts, exact stdout bytes, and native result 42, including one %d-byte standalone Go-built fairyc.exe quickstart and output execution, 18 run-host outcomes covered the safe u8-to-i32 quickstart, signed results, and five trap categories, %d LLP64 layout cases passed %d MSVC probe executions, %d Microsoft x64 ABI cases passed %d separate-translation-unit MSVC probe executions, %d Windows x64 unwind cases passed %d actual exception-dispatch and virtual-unwind executions, %d PE32+ target or probe images passed structural and unwind inspection, and every MSVC compilation object passed COFF inspection across low/optimized with msvc %s\n", len(artifact), len(peReport.Sections), unwindReport.RuntimeFunctions, unwindReport.Version2Infos, targetBuildExecutions, targetRunExecutions, targetExecExecutions, standaloneCompilerBytes, layoutCases, layoutExecutions, abiCases, abiExecutions, unwindCases, unwindExecutions, 1+targetBuildExecutions+targetRunExecutions+targetExecExecutions+layoutExecutions+abiExecutions+unwindExecutions, toolchain.Version)
	return nil
}

func runStandaloneCompiler(repositoryRoot, sourceRoot string) (int64, error) {
	temporaryRoot, err := os.MkdirTemp("", "fairyc-standalone-compiler-smoke-")
	if err != nil {
		return 0, fmt.Errorf("create standalone compiler directory: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	executablePath := filepath.Join(temporaryRoot, "fairyc.exe")
	buildContext, cancelBuild := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancelBuild()
	build := exec.CommandContext(buildContext, "go", "build", "-mod=readonly", "-trimpath", "-o", executablePath, "./cmd/fairyc")
	build.Dir = repositoryRoot
	build.Stdin = nil
	var buildStdout boundedBuffer
	var buildStderr boundedBuffer
	buildStdout.limit = subprocessOutputLimit
	buildStderr.limit = subprocessOutputLimit
	build.Stdout = &buildStdout
	build.Stderr = &buildStderr
	if err := build.Run(); err != nil {
		if errors.Is(buildContext.Err(), context.DeadlineExceeded) {
			return 0, fmt.Errorf("standalone compiler build exceeded timeout")
		}
		return 0, fmt.Errorf("build standalone compiler: %w; stdout %q; stderr %q", err, buildStdout.String(), buildStderr.String())
	}
	if buildStdout.Len() != 0 || buildStderr.Len() != 0 {
		return 0, fmt.Errorf("standalone compiler build produced unexpected output: stdout %q; stderr %q", buildStdout.String(), buildStderr.String())
	}
	info, err := os.Stat(executablePath)
	if err != nil {
		return 0, fmt.Errorf("stat standalone compiler: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		return 0, fmt.Errorf("standalone compiler is not a nonempty regular file")
	}

	helpCode, helpStdout, helpStderr, err := runStandaloneCommand(sourceRoot, executablePath, "--help")
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler help: %w", err)
	}
	if helpCode != cli.ExitSuccess || helpStderr != "" ||
		!strings.Contains(helpStdout, "fairyc experimental run-target") ||
		!strings.Contains(helpStdout, "fairyc experimental exec-target") {
		return 0, fmt.Errorf("standalone compiler help returned code %d, stdout %q, stderr %q", helpCode, helpStdout, helpStderr)
	}

	runCode, runStdout, runStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "run-target", checkedInWindowsQuickstartPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler quickstart: %w", err)
	}
	if runCode != cli.ExitSuccess || runStderr != "" || !strings.HasPrefix(runStdout, "native result: 42 (x86_64-windows-msvc, msvc ") {
		return 0, fmt.Errorf("standalone compiler run-target returned code %d, stdout %q, stderr %q", runCode, runStdout, runStderr)
	}

	execCode, execStdout, execStderr, err := runStandaloneCommand(
		sourceRoot,
		executablePath,
		"experimental", "exec-target", checkedInWindowsOutputPath,
		"--source", checkedInWindowsOutputSupportPath,
		"--target", "x86_64-windows-msvc", "--profile", "optimized",
	)
	if err != nil {
		return 0, fmt.Errorf("execute standalone compiler output program: %w", err)
	}
	if execCode != 42 || execStdout != "Fairy C says hi\n" || execStderr != "" {
		return 0, fmt.Errorf("standalone compiler exec-target returned code %d, stdout %q, stderr %q", execCode, execStdout, execStderr)
	}
	return info.Size(), nil
}

func runStandaloneCommand(directory, executablePath string, arguments ...string) (int, string, string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath, arguments...)
	command.Dir = directory
	command.Stdin = nil
	var stdout boundedBuffer
	var stderr boundedBuffer
	stdout.limit = subprocessOutputLimit
	stderr.limit = subprocessOutputLimit
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return -1, stdout.String(), stderr.String(), fmt.Errorf("command exceeded timeout")
	}
	if err == nil {
		return 0, stdout.String(), stderr.String(), nil
	}
	var exitError *exec.ExitError
	if !errors.As(err, &exitError) {
		return -1, stdout.String(), stderr.String(), err
	}
	return exitError.ExitCode(), stdout.String(), stderr.String(), nil
}

type boundedBuffer struct {
	buffer    bytes.Buffer
	limit     int
	truncated bool
}

func (buffer *boundedBuffer) Write(value []byte) (int, error) {
	originalLength := len(value)
	remaining := buffer.limit - buffer.buffer.Len()
	if remaining > 0 {
		if len(value) > remaining {
			value = value[:remaining]
		}
		_, _ = buffer.buffer.Write(value)
	}
	if originalLength > remaining {
		buffer.truncated = true
	}
	return originalLength, nil
}

func (buffer *boundedBuffer) Len() int {
	return buffer.buffer.Len()
}

func (buffer *boundedBuffer) String() string {
	if buffer.truncated {
		return buffer.buffer.String() + "\n[output truncated]"
	}
	return buffer.buffer.String()
}

func runWindowsUnwindProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit := windowsunwindprobe.Generate()
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCUnwindProbe(toolchain, profile, unit.CatcherSource, unit.FrameSource, unit.RaiserSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 3 || artifact.Objects[0].Name != "catcher.obj" || artifact.Objects[1].Name != "frame.obj" || artifact.Objects[2].Name != "raiser.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe PE executable image with %s: %w", profile.Name, err)
		}
		report, err := pecoff.InspectAMD64Unwind(artifact.Bytes)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC unwind probe unwind data with %s: %w", profile.Name, err)
		}
		if report.HandlerInfos == 0 {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s has no handler-bearing unwind records", profile.Name)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCUnwindProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC unwind probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC unwind probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runWindowsLayoutProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowslayoutprobe.Generate(filepath.FromSlash(windowsLayoutManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC layout probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCProbe(toolchain, profile, unit.Source)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 1 || artifact.Objects[0].Name != "probe.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC layout probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC layout probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC layout probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runWindowsABIProbe(toolchain cdriver.WindowsHostToolchain) (int, int, error) {
	unit, err := windowsabiprobe.Generate(filepath.FromSlash(windowsABIManifestPath))
	if err != nil {
		return 0, 0, fmt.Errorf("generate Windows MSVC ABI probe: %w", err)
	}
	executions := 0
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		artifact, err := cdriver.BuildWindowsMSVCABIProbe(toolchain, profile, unit.CallerSource, unit.CalleeSource)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("build Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if len(artifact.Objects) != 2 || artifact.Objects[0].Name != "caller.obj" || artifact.Objects[1].Name != "callee.obj" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned unexpected COFF object receipts", profile.Name)
		}
		if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe PE executable image with %s: %w", profile.Name, err)
		}
		if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
			return executions, len(unit.Cases), fmt.Errorf("inspect Windows MSVC ABI probe unwind data with %s: %w", profile.Name, err)
		}
		outcome, err := cdriver.ExecuteWindowsMSVCABIProbe(artifact)
		if err != nil {
			return executions, len(unit.Cases), fmt.Errorf("execute Windows MSVC ABI probe with %s: %w", profile.Name, err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			return executions, len(unit.Cases), fmt.Errorf("Windows MSVC ABI probe with %s returned %d, stdout %q, and stderr %q", profile.Name, outcome.ExitCode, outcome.Stdout, outcome.Stderr)
		}
		executions++
	}
	return executions, len(unit.Cases), nil
}

func runHostCase(root, name, source, support, profile string, wantCode int, wantStdoutPrefix, wantStderr string) error {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	arguments := []string{"experimental", "run-host", source}
	if support != "" {
		arguments = append(arguments, "--source", support)
	}
	arguments = append(arguments, "--profile", profile)
	code := cli.Run(arguments, root, &stdout, &stderr)
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
