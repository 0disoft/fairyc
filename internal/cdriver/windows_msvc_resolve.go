package cdriver

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync"
)

var cachedWindowsHostMSVC = memoizeWindowsHostResolver(discoverWindowsHostMSVC)

// ResolveWindowsHostMSVC locates the latest complete x64 MSVC and Windows SDK
// installation once per process. The first result, including failures and its
// environment snapshot, persists until process exit. Each caller owns its copy
// of Environment; mutating it cannot alter another caller's toolchain.
func ResolveWindowsHostMSVC() (WindowsHostToolchain, error) {
	return cachedWindowsHostMSVC()
}

func memoizeWindowsHostResolver(discover func() (WindowsHostToolchain, error)) func() (WindowsHostToolchain, error) {
	once := sync.OnceValues(discover)
	return func() (WindowsHostToolchain, error) {
		toolchain, err := once()
		toolchain.Environment = slices.Clone(toolchain.Environment)
		return toolchain, err
	}
}

func discoverWindowsHostMSVC() (WindowsHostToolchain, error) {
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
