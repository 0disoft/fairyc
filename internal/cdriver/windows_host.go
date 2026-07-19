package cdriver

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

// WindowsHostToolchain is one locally resolved MSVC installation for target-neutral host smoke.
// It is deliberately not a Fairy C target or ABI-conformance identity.
type WindowsHostToolchain struct {
	Name        string
	Path        string
	Version     string
	Environment []string
}

// WindowsHostArtifact is one bounded PE executable produced for local MSVC validation.
// It is not a distributable target artifact or a target-conformance receipt.
type WindowsHostArtifact struct {
	Bytes     []byte
	Toolchain WindowsHostToolchain
	Profile   Profile
}

// WindowsHostOutcome is one bounded local MSVC execution result. ExitCode preserves the program's i32 bits.
type WindowsHostOutcome struct {
	ExitCode int32
	Output   string
}

// ResolveWindowsHostMSVC locates the latest complete x64 MSVC and Windows SDK installation.
func ResolveWindowsHostMSVC() (WindowsHostToolchain, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostToolchain{}, fmt.Errorf("MSVC host smoke requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	programFilesX86, err := resolveProgramFilesX86()
	if err != nil {
		return WindowsHostToolchain{}, err
	}
	vswhere := filepath.Join(programFilesX86, "Microsoft Visual Studio", "Installer", "vswhere.exe")
	if info, err := os.Stat(vswhere); err != nil || !info.Mode().IsRegular() {
		return WindowsHostToolchain{}, fmt.Errorf("MSVC host smoke requires Visual Studio Installer vswhere.exe")
	}
	installationOutput, err := runWindowsHostCommand(vswhere, nil, "",
		"-latest", "-products", "*", "-requires", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath")
	if err != nil {
		return WindowsHostToolchain{}, fmt.Errorf("locate Visual Studio: %w", err)
	}
	installation := strings.TrimSpace(installationOutput)
	if installation == "" || !filepath.IsAbs(installation) {
		return WindowsHostToolchain{}, fmt.Errorf("Visual Studio Installer returned an invalid installation path")
	}
	versionPath := filepath.Join(installation, "VC", "Auxiliary", "Build", "Microsoft.VCToolsVersion.default.txt")
	versionBytes, err := os.ReadFile(versionPath)
	if err != nil {
		return WindowsHostToolchain{}, fmt.Errorf("read MSVC tool version: %w", err)
	}
	version := strings.TrimSpace(string(versionBytes))
	if version == "" || strings.ContainsAny(version, "\\/") {
		return WindowsHostToolchain{}, fmt.Errorf("MSVC tool version file is invalid")
	}
	msvcRoot := filepath.Join(installation, "VC", "Tools", "MSVC", version)
	compilerPath := filepath.Join(msvcRoot, "bin", "Hostx64", "x64", "cl.exe")
	if info, err := os.Stat(compilerPath); err != nil || !info.Mode().IsRegular() {
		return WindowsHostToolchain{}, fmt.Errorf("MSVC x64 compiler is missing for tool version %s", version)
	}
	sdkRoot := filepath.Join(programFilesX86, "Windows Kits", "10")
	sdkVersion, err := latestWindowsSDKVersion(sdkRoot)
	if err != nil {
		return WindowsHostToolchain{}, err
	}
	environment, err := windowsHostEnvironment(os.Environ(), msvcRoot, sdkRoot, sdkVersion)
	if err != nil {
		return WindowsHostToolchain{}, err
	}
	return WindowsHostToolchain{Name: "msvc", Path: compilerPath, Version: version, Environment: environment}, nil
}

func resolveProgramFilesX86() (string, error) {
	var candidates []string
	if value := os.Getenv("ProgramFiles(x86)"); value != "" {
		candidates = append(candidates, value)
	}
	if value := os.Getenv("ProgramFiles"); value != "" {
		candidates = append(candidates, value+" (x86)")
	}
	if executable, err := os.Executable(); err == nil {
		if volume := filepath.VolumeName(executable); volume != "" {
			candidates = append(candidates, filepath.Join(volume+string(os.PathSeparator), "Program Files (x86)"))
		}
	}
	seen := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		clean := filepath.Clean(candidate)
		key := strings.ToLower(clean)
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		vswhere := filepath.Join(clean, "Microsoft Visual Studio", "Installer", "vswhere.exe")
		if info, err := os.Stat(vswhere); err == nil && info.Mode().IsRegular() {
			return clean, nil
		}
	}
	return "", fmt.Errorf("MSVC host smoke cannot locate Program Files (x86) with Visual Studio Installer")
}

// BuildWindowsHost compiles generated C for a target-neutral Windows host smoke only.
func BuildWindowsHost(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC host build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVC(toolchain, profile, []windowsMSVCSource{
		{Name: "unit.c", Contents: unitSource},
		{Name: "runtime.c", Contents: runtimeSource},
	})
}

// BuildWindowsMSVCProbe compiles one closed target-evidence probe with local x64 MSVC.
func BuildWindowsMSVCProbe(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if probeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe source must be non-empty")
	}
	return buildWindowsMSVC(toolchain, profile, []windowsMSVCSource{{Name: "probe.c", Contents: probeSource}})
}

