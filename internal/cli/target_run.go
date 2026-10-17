package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

type targetRunOptions struct {
	source            string
	additionalSources []string
	targetID          target.ID
	profile           cdriver.Profile
}

func runTarget(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseTargetRunOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsTarget(programSourceSet(options.source, options.additionalSources), options.targetID, options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
	if err != nil {
		return buildFailure(stderr, err)
	}
	if outcome.Stdout != "" || outcome.Stderr != "" {
		return buildFailure(stderr, fmt.Errorf("Windows target executable produced unexpected output"))
	}
	if category, trapped := windowsHostTrapCategory(outcome.ExitCode); trapped {
		if _, err := fmt.Fprintf(stderr, "trap: %s\n", category); err != nil {
			return ExitInternalFailure
		}
		return ExitProgramTrap
	}
	if _, err := fmt.Fprintf(
		stdout,
		"native result: %d (%s, msvc %s, %s)\n",
		outcome.ExitCode,
		options.targetID,
		artifact.Toolchain.Version,
		artifact.Profile.Name,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseTargetRunOptions(arguments []string) (targetRunOptions, error) {
	return parseTargetExecutionOptions("run-target", arguments)
}

func parseTargetExecutionOptions(command string, arguments []string) (targetRunOptions, error) {
	if len(arguments) == 0 {
		return targetRunOptions{}, fmt.Errorf("experimental command %q expects one entry source file and explicit --target and --profile values", command)
	}
	options := targetRunOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return targetRunOptions{}, fmt.Errorf("experimental command %q expects the source file before its flags", command)
	}
	_, err := parseCommandFlags(arguments[1:], flagSchema{Command: command, Experimental: true,
		Flags: []flagDefinition{{Name: "--source", Kind: flagSlice}, {Name: "--target", Kind: flagScalar}, {Name: "--profile", Kind: flagScalar}}, Required: []string{"--target", "--profile"}}, func(flag, value string) error {
		switch flag {
		case "--source":
			if err := appendProgramSource(command, &options.additionalSources, value); err != nil {
				return err
			}
		case "--target":
			if target.ID(value) != target.X8664WindowsMSVC {
				return fmt.Errorf("experimental %s currently requires --target %s", command, target.X8664WindowsMSVC)
			}
			options.targetID = target.X8664WindowsMSVC
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return fmt.Errorf("experimental command %q requires --profile low or --profile optimized", command)
			}
		}
		return nil
	})
	if err != nil {
		return targetRunOptions{}, err
	}

	return options, nil
}
