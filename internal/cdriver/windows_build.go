package cdriver

import (
	"fmt"
	"runtime"
)

// WindowsMSVCBuildOptions selects bounded build outputs and link behavior.
// Zero preserves the ordinary build. Prebuilt objects retain the shared builder's
// existing structural validation; these options introduce no CLI flags.
type WindowsMSVCBuildOptions struct {
	CaptureMap               bool
	CaptureLibraryLog        bool
	CaptureProgramDatabase   bool
	DisableICF               bool
	EmbeddedCodeView         bool
	PreserveOptimizedLinking bool
	PrebuiltObjects          []WindowsHostObject
}

// BuildWindowsHost compiles generated C for a target-neutral Windows host smoke.
func BuildWindowsHost(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsHostWithOptions(toolchain, profile, unitSource, runtimeSource, WindowsMSVCBuildOptions{})
}

// BuildWindowsHostWithOptions is the common host-unit build boundary.
func BuildWindowsHostWithOptions(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string, options WindowsMSVCBuildOptions) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC host build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{{Name: "unit.c", Contents: unitSource}, {Name: "runtime.c", Contents: runtimeSource}}, options)
}

// BuildWindowsHostWithMap retains a link map through the shared options path.
func BuildWindowsHostWithMap(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsHostWithOptions(toolchain, profile, unitSource, runtimeSource, WindowsMSVCBuildOptions{CaptureMap: true})
}

// BuildWindowsHostWithDebugMap retains the existing PDB and map profile.
func BuildWindowsHostWithDebugMap(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsHostWithOptions(toolchain, profile, unitSource, runtimeSource, debugMapBuildOptions())
}

// BuildWindowsHostWithLibraryLog retains bounded library-selection output.
func BuildWindowsHostWithLibraryLog(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsHostWithOptions(toolchain, profile, unitSource, runtimeSource, WindowsMSVCBuildOptions{CaptureMap: true, CaptureLibraryLog: true})
}

func debugMapBuildOptions() WindowsMSVCBuildOptions {
	return WindowsMSVCBuildOptions{CaptureMap: true, CaptureProgramDatabase: true, EmbeddedCodeView: true, PreserveOptimizedLinking: true}
}

// BuildWindowsMSVCTarget compiles a unit emitted for the declared target profile.
func BuildWindowsMSVCTarget(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC target build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("target C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{{Name: "unit.c", Contents: unitSource}, {Name: "runtime.c", Contents: runtimeSource}}, WindowsMSVCBuildOptions{})
}

// BuildWindowsMSVCProbe compiles one closed target-evidence probe.
func BuildWindowsMSVCProbe(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCProbeWithOptions(toolchain, profile, probeSource, WindowsMSVCBuildOptions{})
}

// BuildWindowsMSVCProbeWithOptions is the common single-probe build boundary.
func BuildWindowsMSVCProbeWithOptions(toolchain WindowsHostToolchain, profile Profile, probeSource string, options WindowsMSVCBuildOptions) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if probeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{{Name: "probe.c", Contents: probeSource}}, options)
}

// BuildWindowsMSVCProbeWithMap retains the MSVC link map.
func BuildWindowsMSVCProbeWithMap(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCProbeWithOptions(toolchain, profile, probeSource, WindowsMSVCBuildOptions{CaptureMap: true})
}

// BuildWindowsMSVCProbeWithDebugMap retains the existing PDB and map profile.
func BuildWindowsMSVCProbeWithDebugMap(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCProbeWithOptions(toolchain, profile, probeSource, debugMapBuildOptions())
}

// BuildWindowsMSVCProbeWithLibraryLog retains bounded library-selection output.
func BuildWindowsMSVCProbeWithLibraryLog(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCProbeWithOptions(toolchain, profile, probeSource, WindowsMSVCBuildOptions{CaptureMap: true, CaptureLibraryLog: true})
}
