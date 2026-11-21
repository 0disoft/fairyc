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
		arguments := append(msvcCompileArguments(profile, "/std:c11"), "/c", path, "/Fo:header-check.obj")
		_, err := runWindowsHostCommand(toolchain.Path, toolchain.Environment, dir, arguments...)
		return err
	})
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
