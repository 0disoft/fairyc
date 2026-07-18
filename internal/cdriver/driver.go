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
)

// Profile is one closed Stage 0 C compiler profile.
type Profile struct {
	Name             string
	optimizationFlag string
}

var (
	// LowProfile preserves the low-optimization differential and development path.
	LowProfile = Profile{Name: "low", optimizationFlag: "-O0"}
	// OptimizedProfile exercises ordinary optimization without LTO.
	OptimizedProfile = Profile{Name: "optimized", optimizationFlag: "-O2"}
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
	Bytes     []byte
	Toolchain Toolchain
	Profile   Profile
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
	if profile != LowProfile && profile != OptimizedProfile {
		return Artifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
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
	arguments := []string{
		"-std=c11",
		"-pedantic-errors",
		"-Wall",
		"-Wextra",
		"-Werror",
		"-fno-common",
		profile.optimizationFlag,
		unitPath,
		runtimePath,
		"-o",
		executablePath,
	}
	if _, err := runBounded(toolchain.Path, arguments...); err != nil {
		return Artifact{}, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	contents, err := readArtifact(executablePath)
	if err != nil {
		return Artifact{}, err
	}
	return Artifact{Bytes: contents, Toolchain: toolchain, Profile: profile}, nil
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
