// Package cbackendnative compiles and executes constrained C fixtures on the declared Linux target.
package cbackendnative

import (
	"bytes"
	"context"
	"encoding/binary"
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
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	reportSchemaVersion = "fairyc.native-c-report/v4"
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
fn choose() -> Option<Token> { Option::Some(Token { value: 41 }) }
fn increment(input: Result<Token, i32>) -> Result<Token, i32> {
	let token: Token = input?;
	Result::Ok(Token { value: token.value + 1 })
}
fn main() -> i32 {
	let token: Token = match choose() {
		Option::Some(token) => token,
		Option::None => Token { value: 0 },
	};
	match increment(Result::Ok(token)) {
		Result::Ok(token) => token.value,
		Result::Err(error) => -error,
    }
}
`

// Compiler records the bounded toolchain fingerprint used by one native run.
type Compiler struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// Report is the deterministic success record emitted by the native fixture command.
type Report struct {
	SchemaVersion         string     `json:"schemaVersion"`
	Target                string     `json:"target"`
	Libc                  string     `json:"libc"`
	Compilers             []Compiler `json:"compilers"`
	Profiles              []string   `json:"profiles"`
	Cases                 []string   `json:"cases"`
	Executions            int        `json:"executions"`
	DropTraceExecutions   int        `json:"dropTraceExecutions"`
	LayoutCases           []string   `json:"layoutCases"`
	LayoutProbeExecutions int        `json:"layoutProbeExecutions"`
	Sanitizer             string     `json:"sanitizer"`
	SanitizerExecutions   int        `json:"sanitizerExecutions"`
	BuildSmokeExecutions  int        `json:"buildSmokeExecutions"`
}

// Run compiles every emission fixture with Clang and GCC at low and optimized profiles, compares
// the Linux process outcome and source-linked drop trace with the MIR-interpreter oracle, executes
// every production unit under the closed address/undefined sanitizer mode, and exercises the
// product build CLI.
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
	traceRuntimeSource, err := cruntime.DropTraceSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind scalar runtime: %w", err)
	}
	scalarRuntimeSource, err := cruntime.ScalarSource(trapExitBase)
	if err != nil {
		return Report{}, fmt.Errorf("bind uninstrumented scalar runtime: %w", err)
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
		Sanitizer:     cdriver.AddressUndefinedSanitizerMode,
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
	layoutManifest := filepath.Clean(filepath.Join(filepath.Dir(absoluteManifest), "..", "layout", "manifest-v0.json"))
	layoutTarget, layoutCases, err := layoutfixture.NativeProbeCases(layoutManifest)
	if err != nil {
		return Report{}, fmt.Errorf("load native layout probe cases: %w", err)
	}
	if layoutTarget != target.X8664LinuxGNU {
		return Report{}, fmt.Errorf("native layout probe target is %q, want %q", layoutTarget, target.X8664LinuxGNU)
	}
	layoutProbe, err := generateLayoutProbe(layoutTarget, layoutCases)
	if err != nil {
		return Report{}, fmt.Errorf("generate native layout probe: %w", err)
	}
	report.LayoutCases = slices.Clone(layoutProbe.Cases)
	layoutExecutions, err := runLayoutProbe(temporaryRoot, toolchains, layoutProbe)
	if err != nil {
		return Report{}, err
	}
	report.LayoutProbeExecutions = layoutExecutions
	for caseIndex, fixture := range fixtureReport.Emitted {
		if fixture.Unit.Target != string(target.X8664LinuxGNU) || fixture.Unit.Dialect != cbackend.DialectC11 {
			return Report{}, fmt.Errorf("case %s emitted unexpected profile %s/%s", fixture.ID, fixture.Unit.Target, fixture.Unit.Dialect)
		}
		report.Cases = append(report.Cases, fixture.ID)
		expectedExit, err := expectedExitCode(fixture)
		if err != nil {
			return Report{}, err
		}
		expectedDrops, err := expectedDropSiteIDs(fixture)
		if err != nil {
			return Report{}, err
		}
		for compilerIndex, toolchain := range toolchains {
			for profileIndex, profile := range nativeProfiles {
				artifact, err := cdriver.Build(toolchain, profile, fixture.TraceUnit.Source, traceRuntimeSource)
				if err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("program-%02d-%d-%d", caseIndex, compilerIndex, profileIndex))
				if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
					return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				if err := execute(fixture.ID, executablePath, expectedExit, expectedDrops); err != nil {
					return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, profile.Name, err)
				}
				report.Executions++
				report.DropTraceExecutions++
			}
			sanitizedArtifact, err := cdriver.BuildSanitized(toolchain, fixture.Unit.Source, scalarRuntimeSource)
			if err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			sanitizedPath := filepath.Join(temporaryRoot, fmt.Sprintf("sanitized-%02d-%d", caseIndex, compilerIndex))
			if err := os.WriteFile(sanitizedPath, sanitizedArtifact.Bytes, 0o700); err != nil {
				return Report{}, fmt.Errorf("materialize %s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			if err := executeSanitized(fixture.ID, sanitizedPath, expectedExit); err != nil {
				return Report{}, fmt.Errorf("%s with %s/%s: %w", fixture.ID, toolchain.Name, cdriver.AddressUndefinedSanitizerMode, err)
			}
			report.SanitizerExecutions++
		}
	}
	buildSmoke, err := runBuildSmoke(temporaryRoot)
	if err != nil {
		return Report{}, err
	}
	report.BuildSmokeExecutions = buildSmoke
	return report, nil
}

func runLayoutProbe(temporaryRoot string, toolchains []cdriver.Toolchain, probe layoutProbeUnit) (int, error) {
	executions := 0
	for compilerIndex, toolchain := range toolchains {
		for profileIndex, profile := range nativeProfiles {
			artifact, err := cdriver.Build(toolchain, profile, probe.Source, layoutProbeRuntimeSource)
			if err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executablePath := filepath.Join(temporaryRoot, fmt.Sprintf("layout-probe-%d-%d", compilerIndex, profileIndex))
			if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
				return 0, fmt.Errorf("materialize layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			if err := execute("layout-probe", executablePath, 0, nil); err != nil {
				return 0, fmt.Errorf("layout probe with %s/%s: %w", toolchain.Name, profile.Name, err)
			}
			executions++
		}
	}
	return executions, nil
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
			if err := execute("product-build-smoke", filepath.Join(sourceRoot, filepath.FromSlash(output)), 42, nil); err != nil {
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

func execute(caseID, executablePath string, expectedExit int, expectedDrops []uint32) error {
	result, err := executeProcess(executablePath, nil)
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("native executable produced unexpected stdout %q", string(result.stdout))
	}
	actualDrops, err := decodeDropTrace(result.stderr)
	if err != nil {
		return fmt.Errorf("native drop trace for %s: %w", caseID, err)
	}
	if !slices.Equal(actualDrops, expectedDrops) {
		return fmt.Errorf("native drop trace for %s: got %v, want %v", caseID, actualDrops, expectedDrops)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("native exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

func executeSanitized(caseID, executablePath string, expectedExit int) error {
	result, err := executeProcess(executablePath, sanitizerEnvironment(os.Environ()))
	if err != nil {
		return err
	}
	if len(result.stdout) != 0 {
		return fmt.Errorf("sanitized executable produced unexpected stdout %q", string(result.stdout))
	}
	if len(result.stderr) != 0 {
		diagnostic := strings.ReplaceAll(strings.TrimSpace(string(result.stderr)), filepath.Dir(executablePath), "<temporary>")
		return fmt.Errorf("sanitizer reported for %s: %s", caseID, diagnostic)
	}
	if result.exitCode != expectedExit {
		return fmt.Errorf("sanitized exit for %s: got %d, want %d", caseID, result.exitCode, expectedExit)
	}
	return nil
}

type processResult struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func executeProcess(executablePath string, environment []string) (processResult, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), executionTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	if environment != nil {
		command.Env = environment
	}
	stdout := boundedBuffer{limit: maxProcessOutput}
	stderr := boundedBuffer{limit: maxProcessOutput}
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return processResult{}, fmt.Errorf("native executable exceeded %s", executionTimeout)
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return processResult{}, fmt.Errorf("execute native program: %w", err)
		}
		exitCode = exitError.ExitCode()
	}
	if stdout.truncated || stderr.truncated || len(stdout.bytes)+len(stderr.bytes) > maxProcessOutput {
		return processResult{}, fmt.Errorf("native executable exceeded %d output bytes", maxProcessOutput)
	}
	return processResult{exitCode: exitCode, stdout: bytes.Clone(stdout.bytes), stderr: bytes.Clone(stderr.bytes)}, nil
}

func sanitizerEnvironment(base []string) []string {
	result := make([]string, 0, len(base)+3)
	for _, entry := range base {
		name, _, _ := strings.Cut(entry, "=")
		if name == "ASAN_OPTIONS" || name == "LSAN_OPTIONS" || name == "UBSAN_OPTIONS" {
			continue
		}
		result = append(result, entry)
	}
	return append(
		result,
		"ASAN_OPTIONS=abort_on_error=1:color=never:detect_leaks=1:halt_on_error=1:symbolize=0",
		"LSAN_OPTIONS=exitcode=23:report_objects=0",
		"UBSAN_OPTIONS=color=never:halt_on_error=1:print_stacktrace=0",
	)
}

func expectedDropSiteIDs(fixture cbackendfixture.EmittedCase) ([]uint32, error) {
	result := make([]uint32, 0, len(fixture.DropTrace))
	for _, event := range fixture.DropTrace {
		found := false
		for _, site := range fixture.TraceUnit.DropSites {
			if site.ResourceType != string(event.ResourceType) || site.Source != event.Binding.Source ||
				site.StartByte != event.Binding.StartByte || site.EndByte != event.Binding.EndByte {
				continue
			}
			result = append(result, site.ID)
			found = true
			break
		}
		if !found {
			return nil, fmt.Errorf("drop trace for %s has no C site for %s at %s:%d-%d", fixture.ID, event.ResourceType, event.Binding.Source, event.Binding.StartByte, event.Binding.EndByte)
		}
	}
	return result, nil
}

func decodeDropTrace(contents []byte) ([]uint32, error) {
	if len(contents)%4 != 0 {
		return nil, fmt.Errorf("record stream has %d bytes, want a multiple of 4", len(contents))
	}
	result := make([]uint32, 0, len(contents)/4)
	for offset := 0; offset < len(contents); offset += 4 {
		result = append(result, binary.LittleEndian.Uint32(contents[offset:offset+4]))
	}
	return result, nil
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
