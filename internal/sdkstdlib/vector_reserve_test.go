package sdkstdlib_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

// Exercise the shared raw helpers through concrete scalar, nominal and owning
// element types, including the discarded T temporaries in truncate and cleanup.
// This is a source/Flow/MIR check; it does not invoke the native toolchain.
func TestVectorSharedChecksPreserveConcreteElements(t *testing.T) {
	for _, sample := range []struct{ name, declaration, element, value string }{
		{"scalar", "", "u32", "7"},
		{"nominal", "struct Item { i32 code; }", "Item", "Item { .code = 7 }"},
		{"resource", "resource Item { i32 code; cleanup {} }", "Item", "Item { .code = 7 }"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			text := strings.NewReplacer("ELEMENT", sample.element, "VALUE", sample.value, "DECLARATION", sample.declaration).Replace(`module main;
import std::vec as v;
DECLARATION
fn void exercise() {
    var values = v::empty<ELEMENT>();
    assert(switch (v::reserve<ELEMENT>(ref mut values, 4)) { case .ok(_): { true } case .err(_): { false } });
    assert(switch (v::push<ELEMENT>(ref mut values, VALUE)) { case .ok(_): { true } case .err(_): { false } });
    let ref ELEMENT first = v::at<ELEMENT>(ref values, 0);
    let ref mut ELEMENT first_mut = v::at_mut<ELEMENT>(ref mut values, 0);
    let ELEMENT popped = switch (v::pop<ELEMENT>(ref mut values)) { case .some(item): { item } case .none: { assert(false); } };
    assert(switch (v::push<ELEMENT>(ref mut values, popped)) { case .ok(_): { true } case .err(_): { false } });
    assert(v::truncate<ELEMENT>(ref mut values, 0) == 1);
    assert(v::clear<ELEMENT>(ref mut values) == 0);
    assert(switch (v::push<ELEMENT>(ref mut values, VALUE)) { case .ok(_): { true } case .err(_): { false } });
}
fn i32 main() { exercise(); return 0; }
`)
			tree, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
			if err != nil || tree.AST == nil || len(tree.Tree.Diagnostics) != 0 {
				t.Fatalf("client syntax: %v %+v", err, tree.Tree.Diagnostics)
			}
			definition, err := sdkstdlib.LoadVectorCandidate()
			if err != nil {
				t.Fatal(err)
			}
			input := semantic.PackageInput{
				Files:   []semantic.File{{Package: 1, Path: "main.fy", Module: tree.AST}},
				Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "std::vec", ToPackage: 2}},
			}
			seen := map[string]bool{}
			for _, file := range definition.Sources {
				input.Files = append(input.Files, semantic.File{Package: 2, Path: file.Path, Module: file.Module})
				for _, imported := range file.Module.Imports {
					if !seen[imported.Path] {
						seen[imported.Path] = true
						input.Imports = append(input.Imports, semantic.PackageImport{FromPackage: 2, ImportPath: imported.Path, ToPackage: 2})
					}
				}
			}
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{}
			for id, uid := range map[hir.PackageID]string{1: "vector-reserve-client", 2: sdkstdlib.PackageUID} {
				coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: uid, VersionIdentity: "1"})
				if err != nil {
					t.Fatal(err)
				}
				coordinates[id] = coordinate
			}
			analysis, err := compiler.AnalyzeSDKStoragePackages(input, coordinates, genericplan.DefaultPlanLimits(), []hir.PackageID{2})
			if err != nil {
				t.Fatal(err)
			}
			result, err := compiler.LowerAnalyzedStorageTransfers(analysis.Analysis)
			if err != nil || !result.Verified {
				t.Fatalf("concrete element source/Flow/MIR: %v %+v", err, result.Semantic.Diagnostics)
			}
		})
	}
}
