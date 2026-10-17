package cdriver

import (
	"runtime"
	"testing"
)

func TestWindowsHostBuildOptionsAndCompatibilityWrappers(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	const unit = `int answer(void) { return 42; }`
	const support = `int answer(void); int main(void) { return answer() != 42; }`
	tests := []struct {
		name                    string
		build                   func() (WindowsHostArtifact, error)
		mapFile, debug, library bool
	}{
		{"ordinary", func() (WindowsHostArtifact, error) {
			return BuildWindowsHost(toolchain, OptimizedProfile, unit, support)
		}, false, false, false},
		{"map", func() (WindowsHostArtifact, error) {
			return BuildWindowsHostWithMap(toolchain, OptimizedProfile, unit, support)
		}, true, false, false},
		{"debug-map", func() (WindowsHostArtifact, error) {
			return BuildWindowsHostWithDebugMap(toolchain, OptimizedProfile, unit, support)
		}, true, true, false},
		{"library", func() (WindowsHostArtifact, error) {
			return BuildWindowsHostWithLibraryLog(toolchain, OptimizedProfile, unit, support)
		}, true, false, true},
		{"combined", func() (WindowsHostArtifact, error) {
			options := debugMapBuildOptions()
			options.CaptureLibraryLog = true
			return BuildWindowsHostWithOptions(toolchain, OptimizedProfile, unit, support, options)
		}, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			artifact, err := tt.build()
			if err != nil {
				t.Fatal(err)
			}
			if (len(artifact.LinkMap) != 0) != tt.mapFile || (len(artifact.ProgramDatabase) != 0) != tt.debug || (len(artifact.LinkerVerbose) != 0) != tt.library {
				t.Fatal("build output policy mismatch")
			}
			outcome, err := ExecuteWindowsHost(artifact)
			if err != nil || outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
				t.Fatalf("build execution failed: %v", err)
			}
		})
	}
}
