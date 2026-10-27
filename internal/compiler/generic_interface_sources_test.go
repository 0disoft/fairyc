package compiler

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// Model the interface-publication boundary with interleaved, typed HIR owners.
// Parsing, expansion and Flow are outside this focused benchmark.
func genericInterfaceSourcesFixture(tb testing.TB, count int) (semantic.PackageInput, map[hir.PackageID]packagegraph.CoordinateID, genericsource.PackageExpansion, Analysis) {
	tb.Helper()
	input := semantic.PackageInput{}
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, count)
	expansion := genericsource.PackageExpansion{GeneratedNames: []string{"_fy_generic_hidden"}}
	analysis := Analysis{Semantic: semantic.Result{CompletedThrough: semantic.PhaseFlow}}
	for index := range count {
		id := hir.PackageID(index + 1)
		coordinate, err := packagegraph.ComputeCoordinateID(packagegraph.Coordinate{PackageUID: fmt.Sprintf("publish.%d", id), VersionIdentity: "1"})
		if err != nil {
			tb.Fatal(err)
		}
		coordinates[id] = coordinate
		input.Files = append(input.Files, semantic.File{Package: id})
		if index%8 == 0 {
			analysis.Semantic.HIR.StateLayouts = append(analysis.Semantic.HIR.StateLayouts, hir.StateLayout{
				Package: id, ContractUID: fmt.Sprintf("state.%d", id),
				Fields: []hir.StateField{{Name: "count", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()}},
			})
		}
	}
	for module := 3; module >= 0; module-- {
		path := fmt.Sprintf("m%d", module)
		for index := range count {
			id := hir.PackageID(index + 1)
			analysis.Semantic.HIR.Modules = append(analysis.Semantic.HIR.Modules, hir.Module{
				Package: id, Path: path,
				Functions: []hir.Function{
					{Name: "read", Public: true, ResultType: hir.TypeU32},
					{Name: "_fy_generic_hidden", Public: true, ResultType: hir.TypeU32},
				},
			})
			expansion.Templates = append(expansion.Templates, genericsource.PublicTemplate{
				Owner: genericPackageOwner(coordinates[id]), Module: path, Name: "id",
				Closure: genericplan.Digest(sha256.Sum256([]byte(fmt.Sprintf("%d/%s", id, path)))),
			})
		}
	}
	return input, coordinates, expansion, analysis
}

func TestPublishGenericInterfacesMatchIndependentExtraction(t *testing.T) {
	input, coordinates, expansion, analysis := genericInterfaceSourcesFixture(t, 16)
	wantOrdinary := map[hir.PackageID]moduleinterface.Hash{}
	wantGeneric := map[hir.PackageID]moduleinterface.GenericHash{}
	for _, file := range input.Files {
		descriptor, err := moduleinterface.Extract(analysis.Semantic.HIR, file.Package, coordinates)
		if err != nil {
			t.Fatal(err)
		}
		for i := range descriptor.Modules {
			descriptor.Modules[i].Items = slices.DeleteFunc(descriptor.Modules[i].Items, func(item moduleinterface.Item) bool { return item.Name == "_fy_generic_hidden" })
		}
		var templates []moduleinterface.GenericTemplate
		for _, template := range expansion.Templates {
			if template.Owner == genericPackageOwner(coordinates[file.Package]) {
				templates = append(templates, moduleinterface.GenericTemplate{Module: template.Module, Name: template.Name, Closure: template.Closure})
			}
		}
		wantOrdinary[file.Package], err = moduleinterface.Compute(descriptor)
		if err != nil {
			t.Fatal(err)
		}
		wantGeneric[file.Package], err = moduleinterface.ComputeGeneric(descriptor, templates)
		if err != nil {
			t.Fatal(err)
		}
	}
	beforeModules := slices.Clone(analysis.Semantic.HIR.Modules)
	beforeLayouts := slices.Clone(analysis.Semantic.HIR.StateLayouts)
	beforeTemplates := slices.Clone(expansion.Templates)
	got, err := publishGenericAnalysis(input, coordinates, expansion, analysis)
	if err != nil || !maps.Equal(got.OrdinaryInterfaces, wantOrdinary) || !maps.Equal(got.Interfaces, wantGeneric) {
		t.Fatalf("publication differs from independent extraction: %v", err)
	}
	if !reflect.DeepEqual(beforeModules, analysis.Semantic.HIR.Modules) || !reflect.DeepEqual(beforeLayouts, analysis.Semantic.HIR.StateLayouts) || !reflect.DeepEqual(beforeTemplates, expansion.Templates) {
		t.Fatal("publication reordered or changed its source inputs")
	}
	slices.Reverse(input.Files)
	slices.Reverse(analysis.Semantic.HIR.Modules)
	slices.Reverse(analysis.Semantic.HIR.StateLayouts)
	slices.Reverse(expansion.Templates)
	// A repeated source owner must still publish exactly one interface.
	input.Files = append(input.Files, input.Files[0])
	again, err := publishGenericAnalysis(input, coordinates, expansion, analysis)
	if err != nil || !maps.Equal(again.OrdinaryInterfaces, wantOrdinary) || !maps.Equal(again.Interfaces, wantGeneric) {
		t.Fatalf("source ordering or repeated owner changed publication: %v", err)
	}
}

func TestPublishGenericInterfacesPreserveSelectedOwnerErrors(t *testing.T) {
	for _, mode := range []string{"missing coordinate", "missing module", "duplicate module", "bad layout", "duplicate layout"} {
		t.Run(mode, func(t *testing.T) {
			input, coordinates, expansion, analysis := genericInterfaceSourcesFixture(t, 16)
			switch mode {
			case "missing coordinate":
				delete(coordinates, 1)
			case "missing module":
				analysis.Semantic.HIR.Modules = slices.DeleteFunc(analysis.Semantic.HIR.Modules, func(m hir.Module) bool { return m.Package == 1 })
			case "duplicate module":
				analysis.Semantic.HIR.Modules = append(analysis.Semantic.HIR.Modules, analysis.Semantic.HIR.Modules[0])
			case "bad layout":
				analysis.Semantic.HIR.StateLayouts[0].ContractUID = ""
			case "duplicate layout":
				analysis.Semantic.HIR.StateLayouts = append(analysis.Semantic.HIR.StateLayouts, analysis.Semantic.HIR.StateLayouts[0])
			}
			_, want := moduleinterface.Extract(analysis.Semantic.HIR, 1, coordinates)
			_, got := publishGenericAnalysis(input, coordinates, expansion, analysis)
			if want == nil || got == nil || got.Error() != want.Error() {
				t.Fatalf("publication error=%v, independent error=%v", got, want)
			}
			// Malformed records owned by an unselected package remain irrelevant.
			input.Files = input.Files[1:]
			gotResult, err := publishGenericAnalysis(input, coordinates, expansion, analysis)
			if err != nil || len(gotResult.Interfaces) != 15 {
				t.Fatalf("unselected owner's records affected publication: %v", err)
			}
		})
	}
}

func BenchmarkPublishGenericInterfaces(b *testing.B) {
	for _, count := range []int{32, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			input, coordinates, expansion, analysis := genericInterfaceSourcesFixture(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				got, err := publishGenericAnalysis(input, coordinates, expansion, analysis)
				if err != nil || len(got.Interfaces) != count {
					b.Fatalf("interfaces=%d, error=%v", len(got.Interfaces), err)
				}
			}
		})
	}
}
