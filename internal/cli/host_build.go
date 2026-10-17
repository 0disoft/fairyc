package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cdriver"
)

type hostBuildOptions struct {
	source            string
	additionalSources []string
	profile           cdriver.Profile
	output            string
}

func runHostBuild(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseHostBuildOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	artifact, code := compileWindowsHost(programSourceSet(options.source, options.additionalSources), options.profile, sourceRoot, stderr)
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
		return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" expects one entry source file and explicit --profile and --output values")
	}
	options := hostBuildOptions{source: arguments[0]}
	if options.source == "" || strings.HasPrefix(options.source, "-") {
		return hostBuildOptions{}, fmt.Errorf("experimental command \"build-host\" expects the source file before its flags")
	}
	_, err := parseCommandFlags(arguments[1:], flagSchema{Command: "build-host", Experimental: true,
		Flags: []flagDefinition{
			{Name: "--source", Kind: flagSlice},
			{Name: "--profile", Kind: flagScalar},
			{Name: "--output", Kind: flagScalar},
		}, Required: []string{"--profile", "--output"}}, func(flag, value string) error {
		switch flag {
		case "--source":
			if err := appendProgramSource("build-host", &options.additionalSources, value); err != nil {
				return err
			}
		case "--profile":
			switch value {
			case cdriver.LowProfile.Name:
				options.profile = cdriver.LowProfile
			case cdriver.OptimizedProfile.Name:
				options.profile = cdriver.OptimizedProfile
			default:
				return fmt.Errorf("experimental command \"build-host\" requires --profile low or --profile optimized")
			}
		case "--output":
			normalized, err := validateBuildOutput(value)
			if err != nil {
				return err
			}
			if !strings.HasSuffix(strings.ToLower(normalized), ".exe") {
				return fmt.Errorf("experimental build-host output must end with .exe")
			}
			options.output = normalized
		}
		return nil
	})
	if err != nil {
		return hostBuildOptions{}, err
	}

	return options, nil
}
