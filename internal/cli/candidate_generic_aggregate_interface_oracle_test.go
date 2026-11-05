package cli

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

type candidateAggregateInterfaceExpectation struct {
	owner           uint64
	ordinary        []byte
	ordinaryHash    string
	sourceHash      string
	specializations []string
}

// candidateAggregateInterfaceExpectations independently derives the three
// package interfaces used by the aggregate interface probe. Production hashes
// remain the comparison authority; this helper only reconstructs their bytes.
func candidateAggregateInterfaceExpectations(t *testing.T, fixture candidateAggregateLayoutFixture) []candidateAggregateInterfaceExpectation {
	t.Helper()
	maker := fixture.maker
	if maker == "" {
		maker = aggregateLayoutMaker
	}
	texts := []string{fixture.root, maker, aggregatePlanHelper}
	paths := []string{"root.fy", "maker.fy", "helper.fy"}

	var input semantic.PackageInput
	prepared := make([]genericsource.PackageModule, 0, len(texts))
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, len(texts))
	for i, text := range texts {
		parsed, err := frontend.AnalyzeSyntax(source.NewFile(paths[i], []byte(text)), source.DefaultLimits())
		if err != nil || parsed.AST == nil || !parsed.Tree.ErrorFree() {
			t.Fatalf("aggregate interface %q syntax %s: %v %+v", fixture.name, paths[i], err, parsed.Tree.Diagnostics)
		}
		id := hir.PackageID(i + 1)
		coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-" + strings.Repeat(fmt.Sprint(i+1), 64))
		if err != nil {
			t.Fatal(err)
		}
		coordinates[id] = coordinate
		input.Files = append(input.Files, semantic.File{Package: id, Module: parsed.AST, Path: paths[i]})
		prepared = append(prepared, genericsource.PackageModule{
			Module:  parsed.AST,
			Owner:   genericplan.PackageID(fmt.Sprintf("package_coordinate_v1_%x", coordinate[:])),
			Imports: map[string]genericplan.PackageID{},
		})
	}
	input.Imports = []semantic.PackageImport{
		{FromPackage: 1, ImportPath: "maker", ToPackage: 2},
		{FromPackage: 2, ImportPath: "helper", ToPackage: 3},
	}
	prepared[0].Imports["maker"] = prepared[1].Owner
	prepared[1].Imports["helper"] = prepared[2].Owner

	production, err := compiler.AnalyzeGenericPackages(input, coordinates, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatalf("aggregate interface %q analysis: %v", fixture.name, err)
	}
	checked := production.Analysis.Semantic
	if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		t.Fatalf("aggregate interface %q incomplete: phase=%s failed=%s diagnostics=%+v unsupported=%+v", fixture.name, checked.CompletedThrough, checked.FailedPhase, checked.Diagnostics, checked.Unsupported)
	}

	expansion, err := genericsource.PreparePackages(prepared, genericplan.DefaultPlanLimits())
	if err != nil {
		t.Fatalf("aggregate interface %q generated names: %v", fixture.name, err)
	}
	generated := make(map[string]bool, len(expansion.GeneratedNames))
	for _, name := range expansion.GeneratedNames {
		generated[name] = true
	}

	expectations := make([]candidateAggregateInterfaceExpectation, 0, len(input.Files))
	for id := hir.PackageID(1); id <= hir.PackageID(len(input.Files)); id++ {
		var owned hir.Package
		for _, module := range checked.HIR.Modules {
			if module.Package == id {
				owned.Modules = append(owned.Modules, module)
			}
		}
		for _, layout := range checked.HIR.StateLayouts {
			if layout.Package == id {
				owned.StateLayouts = append(owned.StateLayouts, layout)
			}
		}
		descriptor, err := moduleinterface.Extract(owned, id, coordinates)
		if err != nil {
			t.Fatalf("aggregate interface %q package %d extraction: %v", fixture.name, id, err)
		}
		for module := range descriptor.Modules {
			descriptor.Modules[module].Items = slices.DeleteFunc(descriptor.Modules[module].Items, func(item moduleinterface.Item) bool {
				return generated[item.Name]
			})
			for _, item := range descriptor.Modules[module].Items {
				if generated[item.Name] {
					t.Fatalf("aggregate interface %q package %d exports generated source %q", fixture.name, id, item.Name)
				}
			}
		}
		encoded, err := moduleinterface.Encode(descriptor)
		if err != nil {
			t.Fatalf("aggregate interface %q package %d encoding: %v", fixture.name, id, err)
		}
		ordinary, err := moduleinterface.Compute(descriptor)
		if err != nil {
			t.Fatalf("aggregate interface %q package %d hash: %v", fixture.name, id, err)
		}
		if want, found := production.OrdinaryInterfaces[id]; !found || ordinary != want {
			t.Fatalf("aggregate interface %q package %d ordinary hash = %s, production = %s", fixture.name, id, ordinary, want)
		}
		sourceHash, found := production.Interfaces[id]
		if !found {
			t.Fatalf("aggregate interface %q package %d has no production source hash", fixture.name, id)
		}
		ids := slices.Clone(production.Specializations[id])
		slices.Sort(ids)
		specializations := make([]string, len(ids))
		for i, specialization := range ids {
			specializations[i] = string(specialization)
		}
		expectations = append(expectations, candidateAggregateInterfaceExpectation{
			owner:           uint64(id),
			ordinary:        encoded,
			ordinaryHash:    ordinary.String(),
			sourceHash:      sourceHash.String(),
			specializations: specializations,
		})
	}
	return expectations
}
