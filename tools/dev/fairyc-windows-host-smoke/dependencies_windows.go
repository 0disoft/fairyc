//go:build windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/0disoft/fairyc/internal/pecoff"
	"golang.org/x/sys/windows"
)

type dependencyModule struct {
	Base   uint64 `json:"base"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
}

type dependencyResolution struct {
	Requested string `json:"requested"`
	Path      string `json:"path"`
	SHA256    string `json:"sha256"`
	Functions int    `json:"functions"`
}

type windowsDependencyReceipt struct {
	Case         string                     `json:"case"`
	Profile      string                     `json:"profile"`
	ImageSHA256  string                     `json:"imageSHA256"`
	Imports      pecoff.DependencyReport    `json:"imports"`
	Resolutions  []dependencyResolution     `json:"systemLoaderResolutions"`
	Modules      []dependencyModule         `json:"loadEvents"`
	Unloads      []uint64                   `json:"unloadEvents"`
	Exit         int32                      `json:"exit"`
	StdoutSHA256 string                     `json:"stdoutSHA256"`
	StderrSHA256 string                     `json:"stderrSHA256"`
	StaticCRT    []pecoff.MSVCLibrarySymbol `json:"staticCRT,omitempty"`
	Dynamic      []dependencyResolution     `json:"declaredDynamic,omitempty"`
}

func dependencySystemFile(path string) (dependencyModule, error) {
	path = filepath.Clean(strings.TrimPrefix(path, `\\?\`))
	system, err := queryWindowsSystemDirectory()
	if err != nil {
		return dependencyModule{}, err
	}
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Dir(path), system) {
		return dependencyModule{}, fmt.Errorf("DLL outside the declared system directory: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return dependencyModule{}, err
	}
	if len(data) == 0 || len(data) > 64*1024*1024 {
		return dependencyModule{}, fmt.Errorf("DLL has invalid size")
	}
	return dependencyModule{Path: path, SHA256: debugDigest(data)}, nil
}

func resolveDependencyLibrary(library pecoff.ImportedLibrary) (dependencyResolution, error) {
	if library.Name != strings.ToLower(filepath.Base(library.Name)) || strings.ContainsAny(library.Name, `/\:`) || !strings.HasSuffix(library.Name, ".dll") {
		return dependencyResolution{}, fmt.Errorf("DLL resolution requires a basename")
	}
	handle, err := windows.LoadLibraryEx(library.Name, 0, windows.LOAD_LIBRARY_SEARCH_SYSTEM32)
	if err != nil {
		return dependencyResolution{}, err
	}
	defer windows.FreeLibrary(handle)
	var path [32768]uint16
	count, err := windows.GetModuleFileName(handle, &path[0], uint32(len(path)))
	if err != nil || count == 0 || int(count) >= len(path) {
		return dependencyResolution{}, fmt.Errorf("DLL resolved path unavailable: %v", err)
	}
	module, err := dependencySystemFile(windows.UTF16ToString(path[:count]))
	if err != nil {
		return dependencyResolution{}, err
	}
	for _, function := range library.Functions {
		var address uintptr
		if function.Name != "" {
			address, err = windows.GetProcAddress(handle, function.Name)
		} else {
			address, err = windows.GetProcAddressByOrdinal(handle, uintptr(function.Ordinal))
		}
		if err != nil || address == 0 {
			return dependencyResolution{}, fmt.Errorf("resolve %s!%s/%d: %v", library.Name, function.Name, function.Ordinal, err)
		}
	}
	return dependencyResolution{Requested: library.Name, Path: module.Path, SHA256: module.SHA256, Functions: len(library.Functions)}, nil
}

func traceDependencyProcess(tracer, imagePath string, args []string) (windowsDependencyReceipt, []byte, []byte, error) {
	var receipt windowsDependencyReceipt
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return receipt, nil, nil, err
	}
	receipt.ImageSHA256 = debugDigest(data)
	receipt.Imports, err = pecoff.InspectAMD64Dependencies(data)
	if err != nil {
		return receipt, nil, nil, err
	}
	root, err := os.MkdirTemp("", "fairyc-dependency-trace-")
	if err != nil {
		return receipt, nil, nil, err
	}
	defer os.RemoveAll(root)
	outPath, errPath := filepath.Join(root, "out.bin"), filepath.Join(root, "err.bin")
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, tracer, append([]string{imagePath, outPath, errPath}, args...)...)
	command.Dir = root
	// No compiler installation, developer PATH, HOME or symbol-server configuration.
	system, err := queryWindowsSystemDirectory()
	if err != nil {
		return receipt, nil, nil, err
	}
	command.Env = []string{"SystemRoot=" + filepath.Dir(system), "WINDIR=" + filepath.Dir(system), "PATH=", "TEMP=" + root, "TMP=" + root}
	output, err := command.CombinedOutput()
	if err != nil {
		return receipt, nil, nil, fmt.Errorf("DLL tracer: %v: %s", err, output)
	}
	exited := false
	for _, line := range strings.Split(strings.TrimSpace(string(output)), "\n") {
		fields := strings.Split(strings.TrimSuffix(line, "\r"), "\t")
		if len(fields) < 2 || exited {
			return receipt, nil, nil, fmt.Errorf("invalid DLL trace line %q", line)
		}
		switch fields[0] {
		case "DLL":
			if len(fields) != 3 || len(receipt.Modules) >= 512 {
				return receipt, nil, nil, fmt.Errorf("invalid DLL load event")
			}
			base, err := strconv.ParseUint(fields[1], 16, 64)
			if err != nil || base == 0 {
				return receipt, nil, nil, fmt.Errorf("invalid DLL load base")
			}
			module, err := dependencySystemFile(fields[2])
			if err != nil {
				return receipt, nil, nil, err
			}
			module.Base = base
			receipt.Modules = append(receipt.Modules, module)
		case "UNLOAD":
			if len(fields) != 2 {
				return receipt, nil, nil, fmt.Errorf("invalid DLL unload event")
			}
			base, err := strconv.ParseUint(fields[1], 16, 64)
			if err != nil {
				return receipt, nil, nil, err
			}
			receipt.Unloads = append(receipt.Unloads, base)
		case "EXIT":
			if len(fields) != 2 {
				return receipt, nil, nil, fmt.Errorf("invalid DLL exit event")
			}
			code, err := strconv.ParseUint(fields[1], 10, 32)
			if err != nil {
				return receipt, nil, nil, err
			}
			receipt.Exit = int32(uint32(code))
			exited = true
		default:
			return receipt, nil, nil, fmt.Errorf("unrecognized DLL trace event")
		}
	}
	if !exited || len(receipt.Modules) == 0 {
		return receipt, nil, nil, fmt.Errorf("incomplete DLL load trace")
	}
	stdout, err := os.ReadFile(outPath)
	if err != nil {
		return receipt, nil, nil, err
	}
	stderr, err := os.ReadFile(errPath)
	if err != nil {
		return receipt, nil, nil, err
	}
	receipt.StdoutSHA256, receipt.StderrSHA256 = debugDigest(stdout), debugDigest(stderr)
	return receipt, stdout, stderr, nil
}

func validateDependencyResolutions(receipt *windowsDependencyReceipt) error {
	for _, library := range receipt.Imports.Libraries {
		resolution, err := resolveDependencyLibrary(library)
		if err != nil {
			return err
		}
		found := false
		for _, module := range receipt.Modules {
			if strings.EqualFold(module.Path, resolution.Path) && module.SHA256 == resolution.SHA256 {
				found = true
			}
		}
		if !found {
			return fmt.Errorf("import resolution %s was not observed in the child process", library.Name)
		}
		receipt.Resolutions = append(receipt.Resolutions, resolution)
	}
	return nil
}
