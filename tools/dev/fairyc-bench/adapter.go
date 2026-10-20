package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
)

type adapter struct {
	language   string
	executable string
	env        []string
	toolchain  Toolchain
}

func discover(ctx context.Context, o Options, language string) adapter {
	a := adapter{language: language, toolchain: Toolchain{Language: language, Status: "ready"}}
	var versionArgs []string
	switch language {
	case "fairy":
		a.executable = o.Fairy
		if a.executable == "" {
			a.executable = filepath.Join(o.Output, "tooling", "fairyc"+executableSuffix())
			_ = os.MkdirAll(filepath.Dir(a.executable), 0o755)
			cmd, out, err := execute(ctx, o.Timeout, ".", nil, nil, "go", "build", "-mod=readonly", "-o", a.executable, "./cmd/fairyc")
			if err != nil {
				a.toolchain.Status = "unavailable"
				a.toolchain.Reason = "build compiler: " + out + cmd.Log
				return a
			}
		}
		a.toolchain.Version = "checkout " + gitValue("rev-parse", "HEAD")
		if runtime.GOOS == "windows" {
			tc, err := cdriver.ResolveWindowsHostMSVC()
			if err != nil {
				a.toolchain.Status = "unavailable"
				a.toolchain.Reason = err.Error()
				return a
			}
			a.toolchain.Backend = "MSVC " + tc.Version + "; " + sdkIdentity(tc.Environment)
		} else {
			a.toolchain.Backend = "system C compiler; see C toolchain row"
		}
	case "rust":
		a.executable = "cargo"
		versionArgs = []string{"--version"}
		a.toolchain.Backend = strings.TrimSpace(commandOutput("rustc", "-Vv"))
	case "zig":
		a.executable = o.Zig
		versionArgs = []string{"version"}
		a.toolchain.Backend = "zig build; libc linked for getchar/printf"
	case "c":
		a.executable = o.CC
		if a.executable == "" && runtime.GOOS == "windows" {
			tc, err := cdriver.ResolveWindowsHostMSVC()
			if err != nil {
				a.toolchain.Status = "unavailable"
				a.toolchain.Reason = err.Error()
				return a
			}
			a.executable = tc.Path
			for _, entry := range tc.Environment {
				name, _, _ := strings.Cut(entry, "=")
				if strings.EqualFold(name, "PATH") || strings.EqualFold(name, "INCLUDE") || strings.EqualFold(name, "LIB") || strings.EqualFold(name, "LIBPATH") {
					a.env = append(a.env, entry)
				}
			}
			a.toolchain.Version = "MSVC " + tc.Version
			a.toolchain.Backend = sdkIdentity(tc.Environment) + "; direct compile+link, full rebuild on every invocation"
		} else {
			if a.executable == "" {
				a.executable = "cc"
			}
			versionArgs = []string{"--version"}
			a.toolchain.Backend = "direct compile+link, full rebuild on every invocation"
		}
	}
	path, err := exec.LookPath(a.executable)
	if err != nil {
		a.toolchain.Status = "unavailable"
		a.toolchain.Reason = err.Error()
		return a
	}
	a.executable, _ = filepath.Abs(path)
	if len(versionArgs) > 0 {
		cmd, out, err := execute(ctx, o.Timeout, ".", a.env, nil, append([]string{a.executable}, versionArgs...)...)
		if err != nil {
			a.toolchain.Status = "unavailable"
			a.toolchain.Reason = cmd.Log
			return a
		}
		a.toolchain.Version = strings.TrimSpace(out)
	}
	if b, err := os.ReadFile(a.executable); err == nil {
		a.toolchain.ExecutableHash = digest(b)
		a.toolchain.ExecutableBytes = int64(len(b))
	}
	a.toolchain.Executable = filepath.Base(a.executable)
	return a
}

func sdkIdentity(env []string) string {
	for _, entry := range env {
		name, value, _ := strings.Cut(entry, "=")
		if strings.EqualFold(name, "LIB") {
			for _, p := range strings.Split(value, ";") {
				if strings.Contains(p, "Windows Kits") {
					return "Windows SDK " + filepath.Base(filepath.Dir(filepath.Dir(p)))
				}
			}
		}
	}
	return "Windows SDK version unknown"
}

func (a adapter) binary(project, profile string) string {
	suffix := executableSuffix()
	switch a.language {
	case "rust":
		dir := "debug"
		if profile == "release" {
			dir = "release"
		}
		return filepath.Join(project, "target", dir, "bench"+suffix)
	case "zig":
		return filepath.Join(project, "zig-out", "bin", "bench"+suffix)
	default:
		return filepath.Join(project, "build", "bench"+suffix)
	}
}

func (a adapter) build(project, state, profile string) ([]string, []string, error) {
	env := append(append([]string(nil), a.env...), isolatedEnvironment(state)...)
	args := []string{a.executable}
	switch a.language {
	case "fairy":
		args = append(args, "build")
		if profile == "release" {
			args = append(args, "--release")
		}
	case "rust":
		args = append(args, "build", "--offline", "--color", "never")
		env = append(env, "CARGO_TARGET_DIR="+filepath.Join(project, "target"))
		if profile == "release" {
			args = append(args, "--release")
		}
		if profile == "small-dev" {
			env = append(env, "CARGO_PROFILE_DEV_DEBUG=line-tables-only", "CARGO_PROFILE_DEV_INCREMENTAL=false")
		}
	case "zig":
		opt := "Debug"
		if profile == "release" {
			opt = "ReleaseFast"
		}
		args = append(args, "build", "-Doptimize="+opt, "--cache-dir", filepath.Join(project, ".zig-cache"), "--global-cache-dir", filepath.Join(state, "zig-global"))
	case "c":
		if err := os.MkdirAll(filepath.Join(project, "build"), 0o755); err != nil {
			return nil, nil, err
		}
		paths, err := filepath.Glob(filepath.Join(project, "src", "*.c"))
		if err != nil {
			return nil, nil, err
		}
		sort.Strings(paths)
		if strings.EqualFold(filepath.Base(a.executable), "cl.exe") {
			args = append(args, "/nologo", "/std:c11", "/MD", "/Fo"+filepath.Join(project, "build")+string(filepath.Separator), "/Fe"+a.binary(project, profile))
			if profile == "release" {
				args = append(args, "/O2")
			} else {
				args = append(args, "/Od", "/Zi", "/Fd"+filepath.Join(project, "build", "compile.pdb"))
			}
			args = append(args, paths...)
			args = append(args, "/link", "/INCREMENTAL:NO")
			if profile != "release" {
				args = append(args, "/DEBUG", "/PDB:"+filepath.Join(project, "build", "bench.pdb"))
			}
		} else {
			args = append(args, "-std=c11")
			if profile == "release" {
				args = append(args, "-O2")
			} else {
				args = append(args, "-O0", "-g")
			}
			args = append(args, paths...)
			args = append(args, "-o", a.binary(project, profile))
		}
	default:
		return nil, nil, fmt.Errorf("unknown language: %s", a.language)
	}
	return args, env, nil
}
