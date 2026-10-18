package cdriver

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/0disoft/fairyc/internal/pecoff"
	"github.com/0disoft/fairyc/internal/target"
)

type windowsNativeToolchain struct{ toolchain WindowsHostToolchain }

// ClassifyWindowsNativeOutcome validates private receipts from low-level probes
// using the same rules as NativeToolchain.Execute.
func ClassifyWindowsNativeOutcome(outcome WindowsHostOutcome) (NativeOutcome, error) {
	return classifyNativeOutcome("Windows workspace", outcome.ExitCode, outcome.Stdout, outcome.Stderr, outcome.TrapControl)
}

func resolveWindowsNative(name string) (NativeToolchain, error) {
	if name != "" {
		return nil, fmt.Errorf("Windows native target does not accept --cc")
	}
	toolchain, err := ResolveWindowsHostMSVC()
	if err != nil {
		return nil, err
	}
	return windowsNativeToolchain{toolchain: toolchain}, nil
}

func (toolchain windowsNativeToolchain) Build(options NativeBuildOptions) (NativeArtifact, error) {
	artifact, err := BuildWindowsMSVCTargetCached(toolchain.toolchain, options.Profile, options.UnitSource, options.RuntimeSource)
	if err != nil {
		return NativeArtifact{}, err
	}
	if _, err := pecoff.InspectExecutable(artifact.Bytes); err != nil {
		return NativeArtifact{}, fmt.Errorf("inspect Windows target PE: %w", err)
	}
	if _, err := pecoff.InspectAMD64Unwind(artifact.Bytes); err != nil {
		return NativeArtifact{}, fmt.Errorf("inspect Windows target unwind data: %w", err)
	}
	return NativeArtifact{Artifact: Artifact{
		Bytes: artifact.Bytes, RuntimeCache: artifact.RuntimeCache, Profile: artifact.Profile, ConsumerMode: artifact.ConsumerMode,
		Toolchain: Toolchain{Target: target.X8664WindowsMSVC, Name: "msvc", Version: artifact.Toolchain.Version, Path: artifact.Toolchain.Path},
	}, TrapControl: options.TrapControl}, nil
}

func (toolchain windowsNativeToolchain) Execute(artifact NativeArtifact, options NativeExecutionOptions) (NativeOutcome, error) {
	if err := validateNativeArtifact(artifact, target.X8664WindowsMSVC); err != nil {
		return NativeOutcome{}, err
	}
	if options.Timeout == 0 {
		options.Timeout = commandTimeout
	}
	if options.Input == nil {
		options.Input = strings.NewReader("")
	}
	if options.Timeout <= 0 || options.Timeout > commandTimeout {
		return NativeOutcome{}, fmt.Errorf("MSVC target execution timeout must be greater than zero and at most %s", commandTimeout)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		return NativeOutcome{}, fmt.Errorf("MSVC target execution requires windows/amd64, got %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	for _, argument := range options.Arguments {
		if strings.ContainsRune(argument, '\x00') {
			return NativeOutcome{}, fmt.Errorf("MSVC target execution rejects NUL arguments")
		}
	}
	outcome, err := executeWindowsMSVCArtifactWithOptions(WindowsHostArtifact{
		Bytes: artifact.Bytes, Profile: artifact.Profile, ConsumerMode: artifact.ConsumerMode, TrapControl: artifact.TrapControl,
	}, "MSVC target", options.Input, options.Arguments, options.Timeout, options.WorkingDirectory)
	if err != nil {
		return NativeOutcome{}, err
	}
	return finishNativeOutcome("Windows workspace", outcome.ExitCode, outcome.Stdout, outcome.Stderr, outcome.TrapControl, options)
}
