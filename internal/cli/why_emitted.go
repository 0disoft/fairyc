package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func runWhyEmitted(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	const command = "why-emitted"
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
	if err != nil {
		var selection cbackend.EmissionSelectorError
		var capability cbackend.CapabilityError
		switch {
		case errors.As(err, &selection):
			return invocationError(stderr, err.Error())
		case errors.As(err, &capability):
			return subsetCapabilityError(stderr, "C-emission", capability)
		default:
			return internalError(stderr)
		}
	}
	if err := json.NewEncoder(stdout).Encode(report); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
