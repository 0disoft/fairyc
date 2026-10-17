package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
)

type hostRunOptions struct {
	source            string
	additionalSources []string
	profile           cdriver.Profile
}

func runHost(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseHostRunOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsHost(programSourceSet(options.source, options.additionalSources), options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsHost(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if outcome.Stdout != "" || outcome.Stderr != "" {
		return buildFailure(stderr, fmt.Errorf("Windows host executable produced unexpected output"))
	}
	if category, trapped := windowsHostTrapCategory(outcome.ExitCode); trapped {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(stdout, "native result: %d (windows/amd64 host-smoke, msvc %s, %s)\n", outcome.ExitCode, artifact.Toolchain.Version, artifact.Profile.Name); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func compileWindowsHost(sources []string, profile cdriver.Profile, sourceRoot string, stderr io.Writer) (cdriver.WindowsHostArtifact, int) {
	lowered, code := lowerProgramSources(sources, sourceRoot, stderr, "Windows host-smoke C-emission")
	if code != ExitSuccess {
		return cdriver.WindowsHostArtifact{}, code
	}
	unit, err := cbackend.Emit(lowered.program, lowered.entryName, cbackend.Stage0Profile())
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return cdriver.WindowsHostArtifact{}, subsetCapabilityError(stderr, "Windows host-smoke C-emission", capability)
		}
		return cdriver.WindowsHostArtifact{}, internalError(stderr)
	}
	if err := validateWindowsHostUnit(unit); err != nil {
		return cdriver.WindowsHostArtifact{}, subsetCapabilityError(stderr, "Windows host-smoke", err)
	}
	runtimeSource, err := productSupportSource(nil)
	if err != nil {
		return cdriver.WindowsHostArtifact{}, internalError(stderr)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, err)
	}
	artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, runtimeSource)
	if err != nil {
		return cdriver.WindowsHostArtifact{}, buildFailure(stderr, err)
	}
	return artifact, ExitSuccess
}

func parseHostRunOptions(arguments []string) (hostRunOptions, error) {
	if len(arguments) == 0 {
		return hostRunOptions{}, fmt.Errorf("experimental command \"run-host\" expects one entry source file and explicit --profile value")
	}
	if arguments[0] == "" || strings.HasPrefix(arguments[0], "-") {
		return hostRunOptions{}, fmt.Errorf("experimental command \"run-host\" expects the entry source file before its flags")
	}
	options := hostRunOptions{source: arguments[0]}
	_, err := parseCommandFlags(arguments[1:], flagSchema{Command: "run-host", Experimental: true, Flags: []flagDefinition{{Name: "--source", Kind: flagSlice}, {Name: "--profile", Kind: flagScalar}}, Required: []string{"--profile"}}, func(flag, value string) error {
		switch flag {
		case "--source":
			if err := appendProgramSource("run-host", &options.additionalSources, value); err != nil {
				return err
			}
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return fmt.Errorf("experimental command \"run-host\" requires --profile low or --profile optimized")
			}
		}
		return nil
	})
	if err != nil {
		return hostRunOptions{}, err
	}
	return options, nil
}

func validateWindowsHostUnit(unit cbackend.Unit) error {
	if len(unit.ForeignSymbols) != 0 {
		return fmt.Errorf("foreign calls and provisional standard-library linkage are not target-neutral")
	}
	for _, feature := range unit.Features {
		if feature == "foreign-c-call" || feature == "foreign-c-struct-result" || feature == "repr-c-struct" || strings.HasPrefix(feature, "c-") {
			return fmt.Errorf("feature %q requires a declared Windows target ABI", feature)
		}
	}
	return nil
}

func windowsHostTrapCategory(exitCode int32) (string, bool) {
	for _, category := range []string{
		"bounds", "integer-overflow", "division-by-zero", "signed-division-overflow",
		"invalid-shift", "assertion-failed", "unreachable",
	} {
		code, found := cbackend.TrapCode(category)
		if found && exitCode == int32(productTrapExitBase)+int32(code) {
			return category, true
		}
	}
	return "", false
}
