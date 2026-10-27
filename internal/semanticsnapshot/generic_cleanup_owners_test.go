package semanticsnapshot_test

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/semanticsnapshot"
)

func TestGenericCleanupOwnersKeepEqualNamesInSeparatePackages(t *testing.T) {
	const text = `module main;
resource Token<T> { T value; cleanup {
    let T copy = this.value; let ref T view = ref copy; assert(*view == this.value);
} }
fn i32 main() { let Token<i32> token = Token<i32> { .value = 7 }; return token.value; }`
	build := func(ids [2]hir.PackageID) semanticsnapshot.Snapshot {
		t.Helper()
		input := semantic.PackageInput{}
		var packages []semanticsnapshot.PackageInput
		coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
		for i, id := range ids {
			role := "dependency"
			if i == 0 {
				role = "root"
			}
			metadata := packageInput(t, id, fmt.Sprintf("example.cleanup.owner.%d", i), fmt.Sprintf("owner-%d", i), "1", "main.fy", text, role)
			packages = append(packages, metadata)
			coordinates[id] = metadata.Coordinate
			input.Files = append(input.Files, semantic.File{Package: id, Path: "main.fy", Module: analyzeSyntax(t, "main.fy", text)})
		}
		// The second build also changes source and metadata discovery order.
		if ids[0] > ids[1] {
			slices.Reverse(input.Files)
			slices.Reverse(packages)
		}
		compiled, _, err := compiler.CheckGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
		if err != nil || !compiled.Verified {
			t.Fatalf("compile resource owners: verified=%t error=%v", compiled.Verified, err)
		}
		snapshot, err := semanticsnapshot.BuildCompiler(compiled, packages)
		if err != nil {
			t.Fatal(err)
		}
		owners := map[string]bool{}
		for _, specialization := range snapshot.Specializations() {
			if specialization.Declaration.Kind != semanticsnapshot.DeclarationResource {
				continue
			}
			if specialization.Aggregate == nil || specialization.Cleanup == nil || specialization.Cleanup.Function != specialization.Declaration || len(specialization.Cleanup.Borrows) != 1 {
				t.Fatalf("resource lost its exact cleanup owner: %+v", specialization)
			}
			owner := specialization.Declaration.PackageInstance
			if specialization.Cleanup.Borrows[0].Span.PackageInstance != owner || owners[owner] {
				t.Fatal("equal resource names shared or duplicated a cleanup owner")
			}
			owners[owner] = true
		}
		if len(owners) != 2 {
			t.Fatalf("resource owner count=%d, want 2", len(owners))
		}
		// The reused coordinate must still be proven by the source/type join.
		compiled.Semantic.HIR.Modules[0].Resources[0].Type = compiled.Semantic.HIR.Modules[1].Resources[0].Type
		if _, err := semanticsnapshot.BuildCompiler(compiled, packages); err == nil {
			t.Fatal("resource type with another package's coordinate was accepted")
		}
		return snapshot
	}
	first := build([2]hir.PackageID{1, 2})
	again := build([2]hir.PackageID{99, 41})
	assertSnapshotsEqual(t, first, again)
	if !reflect.DeepEqual(first.Specializations(), again.Specializations()) {
		t.Fatal("package numbering or discovery order changed cleanup specializations")
	}
}
