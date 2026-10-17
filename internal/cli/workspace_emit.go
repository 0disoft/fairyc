package cli

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/target"
)

const experimentalWorkspaceCCommand = "emit-workspace-c"

type experimentalWorkspaceEmitOptions struct {
	targetID target.ID
	locked   bool
}

func runExperimentalWorkspaceEmit(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	options, err := parseExperimentalWorkspaceEmitOptions(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	// Demoting generated C changes command stability, not the existing workspace
	// diagnostic and exit-code classification shared with stable build/run.
	policy := workspaceCompilePolicy{stableDiagnostics: true}
	lowered, code := lowerWorkspaceExecutable(
		sourceRoot,
		options.locked,
		policy,
		"workspace C transport",
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
	emit := cbackend.Emit
	if lowered.sdkStorage {
		emit = cbackend.EmitStorageTransfers
	}
	unit, err := emit(lowered.program, lowered.entryName, profile)
	if err != nil {
		var capability cbackend.CapabilityError
		if errors.As(err, &capability) {
			return workspaceSubsetCapabilityError(
				policy,
				stderr,
				fmt.Sprintf("%s workspace C transport", options.targetID),
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

func rejectRemovedStableEmit(stderr io.Writer) int {
	return invocationError(
		stderr,
		`command "emit" is not a stable product command; for compiler debugging use "fairyc experimental emit-workspace-c --target <target-id> [--locked]"`,
	)
}

func parseExperimentalWorkspaceEmitOptions(arguments []string) (experimentalWorkspaceEmitOptions, error) {
	options := experimentalWorkspaceEmitOptions{}
	_, err := parseCommandFlags(arguments, flagSchema{Command: experimentalWorkspaceCCommand, Experimental: true,
		Flags: []flagDefinition{{Name: "--locked", Kind: flagSwitch}, {Name: "--target", Kind: flagScalar}}, Required: []string{"--target"},
		Unknown: func(flag string) error {
			return fmt.Errorf(`experimental command %q does not accept argument %q`, experimentalWorkspaceCCommand, flag)
		},
		RequiredError: func(string) error {
			return fmt.Errorf(`experimental command %q requires --target <target-id>`, experimentalWorkspaceCCommand)
		},
	}, func(flag, value string) error {
		switch flag {
		case "--locked":
			options.locked = true
		case "--target":
			candidate := target.ID(value)
			if candidate != target.X8664LinuxGNU && candidate != target.X8664WindowsMSVC {
				return fmt.Errorf(`experimental command %q does not know target %q`, experimentalWorkspaceCCommand, value)
			}
			options.targetID = candidate
		}
		return nil
	})
	if err != nil {
		return experimentalWorkspaceEmitOptions{}, err
	}
	return options, nil
}
