// Package cbackendnative compiles and executes constrained C fixtures on the declared Linux target.
package cbackendnative

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v0"
	trapExitBase        = 192
	maxProcessOutput    = 64 * 1024
	compilerTimeout     = 30 * time.Second
	executionTimeout    = 10 * time.Second
)

type compilerSpec struct {
	Name string
	Path string
}

type profileSpec struct {
	Name string
	Flag string
}

var nativeProfiles = []profileSpec{
	{Name: "low", Flag: "-O0"},
	{Name: "optimized", Flag: "-O2"},
}

// Compiler records the bounded toolchain fingerprint used by one native run.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Report is the deterministic success record emitted by the native fixture command.
type Report struct {
	SchemaVersion string     `json:"schemaVersion"`
	Target        string     `json:"target"`
	Libc          string     `json:"libc"`
	Compilers     []Compiler `json:"compilers"`
	Profiles      []string   `json:"profiles"`
	Cases         []string   `json:"cases"`
	Executions    int        `json:"executions"`
}

// Run compiles every emission fixture with Clang and GCC at low and optimized profiles, then
// compares the Linux process outcome with the MIR-interpreter outcome captured by the fixture runner.
func Run(manifestPath string) (Report, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Report{}, fmt.Errorf("native C differential requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	runtimeSource := filepath.Join(filepath.Dir(absoluteManifest), "native", "runtime.c")
	if info, err := os.Stat(runtimeSource); err != nil || !info.Mode().IsRegular() {
		if err == nil {
			err = fmt.Errorf("not a regular file")
		}
		return Report{}, fmt.Errorf("native test runtime: %w", err)
	}

	fixtureReport, err := cbackendfixture.Run(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if len(fixtureReport.Failures) != 0 || len(fixtureReport.Emitted) != fixtureReport.CasesRun {
		return Report{}, fmt.Errorf("C emission fixtures are not clean: %s", summarizeFailures(fixtureReport.Failures))
	}

	libc, err := probeLibc()
	if err != nil {
		return Report{}, err
	}
	compilers, compilerReports, err := probeCompilers()
	if err != nil {
		return Report{}, err
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-native-c-")
	if err != nil {
		return Report{}, fmt.Errorf("create native fixture temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)

	report := Report{
		SchemaVersion: reportSchemaVersion,
		Target:        cbackend.TargetX8664LinuxGNU,
		Libc:          libc,
		Compilers:     compilerReports,
	}
	for _, profile := range nativeProfiles {
		report.Profiles = append(report.Profiles, profile.Name)
	}
	for caseIndex, fixture := range fixtureReport.Emitted {
		if fixture.Unit.Target != cbackend.TargetX8664LinuxGNU || fixture.Unit.Dialect != cbackend.DialectC11 {
			return Report{}, fmt.Errorf("case %s emitted unexpected profile %s/%s", fixture.ID, fixture.Unit.Target, fixture.Unit.Dialect)
		}
		report.Cases = append(report.Cases, fixture.ID)
		caseDirectory := filepath.Join(temporaryRoot, fmt.Sprintf("case-%02d", caseIndex))
		if err := os.Mkdir(caseDirectory, 0o700); err != nil {
			return Report{}, fmt.Errorf("create case directory: %w", err)
		}
		sourcePath := filepath.Join(caseDirectory, "unit.c")
		if err := os.WriteFile(sourcePath, []byte(fixture.Unit.Source), 0o600); err != nil {
			return Report{}, fmt.Errorf("write generated C for %s: %w", fixture.ID, err)
		}
		expectedExit, err := expectedExitCode(fixture)
		if err != nil {
			return Report{}, err
		}
		for compilerIndex, compiler := range compilers {
			for profileIndex, profile := range nativeProfiles {
				executablePath := filepath.Join(caseDirectory, fmt.Sprintf("program-%d-%d", compilerIndex, profileIndex))
				if err := compile(compiler, profile, sourcePath, runtimeSource, executablePath); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, compiler.Name, profile.Name, err)
				}
				if err := execute(fixture.ID, executablePath, expectedExit); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, compiler.Name, profile.Name, err)
				}
				report.Executions++
			}
		}
	}
	return report, nil
}

func probeLibc() (string, error) {
	path, err := exec.LookPath("getconf")
	if err != nil {
		return "", fmt.Errorf("x86_64-linux-gnu probe requires getconf: %w", err)
	}
	output, err := runBounded(compilerTimeout, path, "GNU_LIBC_VERSION")
	if err != nil {
		return "", fmt.Errorf("probe glibc: %w", err)
	}
	value := strings.TrimSpace(output)
	if !strings.HasPrefix(value, "glibc ") {
		return "", fmt.Errorf("target requires glibc, getconf reported %q", value)
	}
	return value, nil
}

func probeCompilers() ([]compilerSpec, []Compiler, error) {
	var specs []compilerSpec
	var reports []Compiler
	for _, name := range []string{"clang", "gcc"} {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, nil, fmt.Errorf("native C differential requires %s: %w", name, err)
		}
		output, err := runBounded(compilerTimeout, path, "--version")
		if err != nil {
			return nil, nil, fmt.Errorf("probe %s: %w", name, err)
		}
		version := firstLine(output)
		if version == "" {
			return nil, nil, fmt.Errorf("probe %s returned an empty version", name)
		}
		specs = append(specs, compilerSpec{Name: name, Path: path})
		reports = append(reports, Compiler{Name: name, Version: version})
	}
	return specs, reports, nil
}

