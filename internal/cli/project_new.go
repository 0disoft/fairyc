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
	invalid := func() (projectscaffold.Spec, error) {
		return projectscaffold.Spec{}, fmt.Errorf(`command "new" expects <package-name> --uid <package-uid> [--template <console|win32>]`)
	}
	if len(arguments) != 3 && len(arguments) != 5 {
		return invalid()
	}
	spec := projectscaffold.Spec{Name: arguments[0]}
	seen := map[string]bool{}
	for i := 1; i < len(arguments); i += 2 {
		option := arguments[i]
		if seen[option] {
			return invalid()
		}
		seen[option] = true
		switch option {
		case "--uid":
			spec.UID = arguments[i+1]
		case "--template":
			spec.Template = arguments[i+1]
			if spec.Template == "" {
				return invalid()
			}
		default:
			return invalid()
		}
	}
	if !seen["--uid"] {
		return invalid()
	}
	if err := projectscaffold.Validate(spec); err != nil {
		return projectscaffold.Spec{}, fmt.Errorf(`command "new": %w`, err)
	}
	return spec, nil
}
