// Package cdriver resolves and invokes the declared constrained-C toolchain.
package cdriver

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/target"
)

const (
	maxProcessOutput = 64 * 1024
	maxArtifactBytes = 64 * 1024 * 1024
	commandTimeout   = 60 * time.Second

	// AddressUndefinedSanitizerMode identifies the validation-only ASan+UBSan build mode.
	AddressUndefinedSanitizerMode = "address-undefined-o1"
)

// Profile is one closed Stage 0 C compiler profile.
type Profile struct {
	Name             string
	optimizationFlag string
}

// ConsumerMode identifies the C language mode used to validate one unchanged
// generated translation unit. It is separate from the emission dialect,
// target ABI, optimization profile, and validation instrumentation.
type ConsumerMode string

const (
	// C11ConsumerMode is the normative Stage 0 transport consumer mode.
	C11ConsumerMode ConsumerMode = "c11"
	// C17ConsumerMode validates forward compatibility of identical C11 source.
	C17ConsumerMode ConsumerMode = "c17"
)

var (
	// LowProfile preserves the low-optimization differential and development path.
	LowProfile = Profile{Name: "low", optimizationFlag: "-O0"}
	// OptimizedProfile exercises ordinary optimization without LTO.
	OptimizedProfile = Profile{Name: "optimized", optimizationFlag: "-O2"}
	sanitizerProfile = Profile{Name: AddressUndefinedSanitizerMode, optimizationFlag: "-O1"}
)

// Toolchain is one resolved local C compiler plus its target-libc fingerprint.
type Toolchain struct {
	Target  target.ID
	Name    string
	Path    string
	Version string
	Libc    string
}

// Artifact is one bounded linked executable image and its build identity.
type Artifact struct {
	Bytes        []byte
	Toolchain    Toolchain
	Profile      Profile
	ConsumerMode ConsumerMode
}

// CrossArtifact is one executable whose caller and callee objects came from different compilers.
type CrossArtifact struct {
	Bytes           []byte
	CallerToolchain Toolchain
	CalleeToolchain Toolchain
	Profile         Profile
	ConsumerMode    ConsumerMode
}

