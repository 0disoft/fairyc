package compiler_test

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/genericplan"
)

// Compare source-level call targets after resolving each generated generic
// name through the plan that produced it. Source-hash-derived names alone
// cannot identify whether the same SDK operation was called in both forms.
func TestPrivateVectorSDKCandidateScannerCanonicalCallParity(t *testing.T) {
	variants := realScannerVectorVariants(t)
	if len(variants) != 5 || variants[0].name != "explicit index" || variants[4].name != "all vector for" {
		t.Fatal("real scanner variants changed")
	}
	var calls [2][2]map[string]int
	for variantIndex, variant := range []realScannerVectorVariant{variants[0], variants[4]} {
		var plan genericplan.Plan
		program, _, specializations := scannerBudgetFairyProgramWithMainTimingPlan(t, variant,
			realScannerExecutionMain, nil, &plan)
		if specializations != 21 || len(plan.Specializations) != specializations {
			t.Fatalf("%s specialization plan changed: %d/%d", variant.name,
				specializations, len(plan.Specializations))
		}
		canonical := scannerCanonicalSpecializations(plan)
		for functionIndex, name := range []string{"$p1::scanner::report_work", "$p1::scanner::storage"} {
			shape, err := scannerMIRFunctionShape(program, name)
			if err != nil {
				t.Fatal(err)
			}
			calls[variantIndex][functionIndex], err = scannerCanonicalCalls(shape.Calls, canonical)
			if err != nil {
				t.Fatalf("%s %s call target: %v", variant.name, name, err)
			}
		}
	}
	for index, name := range []string{"scanner::report_work", "scanner::storage"} {
		if !maps.Equal(calls[0][index], calls[1][index]) {
			t.Fatalf("%s changed normalized call targets: indexed=%v candidate=%v",
				name, calls[0][index], calls[1][index])
		}
	}
}

func scannerCanonicalSpecializations(plan genericplan.Plan) map[string]string {
	canonical := make(map[string]string, len(plan.Specializations))
	for _, specialization := range plan.Specializations {
		var identity strings.Builder
		identity.WriteString(string(specialization.Declaration))
		for _, binding := range specialization.Bindings {
			fmt.Fprintf(&identity, "|%s=%s", binding.Parameter, binding.Type)
		}
		canonical["_fy_generic_"+string(specialization.ID)] = identity.String()
	}
	return canonical
}

func scannerCanonicalCalls(raw map[string]int, specializations map[string]string) (map[string]int, error) {
	calls := make(map[string]int, len(raw))
	for name, count := range raw {
		if suffix := name[strings.LastIndex(name, "::")+2:]; strings.HasPrefix(suffix, "_fy_generic_") {
			identity, ok := specializations[suffix]
			if !ok {
				return nil, fmt.Errorf("missing generic origin for %s", name)
			}
			name = identity
		}
		calls[name] += count
	}
	return calls, nil
}
