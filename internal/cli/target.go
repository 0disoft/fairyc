package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/0disoft/fairyc/internal/target"
)

func runTargetSpec(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) != 1 {
		return invocationError(stderr, "experimental command \"target-spec\" expects exactly one target id")
	}
	spec, found := target.Lookup(target.ID(arguments[0]))
	if !found {
		return invocationError(stderr, fmt.Sprintf("experimental command \"target-spec\" does not know target %q", arguments[0]))
	}
	if err := target.Validate(spec); err != nil {
		return internalError(stderr)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(spec); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}
