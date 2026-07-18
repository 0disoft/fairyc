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
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v1"
	trapExitBase        = 192
	maxProcessOutput    = 64 * 1024
	executionTimeout    = 10 * time.Second
)

var nativeProfiles = []cdriver.Profile{
	cdriver.LowProfile,
	cdriver.OptimizedProfile,
}

const buildSmokeSource = `module main;
resource Token {
    value: i32;
    drop fn(self: &mut Self) {}
}
fn choose() -> Option<i32> { Option::Some(41) }
fn increment(input: Result<i32, i32>) -> Result<i32, i32> {
    let value: i32 = input?;
    Result::Ok(value + 1)
}
fn main() -> i32 {
    let base: i32 = match choose() {
        Option::Some(value) => value,
        Option::None => 0,
    };
    let value: i32 = match increment(Result::Ok(base)) {
        Result::Ok(value) => value,
        Result::Err(error) => -error,
    };
    let token: Token = Token { value: value };
    consume(token)
}

fn consume(token: Token) -> i32 { token.value }
`

// Compiler records the bounded toolchain fingerprint used by one native run.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Report is the deterministic success record emitted by the native fixture command.
type Report struct {
	SchemaVersion        string     `json:"schemaVersion"`
	Target               string     `json:"target"`
	Libc                 string     `json:"libc"`
	Compilers            []Compiler `json:"compilers"`
	Profiles             []string   `json:"profiles"`
	Cases                []string   `json:"cases"`
	Executions           int        `json:"executions"`
	BuildSmokeExecutions int        `json:"buildSmokeExecutions"`
}

// Run compiles every emission fixture with Clang and GCC at low and optimized profiles, compares
// the Linux process outcome with the MIR-interpreter oracle, and exercises the product build CLI.
func Run(manifestPath string) (Report, error) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		return Report{}, fmt.Errorf("native C differential requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	absoluteManifest, err := filepath.Abs(manifestPath)
	if err != nil {
		return Report{}, fmt.Errorf("resolve manifest: %w", err)
	}
	fixtureReport, err := cbackendfixture.Run(absoluteManifest)
	if err != nil {
		return Report{}, err
	}
	if len(fixtureReport.Failures) != 0 || len(fixtureReport.Emitted) != fixtureReport.CasesRun {
		return Report{}, fmt.Errorf("C emission fixtures are not clean: %s", summarizeFailures(fixtureReport.Failures))
	}
	runtimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind scalar runtime: %w", err)
	}
	toolchains, err := resolveToolchains()
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
		Target:        string(target.X8664LinuxGNU),
		Libc:          toolchains[0].Libc,
	}
	for _, toolchain := range toolchains {
		if toolchain.Libc != report.Libc {
			return Report{}, fmt.Errorf("C toolchains disagree on libc: %q and %q", report.Libc, toolchain.Libc)
		}
		report.Compilers = append(report.Compilers, Compiler{Name: toolchain.Name, Version: toolchain.Version})
	}
	for _, profile := range nativeProfiles {
		report.Profiles = append(report.Profiles, profile.Name)
	}
	for caseIndex, fixture := range fixtureReport.Emitted {
		if fixture.Unit.Target != string(target.X8664LinuxGNU) || fixture.Unit.Dialect != cbackend.DialectC11 {
			return Report{}, fmt.Errorf("case %s emitted unexpected profile %s/%s", fixture.ID, fixture.Unit.Target, fixture.Unit.Dialect)
		}
		report.Cases = append(report.Cases, fixture.ID)
		expectedExit, err := expectedExitCode(fixture)
		if err != nil {
			return Report{}, err
		}
		for compilerIndex, toolchain := range toolchains {
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.Build(toolchain, profile, fixture.Unit.Source, runtimeSource)
				if err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("program-%02d-%d-%d", caseIndex, compilerIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				if err := execute(fixture.ID, executablePath, expectedExit); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				report.Executions++
			}
		}
	}
	buildSmoke, err := runBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.BuildSmokeExecutions = buildSmoke
	return report, nil
}

func runBuildSmoke(temporaryRoot string) (int, error) {
	sourceRoot := filepath.Join(temporaryRoot, "product-build-smoke")
	if err := os.Mkdir(sourceRoot, 0o700); err != nil {
		return 0, fmt.Errorf("create product build smoke root: %w", err)
	}
	if err := os.WriteFile(filepath.Join(sourceRoot, "main.fy"), []byte(buildSmokeSource), 0o600); err != nil {
		return 0, fmt.Errorf("write product build smoke source: %w", err)
	}
	executions := 0
	for _, compiler := range []string{"clang", "gcc"} {
		for _, profile := range nativeProfiles {
			output := fmt.Sprintf("build/smoke-%s-%s", compiler, profile.Name)
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			code := cli.Run([]string{
				"experimental", "build", "main.fy",
				"--cc", compiler,
				"--profile", profile.Name,
				"--output", output,
			}, sourceRoot, &stdout, &stderr)
			if code != cli.ExitSuccess {
				return 0, fmt.Errorf("product build smoke with %s/%s failed with code %d: %s", compiler, profile.Name, code, strings.TrimSpace(stderr.String()))
			}
			expectedReceipt := fmt.Sprintf("built: %s (%s, %s, %s)\n", output, target.X8664LinuxGNU, compiler, profile.Name)
			if stdout.String() != expectedReceipt || stderr.Len() != 0 {
				return 0, fmt.Errorf("product build smoke with %s/%s produced stdout %q and stderr %q", compiler, profile.Name, stdout.String(), stderr.String())
			}
			if err := execute("product-build-smoke", filepath.Join(sourceRoot, filepath.FromSlash(output)), 42); err != nil {
				return 0, fmt.Errorf("product build smoke with %s/%s: %w", compiler, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
}

func resolveToolchains() ([]cdriver.Toolchain, error) {
	toolchains := make([]cdriver.Toolchain, 0, 2)
	for _, name := range []string{"clang", "gcc"} {
		toolchain, err := cdriver.Resolve(target.X8664LinuxGNU, name)
		if err != nil {
			return nil, fmt.Errorf("native C differential requires %s: %w", name, err)
		}
		toolchains = append(toolchains, toolchain)
	}
	return toolchains, nil
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
