package compiler

import (
	"context"
	"slices"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// GenericAnalysis retains source-interface identities separately from concrete
// typed HIR. Interfaces are absent on semantic rejection and have no authority
// to admit current stable cache records or unverified executable artifacts.
type GenericAnalysis struct {
	Analysis           Analysis
	Plan               genericplan.Plan
	Interfaces         map[hir.PackageID]moduleinterface.GenericHash
	OrdinaryInterfaces map[hir.PackageID]moduleinterface.Hash
	Specializations    map[hir.PackageID][]genericplan.SpecializationID
}

func AnalyzeGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (GenericAnalysis, error) {
	return AnalyzeWithPolicy(input, coordinates, limits, admission.Policy[hir.PackageID]{})
}

// AnalyzeWithPolicy is the shared source-to-HIR pipeline for resolved generic
// packages. Source-owned rejections become diagnostics; plumbing failures remain
// errors. Storage owners must already have been authenticated by the caller.
func AnalyzeWithPolicy(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (GenericAnalysis, error) {
	return AnalyzeWithPolicyContext(context.Background(), input, coordinates, limits, policy)
}

func AnalyzeWithPolicyContext(ctx context.Context, input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (result GenericAnalysis, err error) {
	if err := ctx.Err(); err != nil {
		return GenericAnalysis{}, &Error{Stage: StageSemantic, Cause: err}
	}
	defer func() {
		if canceled := ctx.Err(); canceled != nil {
			result, err = GenericAnalysis{}, &Error{Stage: StageSemantic, Cause: canceled}
		}
	}()
	analysis, err := analyzeWithPolicyContext(ctx, input, coordinates, limits, policy)
	if err != nil {
		if rejection, sourceOwned := genericSourceDiagnostic(err, input, coordinates); sourceOwned {
			return GenericAnalysis{Analysis: Analysis{Semantic: semantic.Result{FailedPhase: semantic.PhaseResolve, Diagnostics: []semantic.Diagnostic{rejection}}}}, nil
		}
	}
	return analysis, err
}

// Legacy private proof lanes keep returning source expansion errors directly.
func analyzeWithPolicy(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (GenericAnalysis, error) {
	return analyzeWithPolicyContext(context.Background(), input, coordinates, limits, policy)
}

func analyzeWithPolicyContext(ctx context.Context, input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (GenericAnalysis, error) {
	if err := policy.Validate(); err != nil {
		return GenericAnalysis{}, err
	}
	if rejections := semantic.TestDeclarationDiagnostics(input.Files); len(rejections) != 0 {
		return GenericAnalysis{Analysis: Analysis{Semantic: semantic.Result{FailedPhase: semantic.PhaseResolve, Diagnostics: rejections}}}, nil
	}

	analysis, expansion, err := analyzePackageGraphContext(ctx, input, coordinates, limits, policy)
	if err != nil {
		return GenericAnalysis{}, err
	}
	return publishGenericAnalysis(input, coordinates, expansion, Analysis{Semantic: analysis})
}

// CheckWithPolicy uses one policy for source admission, semantics and MIR.
func CheckWithPolicy(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (Result, genericplan.Plan, error) {
	return CheckWithPolicyContext(context.Background(), input, coordinates, limits, policy)
}

func CheckWithPolicyContext(ctx context.Context, input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (Result, genericplan.Plan, error) {
	analysis, err := AnalyzeWithPolicyContext(ctx, input, coordinates, limits, policy)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	result, err := LowerAnalyzedWithCapabilitiesContext(ctx, analysis.Analysis, policy.Capabilities)
	return result, analysis.Plan, err
}

// Publish source interface identities only after the owning semantic lane has
// completed Flow. This helper grants no executable artifact admission.
func publishGenericAnalysis(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, expansion genericsource.PackageExpansion, analysis Analysis) (GenericAnalysis, error) {
	result := GenericAnalysis{Analysis: analysis, Plan: expansion.Plan}
	checked := analysis.Semantic
	if checked.CompletedThrough != semantic.PhaseFlow || checked.FailedPhase != semantic.PhaseNone || len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
		return result, nil
	}
	generated := map[string]bool{}
	result.Analysis.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	for _, name := range expansion.GeneratedNames {
		generated[name] = true
	}
	interfaces := map[hir.PackageID]moduleinterface.GenericHash{}
	ordinary := map[hir.PackageID]moduleinterface.Hash{}
	specializations := map[hir.PackageID][]genericplan.SpecializationID{}
	for _, file := range input.Files {
		if _, done := interfaces[file.Package]; done {
			continue
		}
		descriptor, err := moduleinterface.Extract(checked.HIR, file.Package, coordinates)
		if err != nil {
			return GenericAnalysis{}, err
		}
		for i := range descriptor.Modules {
			descriptor.Modules[i].Items = slices.DeleteFunc(descriptor.Modules[i].Items, func(item moduleinterface.Item) bool { return generated[item.Name] })
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
		ordinary[file.Package], err = moduleinterface.Compute(descriptor)
		if err != nil {
			return GenericAnalysis{}, err
		}
		specializations[file.Package] = slices.Clone(expansion.Specializations[owner])
	}
	result.Interfaces = interfaces
	result.OrdinaryInterfaces = ordinary
	result.Specializations = specializations
	owners := map[hir.PackageID]genericplan.PackageID{}
	for id, coordinate := range coordinates {
		owners[id] = genericPackageOwner(coordinate)
	}
	var err error
	result.Analysis.GenericProvenance, err = genericProvenance(input, owners, expansion, checked.NominalTypes)
	if err != nil {
		return GenericAnalysis{}, err
	}
	result.Analysis.GenericProvenance.interfaces = retainGenericPackageInterfaces(ordinary, interfaces, coordinates)
	return result, nil
}
