package cli

import (
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/projectscaffold"
)

func runStableNew(arguments []string, sourceRoot string, stdout, stderr io.Writer) int {
	spec, err := parseNewSpec(arguments)
	if err != nil {
		return invocationError(stderr, err.Error())
	}
	if err := projectscaffold.Create(sourceRoot, spec); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "fairyc: project creation failed: %s\n", err); writeErr != nil {
			return ExitInternalFailure
		}
		return ExitPackageFailure
	}
	if _, err := fmt.Fprintf(stdout, "created Fairy C package %q at %s\n", spec.Name, spec.Name); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func parseNewSpec(arguments []string) (projectscaffold.Spec, error) {
	if len(arguments) != 3 || arguments[1] != "--uid" {
		return projectscaffold.Spec{}, fmt.Errorf(`command "new" expects <package-name> --uid <package-uid>`)
	}
	spec := projectscaffold.Spec{Name: arguments[0], UID: arguments[2]}
	if err := projectscaffold.Validate(spec); err != nil {
		return projectscaffold.Spec{}, fmt.Errorf(`command "new": %w`, err)
	}
	return spec, nil
}