// BuildWindowsMSVCABIProbe compiles one caller and one callee translation unit
// separately before linking them with local x64 MSVC.
func BuildWindowsMSVCABIProbe(toolchain WindowsHostToolchain, profile Profile, callerSource, calleeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC ABI probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if callerSource == "" || calleeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC ABI caller and callee sources must be non-empty")
	}
	return buildWindowsMSVC(toolchain, profile, []windowsMSVCSource{
		{Name: "caller.c", Contents: callerSource},
		{Name: "callee.c", Contents: calleeSource},
	})
}

type windowsMSVCSource struct {
	Name     string
	Contents string
}

func buildWindowsMSVC(toolchain WindowsHostToolchain, profile Profile, sources []windowsMSVCSource) (WindowsHostArtifact, error) {
	if profile != LowProfile && profile != OptimizedProfile {
		return WindowsHostArtifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return WindowsHostArtifact{}, fmt.Errorf("invalid resolved MSVC host toolchain")
	}
	if len(sources) == 0 {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC build has no C sources")
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-msvc-host-")
	if err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("create MSVC host temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	executablePath := filepath.Join(temporaryRoot, "program.exe")
	sourcePaths := make([]string, 0, len(sources))
	for _, source := range sources {
		if source.Name == "" || filepath.Base(source.Name) != source.Name || filepath.Ext(source.Name) != ".c" || source.Contents == "" {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC build source is invalid")
		}
		path := filepath.Join(temporaryRoot, source.Name)
		if err := os.WriteFile(path, []byte(source.Contents), 0o600); err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("write MSVC source %s: %w", source.Name, err)
		}
		sourcePaths = append(sourcePaths, path)
	}
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	arguments := []string{
		"/nologo", "/std:c11", "/TC", "/W4", "/WX", "/MT", optimization,
	}
	arguments = append(arguments, sourcePaths...)
	arguments = append(arguments, "/Fe:"+executablePath, "/link", "/INCREMENTAL:NO")
	if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, temporaryRoot, arguments...); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	contents, err := readArtifact(executablePath)
	if err != nil {
		return WindowsHostArtifact{}, err
	}
	return WindowsHostArtifact{Bytes: contents, Toolchain: toolchain, Profile: profile}, nil
}

// ExecuteWindowsHost runs one temporary PE and returns its exact i32 exit bits and bounded output.
func ExecuteWindowsHost(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC host execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC host")
}

// ExecuteWindowsMSVCProbe runs one locally compiled target-evidence probe.
func ExecuteWindowsMSVCProbe(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC layout probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC layout probe")
}

// ExecuteWindowsMSVCABIProbe runs one linked caller/callee ABI probe.
func ExecuteWindowsMSVCABIProbe(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC ABI probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC ABI probe")
}

func executeWindowsMSVCArtifact(artifact WindowsHostArtifact, label string) (WindowsHostOutcome, error) {
	if len(artifact.Bytes) == 0 || len(artifact.Bytes) > maxArtifactBytes {
		return WindowsHostOutcome{}, fmt.Errorf("%s executable has invalid size %d", label, len(artifact.Bytes))
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-msvc-run-")
	if err != nil {
		return WindowsHostOutcome{}, fmt.Errorf("create %s execution temp root: %w", label, err)
	}
	defer os.RemoveAll(temporaryRoot)
	executablePath := filepath.Join(temporaryRoot, "program.exe")
	if err := os.WriteFile(executablePath, artifact.Bytes, 0o700); err != nil {
		return WindowsHostOutcome{}, fmt.Errorf("materialize MSVC host executable: %w", err)
	}
	contextValue, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	command.Stdin = nil
	command.Dir = temporaryRoot
	var output boundedBuffer
	command.Stdout = &output
	command.Stderr = &output
	err = command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return WindowsHostOutcome{}, fmt.Errorf("%s executable exceeded %s", label, commandTimeout)
	}
	if output.truncated {
		return WindowsHostOutcome{}, fmt.Errorf("%s executable exceeded %d output bytes", label, maxProcessOutput)
	}
	exitCode := 0
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return WindowsHostOutcome{}, fmt.Errorf("execute %s artifact: %w", label, err)
		}
		exitCode = exitError.ExitCode()
	}
	return WindowsHostOutcome{ExitCode: int32(uint32(exitCode)), Output: output.String()}, nil
}

