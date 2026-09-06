package semantic_test

import (
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestVisibleNominalFactsUseResolvedProvidersAndVisibility(t *testing.T) {
	input := semantic.PackageInput{Files: []semantic.File{
		parsePackageFile(t, 1, "app_a.fy", "module app_a; import util as u; struct Local { i32 value; } fn i32 main() { return 0; }"),
		parsePackageFile(t, 2, "app_b.fy", "module app_b; import util as u; fn i32 main() { return 0; }"),
		parsePackageFile(t, 3, "util.fy", "module util; pub struct Token { i32 value; } struct Hidden { i32 value; }"),
		parsePackageFile(t, 4, "util.fy", "module util; pub struct Token { u32 value; }"),
	}, Imports: []semantic.PackageImport{
		{FromPackage: 1, ImportPath: "util", ToPackage: 3},
		{FromPackage: 2, ImportPath: "util", ToPackage: 4},
	}}
	result, err := semantic.AnalyzePackages(input)
	if err != nil || result.CompletedThrough != semantic.PhaseFlow {
		t.Fatalf("analysis: %+v %v", result, err)
	}
	var left, right semantic.NominalTypeFact
	for _, fact := range result.NominalTypes {
		if fact.Name == "u::Hidden" {
			t.Fatal("private imported type entered visible facts")
		}
		if fact.Name == "u::Token" && fact.Package == 1 {
			left = fact
		}
		if fact.Name == "u::Token" && fact.Package == 2 {
			right = fact
		}
	}
	if left.CanonicalType == "" || right.CanonicalType == "" || left.CanonicalType == right.CanonicalType {
		t.Fatalf("provider identity lost: %+v %+v", left, right)
	}
	slices.Reverse(input.Files)
	reordered, err := semantic.AnalyzePackages(input)
	if err != nil || !reflect.DeepEqual(result.NominalTypes, reordered.NominalTypes) {
		t.Fatal("file order changed nominal facts")
	}
	input.Files = append(input.Files, parsePackageFile(t, 9, "bad.fy", "module bad; fn i32 bad() { return missing; }"))
	rejected, err := semantic.AnalyzePackages(input)
	if err != nil || len(rejected.NominalTypes) != 0 || rejected.FailedPhase == semantic.PhaseNone {
		t.Fatalf("rejected analysis published nominal facts: %+v %v", rejected, err)
	}
}