// Resolve requires the declared Linux/amd64 glibc host and one closed compiler identity.
func Resolve(targetID target.ID, name string) (Toolchain, error) {
	if name != "clang" && name != "gcc" {
		return Toolchain{}, fmt.Errorf("unsupported C compiler %q; expected clang or gcc", name)
	}
	spec, found := target.Lookup(targetID)
	if !found {
		return Toolchain{}, fmt.Errorf("unsupported C target %q", targetID)
	}
	if err := target.Validate(spec); err != nil {
		return Toolchain{}, fmt.Errorf("invalid C target %q: %w", targetID, err)
	}
	if runtime.GOOS != spec.OperatingSystem || runtime.GOARCH != "amd64" {
		return Toolchain{}, fmt.Errorf("C toolchain requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	getconf, err := exec.LookPath("getconf")
	if err != nil {
		return Toolchain{}, fmt.Errorf("x86_64-linux-gnu requires getconf: %w", err)
	}
	libcOutput, err := runBounded(getconf, "GNU_LIBC_VERSION")
	if err != nil {
		return Toolchain{}, fmt.Errorf("probe glibc: %w", err)
	}
	libc := strings.TrimSpace(libcOutput)
	if !strings.HasPrefix(libc, "glibc ") {
		return Toolchain{}, fmt.Errorf("target requires glibc, getconf reported %q", libc)
	}
	path, err := exec.LookPath(name)
	if err != nil {
		return Toolchain{}, fmt.Errorf("resolve %s: %w", name, err)
	}
	versionOutput, err := runBounded(path, "--version")
	if err != nil {
		return Toolchain{}, fmt.Errorf("probe %s: %w", name, err)
	}
	version := firstLine(versionOutput)
	if version == "" {
		return Toolchain{}, fmt.Errorf("probe %s returned an empty version", name)
	}
	return Toolchain{Target: targetID, Name: name, Path: path, Version: version, Libc: libc}, nil
}

// Build compiles and links one generated translation unit with one embedded runtime source.
func Build(toolchain Toolchain, profile Profile, unitSource, runtimeSource string) (Artifact, error) {
	if profile != LowProfile && profile != OptimizedProfile {
		return Artifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	return build(toolchain, profile, nil, unitSource, runtimeSource)
}

// BuildSanitized compiles and links one generated translation unit for the closed native
// conformance lane. It is deliberately separate from product optimization profiles.
func BuildSanitized(toolchain Toolchain, unitSource, runtimeSource string) (Artifact, error) {
	return build(toolchain, sanitizerProfile, sanitizerInstrumentationFlags(), unitSource, runtimeSource)
}

// BuildCrossCompiler compiles caller and callee translation units with different closed
// toolchains, then links them through the caller driver for native C ABI conformance checks.
func BuildCrossCompiler(caller, callee Toolchain, profile Profile, callerSource, calleeSource string) (CrossArtifact, error) {
	return buildCrossCompiler(caller, callee, profile, callerSource, calleeSource, "")
}

// BuildCrossCompilerWithRuntime adds one caller-compiled runtime translation unit to the same
// closed cross-compiler link. This keeps generated caller support separate from the foreign callee.
func BuildCrossCompilerWithRuntime(caller, callee Toolchain, profile Profile, callerSource, calleeSource, runtimeSource string) (CrossArtifact, error) {
	if runtimeSource == "" {
		return CrossArtifact{}, fmt.Errorf("C ABI cross build runtime source must be non-empty")
	}
	return buildCrossCompiler(caller, callee, profile, callerSource, calleeSource, runtimeSource)
}

func buildCrossCompiler(caller, callee Toolchain, profile Profile, callerSource, calleeSource, runtimeSource string) (CrossArtifact, error) {
	if profile != LowProfile && profile != OptimizedProfile {
		return CrossArtifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	if err := validateCrossToolchains(caller, callee); err != nil {
		return CrossArtifact{}, err
	}
	if callerSource == "" || calleeSource == "" {
		return CrossArtifact{}, fmt.Errorf("C ABI caller and callee sources must be non-empty")
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-cdriver-cross-")
	if err != nil {
		return CrossArtifact{}, fmt.Errorf("create cross-compiler temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	callerPath := filepath.Join(temporaryRoot, "caller.c")
	calleePath := filepath.Join(temporaryRoot, "callee.c")
	callerObject := filepath.Join(temporaryRoot, "caller.o")
	calleeObject := filepath.Join(temporaryRoot, "callee.o")
	runtimePath := filepath.Join(temporaryRoot, "runtime.c")
	runtimeObject := filepath.Join(temporaryRoot, "runtime.o")
	executablePath := filepath.Join(temporaryRoot, "program")
	if err := os.WriteFile(callerPath, []byte(callerSource), 0o600); err != nil {
		return CrossArtifact{}, fmt.Errorf("write C ABI caller: %w", err)
	}
	if err := os.WriteFile(calleePath, []byte(calleeSource), 0o600); err != nil {
		return CrossArtifact{}, fmt.Errorf("write C ABI callee: %w", err)
	}
	if _, err := runBounded(caller.Path, objectArguments(profile, callerPath, callerObject)...); err != nil {
		return CrossArtifact{}, fmt.Errorf("compile caller with %s: %s", caller.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	if _, err := runBounded(callee.Path, objectArguments(profile, calleePath, calleeObject)...); err != nil {
		return CrossArtifact{}, fmt.Errorf("compile callee with %s: %s", callee.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	linkInputs := []string{callerObject, calleeObject}
	if runtimeSource != "" {
		if err := os.WriteFile(runtimePath, []byte(runtimeSource), 0o600); err != nil {
			return CrossArtifact{}, fmt.Errorf("write C ABI caller runtime: %w", err)
		}
		if _, err := runBounded(caller.Path, objectArguments(profile, runtimePath, runtimeObject)...); err != nil {
			return CrossArtifact{}, fmt.Errorf("compile caller runtime with %s: %s", caller.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		linkInputs = append(linkInputs, runtimeObject)
	}
	linkArguments := append(linkInputs, "-o", executablePath)
	if _, err := runBounded(caller.Path, linkArguments...); err != nil {
		return CrossArtifact{}, fmt.Errorf("link cross-compiler executable with %s: %s", caller.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	contents, err := readArtifact(executablePath)
	if err != nil {
		return CrossArtifact{}, err
	}
	return CrossArtifact{Bytes: contents, CallerToolchain: caller, CalleeToolchain: callee, Profile: profile, ConsumerMode: C11ConsumerMode}, nil
}

func validateCrossToolchains(caller, callee Toolchain) error {
	if caller.Target != callee.Target || caller.Target == "" {
		return fmt.Errorf("C ABI caller and callee targets do not match")
	}
	spec, found := target.Lookup(caller.Target)
	if !found {
		return fmt.Errorf("C ABI cross build has unsupported target %q", caller.Target)
	}
	if runtime.GOOS != spec.OperatingSystem || runtime.GOARCH != "amd64" {
		return fmt.Errorf("C ABI cross build requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if caller.Name != "clang" && caller.Name != "gcc" || callee.Name != "clang" && callee.Name != "gcc" ||
		caller.Name == callee.Name || caller.Path == "" || callee.Path == "" {
		return fmt.Errorf("C ABI cross build requires distinct resolved Clang and GCC toolchains")
	}
	if caller.Libc == "" || caller.Libc != callee.Libc {
		return fmt.Errorf("C ABI caller and callee libc fingerprints do not match")
	}
	return nil
}

func objectArguments(profile Profile, sourcePath, objectPath string) []string {
	return []string{
		"-std=c11", "-pedantic-errors", "-Wall", "-Wextra", "-Werror", "-fno-common",
		profile.optimizationFlag, "-c", sourcePath, "-o", objectPath,
	}
}

func sanitizerInstrumentationFlags() []string {
	return []string{
		"-fsanitize=address,undefined",
		"-fno-omit-frame-pointer",
		"-fno-sanitize-recover=all",
	}
}

func build(toolchain Toolchain, profile Profile, instrumentationFlags []string, unitSource, runtimeSource string) (Artifact, error) {
	spec, found := target.Lookup(toolchain.Target)
	if !found {
		return Artifact{}, fmt.Errorf("C build has unsupported target %q", toolchain.Target)
	}
	if runtime.GOOS != spec.OperatingSystem || runtime.GOARCH != "amd64" {
		return Artifact{}, fmt.Errorf("C build requires linux/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if toolchain.Name != "clang" && toolchain.Name != "gcc" || toolchain.Path == "" {
		return Artifact{}, fmt.Errorf("invalid resolved C toolchain")
	}
	if unitSource == "" || runtimeSource == "" {
		return Artifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-cdriver-")
	if err != nil {
		return Artifact{}, fmt.Errorf("create C driver temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	unitPath := filepath.Join(temporaryRoot, "unit.c")
	runtimePath := filepath.Join(temporaryRoot, "runtime.c")
	executablePath := filepath.Join(temporaryRoot, "program")
	if err := os.WriteFile(unitPath, []byte(unitSource), 0o600); err != nil {
		return Artifact{}, fmt.Errorf("write C unit: %w", err)
	}
	if err := os.WriteFile(runtimePath, []byte(runtimeSource), 0o600); err != nil {
		return Artifact{}, fmt.Errorf("write C runtime: %w", err)
	}
	arguments := buildArguments(profile, instrumentationFlags, unitPath, runtimePath, executablePath)
	if _, err := runBounded(toolchain.Path, arguments...); err != nil {
		return Artifact{}, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	contents, err := readArtifact(executablePath)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Bytes: contents, Toolchain: toolchain, Profile: profile, ConsumerMode: C11ConsumerMode}, nil
}

func buildArguments(profile Profile, instrumentationFlags []string, unitPath, runtimePath, executablePath string) []string {
	arguments := []string{
		"-std=c11",
		"-pedantic-errors",
		"-Wall",
		"-Wextra",
		"-Werror",
		"-fno-common",
		profile.optimizationFlag,
	}
	arguments = append(arguments, instrumentationFlags...)
	return append(arguments, unitPath, runtimePath, "-o", executablePath)
}

func readArtifact(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open linked executable: %w", err)
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("inspect linked executable: %w", err)
	}
	if !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > maxArtifactBytes {
		return nil, fmt.Errorf("linked executable has invalid size %d", info.Size())
	}
	contents, err := io.ReadAll(io.LimitReader(file, maxArtifactBytes+1))
	if err != nil {
		return nil, fmt.Errorf("read linked executable: %w", err)
	}
	if len(contents) == 0 || len(contents) > maxArtifactBytes {
		return nil, fmt.Errorf("linked executable has invalid size %d", len(contents))
	}
	return contents, nil
}

func runBounded(name string, arguments ...string) (string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, name, arguments...)
	command.Stdin = nil
	command.Env = append(os.Environ(), "LC_ALL=C", "LANG=C")
	var output boundedBuffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return output.String(), fmt.Errorf("C toolchain command exceeded %s", commandTimeout)
	}
	if output.truncated {
		return output.String(), fmt.Errorf("C toolchain command exceeded %d output bytes", maxProcessOutput)
	}
	if err != nil {
		return output.String(), fmt.Errorf("C toolchain command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}
	return output.String(), nil
}

func firstLine(value string) string {
	line, _, _ := strings.Cut(strings.ReplaceAll(value, "\r\n", "\n"), "\n")
	return strings.TrimSpace(line)
}

type boundedBuffer struct {
	bytes     []byte
	truncated bool
}

func (b *boundedBuffer) Write(value []byte) (int, error) {
	available := maxProcessOutput - len(b.bytes)
	if available > 0 {
		b.bytes = append(b.bytes, value[:min(available, len(value))]...)
	}
	if len(value) > available {
		b.truncated = true
	}
	return len(value), nil
}

func (b *boundedBuffer) String() string {
	return string(b.bytes)
}
