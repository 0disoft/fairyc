package cdriver

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

// CompileLinuxC11Only checks one C11 translation unit using the native build
// flags. It discards the temporary object without linking.
func CompileLinuxC11Only(toolchain Toolchain, profile Profile, source string) error {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" || toolchain.Target != target.X8664LinuxGNU {
		return fmt.Errorf("C11 object check requires linux/amd64")
	}
	if (toolchain.Name != "clang" && toolchain.Name != "gcc") || toolchain.Path == "" {
		return fmt.Errorf("invalid resolved C toolchain")
	}
	return compileC11Only(profile, source, func(dir, path string) error {
		arguments := objectArguments(profile, path, filepath.Join(dir, "header-check.o"))
		_, err := runBounded(toolchain.Path, arguments...)
		return err
	})
}

// CompileWindowsMSVCC11Only checks one C11 translation unit using the native
// build flags and SDK environment. It discards the object without linking.
func CompileWindowsMSVCC11Only(toolchain WindowsHostToolchain, profile Profile, source string) error {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return fmt.Errorf("MSVC C11 object check requires windows/amd64")
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return fmt.Errorf("invalid resolved MSVC host toolchain")
	}
	return compileC11Only(profile, source, func(dir, path string) error {
		arguments := append(msvcCompileArguments(profile, msvcC11Standard), "/c", path, "/Fo:header-check.obj")
		_, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, dir, arguments...)
		return err
	})
}

// CompileWindowsMSVCCPP23Object builds a C++23 developer interoperability companion
// with C++ exception handling enabled. The host builder links it as a COFF input.
func CompileWindowsMSVCCPP23Object(toolchain WindowsHostToolchain, profile Profile, sourcePath string) (WindowsHostObject, error) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return WindowsHostObject{}, fmt.Errorf("MSVC C++ object build requires windows/amd64")
	}
	if toolchain.Name != "msvc" || toolchain.Path == "" || toolchain.Version == "" || len(toolchain.Environment) == 0 {
		return WindowsHostObject{}, fmt.Errorf("invalid resolved MSVC host toolchain")
	}
	if profile != LowProfile && profile != OptimizedProfile {
		return WindowsHostObject{}, fmt.Errorf("unsupported C++ profile %q", profile.Name)
	}
	if !filepath.IsAbs(sourcePath) || filepath.Ext(sourcePath) != ".cpp" {
		return WindowsHostObject{}, fmt.Errorf("MSVC C++ object build requires an absolute .cpp path")
	}
	var major, minor int
	if _, err := fmt.Sscanf(toolchain.Version, "%d.%d", &major, &minor); err != nil {
		return WindowsHostObject{}, fmt.Errorf("invalid MSVC toolset version %q", toolchain.Version)
	}
	standard := "/std:c++23preview"
	// MSVC 14.52 introduced the stable spelling.
	if major > 14 || (major == 14 && minor >= 52) {
		standard = "/std:c++23"
	}
	dir, err := os.MkdirTemp("", "fairyc-interop-cpp-")
	if err != nil {
		return WindowsHostObject{}, err
	}
	defer os.RemoveAll(dir)

	const name = "wrapper.obj"
	optimization := "/Od"
	if profile == OptimizedProfile {
		optimization = "/O2"
	}
	if _, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, dir, "/nologo",
		standard, "/TP", "/EHsc", "/W4", "/WX", "/MT", optimization, "/c", sourcePath, "/Fo:"+name); err != nil {
		return WindowsHostObject{}, fmt.Errorf("compile C++ wrapper: %s", strings.ReplaceAll(err.Error(), dir, "<temporary>"))
	}
	object, err := readArtifact(filepath.Join(dir, name))
	return WindowsHostObject{Name: name, Bytes: object}, err
}

func compileC11Only(profile Profile, source string, compile func(dir, path string) error) error {
	if profile != LowProfile && profile != OptimizedProfile {
		return fmt.Errorf("unsupported C profile %q", profile.Name)
	}
	if source == "" {
		return fmt.Errorf("C11 object check requires source")
	}
	dir, err := os.MkdirTemp("", "fairyc-interop-check-")
	if err != nil {
		return fmt.Errorf("create temporary C11 check directory: %w", err)
	}
	defer os.RemoveAll(dir)

	path := filepath.Join(dir, "header-check.c")
	if err := os.WriteFile(path, []byte(source), 0o600); err != nil {
		return fmt.Errorf("write C11 check input: %w", err)
	}
	if err := compile(dir, path); err != nil {
		return fmt.Errorf("%s", strings.ReplaceAll(err.Error(), dir, "<temporary>"))
	}
	return nil
}
