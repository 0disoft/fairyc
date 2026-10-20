package cdriver

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

const linuxToolchainGuide = "see docs/abi/linux-toolchain-v0.md for Ubuntu 24.04 setup"

var (
	linuxCompilerVersion = regexp.MustCompile(`^([0-9]+)(\.[0-9]+){0,2}$`)
	linuxLibcVersion     = regexp.MustCompile(`^glibc ([0-9]+)\.([0-9]+)$`)
	linuxClangVersion    = regexp.MustCompile(`\bclang version (\S+)`)
)

// Resolution checks the native compiler identity and version floors. Headers,
// startup objects, linker and sanitizer readiness belong to the native fixture lane.
// Injected probes let Windows hosts exercise the same admission rules without Linux.
func resolveLinuxToolchain(name string, lookPath func(string) (string, error), run func(string, ...string) (string, error)) (Toolchain, error) {
	getconf, err := lookPath("getconf")
	if err != nil {
		return Toolchain{}, fmt.Errorf("Linux toolchain requires getconf from libc-bin: %w; %s", err, linuxToolchainGuide)
	}
	libcOutput, err := run(getconf, "GNU_LIBC_VERSION")
	if err != nil {
		return Toolchain{}, fmt.Errorf("probe glibc: %w; %s", err, linuxToolchainGuide)
	}
	libc := strings.TrimSpace(libcOutput)
	match := linuxLibcVersion.FindStringSubmatch(libc)
	if match == nil {
		return Toolchain{}, fmt.Errorf("Linux toolchain requires glibc >= 2.39, getconf reported %q; %s", libc, linuxToolchainGuide)
	}
	major, majorErr := strconv.Atoi(match[1])
	minor, minorErr := strconv.Atoi(match[2])
	if majorErr != nil || minorErr != nil || major < 2 || major == 2 && minor < 39 {
		return Toolchain{}, fmt.Errorf("Linux toolchain requires glibc >= 2.39, found %q; %s", libc, linuxToolchainGuide)
	}
	path, err := lookPath(name)
	if err != nil {
		return Toolchain{}, fmt.Errorf("resolve %s on PATH: %w; %s", name, err, linuxToolchainGuide)
	}
	versionOutput, err := run(path, "--version")
	if err != nil {
		return Toolchain{}, fmt.Errorf("probe %s: %w; %s", name, err, linuxToolchainGuide)
	}
	version := firstLine(versionOutput)
	numeric := ""
	minimum := 13
	if name == "gcc" {
		if strings.Contains(strings.ToLower(version), "clang") {
			return Toolchain{}, fmt.Errorf("gcc resolves to Clang rather than an independent GCC compiler; %s", linuxToolchainGuide)
		}
		numeric, err = run(path, "-dumpfullversion", "-dumpversion")
		if err != nil {
			return Toolchain{}, fmt.Errorf("probe GCC version: %w; %s", err, linuxToolchainGuide)
		}
		numeric = strings.TrimSpace(numeric)
	} else {
		minimum = 18
		if match := linuxClangVersion.FindStringSubmatch(version); match != nil {
			numeric = match[1]
		}
	}
	match = linuxCompilerVersion.FindStringSubmatch(numeric)
	if version == "" || match == nil {
		return Toolchain{}, fmt.Errorf("cannot identify %s version from %q (%q); %s", name, version, numeric, linuxToolchainGuide)
	}
	major, err = strconv.Atoi(match[1])
	if err != nil || major < minimum {
		return Toolchain{}, fmt.Errorf("Linux toolchain requires %s >= %d, found %q; %s", name, minimum, version, linuxToolchainGuide)
	}
	tripleOutput, err := run(path, "-dumpmachine")
	if err != nil {
		return Toolchain{}, fmt.Errorf("probe %s target: %w; %s", name, err, linuxToolchainGuide)
	}
	triple := strings.TrimSpace(tripleOutput)
	if len(strings.Fields(triple)) != 1 || !strings.HasPrefix(triple, "x86_64-") || !strings.HasSuffix(triple, "-linux-gnu") {
		return Toolchain{}, fmt.Errorf("%s targets %q; expected native x86_64-linux-gnu (no musl, x32, or cross target); %s", name, triple, linuxToolchainGuide)
	}
	return Toolchain{Target: target.X8664LinuxGNU, Name: name, Path: path, Version: version, Libc: libc}, nil
}

func selectLinuxToolchain(name string, resolve func(string) (Toolchain, error)) (Toolchain, error) {
	if name != "" {
		return resolve(name)
	}
	var failures []string
	for _, candidate := range []string{"gcc", "clang"} {
		toolchain, err := resolve(candidate)
		if err == nil {
			return toolchain, nil
		}
		failures = append(failures, candidate+": "+err.Error())
	}
	return Toolchain{}, fmt.Errorf("Linux native commands require GCC >= 13 or Clang >= 18 on PATH: %s", strings.Join(failures, "; "))
}
