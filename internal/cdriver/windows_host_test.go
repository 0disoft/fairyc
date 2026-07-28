package cdriver

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestExecuteWindowsMSVCTargetWithInputRejectsNilInput(t *testing.T) {
	t.Parallel()
	if _, err := ExecuteWindowsMSVCTargetWithInput(WindowsHostArtifact{}, nil); err == nil ||
		!strings.Contains(err.Error(), "input must not be nil") {
		t.Fatalf("ExecuteWindowsMSVCTargetWithInput(nil) error = %v", err)
	}
}

func TestLatestWindowsSDKVersionRequiresCompleteHighestVersion(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	for _, version := range []string{"10.0.19041.0", "10.0.26100.0"} {
		for _, relative := range []string{
			filepath.Join("Include", version, "ucrt"),
			filepath.Join("Include", version, "shared"),
			filepath.Join("Include", version, "um"),
			filepath.Join("Lib", version, "ucrt", "x64"),
			filepath.Join("Lib", version, "um", "x64"),
		} {
			if err := os.MkdirAll(filepath.Join(root, relative), 0o755); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := os.MkdirAll(filepath.Join(root, "Include", "10.0.99999.0", "ucrt"), 0o755); err != nil {
		t.Fatal(err)
	}
	version, err := latestWindowsSDKVersion(root)
	if err != nil {
		t.Fatal(err)
	}
	if version != "10.0.26100.0" {
		t.Fatalf("version = %q", version)
	}
}

func TestWindowsHostEnvironmentIsClosedAndReplacesInheritedValues(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	msvcRoot := filepath.Join(root, "msvc")
	sdkRoot := filepath.Join(root, "sdk")
	version := "10.0.26100.0"
	for _, directory := range []string{
		filepath.Join(msvcRoot, "bin", "Hostx64", "x64"),
		filepath.Join(msvcRoot, "include"),
		filepath.Join(msvcRoot, "lib", "x64"),
		filepath.Join(sdkRoot, "bin", version, "x64"),
		filepath.Join(sdkRoot, "Include", version, "ucrt"),
		filepath.Join(sdkRoot, "Include", version, "shared"),
		filepath.Join(sdkRoot, "Include", version, "um"),
		filepath.Join(sdkRoot, "Lib", version, "ucrt", "x64"),
		filepath.Join(sdkRoot, "Lib", version, "um", "x64"),
	} {
		if err := os.MkdirAll(directory, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	environment, err := windowsHostEnvironment([]string{"Path=old-path", "INCLUDE=old-include", "LIB=old-lib", "KEEP=value"}, msvcRoot, sdkRoot, version)
	if err != nil {
		t.Fatal(err)
	}
	if environmentValue(environment, "INCLUDE") == "old-include" || environmentValue(environment, "LIB") == "old-lib" || !strings.HasSuffix(environmentValue(environment, "PATH"), "old-path") || environmentValue(environment, "KEEP") != "value" {
		t.Fatalf("environment = %q", environment)
	}
}

func TestWindowsHostDriverFailsClosedOnIneligibleHost(t *testing.T) {
	if runtime.GOOS == "windows" && runtime.GOARCH == "amd64" {
		t.Skip("live MSVC resolution belongs to the configured Windows host-smoke intent")
	}
	if _, err := ResolveWindowsHostMSVC(); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("ResolveWindowsHostMSVC() error = %v", err)
	}
	if _, err := BuildWindowsHost(WindowsHostToolchain{}, LowProfile, "unit", "runtime"); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("BuildWindowsHost() error = %v", err)
	}
	if _, err := BuildWindowsMSVCTarget(WindowsHostToolchain{}, LowProfile, "unit", "runtime"); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("BuildWindowsMSVCTarget() error = %v", err)
	}
	if _, err := BuildWindowsMSVCProbe(WindowsHostToolchain{}, LowProfile, "probe"); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("BuildWindowsMSVCProbe() error = %v", err)
	}
	if _, err := BuildWindowsMSVCABIProbe(WindowsHostToolchain{}, LowProfile, "caller", "callee"); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("BuildWindowsMSVCABIProbe() error = %v", err)
	}
	if _, err := BuildWindowsMSVCUnwindProbe(WindowsHostToolchain{}, LowProfile, "catcher", "frame", "raiser"); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("BuildWindowsMSVCUnwindProbe() error = %v", err)
	}
	if _, err := ExecuteWindowsHost(WindowsHostArtifact{Bytes: []byte("image")}); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("ExecuteWindowsHost() error = %v", err)
	}
	if _, err := ExecuteWindowsMSVCProbe(WindowsHostArtifact{Bytes: []byte("image")}); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("ExecuteWindowsMSVCProbe() error = %v", err)
	}
	if _, err := ExecuteWindowsMSVCABIProbe(WindowsHostArtifact{Bytes: []byte("image")}); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("ExecuteWindowsMSVCABIProbe() error = %v", err)
	}
	if _, err := ExecuteWindowsMSVCUnwindProbe(WindowsHostArtifact{Bytes: []byte("image")}); err == nil || !strings.Contains(err.Error(), "requires windows/amd64") {
		t.Fatalf("ExecuteWindowsMSVCUnwindProbe() error = %v", err)
	}
}

func TestMSVCStandardFlagKeepsConsumerModesClosed(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		mode ConsumerMode
		want string
	}{
		{mode: C11ConsumerMode, want: "/std:c11"},
		{mode: C17ConsumerMode, want: "/std:c17"},
	} {
		got, err := msvcStandardFlag(testCase.mode)
		if err != nil || got != testCase.want {
			t.Fatalf("msvcStandardFlag(%q) = %q, %v", testCase.mode, got, err)
		}
	}
	if _, err := msvcStandardFlag("c23"); err == nil || !strings.Contains(err.Error(), "unsupported C consumer mode") {
		t.Fatalf("msvcStandardFlag(c23) error = %v", err)
	}
}

func TestResolveProgramFilesX86FindsVisualStudioInstallerWhenAvailable(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Program Files discovery is Windows-specific")
	}
	root, err := resolveProgramFilesX86()
	if err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(root, "Microsoft Visual Studio", "Installer", "vswhere.exe")); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("Visual Studio Installer discovery = %q, %v", root, err)
	}
}

func TestBoundedProcessOutputSeparatesStreamsUnderSharedLimit(t *testing.T) {
	t.Parallel()
	output := newBoundedProcessOutput(5)

	if written, err := output.stdoutWriter().Write([]byte("abcd")); err != nil || written != 4 {
		t.Fatalf("stdout Write() = %d, %v", written, err)
	}
	if written, err := output.stderrWriter().Write([]byte("xyz")); err != nil || written != 3 {
		t.Fatalf("stderr Write() = %d, %v", written, err)
	}
	stdout, stderr, truncated := output.values()
	if stdout != "abcd" || stderr != "x" || !truncated {
		t.Fatalf("output = stdout %q, stderr %q, truncated %t", stdout, stderr, truncated)
	}
}

func TestBoundedProcessOutputAcceptsExactSharedLimit(t *testing.T) {
	t.Parallel()
	output := newBoundedProcessOutput(5)

	_, _ = output.stdoutWriter().Write([]byte("abc"))
	_, _ = output.stderrWriter().Write([]byte("de"))
	stdout, stderr, truncated := output.values()
	if stdout != "abc" || stderr != "de" || truncated {
		t.Fatalf("output = stdout %q, stderr %q, truncated %t", stdout, stderr, truncated)
	}
}
