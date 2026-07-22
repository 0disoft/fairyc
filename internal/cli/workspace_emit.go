package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/target"
)

type stableEmitOptions struct {
	targetID target.ID
	locked   bool
}

func runStableEmit(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseStableEmitOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	lowered, code := lowerWorkspaceExecutable(
		sourceRoot,
		options.locked,
		workspaceCompilePolicy{stableDiagnostics: true},
		"C-emission",
		stderr,
	)
	if code != ExitSuccess {
		return code
	}
	var profile cbackend.Profile
	switch options.targetID {
	case target.X8664LinuxGNU:
		profile = cbackend.Stage0Profile()
	case target.X8664WindowsMSVC:
		profile = cbackend.Stage0WindowsMSVCProfile()
	default:
		return internalError(stderr)
	}
	unit, err := cbackend.Emit(lowered.program, lowered.entryName, profile)
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return workspaceSubsetCapabilityError(
				workspaceCompilePolicy{stableDiagnostics: true},
				stderr,
				fmt.Sprintf("%s C-emission", options.targetID),
				capability,
			)
		}
		return internalError(stderr)
	}
	if unit.Target != string(options.targetID) || unit.Dialect != cbackend.DialectC11 ||
		unit.Instrumentation != "" || unit.Source == "" || !strings.HasSuffix(unit.Source, "\n") {
		return internalError(stderr)
	}
	if _, err := io.WriteString(stdout, unit.Source); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseStableEmitOptions(arguments []string) (stableEmitOptions, error) {
	if len(arguments) == 0 {
		return stableEmitOptions{}, fmt.Errorf(`command "emit" expects form c and required --target <target-id>`)
	}
	if arguments[0] != "c" {
		return stableEmitOptions{}, fmt.Errorf(`command "emit" supports only form c`)
	}
	options := stableEmitOptions{}
	seen := make(map[string]bool, 2)
	for index := 1; index < len(arguments); {
		flag := arguments[index]
		switch flag {
		case "--locked":
			if seen[flag] {
				return stableEmitOptions{}, fmt.Errorf(`command "emit" received --locked more than once`)
			}
			seen[flag] = true
			options.locked = true
			index++
		case "--target":
			if seen[flag] {
				return stableEmitOptions{}, fmt.Errorf(`command "emit" received --target more than once`)
			}
			if index+1 >= len(arguments) || strings.HasPrefix(arguments[index+1], "--") {
				return stableEmitOptions{}, fmt.Errorf(`command "emit" requires a value for --target`)
			}
			seen[flag] = true
			candidate := target.ID(arguments[index+1])
			if candidate != target.X8664LinuxGNU && candidate != target.X8664WindowsMSVC {
				return stableEmitOptions{}, fmt.Errorf(`command "emit" does not know target %q`, arguments[index+1])
			}
			options.targetID = candidate
			index += 2
		default:
			return stableEmitOptions{}, fmt.Errorf(`command "emit" does not accept argument %q`, flag)
		}
	}
	if !seen["--target"] {
		return stableEmitOptions{}, fmt.Errorf(`command "emit" requires --target <target-id>`)
	}
	return options, nil
}
