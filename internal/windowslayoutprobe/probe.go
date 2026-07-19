// Package windowslayoutprobe generates the closed MSVC C layout oracle for the
// declared x86_64 Windows profile.
package windowslayoutprobe

import (
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/msvccprobe"
	"github.com/0disoft/fairyc/internal/target"
)

// Unit is one deterministic MSVC C translation unit derived from validated fixtures.
type Unit struct {
	Target target.ID
	Cases  []string
	Source string
}

// Generate validates the Windows fixture manifest and translates every pass case
// into C11 size, alignment, field-size, and offset assertions.
func Generate(manifestPath string) (Unit, error) {
	targetID, cases, err := layoutfixture.NativeProbeCases(manifestPath)
	if err != nil {
		return Unit{}, err
	}
	if targetID != target.X8664WindowsMSVC {
		return Unit{}, fmt.Errorf("Windows MSVC layout probe target is %q, want %q", targetID, target.X8664WindowsMSVC)
	}
	spec, found := target.Lookup(targetID)
	if !found {
		return Unit{}, fmt.Errorf("Windows MSVC layout probe target %q is unknown", targetID)
	}
	engine, err := layout.New(spec)
	if err != nil {
		return Unit{}, err
	}
	typeEmitter, err := msvccprobe.NewTypeEmitter(spec)
	if err != nil {
		return Unit{}, err
	}
	caseIDs := make([]string, 0, len(cases))
	previous := ""
	for _, probeCase := range cases {
		if probeCase.ID == "" || previous != "" && previous >= probeCase.ID {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case IDs are empty, duplicate, or unsorted")
		}
		previous = probeCase.ID
		actual, err := engine.Of(probeCase.Type)
		if err != nil {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q: %w", probeCase.ID, err)
		}
		if !equalResult(actual, probeCase.Layout) {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q drifted from its fixture result", probeCase.ID)
		}
		if _, err := typeEmitter.EmitType(probeCase.Type); err != nil {
			return Unit{}, fmt.Errorf("Windows MSVC layout probe case %q: %w", probeCase.ID, err)
		}
		caseIDs = append(caseIDs, probeCase.ID)
	}
	var output strings.Builder
	output.WriteString("/* fairyc Windows MSVC layout probe: numeric private types, ISO C11 */\n")
	output.WriteString("#include <stddef.h>\n#include <stdint.h>\n\n")
	output.WriteString(typeEmitter.Source())
	output.WriteString("int main(void) { return 0; }\n")
	return Unit{Target: targetID, Cases: caseIDs, Source: output.String()}, nil
}

func equalResult(left, right layout.Result) bool {
	return left.SizeBytes == right.SizeBytes && left.AlignmentBytes == right.AlignmentBytes &&
		slices.Equal(left.Fields, right.Fields) && slices.Equal(left.Padding, right.Padding)
}