func latestWindowsSDKVersion(sdkRoot string) (string, error) {
	includeRoot := filepath.Join(sdkRoot, "Include")
	entries, err := os.ReadDir(includeRoot)
	if err != nil {
		return "", fmt.Errorf("read Windows SDK include root: %w", err)
	}
	var candidates []string
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "10.") {
			continue
		}
		version := entry.Name()
		required := []string{
			filepath.Join(sdkRoot, "Include", version, "ucrt"),
			filepath.Join(sdkRoot, "Include", version, "shared"),
			filepath.Join(sdkRoot, "Include", version, "um"),
			filepath.Join(sdkRoot, "Lib", version, "ucrt", "x64"),
			filepath.Join(sdkRoot, "Lib", version, "um", "x64"),
		}
		complete := true
		for _, path := range required {
			if info, statErr := os.Stat(path); statErr != nil || !info.IsDir() {
				complete = false
				break
			}
		}
		if complete {
			candidates = append(candidates, version)
		}
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("MSVC host smoke requires a complete Windows 10/11 SDK")
	}
	slices.Sort(candidates)
	return candidates[len(candidates)-1], nil
}

func windowsHostEnvironment(base []string, msvcRoot, sdkRoot, sdkVersion string) ([]string, error) {
	msvcBin := filepath.Join(msvcRoot, "bin", "Hostx64", "x64")
	sdkBin := filepath.Join(sdkRoot, "bin", sdkVersion, "x64")
	include := []string{
		filepath.Join(msvcRoot, "include"),
		filepath.Join(sdkRoot, "Include", sdkVersion, "ucrt"),
		filepath.Join(sdkRoot, "Include", sdkVersion, "shared"),
		filepath.Join(sdkRoot, "Include", sdkVersion, "um"),
	}
	library := []string{
		filepath.Join(msvcRoot, "lib", "x64"),
		filepath.Join(sdkRoot, "Lib", sdkVersion, "ucrt", "x64"),
		filepath.Join(sdkRoot, "Lib", sdkVersion, "um", "x64"),
	}
	for _, path := range append(append([]string{msvcBin, sdkBin}, include...), library...) {
		info, err := os.Stat(path)
		if err != nil || !info.IsDir() {
			return nil, fmt.Errorf("MSVC host toolchain directory is missing: %s", path)
		}
	}
	environment := slices.Clone(base)
	environment = replaceWindowsEnvironment(environment, "PATH", strings.Join([]string{msvcBin, sdkBin, environmentValue(base, "PATH")}, string(os.PathListSeparator)))
	environment = replaceWindowsEnvironment(environment, "INCLUDE", strings.Join(include, string(os.PathListSeparator)))
	environment = replaceWindowsEnvironment(environment, "LIB", strings.Join(library, string(os.PathListSeparator)))
	return environment, nil
}

func environmentValue(environment []string, key string) string {
	for _, entry := range environment {
		name, value, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(name, key) {
			return value
		}
	}
	return ""
}

func replaceWindowsEnvironment(environment []string, key, value string) []string {
	result := make([]string, 0, len(environment)+1)
	for _, entry := range environment {
		name, _, found := strings.Cut(entry, "=")
		if found && strings.EqualFold(name, key) {
			continue
		}
		result = append(result, entry)
	}
	return append(result, key+"="+value)
}

func runWindowsHostCommand(name string, environment []string, directory string, arguments ...string) (string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, name, arguments...)
	command.Stdin = nil
	if len(environment) != 0 {
		command.Env = environment
	}
	command.Dir = directory
	var output boundedBuffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return output.String(), fmt.Errorf("MSVC host command exceeded %s", commandTimeout)
	}
	if output.truncated {
		return output.String(), fmt.Errorf("MSVC host command exceeded %d output bytes", maxProcessOutput)
	}
	if err != nil {
		return output.String(), fmt.Errorf("MSVC host command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}
	return output.String(), nil
}
