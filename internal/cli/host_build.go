package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
)

type hostBuildOptions struct {
	source  string
	profile cdriver.Profile
	output  string
}

func runHostBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseHostBuildOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsHost(options.source, options.profile, sourceRoot, stderr)
	if code != ExitSuccess {
		return code
	}
	if err := writeBuildArtifact(sourceRoot, options.output, artifact.Bytes); err != nil {
		return buildFailure(stderr, err)
	}
	if _, err := fmt.Fprintf(
		stdout,
		"built: %s (windows/amd64 host-smoke, msvc %s, %s)\n",
		options.output,
		artifact.Toolchain.Version,
		artifact.Profile.Name,
	); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseHostBuildOptions(arguments []string) (hostBuildOptions, error) {
	if len(arguments) == 0 {
		return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" expects one source file and explicit --profile and --output values")
	}
	options := hostBuildOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" expects the source file before its flags")
	}
	seen := make(map[string]bool, 2)
	for index := 1; index < len(arguments); {
		flag := arguments[index]
		if flag != "--profile" && flag != "--output" {
			return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" does not accept flag %q", flag)
		}
		if seen[flag] {
			return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" received %s more than once", flag)
		}
		if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
			return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" requires a value for %s", flag)
		}
		seen[flag] = true
		value := arguments[index+1]
		switch flag {
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" requires --profile low or --profile optimized")
			}
		case "--output":
			normalized, err := validateBuildOutput(value)
			if err != nil {
				return hostBuildOptions{}, err
			}
			if !strings.HasSuffix(strings.ToLower(normalized), ".exe") {
				return hostBuildOptions{}, fmt.Errorf("experimental build-host output must end with .exe")
			}
			options.output = normalized
		}
		index += 2
	}
	for _, flag := range []string{"--profile", "--output"} {
		if !seen[flag] {
			return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" requires %s", flag)
		}
	}
	return options, nil
}
