package cdriver

import (
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

// This file intentionally has no GOOS build constraint: target admission and
// unsupported-host errors must work on every compiler host.
type linuxNativeToolchain struct{ toolchain Toolchain }

func resolveLinuxNative(name string) (result NativeToolchain, err error) {
	defer func() { err = driverError(ErrorToolchain, "resolveLinuxNative", err) }()
	toolchain, err := selectLinuxToolchain(name, func(candidate string) (Toolchain, error) {
		return Resolve(target.X8664LinuxGNU, candidate)
	})
	if err != nil {
		return nil, err
	}
	return linuxNativeToolchain{toolchain: toolchain}, nil
}

func (toolchain linuxNativeToolchain) Build(options NativeBuildOptions) (result NativeArtifact, err error) {
	defer func() { err = driverError(ErrorBuild, "Build", err) }()
	artifact, err := BuildCached(toolchain.toolchain, options.Profile, options.UnitSource, options.RuntimeSource)
	return NativeArtifact{Artifact: artifact, TrapControl: options.TrapControl}, err
}

func (toolchain linuxNativeToolchain) Execute(artifact NativeArtifact, options NativeExecutionOptions) (result NativeOutcome, err error) {
	defer func() { err = driverError(ErrorExecution, "Execute", err) }()
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
	return finishNativeOutcome("Linux workspace", outcome.ExitCode, outcome.Stdout, outcome.Stderr, receipt, options)
}
