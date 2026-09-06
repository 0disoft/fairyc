package compiler

import (
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// GenericAnalysis retains source-interface identities separately from concrete
// typed HIR. Interfaces are absent on semantic rejection and have no authority
// to admit current stable cache records or unverified executable artifacts.
type GenericAnalysis struct {
	Analysis   Analysis
	Plan       genericplan.Plan
	Interfaces map[hir.PackageID]moduleinterface.GenericHash
}

func AnalyzeGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (GenericAnalysis, error) {
	prepared, expansion, err := prepareGenericPackages(input, coordinates, limits)
	if err != nil {
		return GenericAnalysis{}, err
	}
	analysis, err := AnalyzePackages(prepared)
	if err != nil {
		return GenericAnalysis{}, err
	}
	result := GenericAnalysis{Analysis: analysis, Plan: expansion.Plan}
	checked := analysis.Semantic
	if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		return result, nil
	}
	generated := map[string]bool{}
	for _, name := range expansion.GeneratedNames {
		generated[name] = true
	}
	interfaces := map[hir.PackageID]moduleinterface.GenericHash{}
	for _, file := range input.Files {
		if _, done := interfaces[file.Package]; done {
			continue
		}
		descriptor, err := moduleinterface.Extract(checked.HIR, file.Package, coordinates)
		if err != nil {
			return GenericAnalysis{}, err
		}
		for i := range descriptor.Modules {
			descriptor.Modules[i].Items = slices.DeleteFunc(descriptor.Modules[i].Items, func(item moduleinterface.Item) bool { return item.Kind == "function" && generated[item.Name] })
		}
		var templates []moduleinterface.GenericTemplate
		owner := genericPackageOwner(coordinates[file.Package])
		for _, template := range expansion.Templates {
			if template.Owner == owner {
				templates = append(templates, moduleinterface.GenericTemplate{Module: template.Module, Name: template.Name, Closure: template.Closure})
			}
		}
		hash, err := moduleinterface.ComputeGeneric(descriptor, templates)
		if err != nil {
			return GenericAnalysis{}, err
		}
		interfaces[file.Package] = hash
	}
	result.Interfaces = interfaces
	return result, nil
}