func compile(compiler compilerSpec, profile profileSpec, sourcePath, runtimeSource, executablePath string) error {
	arguments := []string{
		"-std=c11",
		"-pedantic-errors",
		"-Wall",
		"-Wextra",
		"-Werror",
		"-fno-common",
		profile.Flag,
		fmt.Sprintf("-DFY_TEST_TRAP_EXIT_BASE=%d", trapExitBase),
		sourcePath,
		runtimeSource,
		"-o",
		executablePath,
	}
	_, err := runBounded(compilerTimeout, compiler.Path, arguments...)
	return err
}

func execute(caseID, executablePath string, expectedExit int) error {
	contextValue, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	var output boundedBuffer
	output.limit = maxProcessOutput
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return fmt.Errorf("native executable exceeded %s", executionTimeout)
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return fmt.Errorf("execute native program: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	if output.truncated {
		return fmt.Errorf("native executable exceeded %d output bytes", maxProcessOutput)
	}
	if len(output.bytes) != 0 {
		return fmt.Errorf("native executable produced unexpected output %q", string(output.bytes))
	}
	if exitCode != expectedExit {
		return fmt.Errorf("native exit for %s: got %d, want %d", caseID, exitCode, expectedExit)
	}
	return nil
}

func expectedExitCode(fixture cbackendfixture.EmittedCase) (int, error) {
	switch fixture.Outcome {
	case "return":
		if fixture.Value < 0 || fixture.Value >= trapExitBase {
			return 0, fmt.Errorf("native process-status oracle cannot represent return %d in %s", fixture.Value, fixture.ID)
		}
		return int(fixture.Value), nil
	case "trap":
		code, ok := cbackend.TrapCode(fixture.TrapCategory)
		if !ok {
			return 0, fmt.Errorf("unknown trap category %q in %s", fixture.TrapCategory, fixture.ID)
		}
		return trapExitBase + int(code), nil
	default:
		return 0, fmt.Errorf("unsupported native outcome %q in %s", fixture.Outcome, fixture.ID)
	}
}

func runBounded(timeout time.Duration, name string, arguments ...string) (string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(contextValue, name, arguments...)
	command.Stdin = nil
	var output boundedBuffer
	output.limit = maxProcessOutput
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return string(output.bytes), fmt.Errorf("command exceeded %s", timeout)
	}
	if output.truncated {
		return string(output.bytes), fmt.Errorf("command exceeded %d output bytes", maxProcessOutput)
	}
	if err != nil {
		return string(output.bytes), fmt.Errorf("command failed: %w: %s", err, strings.TrimSpace(string(output.bytes)))
	}
	return string(output.bytes), nil
}

func summarizeFailures(failures []cbackendfixture.Failure) string {
	if len(failures) == 0 {
		return "emitted case count mismatch"
	}
	values := make([]string, 0, len(failures))
	for _, failure := range failures {
		values = append(values, failure.CaseID+": "+failure.Detail)
	}
	slices.Sort(values)
	return strings.Join(values, "; ")
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	return strings.TrimSpace(line)
}

type boundedBuffer struct {
	limit     int
	bytes     []byte
	truncated bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	available := b.limit - len(b.bytes)
	if available > 0 {
		b.bytes = append(b.bytes, value[:min(available, len(value))]...)
	}
	if len(value) > available {
		b.truncated = true
	}
	return len(value), nil
}

func (b *boundedBuffer) String() string {
	return string(bytes.Clone(b.bytes))
}
