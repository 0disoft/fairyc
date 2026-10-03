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
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/pecoff"
)

// WindowsHostToolchain is one locally resolved MSVC installation for target-neutral host smoke.
// It is deliberately not a Fairy C target or ABI-conformance identity.
type WindowsHostToolchain struct {
	Name        string
	Path        string
	Version     string
	Environment []string
}

// WindowsHostObject is one bounded and structurally validated intermediate COFF object.
type WindowsHostObject struct {
	Name  string
	Bytes []byte
}

// WindowsHostArtifact is one bounded PE executable plus its validated COFF objects and optional
// private validation-only linker map and program database.
// It is not a distributable target artifact or a target-conformance receipt.
type WindowsHostArtifact struct {
	RuntimeCache    RuntimeObjectCacheStatus
	Bytes           []byte
	Objects         []WindowsHostObject
	LinkMap         []byte
	ProgramDatabase []byte
	LinkerVerbose   []byte
	Toolchain       WindowsHostToolchain
	Profile         Profile
	ConsumerMode    ConsumerMode
	TrapControl     bool
}

// WindowsMSVCArchiveArtifact preserves one exact input-order link result together with the
// inspected archive and linker map that explain archive extraction and COMDAT selection.
type WindowsMSVCArchiveArtifact struct {
	WindowsHostArtifact
	Variant     string
	ArchiveName string
	Archive     []byte
}

// WindowsMSVCWeakSearchArtifact preserves the exact archive and link map used to distinguish
// IMAGE_WEAK_EXTERN_SEARCH_NOLIBRARY from IMAGE_WEAK_EXTERN_SEARCH_LIBRARY.
type WindowsMSVCWeakSearchArtifact struct {
	WindowsHostArtifact
	ArchiveName string
	Archive     []byte
}

// WindowsHostOutcome is one bounded local MSVC execution result. ExitCode preserves the program's i32 bits.
// Stdout and Stderr preserve stream identity but do not claim ordering across the two OS pipes.
// TrapControl is compiler-private control data received outside both program streams.
type WindowsHostOutcome struct {
	ExitCode    int32
	Stdout      string
	Stderr      string
	TrapControl string
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
	return buildWindowsHostWithMap(toolchain, profile, unitSource, runtimeSource, false)
}

// BuildWindowsHostWithMap retains the MSVC link map for local size attribution.
func BuildWindowsHostWithMap(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	return buildWindowsHostWithMap(toolchain, profile, unitSource, runtimeSource, true)
}

// BuildWindowsHostWithDebugMap retains a PDB and map for private native size attribution.
func BuildWindowsHostWithDebugMap(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC host build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "unit.c", Contents: unitSource},
		{Name: "runtime.c", Contents: runtimeSource},
	}, windowsMSVCBuildOptions{CaptureMap: true, CaptureProgramDatabase: true, EmbeddedCodeView: true, PreserveOptimizedLinking: true})
}

// BuildWindowsHostWithLibraryLog retains bounded MSVC library-selection output.
func BuildWindowsHostWithLibraryLog(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC host build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "unit.c", Contents: unitSource},
		{Name: "runtime.c", Contents: runtimeSource},
	}, windowsMSVCBuildOptions{CaptureMap: true, CaptureLibraryLog: true})
}

func buildWindowsHostWithMap(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string, captureMap bool) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC host build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "unit.c", Contents: unitSource},
		{Name: "runtime.c", Contents: runtimeSource},
	}, windowsMSVCBuildOptions{CaptureMap: captureMap})
}

// BuildWindowsMSVCTarget compiles a unit emitted for the declared
// x86_64-windows-msvc profile. Local paths remain toolchain-resolution data.
func BuildWindowsMSVCTarget(toolchain WindowsHostToolchain, profile Profile, unitSource, runtimeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC target build requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if unitSource == "" || runtimeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("target C unit and runtime source must be non-empty")
	}
	return buildWindowsMSVC(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "unit.c", Contents: unitSource},
		{Name: "runtime.c", Contents: runtimeSource},
	})
}

// BuildWindowsMSVCProbe compiles one closed target-evidence probe with local x64 MSVC.
func BuildWindowsMSVCProbe(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return buildWindowsMSVCProbeWithMap(toolchain, profile, probeSource, false)
}

// BuildWindowsMSVCProbeWithMap retains the MSVC link map for local C comparison.
func BuildWindowsMSVCProbeWithMap(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	return buildWindowsMSVCProbeWithMap(toolchain, profile, probeSource, true)
}

