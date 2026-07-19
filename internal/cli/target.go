package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/0disoft/fairyc/internal/target"
)

func runTargets(arguments []string, stdout, stderr io.Writer) int {
	if len(arguments) != 0 {
		return invocationError(stderr, "command \"targets\" does not accept arguments")
	}
	specs := target.All()
	if len(specs) == 0 {
		return internalError(stderr)
	}
	var output strings.Builder
	for index, spec := range specs {
		if err := target.Validate(spec); err != nil {
			return internalError(stderr)
		}
		if index > 0 {
			output.WriteByte('\n')
		}
		fmt.Fprintf(&output, "%s\n", spec.ID)
		fmt.Fprintf(&output, "  schema: %s\n", spec.SchemaVersion)
		fmt.Fprintf(&output, "  emission subset: %s\n", spec.Evidence.EmissionSubset)
		fmt.Fprintf(&output, "  emittable: %s\n", yesNo(spec.Evidence.Emittable))
		fmt.Fprintf(&output, "  conformant: %s\n", yesNo(spec.Evidence.Conformant))
		fmt.Fprintf(&output, "  supported: %s\n", yesNo(spec.Evidence.Supported))
		output.WriteString("  missing evidence:\n")
		for _, missing := range spec.Evidence.Missing {
			fmt.Fprintf(&output, "    - %s\n", missing)
		}
	}
	if _, err := io.WriteString(stdout, output.String()); err != nil {
		return outputError(stderr)
	}
	return ExitSuccess
}

func yesNo(value bool) string {
	if value {
		return "yes"
	}
	return "no"
}

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
