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
	"sync"
	"time"
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
	installationOutput, err := queryMSVCInstallation(commandTimeout, func(timeout time.Duration) (string, error) {
		return runWindowsHostCommandWithLimitAndTimeout(vswhere, nil, "", maxProcessOutput, timeout,
			"-latest", "-products", "*", "-requires", "Microsoft.VisualStudio.Component.VC.Tools.x86.x64", "-property", "installationPath")
	})
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

type msvcQueryOptions struct {
	Now     func() time.Time
	Context context.Context
}

// The query owns its synchronous I/O and must honor the supplied timeout. The
// optional clock/context control admission, retry and acceptance of its result.
func queryMSVCInstallation(budget time.Duration, query func(time.Duration) (string, error), options ...msvcQueryOptions) (string, error) {
	if len(options) > 1 {
		return "", fmt.Errorf("MSVC installation query accepts at most one options value")
	}
	now := time.Now
	ctx := context.Background()
	if len(options) != 0 {
		if options[0].Now != nil {
			now = options[0].Now
		}
		if options[0].Context != nil {
			ctx = options[0].Context
		}
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if budget <= 0 {
		return "", context.DeadlineExceeded
	}
	deadline := now().Add(budget)
	remaining := func() (time.Duration, error) {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		left := deadline.Sub(now())
		if left <= 0 {
			return 0, context.DeadlineExceeded
		}
		// An injected clock must never increase a single query's timeout.
		return min(left, budget), nil
	}
	var queryErr error
	for attempt := 0; attempt < 2; attempt++ {
		left, err := remaining()
		if err != nil {
			return "", errors.Join(queryErr, err)
		}
		output, err := query(left)
		queryErr = err
		// Reject late success as well as late failure, even if the underlying
		// process exits before its timeout notification has been delivered.
		if _, stop := remaining(); stop != nil {
			return "", errors.Join(queryErr, stop)
		}
		if !errors.Is(queryErr, exec.ErrWaitDelay) {
			return output, queryErr
		}
		// An inherited pipe may outlive the query process. Discard its incomplete
		// output and repeat only this read-only lookup, once, in the same budget.
	}
	return "", queryErr
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
