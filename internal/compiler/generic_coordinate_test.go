package compiler_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func genericCoordinate(t *testing.T) packagegraph.CoordinateID {
	t.Helper()
	id, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: "generic-fixture", VersionIdentity: "1"})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func TestGenericSourceCoordinateIdentity(t *testing.T) {
	coordinate := genericCoordinate(t)
	compile := func(number hir.PackageID, owner packagegraph.CoordinateID, text string) genericplan.Plan {
		t.Helper()
		file := parseCompilerFile(t, number, "main.fy", text)
		result, plan, err := compiler.CheckGenericFunctionsForPackage(file, owner, genericplan.DefaultPlanLimits())
		if err != nil || !result.Verified {
			t.Fatalf("compile: %v %+v", err, result.Semantic)
		}
		if len(plan.Specializations) != 3 {
			t.Fatalf("expected three specializations: %+v", plan)
		}
		return plan
	}
	text := genericProgram(t)
	first := compile(hir.DefaultPackage, coordinate, text)
	renumbered := compile(hir.PackageID(7), coordinate, text)
	if !reflect.DeepEqual(first, renumbered) {
		t.Fatal("semantic package numbering changed the specialization plan")
	}
	for _, index := range []int{0, len(coordinate) - 1} {
		other := coordinate
		other[index] ^= 1
		changed := compile(hir.DefaultPackage, other, text)
		for i := range first.Specializations {
			if first.Specializations[i].ID == changed.Specializations[i].ID {
				t.Fatal("coordinate bytes omitted from specialization identity")
			}
		}
	}
	changed := compile(hir.DefaultPackage, coordinate, strings.Replace(text, "return value;", "let T copy = value; return copy;", 1))
	for i := range first.Specializations {
		if first.Specializations[i].ID == changed.Specializations[i].ID {
			t.Fatal("same coordinate concealed a transitive template change")
		}
	}
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)
	result, _, err := compiler.CheckGenericFunctionsForPackage(file, coordinate, genericplan.DefaultPlanLimits())
	if err != nil || !result.Verified {
		t.Fatalf("compile: %v", err)
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 {
		t.Fatalf("execution: %+v %v", outcome, err)
	}
	result, plan, err := compiler.CheckGenericFunctionsForPackage(file, packagegraph.CoordinateID{}, genericplan.DefaultPlanLimits())
	if err == nil || result.Verified || len(plan.Specializations) != 0 {
		t.Fatal("missing package coordinate admitted")
	}
}
