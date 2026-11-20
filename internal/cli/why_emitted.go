package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/target"
)

func runWhyEmitted(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	const command = "why-emitted"
	if len(arguments) > 0 && arguments[0] == "--workspace" {
		return runWorkspaceWhyEmitted(arguments[1:], sourceRoot, stdout, stderr)
	}
	if len(arguments) == 0 || arguments[0] == "" || strings.HasPrefix(arguments[0], "-") {
		return invocationError(stderr, "experimental command \"why-emitted\" expects the entry source file before its flags")
	}
	var selector string
	var additional []string
	_, err := parseCommandFlags(arguments[1:], flagSchema{
		Command: command, Experimental: true, Required: []string{"--symbol"},
		Flags: []flagDefinition{{Name: "--symbol", Kind: flagScalar}, {Name: "--source", Kind: flagSlice}},
	}, func(flag, value string) error {
		if flag == "--source" {
			return appendProgramSource(command, &additional, value)
		}
		if strings.TrimSpace(value) == "" {
			return fmt.Errorf("why-emitted requires a nonempty symbol name")
		}
		selector = value
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	lowered, code := lowerProgramSources(programSourceSet(arguments[0], additional), sourceRoot, stderr, "C-emission")
	if code != ExitSuccess {
		return code
	}
	_, report, err := cbackend.EmitWithExplanation(lowered.program, lowered.entryName, cbackend.Stage0Profile(), selector)
	return writeEmissionExplanation(report, err, workspaceCompilePolicy{}, "C-emission", stdout, stderr)
}

func runWorkspaceWhyEmitted(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	var profile cbackend.Profile
	var selector string
	var locked bool
	_, err := parseCommandFlags(arguments, flagSchema{
		Command: "why-emitted", Experimental: true, Required: []string{"--target", "--symbol"},
		Flags: []flagDefinition{
			{Name: "--target", Kind: flagScalar}, {Name: "--symbol", Kind: flagScalar}, {Name: "--locked", Kind: flagSwitch},
		},
	}, func(flag, value string) error {
		switch flag {
		case "--target":
			var ok bool
			profile, ok = workspaceEmissionProfile(target.ID(value))
			if !ok {
				return fmt.Errorf("experimental command %q does not know target %q", "why-emitted", value)
			}
		case "--symbol":
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("why-emitted requires a nonempty symbol name")
			}
			selector = value
		case "--locked":
			locked = true
		}
		return nil
	})
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	policy := workspaceCompilePolicy{stableDiagnostics: true}
	lowered, code := lowerWorkspaceExecutable(sourceRoot, locked, policy, "workspace C transport", stderr)
	if code != ExitSuccess {
		return code
	}
	emit := cbackend.EmitWithExplanation
	if lowered.sdkStorage {
		emit = cbackend.EmitStorageTransfersWithExplanation
	}
	_, report, err := emit(lowered.program, lowered.entryName, profile, selector)
	return writeEmissionExplanation(report, err, policy, fmt.Sprintf("%s workspace C transport", profile.Target), stdout, stderr)
}

func writeEmissionExplanation(report cbackend.EmissionExplanation, err error, policy workspaceCompilePolicy, subset string, stdout, stderr io.Writer) int {
	if err != nil {
		var selection cbackend.EmissionSelectorError
		var capability cbackend.CapabilityError
		switch {
		case errors.As(err, &selection):
			return invocationError(stderr, err.Error())
		case errors.As(err, &capability):
			return workspaceSubsetCapabilityError(policy, stderr, subset, capability)
		default:
			return internalError(stderr)
		}
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
