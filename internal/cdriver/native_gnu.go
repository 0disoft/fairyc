package cdriver

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

// This file intentionally has no GOOS build constraint: target admission and
// unsupported-host errors must work on every compiler host.
type linuxNativeToolchain struct{ toolchain Toolchain }

func resolveLinuxNative(name string) (NativeToolchain, error) {
	if name == "" {
		for _, candidate := range []string{"gcc", "clang"} {
			if _, err := exec.LookPath(candidate); err == nil {
				name = candidate
				break
			}
		}
		if name == "" {
			return nil, fmt.Errorf("Linux native commands require GCC or Clang on PATH")
		}
	}
	toolchain, err := Resolve(target.X8664LinuxGNU, name)
	if err != nil {
		return nil, err
	}
	return linuxNativeToolchain{toolchain: toolchain}, nil
}

func (toolchain linuxNativeToolchain) Build(options NativeBuildOptions) (NativeArtifact, error) {
	artifact, err := BuildCached(toolchain.toolchain, options.Profile, options.UnitSource, options.RuntimeSource)
	return NativeArtifact{Artifact: artifact, TrapControl: options.TrapControl}, err
}

func (toolchain linuxNativeToolchain) Execute(artifact NativeArtifact, options NativeExecutionOptions) (NativeOutcome, error) {
	if err := validateNativeArtifact(artifact, target.X8664LinuxGNU); err != nil {
		return NativeOutcome{}, err
	}
	if options.Timeout == 0 {
		options.Timeout = commandTimeout
	}
	if options.Input == nil {
		options.Input = strings.NewReader("")
	}
	outcome, err := ExecuteLinuxTargetWithOptions(artifact.Artifact, LinuxExecutionOptions{
		Input: options.Input, Arguments: options.Arguments, WorkingDirectory: options.WorkingDirectory, Timeout: options.Timeout,
	})
	if err != nil {
		return NativeOutcome{}, err
	}
	receipt := ""
	if outcome.TrapCategory != 0 {
		receipt = string([]byte{outcome.TrapCategory})
	}
	return classifyNativeOutcome("Linux", outcome.ExitCode, outcome.Stdout, outcome.Stderr, receipt)
}