// BuildWindowsMSVCProbeWithDebugMap retains private C symbols for a local size comparison.
func BuildWindowsMSVCProbeWithDebugMap(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if probeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode,
		[]windowsMSVCSource{{Name: "probe.c", Contents: probeSource}},
		windowsMSVCBuildOptions{CaptureMap: true, CaptureProgramDatabase: true, EmbeddedCodeView: true, PreserveOptimizedLinking: true})
}

// BuildWindowsMSVCProbeWithLibraryLog retains bounded MSVC library-selection output.
func BuildWindowsMSVCProbeWithLibraryLog(toolchain WindowsHostToolchain, profile Profile, probeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if probeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode,
		[]windowsMSVCSource{{Name: "probe.c", Contents: probeSource}},
		windowsMSVCBuildOptions{CaptureMap: true, CaptureLibraryLog: true})
}

func buildWindowsMSVCProbeWithMap(toolchain WindowsHostToolchain, profile Profile, probeSource string, captureMap bool) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if probeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC layout probe source must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{{Name: "probe.c", Contents: probeSource}}, windowsMSVCBuildOptions{CaptureMap: captureMap})
}

// BuildWindowsMSVCABIProbe compiles one caller and one callee translation unit
// separately before linking them with local x64 MSVC.
func BuildWindowsMSVCABIProbe(toolchain WindowsHostToolchain, profile Profile, callerSource, calleeSource string) (WindowsHostArtifact, error) {
	return BuildWindowsMSVCABIProbeWithConsumerMode(toolchain, profile, C11ConsumerMode, callerSource, calleeSource)
}

// BuildWindowsMSVCABIProbeWithConsumerMode compiles one unchanged C11 caller
// and callee under an explicit validation consumer mode. C17 is evidence only
// and does not create another Fairy C emission profile.
func BuildWindowsMSVCABIProbeWithConsumerMode(toolchain WindowsHostToolchain, profile Profile, consumerMode ConsumerMode, callerSource, calleeSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC ABI probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if callerSource == "" || calleeSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC ABI caller and callee sources must be non-empty")
	}
	return buildWindowsMSVC(toolchain, profile, consumerMode, []windowsMSVCSource{
		{Name: "caller.c", Contents: callerSource},
		{Name: "callee.c", Contents: calleeSource},
	})
}

// BuildWindowsMSVCRelocationProbe compiles one caller and one target translation unit separately
// before linking them with local x64 MSVC for object-to-image relocation inspection.
func BuildWindowsMSVCRelocationProbe(toolchain WindowsHostToolchain, profile Profile, callerSource, targetSource string, variantObject, weakObject, overflowObject, bigObject []byte) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC relocation probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if callerSource == "" || targetSource == "" || len(variantObject) == 0 || len(weakObject) == 0 || len(overflowObject) == 0 || len(bigObject) == 0 {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC relocation caller, target, variant, weak, overflow, and bigobj inputs must be non-empty")
	}
	if _, err := pecoff.InspectObject(variantObject); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("inspect generated MSVC REL32 variants object: %w", err)
	}
	if _, err := pecoff.InspectObject(weakObject); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("inspect generated MSVC weak-external object: %w", err)
	}
	if _, err := pecoff.InspectObject(overflowObject); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("inspect generated MSVC relocation-overflow object: %w", err)
	}
	if _, err := pecoff.InspectObject(bigObject); err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("inspect generated MSVC bigobj: %w", err)
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "caller.c", Contents: callerSource},
		{Name: "target.c", Contents: targetSource},
	}, windowsMSVCBuildOptions{
		CaptureMap:             true,
		CaptureProgramDatabase: true,
		DisableICF:             true,
		EmbeddedCodeView:       true,
		PrebuiltObjects: []WindowsHostObject{
			{Name: "variants.obj", Bytes: slices.Clone(variantObject)},
			{Name: "weak.obj", Bytes: slices.Clone(weakObject)},
			{Name: "overflow.obj", Bytes: slices.Clone(overflowObject)},
			{Name: "bigobj.obj", Bytes: slices.Clone(bigObject)},
		},
	})
}

