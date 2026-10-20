package cdriver

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestLinuxToolchainBaselineAdmission(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name, compiler, libc, version, numeric, triple, missing, wantError string
	}{
		{name: "gcc reference", compiler: "gcc", libc: "glibc 2.39", version: "gcc (Ubuntu) 13.3.0", numeric: "13.3.0", triple: "x86_64-linux-gnu"},
		{name: "clang reference", compiler: "clang", libc: "glibc 2.39", version: "Ubuntu clang version 18.1.3 (1ubuntu1)", triple: "x86_64-pc-linux-gnu"},
		{name: "newer versions", compiler: "gcc", libc: "glibc 2.41", version: "gcc (GNU) 15.2.0", numeric: "15.2.0", triple: "x86_64-redhat-linux-gnu"},
		{name: "old libc", compiler: "gcc", libc: "glibc 2.38", wantError: "glibc >= 2.39"},
		{name: "not glibc", compiler: "gcc", libc: "musl 1.2.5", wantError: "glibc >= 2.39"},
		{name: "old gcc", compiler: "gcc", libc: "glibc 2.39", version: "gcc (GNU) 12.3.0", numeric: "12.3.0", wantError: "gcc >= 13"},
		{name: "old clang", compiler: "clang", libc: "glibc 2.39", version: "clang version 17.0.6", wantError: "clang >= 18"},
		{name: "unknown version", compiler: "clang", libc: "glibc 2.39", version: "development build", wantError: "cannot identify clang version"},
		{name: "malformed clang version", compiler: "clang", libc: "glibc 2.39", version: "clang version 18.1.3git", wantError: "cannot identify clang version"},
		{name: "malformed gcc version", compiler: "gcc", libc: "glibc 2.39", version: "gcc (GNU) 13.3.0", numeric: "13.3.0 junk", wantError: "cannot identify gcc version"},
		{name: "not independent gcc", compiler: "gcc", libc: "glibc 2.39", version: "clang version 18.1.3", wantError: "rather than an independent GCC"},
		{name: "cross compiler", compiler: "gcc", libc: "glibc 2.39", version: "gcc 13.3.0", numeric: "13.3.0", triple: "aarch64-linux-gnu", wantError: "expected native x86_64-linux-gnu"},
		{name: "musl compiler", compiler: "gcc", libc: "glibc 2.39", version: "gcc 13.3.0", numeric: "13.3.0", triple: "x86_64-linux-musl", wantError: "expected native x86_64-linux-gnu"},
		{name: "x32 compiler", compiler: "gcc", libc: "glibc 2.39", version: "gcc 13.3.0", numeric: "13.3.0", triple: "x86_64-linux-gnux32", wantError: "expected native x86_64-linux-gnu"},
		{name: "missing libc probe", compiler: "gcc", missing: "getconf", wantError: "requires getconf"},
		{name: "missing compiler", compiler: "clang", libc: "glibc 2.39", missing: "clang", wantError: "resolve clang on PATH"},
	} {
		t.Run(test.name, func(t *testing.T) {
			look := func(name string) (string, error) {
				if name == test.missing {
					return "", exec.ErrNotFound
				}
				return "/tools/" + name, nil
			}
			run := func(path string, args ...string) (string, error) {
				switch strings.Join(args, " ") {
				case "GNU_LIBC_VERSION":
					return test.libc + "\n", nil
				case "--version":
					return test.version + "\nCopyright\n", nil
				case "-dumpfullversion -dumpversion":
					return test.numeric + "\n", nil
				case "-dumpmachine":
					return test.triple + "\n", nil
				default:
					return "", fmt.Errorf("unexpected probe %s %v", path, args)
				}
			}
			got, err := resolveLinuxToolchain(test.compiler, look, run)
			if test.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantError) || !strings.Contains(err.Error(), linuxToolchainGuide) {
					t.Fatalf("Resolve() = %+v, %v; want %s with setup hint", got, err, test.wantError)
				}
				return
			}
			if err != nil || got.Name != test.compiler || got.Version != test.version || got.Libc != test.libc {
				t.Fatalf("Resolve() = %+v, %v", got, err)
			}
		})
	}
}

func TestLinuxToolchainSelectionFallsBackOnlyForAutomaticChoice(t *testing.T) {
	resolve := func(name string) (Toolchain, error) {
		if name == "gcc" {
			return Toolchain{}, fmt.Errorf("gcc too old")
		}
		return Toolchain{Name: name}, nil
	}
	got, err := selectLinuxToolchain("", resolve)
	if err != nil || got.Name != "clang" {
		t.Fatalf("automatic selection = %+v, %v", got, err)
	}
	if _, err := selectLinuxToolchain("gcc", resolve); err == nil || !strings.Contains(err.Error(), "gcc too old") {
		t.Fatalf("explicit GCC silently fell back: %v", err)
	}
	_, err = selectLinuxToolchain("", func(name string) (Toolchain, error) {
		return Toolchain{}, fmt.Errorf("missing %s", name)
	})
	if err == nil || !strings.Contains(err.Error(), "missing gcc") || !strings.Contains(err.Error(), "missing clang") {
		t.Fatalf("missing both compilers: %v", err)
	}
}
