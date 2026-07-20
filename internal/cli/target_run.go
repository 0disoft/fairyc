package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

type targetRunOptions struct {
	source   string
	targetID target.ID
	profile  cdriver.Profile
}

func runTarget(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseTargetRunOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsTarget(options.source, options.targetID, options.profile, sourceRoot, stderr)
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
	if len(arguments) == 0 {
		return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" expects one source file and explicit --target and --profile values")
	}
	options := targetRunOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" expects the source file before its flags")
	}
	seen := make(map[string]bool, 2)
	for index := 1; index < len(arguments); {
		flag := arguments[index]
		if flag != "--target" && flag != "--profile" {
			return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" does not accept flag %q", flag)
		}
		if seen[flag] {
			return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" received %s more than once", flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[index+1]
		switch flag {
		case "--target":
			if target.ID(value) != target.X8664WindowsMSVC {
				return targetRunOptions{}, fmt.Errorf("experimental run-target currently requires --target %s", target.X8664WindowsMSVC)
			}
			options.targetID = target.X8664WindowsMSVC
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" requires --profile low or --profile optimized")
			}
		}
		index += 2
	}
	for _, flag := range []string{"--target", "--profile"} {
		if !seen[flag] {
			return targetRunOptions{}, fmt.Errorf("experimental command \"run-target\" requires %s", flag)
		}
	}
	return options, nil
}