// BuildWindowsMSVCArchiveProbe compiles one C11 caller, one C11 unused member, and two C++20
// COMDAT candidates separately, archives the demanded and unused members, then links the
// demanded and unused members with LIB, and links direct-first and archive-first input orders with
// exact MSVC map files. IMAGE_COMDAT_SELECT_ANY does not promise which candidate wins.
func BuildWindowsMSVCArchiveProbe(toolchain WindowsHostToolchain, profile Profile, callerSource, chosenSource, unusedSource, directSource string) ([]WindowsMSVCArchiveArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return nil, fmt.Errorf("MSVC archive probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if callerSource == "" || chosenSource == "" || unusedSource == "" || directSource == "" {
		return nil, fmt.Errorf("MSVC archive probe sources must be non-empty")
	}
	if profile != LowProfile && profile != OptimizedProfile {
		return nil, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return nil, fmt.Errorf("invalid resolved MSVC host toolchain")
	}

	temporaryRoot, err := os.MkdirTemp("", "fairyc-msvc-archive-")
	if err != nil {
		return nil, fmt.Errorf("create MSVC archive temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	sources := []windowsMSVCSource{
		{Name: "caller.c", Contents: callerSource},
		{Name: "chosen.cpp", Contents: chosenSource},
		{Name: "unused.c", Contents: unusedSource},
		{Name: "direct.cpp", Contents: directSource},
	}
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	objects := make([]WindowsHostObject, 0, len(sources))
	objectPaths := make(map[string]string, len(sources))
	for _, source := range sources {
		sourcePath := filepath.Join(temporaryRoot, source.Name)
		if err := os.WriteFile(sourcePath, []byte(source.Contents), 0o600); err != nil {
			return nil, fmt.Errorf("write MSVC archive source %s: %w", source.Name, err)
		}
		objectName := strings.TrimSuffix(source.Name, filepath.Ext(source.Name)) + ".obj"
		objectPath := filepath.Join(temporaryRoot, objectName)
		functionPackaging := "/Gy"
		if source.Name == "unused.c" {
			// Keep the unused member's sentinel in an ordinary, indivisible code section. Its
			// absence from the final map then cannot be explained by /OPT:REF collecting a
			// per-function COMDAT after the archive member had contributed to the image.
			functionPackaging = "/Gy-"
		}
		languageStandard := "/std:c11"
		languageMode := "/TC"
		bufferSecurity := "/GS"
		if filepath.Ext(source.Name) == ".cpp" {
			languageStandard = "/std:c++20"
			languageMode = "/TP"
			// Keep the competing unwind payloads free of __GSHandlerCheck relocations so
			// the section-collection oracle compares only self-contained unwind bytes.
			bufferSecurity = "/GS-"
		}
		arguments := []string{
			"/nologo", languageStandard, languageMode, "/W4", "/WX", "/MT", bufferSecurity, optimization, functionPackaging, "/c",
			sourcePath, "/Fo:" + objectPath,
		}
		if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, temporaryRoot, arguments...); err != nil {
			return nil, fmt.Errorf("compile MSVC archive source %s: %s", source.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		objectBytes, err := readArtifact(objectPath)
		if err != nil {
			return nil, fmt.Errorf("read MSVC archive object %s: %s", objectName, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		if _, err := pecoff.InspectObject(objectBytes); err != nil {
			return nil, fmt.Errorf("inspect MSVC archive object %s: %w", objectName, err)
		}
		objects = append(objects, WindowsHostObject{Name: objectName, Bytes: objectBytes})
		objectPaths[objectName] = objectPath
	}

	librarianPath := filepath.Join(filepath.Dir(toolchain.Path), "lib.exe")
	if info, err := os.Stat(librarianPath); err != nil || !info.Mode().IsRegular() {
		return nil, fmt.Errorf("MSVC archive probe requires x64 lib.exe beside cl.exe")
	}
	archiveName := "choice.lib"
	archivePath := filepath.Join(temporaryRoot, archiveName)
	if _, err := runWindowsHostCommand(librarianPath, toolchain.Environment, temporaryRoot,
		"/NOLOGO", "/OUT:"+archivePath, "chosen.obj", "unused.obj"); err != nil {
		return nil, fmt.Errorf("create MSVC archive: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	archiveBytes, err := readArtifact(archivePath)
	if err != nil {
		return nil, fmt.Errorf("read MSVC archive: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	if _, err := pecoff.InspectArchive(archiveBytes); err != nil {
		return nil, fmt.Errorf("inspect MSVC archive: %w", err)
	}

	variants := []struct {
		name   string
		inputs []string
	}{
		{name: "direct-first", inputs: []string{objectPaths["caller.obj"], objectPaths["direct.obj"], archivePath}},
		{name: "archive-first", inputs: []string{objectPaths["caller.obj"], archivePath, objectPaths["direct.obj"]}},
	}
	artifacts := make([]WindowsMSVCArchiveArtifact, 0, len(variants))
	for _, variant := range variants {
		executablePath := filepath.Join(temporaryRoot, variant.name+".exe")
		mapPath := filepath.Join(temporaryRoot, variant.name+".map")
		arguments := []string{"/nologo", "/Fe:" + executablePath}
		arguments = append(arguments, variant.inputs...)
		arguments = append(arguments, "/link", "/INCREMENTAL:NO", "/OPT:REF", "/OPT:NOICF", "/MAP:"+mapPath)
		_, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, temporaryRoot, arguments...)
		if err != nil {
			return nil, fmt.Errorf("link MSVC archive variant %s: %s", variant.name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		executableBytes, err := readArtifact(executablePath)
		if err != nil {
			return nil, fmt.Errorf("read MSVC archive executable %s: %s", variant.name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		if _, err := pecoff.InspectExecutable(executableBytes); err != nil {
			return nil, fmt.Errorf("inspect MSVC archive executable %s: %w", variant.name, err)
		}
		mapBytes, err := readArtifact(mapPath)
		if err != nil {
			return nil, fmt.Errorf("read MSVC archive map %s: %s", variant.name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		artifacts = append(artifacts, WindowsMSVCArchiveArtifact{
			WindowsHostArtifact: WindowsHostArtifact{
				Bytes: executableBytes, Objects: slices.Clone(objects), Toolchain: toolchain,
				Profile: profile, ConsumerMode: C11ConsumerMode, LinkMap: mapBytes,
			},
			Variant: variant.name, ArchiveName: archiveName, Archive: slices.Clone(archiveBytes),
		})
	}
	return artifacts, nil
}

// BuildWindowsMSVCWeakSearchProbe compiles one caller and two independent archive members, then
// links them with one generated weak-search object and captures the exact archive and map.
func BuildWindowsMSVCWeakSearchProbe(toolchain WindowsHostToolchain, profile Profile, callerSource, noLibrarySource, librarySource string, weakObject []byte) (WindowsMSVCWeakSearchArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("MSVC weak-search probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if callerSource == "" || noLibrarySource == "" || librarySource == "" || len(weakObject) == 0 {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("MSVC weak-search probe inputs must be non-empty")
	}
	if profile != LowProfile && profile != OptimizedProfile {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("invalid resolved MSVC host toolchain")
	}
	if _, err := pecoff.InspectObject(weakObject); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("inspect generated MSVC weak-search object: %w", err)
	}
	temporaryRoot, err := os.MkdirTemp("", "fairyc-msvc-weak-search-")
	if err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("create MSVC weak-search temp root: %w", err)
	}
	defer os.RemoveAll(temporaryRoot)
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	sources := []windowsMSVCSource{
		{Name: "caller.c", Contents: callerSource},
		{Name: "nolib.c", Contents: noLibrarySource},
		{Name: "library.c", Contents: librarySource},
	}
	objects := make([]WindowsHostObject, 0, len(sources)+1)
	objectPaths := make(map[string]string, len(sources)+1)
	for _, source := range sources {
		sourcePath := filepath.Join(temporaryRoot, source.Name)
		if err := os.WriteFile(sourcePath, []byte(source.Contents), 0o600); err != nil {
			return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("write MSVC weak-search source %s: %w", source.Name, err)
		}
		objectName := strings.TrimSuffix(source.Name, ".c") + ".obj"
		objectPath := filepath.Join(temporaryRoot, objectName)
		if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, temporaryRoot,
			"/nologo", "/std:c11", "/TC", "/W4", "/WX", "/MT", optimization, "/Gy-", "/c", sourcePath, "/Fo:"+objectPath); err != nil {
			return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("compile MSVC weak-search source %s: %s", source.Name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		objectBytes, err := readArtifact(objectPath)
		if err != nil {
			return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("read MSVC weak-search object %s: %s", objectName, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		if _, err := pecoff.InspectObject(objectBytes); err != nil {
			return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("inspect MSVC weak-search object %s: %w", objectName, err)
		}
		objects = append(objects, WindowsHostObject{Name: objectName, Bytes: objectBytes})
		objectPaths[objectName] = objectPath
	}
	weakObjectName := "weakmode.obj"
	weakObjectPath := filepath.Join(temporaryRoot, weakObjectName)
	if err := os.WriteFile(weakObjectPath, weakObject, 0o600); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("write MSVC weak-search object: %w", err)
	}
	objects = append(objects, WindowsHostObject{Name: weakObjectName, Bytes: slices.Clone(weakObject)})
	objectPaths[weakObjectName] = weakObjectPath

	librarianPath := filepath.Join(filepath.Dir(toolchain.Path), "lib.exe")
	if info, err := os.Stat(librarianPath); err != nil || !info.Mode().IsRegular() {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("MSVC weak-search probe requires x64 lib.exe beside cl.exe")
	}
	archiveName := "weakmode.lib"
	archivePath := filepath.Join(temporaryRoot, archiveName)
	if _, err := runWindowsHostCommand(librarianPath, toolchain.Environment, temporaryRoot,
		"/NOLOGO", "/OUT:"+archivePath, "nolib.obj", "library.obj"); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("create MSVC weak-search archive: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	archiveBytes, err := readArtifact(archivePath)
	if err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("read MSVC weak-search archive: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	if _, err := pecoff.InspectArchive(archiveBytes); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("inspect MSVC weak-search archive: %w", err)
	}
	executablePath := filepath.Join(temporaryRoot, "program.exe")
	mapPath := filepath.Join(temporaryRoot, "program.map")
	if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, temporaryRoot,
		"/nologo", "/Fe:"+executablePath, objectPaths["caller.obj"], weakObjectPath, archivePath,
		"/link", "/INCREMENTAL:NO", "/OPT:REF", "/OPT:NOICF", "/MAP:"+mapPath); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("link MSVC weak-search probe: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	executableBytes, err := readArtifact(executablePath)
	if err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("read MSVC weak-search executable: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	if _, err := pecoff.InspectExecutable(executableBytes); err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("inspect MSVC weak-search executable: %w", err)
	}
	mapBytes, err := readArtifact(mapPath)
	if err != nil {
		return WindowsMSVCWeakSearchArtifact{}, fmt.Errorf("read MSVC weak-search map: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	return WindowsMSVCWeakSearchArtifact{
		WindowsHostArtifact: WindowsHostArtifact{
			Bytes: executableBytes, Objects: objects, LinkMap: mapBytes, Toolchain: toolchain,
			Profile: profile, ConsumerMode: C11ConsumerMode,
		},
		ArchiveName: archiveName, Archive: slices.Clone(archiveBytes),
	}, nil
}

// BuildWindowsMSVCUnwindProbe compiles the catcher, termination frame, and
// raiser as separate translation units before linking them with local x64 MSVC.
func BuildWindowsMSVCUnwindProbe(toolchain WindowsHostToolchain, profile Profile, catcherSource, frameSource, raiserSource string) (WindowsHostArtifact, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC unwind probe requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	if catcherSource == "" || frameSource == "" || raiserSource == "" {
		return WindowsHostArtifact{}, fmt.Errorf("MSVC unwind catcher, frame, and raiser sources must be non-empty")
	}
	return buildWindowsMSVCWithOptions(toolchain, profile, C11ConsumerMode, []windowsMSVCSource{
		{Name: "catcher.c", Contents: catcherSource},
		{Name: "frame.c", Contents: frameSource},
		{Name: "raiser.c", Contents: raiserSource},
	}, windowsMSVCBuildOptions{CaptureMap: true})
}

type windowsMSVCSource struct {
	Name     string
	Contents string
}

type windowsMSVCBuildOptions struct {
	CaptureMap               bool
	CaptureLibraryLog        bool
	CaptureProgramDatabase   bool
	DisableICF               bool
	EmbeddedCodeView         bool
	PreserveOptimizedLinking bool
	PrebuiltObjects          []WindowsHostObject
}

func buildWindowsMSVC(toolchain WindowsHostToolchain, profile Profile, consumerMode ConsumerMode, sources []windowsMSVCSource) (WindowsHostArtifact, error) {
	return buildWindowsMSVCWithOptions(toolchain, profile, consumerMode, sources, windowsMSVCBuildOptions{})
}

func buildWindowsMSVCWithOptions(toolchain WindowsHostToolchain, profile Profile, consumerMode ConsumerMode, sources []windowsMSVCSource, options windowsMSVCBuildOptions) (WindowsHostArtifact, error) {
	if profile != LowProfile && profile != OptimizedProfile {
		return WindowsHostArtifact{}, fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	standardFlag, err := msvcStandardFlag(consumerMode)
	if err != nil {
		return WindowsHostArtifact{}, err
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
	prebuiltPaths := make([]string, 0, len(options.PrebuiltObjects))
	objectNames := make([]string, 0, len(sources))
	seenNames := make(map[string]struct{}, len(sources)+len(options.PrebuiltObjects))
	for _, source := range sources {
		if source.Name == "" || filepath.Base(source.Name) != source.Name || filepath.Ext(source.Name) != ".c" || source.Contents == "" {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC build source is invalid")
		}
		key := strings.ToLower(source.Name)
		if _, duplicate := seenNames[key]; duplicate {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC build repeats source name %q", source.Name)
		}
		seenNames[key] = struct{}{}
		path := filepath.Join(temporaryRoot, source.Name)
		if err := os.WriteFile(path, []byte(source.Contents), 0o600); err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("write MSVC source %s: %w", source.Name, err)
		}
		sourcePaths = append(sourcePaths, path)
		objectNames = append(objectNames, strings.TrimSuffix(source.Name, filepath.Ext(source.Name))+".obj")
	}
	for _, object := range options.PrebuiltObjects {
		if object.Name == "" || filepath.Base(object.Name) != object.Name || !strings.EqualFold(filepath.Ext(object.Name), ".obj") || len(object.Bytes) == 0 {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC prebuilt object is invalid")
		}
		key := strings.ToLower(strings.TrimSuffix(object.Name, filepath.Ext(object.Name)) + ".c")
		if _, duplicate := seenNames[key]; duplicate {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC build repeats object name %q", object.Name)
		}
		seenNames[key] = struct{}{}
		if _, err := pecoff.InspectObject(object.Bytes); err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("inspect MSVC prebuilt object %s: %w", object.Name, err)
		}
		path := filepath.Join(temporaryRoot, object.Name)
		if err := os.WriteFile(path, object.Bytes, 0o600); err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("write MSVC prebuilt object %s: %w", object.Name, err)
		}
		prebuiltPaths = append(prebuiltPaths, path)
		objectNames = append(objectNames, object.Name)
	}
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	arguments := []string{
		"/nologo", standardFlag, "/TC", "/W4", "/WX", "/MT", optimization,
	}
	if options.EmbeddedCodeView {
		arguments = append(arguments, "/Z7")
	}
	arguments = append(arguments, sourcePaths...)
	arguments = append(arguments, "/Fe:"+executablePath, "/link")
	arguments = append(arguments, prebuiltPaths...)
	arguments = append(arguments, "/INCREMENTAL:NO")
	programDatabasePath := ""
	if options.CaptureProgramDatabase {
		if !options.EmbeddedCodeView {
			return WindowsHostArtifact{}, fmt.Errorf("MSVC program database capture requires embedded CodeView input")
		}
		programDatabasePath = filepath.Join(temporaryRoot, "program.pdb")
		arguments = append(arguments, "/DEBUG:FULL", "/PDB:"+programDatabasePath, "/PDBALTPATH:%_PDB%")
	}
	mapPath := ""
	if options.CaptureMap {
		mapPath = filepath.Join(temporaryRoot, "program.map")
		arguments = append(arguments, "/MAP:"+mapPath)
	}
	if options.PreserveOptimizedLinking && profile == OptimizedProfile {
		arguments = append(arguments, "/OPT:REF", "/OPT:ICF")
	}
	if options.DisableICF {
		arguments = append(arguments, "/OPT:NOICF")
	}
	if options.CaptureLibraryLog {
		arguments = append(arguments, "/VERBOSE")
	}
	outputLimit := maxProcessOutput
	if options.CaptureLibraryLog {
		outputLimit = 8 * 1024 * 1024
	}
	output, err := runWindowsHostCommandWithLimit(toolchain.Path, toolchain.Environment, temporaryRoot, outputLimit, arguments...)
	if err != nil {
		return WindowsHostArtifact{}, fmt.Errorf("%s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
	}
	contents, err := readArtifact(executablePath)
	if err != nil {
		return WindowsHostArtifact{}, err
	}
	objects := make([]WindowsHostObject, 0, len(objectNames))
	for _, name := range objectNames {
		objectBytes, err := readArtifact(filepath.Join(temporaryRoot, name))
		if err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("read MSVC COFF object %s: %s", name, strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
		if _, err := pecoff.InspectObject(objectBytes); err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("inspect MSVC COFF object %s: %w", name, err)
		}
		objects = append(objects, WindowsHostObject{Name: name, Bytes: objectBytes})
	}
	var linkMap []byte
	if options.CaptureMap {
		linkMap, err = readArtifact(mapPath)
		if err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("read MSVC link map: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
	}
	var programDatabase []byte
	if options.CaptureProgramDatabase {
		programDatabase, err = readArtifact(programDatabasePath)
		if err != nil {
			return WindowsHostArtifact{}, fmt.Errorf("read MSVC program database: %s", strings.ReplaceAll(err.Error(), temporaryRoot, "<temporary>"))
		}
	}
	var linkerVerbose []byte
	if options.CaptureLibraryLog {
		linkerVerbose = []byte(output)
	}
	return WindowsHostArtifact{
		Bytes: contents, Objects: objects, LinkMap: linkMap, ProgramDatabase: programDatabase,
		LinkerVerbose: linkerVerbose,
		Toolchain:     toolchain, Profile: profile, ConsumerMode: consumerMode,
	}, nil
}

func msvcStandardFlag(mode ConsumerMode) (string, error) {
	switch mode {
	case C11ConsumerMode:
		return "/std:c11", nil
	case C17ConsumerMode:
		return "/std:c17", nil
	default:
		return "", fmt.Errorf("unsupported C consumer mode %q", mode)
	}
}

// ExecuteWindowsHost runs one temporary PE and returns its exact i32 exit bits and bounded output.
func ExecuteWindowsHost(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC host execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC host")
}

// ExecuteWindowsHostWithTimeout gives bounded long-running host probes more time
// without changing the 60-second default for ordinary target execution.
func ExecuteWindowsHostWithTimeout(artifact WindowsHostArtifact, timeout time.Duration) (WindowsHostOutcome, error) {
	if timeout <= 0 || timeout > 2*time.Minute {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC host probe timeout must be greater than zero and at most %s", 2*time.Minute)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC host execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifactWithInputArgsAndTimeout(artifact, "MSVC host", nil, nil, timeout)
}

// ExecuteWindowsMSVCTarget runs one temporary PE built for the declared
// x86_64-windows-msvc target and returns its exact i32 exit bits and bounded output.
func ExecuteWindowsMSVCTarget(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC target")
}

// ExecuteWindowsMSVCTargetWithInput runs one temporary target PE with the caller-owned input
// stream. The driver does not pre-read or retain input bytes.
func ExecuteWindowsMSVCTargetWithInput(artifact WindowsHostArtifact, stdin io.Reader) (WindowsHostOutcome, error) {
	return ExecuteWindowsMSVCTargetWithInputAndArgs(artifact, stdin, nil)
}

// ExecuteWindowsMSVCTargetWithInputAndArgs runs one temporary target PE with the caller-owned
// input stream and exact program arguments after argv[0]. The driver does not pre-read or retain
// input bytes.
func ExecuteWindowsMSVCTargetWithInputAndArgs(artifact WindowsHostArtifact, stdin io.Reader, programArgs []string) (WindowsHostOutcome, error) {
	return ExecuteWindowsMSVCTargetWithInputArgsAndTimeout(artifact, stdin, programArgs, commandTimeout)
}

// ExecuteWindowsMSVCTargetWithInputArgsAndTimeout runs one temporary target PE with an explicit
// bounded execution timeout. The timeout applies only to the program process, not compilation.
func ExecuteWindowsMSVCTargetWithInputArgsAndTimeout(artifact WindowsHostArtifact, stdin io.Reader, programArgs []string, timeout time.Duration) (WindowsHostOutcome, error) {
	if stdin == nil {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution input must not be nil")
	}
	if timeout <= 0 || timeout > commandTimeout {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution timeout must be greater than zero and at most %s", commandTimeout)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC target execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifactWithInputArgsAndTimeout(artifact, "MSVC target", stdin, programArgs, timeout)
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

// ExecuteWindowsMSVCRelocationProbe runs one linked caller/target relocation proof.
func ExecuteWindowsMSVCRelocationProbe(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC relocation probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC relocation probe")
}

// ExecuteWindowsMSVCArchiveProbe runs one archive input-order variant.
func ExecuteWindowsMSVCArchiveProbe(artifact WindowsMSVCArchiveArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC archive probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact.WindowsHostArtifact, "MSVC archive probe "+artifact.Variant)
}

// ExecuteWindowsMSVCWeakSearchProbe runs one weak archive-search executable.
func ExecuteWindowsMSVCWeakSearchProbe(artifact WindowsMSVCWeakSearchArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC weak-search probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact.WindowsHostArtifact, "MSVC weak-search probe")
}

// ExecuteWindowsMSVCUnwindProbe runs one linked structured-exception fixture.
func ExecuteWindowsMSVCUnwindProbe(artifact WindowsHostArtifact) (WindowsHostOutcome, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostOutcome{}, fmt.Errorf("MSVC unwind probe execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	return executeWindowsMSVCArtifact(artifact, "MSVC unwind probe")
}

func executeWindowsMSVCArtifact(artifact WindowsHostArtifact, label string) (WindowsHostOutcome, error) {
	return executeWindowsMSVCArtifactWithInputAndArgs(artifact, label, nil, nil)
}

func executeWindowsMSVCArtifactWithInput(artifact WindowsHostArtifact, label string, stdin io.Reader) (WindowsHostOutcome, error) {
	return executeWindowsMSVCArtifactWithInputAndArgs(artifact, label, stdin, nil)
}

func executeWindowsMSVCArtifactWithInputAndArgs(artifact WindowsHostArtifact, label string, stdin io.Reader, programArgs []string) (WindowsHostOutcome, error) {
	return executeWindowsMSVCArtifactWithInputArgsAndTimeout(artifact, label, stdin, programArgs, commandTimeout)
}

func executeWindowsMSVCArtifactWithInputArgsAndTimeout(artifact WindowsHostArtifact, label string, stdin io.Reader, programArgs []string, timeout time.Duration) (WindowsHostOutcome, error) {
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
	contextValue, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	command := exec.CommandContext(contextValue, executablePath)
	if len(programArgs) != 0 {
		command.Args = append([]string{executablePath}, programArgs...)
	}
	command.Stdin = stdin
	command.Dir = temporaryRoot
	output := newBoundedProcessOutput(maxProcessOutput)
	command.Stdout = output.stdoutWriter()
	command.Stderr = output.stderrWriter()
	trapControl := ""
	trapControlOverflow := false
	if artifact.TrapControl {
		trapControl, trapControlOverflow, err = runWindowsProcessWithTrapControl(command, cruntime.RunTrapControlEnvironment)
	} else {
		err = command.Run()
	}
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return WindowsHostOutcome{}, fmt.Errorf("%s executable exceeded %s", label, timeout)
	}
	if trapControlOverflow {
		return WindowsHostOutcome{}, fmt.Errorf("%s executable exceeded the private trap control bound", label)
	}
	stdout, stderr, truncated := output.values()
	if truncated {
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
	return WindowsHostOutcome{
		ExitCode:    int32(uint32(exitCode)),
		Stdout:      stdout,
		Stderr:      stderr,
		TrapControl: trapControl,
	}, nil
}

type processOutputStream uint8

const (
	processStdout processOutputStream = iota
	processStderr
)

type boundedProcessOutput struct {
	mu        sync.Mutex
	limit     int
	stdout    []byte
	stderr    []byte
	truncated bool
}

type boundedProcessStreamWriter struct {
	output *boundedProcessOutput
	stream processOutputStream
}

func newBoundedProcessOutput(limit int) *boundedProcessOutput {
	return &boundedProcessOutput{limit: limit}
}

func (output *boundedProcessOutput) stdoutWriter() *boundedProcessStreamWriter {
	return &boundedProcessStreamWriter{output: output, stream: processStdout}
}

func (output *boundedProcessOutput) stderrWriter() *boundedProcessStreamWriter {
	return &boundedProcessStreamWriter{output: output, stream: processStderr}
}

func (writer *boundedProcessStreamWriter) Write(value []byte) (int, error) {
	return writer.output.write(writer.stream, value), nil
}

func (output *boundedProcessOutput) write(stream processOutputStream, value []byte) int {
	output.mu.Lock()
	defer output.mu.Unlock()
	originalLength := len(value)
	available := output.limit - len(output.stdout) - len(output.stderr)
	if available > 0 {
		value = value[:min(available, len(value))]
		switch stream {
		case processStdout:
			output.stdout = append(output.stdout, value...)
		case processStderr:
			output.stderr = append(output.stderr, value...)
		default:
			output.truncated = true
			return originalLength
		}
	}
	if originalLength > max(available, 0) {
		output.truncated = true
	}
	return originalLength
}

func (output *boundedProcessOutput) values() (string, string, bool) {
	output.mu.Lock()
	defer output.mu.Unlock()
	return string(output.stdout), string(output.stderr), output.truncated
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
	return runWindowsHostCommandWithLimit(name, environment, directory, maxProcessOutput, arguments...)
}

func runWindowsHostCommandWithLimit(name string, environment []string, directory string, limit int, arguments ...string) (string, error) {
	contextValue, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()
	command := exec.CommandContext(contextValue, name, arguments...)
	command.Stdin = nil
	if len(environment) != 0 {
		command.Env = environment
	}
	command.Dir = directory
	output := boundedBuffer{limit: limit}
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	if errors.Is(contextValue.Err(), context.DeadlineExceeded) {
		return output.String(), fmt.Errorf("MSVC host command exceeded %s", commandTimeout)
	}
	if output.truncated {
		return output.String(), fmt.Errorf("MSVC host command exceeded %d output bytes", limit)
	}
	if err != nil {
		return output.String(), fmt.Errorf("MSVC host command failed: %w: %s", err, strings.TrimSpace(output.String()))
	}
	return output.String(), nil
}
