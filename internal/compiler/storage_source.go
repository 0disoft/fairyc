package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/admission"
	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/semantic"
)

// CheckStorageGenericFunctions is the private source-to-native preparation lane.
// Its MIR may contain symbolic queries rejected by ordinary artifact consumers.
func CheckStorageGenericFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericFunctions(file, coordinate, limits, admission.Capabilities{Storage: admission.StorageQueries, Arguments: admission.NominalArguments})
}

// CheckStorageTransferFunctions explicitly admits unsafe source storage transfers.
// Its output requires the matching private verifier and C emission entry point.
func CheckStorageTransferFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericFunctions(file, coordinate, limits, admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments})
}

func checkStorageGenericFunctions(file semantic.File, coordinate packagegraph.CoordinateID, limits genericplan.PlanLimits, capabilities admission.Capabilities) (Result, genericplan.Plan, error) {
	if coordinate == (packagegraph.CoordinateID{}) {
		return Result{}, genericplan.Plan{}, fmt.Errorf("storage generic source requires a nonzero package coordinate")
	}
	original, owner := file, genericPackageOwner(coordinate)
	expansion, err := genericsource.PrepareModuleWithCapabilities(file.Module, owner, limits, capabilities)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	file.Module = expansion.Modules[0]
	result, err := lowerWithCapabilities(semantic.Analyze([]semantic.File{file}), capabilities)
	if err == nil && result.Verified {
		result.GenericProvenance, err = genericProvenance(semantic.PackageInput{Files: []semantic.File{original}},
			map[hir.PackageID]genericplan.PackageID{file.Package: owner}, expansion, result.Semantic.NominalTypes)
		if err != nil {
			return Result{}, genericplan.Plan{}, err
		}
		result.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	}
	return result, expansion.Plan, err
}

// CheckStorageGenericPackages enables private scalar storage queries across a
// resolved graph. It does not publish a stable interface or artifact-cache key.
func CheckStorageGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericPackages(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageQueries}})
}

// CheckNominalStoragePackages admits source-visible nominal arguments through
// exact private bindings, without adding imports or publishing a stable cache key.
func CheckNominalStoragePackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	return checkStorageGenericPackages(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageQueries, Arguments: admission.NominalArguments}})
}

// CheckStorageTransferPackages is the private closed-graph ownership-transfer
// lane. Its verified MIR requires the matching private C emitter; it does not
// admit storage transfers to stable workspace or artifact consumers.
func CheckStorageTransferPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (Result, genericplan.Plan, error) {
	analysis, err := AnalyzeStorageTransferPackages(input, coordinates, limits)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	result, err := LowerAnalyzedStorageTransfers(analysis.Analysis)
	return result, analysis.Plan, err
}

// LowerAnalyzedStorageTransfers is the private cache fallback boundary. It keeps
// ordinary lowering closed to transfers and publishes provenance only with MIR
// accepted by the transfer verifier.
func LowerAnalyzedStorageTransfers(analysis Analysis) (Result, error) {
	return LowerAnalyzedWithCapabilities(analysis, admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments})
}

// AnalyzeStorageTransferPackages publishes private closed-graph source identities
// after bound nominal Flow analysis, without publishing executable MIR or cache
// admission. Nominal specialization definitions retain their conservative closure.
func AnalyzeStorageTransferPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits) (GenericAnalysis, error) {
	return analyzeWithPolicy(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments}})
}

// AnalyzeOwnedStorageTransferPackages is the owner-restricted integration lane.
// The caller must authenticate the selected source packages; this function does
// not derive authority from a module name, SDK label, or import spelling.
func AnalyzeOwnedStorageTransferPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storageOwners []hir.PackageID) (GenericAnalysis, error) {
	return analyzeOwnedStorageTransferPackages(input, coordinates, limits, storageOwners, admission.OrdinaryTemplates)
}

// AnalyzeSDKStoragePackages preserves scalar-only consumer templates while
// admitting the exact authenticated SDK's generic container implementations.
func AnalyzeSDKStoragePackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storageOwners []hir.PackageID) (GenericAnalysis, error) {
	if len(storageOwners) == 0 {
		return GenericAnalysis{}, fmt.Errorf("storage operation owners must be explicit")
	}
	return AnalyzeWithPolicy(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: admission.SDKOwnerTemplates}, StorageOwners: storageOwners})
}

// AnalyzeSDKVectorCandidatePackages checks vector-loop source and Flow only in
// a compiler-private proof lane. Stable workspace commands keep using
// AnalyzeSDKStoragePackages until native and C-relative evidence is complete.
func AnalyzeSDKVectorCandidatePackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storageOwners []hir.PackageID) (GenericAnalysis, error) {
	if len(storageOwners) == 0 {
		return GenericAnalysis{}, fmt.Errorf("SDK vector candidate requires authenticated storage owners")
	}
	return analyzeWithPolicy(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: admission.SDKOwnerTemplates, Vectors: admission.CandidateVectors}, StorageOwners: storageOwners})
}

func analyzeOwnedStorageTransferPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, storageOwners []hir.PackageID, templates admission.TemplateMode) (GenericAnalysis, error) {
	if len(storageOwners) == 0 {
		return GenericAnalysis{}, fmt.Errorf("storage operation owners must be explicit")
	}
	return analyzeWithPolicy(input, coordinates, limits, admission.Policy[hir.PackageID]{Capabilities: admission.Capabilities{Storage: admission.StorageTransfers, Arguments: admission.NominalArguments, Templates: templates}, StorageOwners: storageOwners})
}

func analyzePackageGraph(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (semantic.Result, genericsource.PackageExpansion, error) {
	prepared, expansion, err := prepareGenericPackageGraph(input, coordinates, limits, policy)
	if err != nil {
		return semantic.Result{}, genericsource.PackageExpansion{}, err
	}
	var analysis semantic.Result
	if policy.Capabilities.AllowsNominalArguments() {
		byOwner := map[genericplan.PackageID]hir.PackageID{}
		for _, file := range input.Files {
			byOwner[genericPackageOwner(coordinates[file.Package])] = file.Package
		}
		bindings := make([]semantic.NominalTypeBinding, 0, len(expansion.NominalBindings))
		for _, binding := range expansion.NominalBindings {
			id, found := byOwner[binding.Owner]
			if !found {
				return semantic.Result{}, genericsource.PackageExpansion{}, fmt.Errorf("nominal binding owner is absent")
			}
			bindings = append(bindings, semantic.NominalTypeBinding{Key: string(binding.ID), Package: id, Module: binding.Module, Name: binding.Name})
		}
		if policy.Capabilities.VectorCandidate() {
			selected := map[genericplan.PackageID]bool{}
			for _, owner := range policy.StorageOwners {
				selected[genericPackageOwner(coordinates[owner])] = true
			}
			vectorDeclarations := map[genericplan.DeclarationID]hir.PackageID{}
			for _, declaration := range expansion.Declarations {
				if declaration.Kind == genericsource.OriginResource && declaration.Module == "std::vec" &&
					declaration.Name == "Vec" && selected[declaration.Owner] {
					vectorDeclarations[declaration.ID] = byOwner[declaration.Owner]
				}
			}
			var vectors []semantic.VectorTypeAdmission
			for _, specialization := range expansion.Origins {
				if owner, found := vectorDeclarations[specialization.Declaration]; found {
					vectors = append(vectors, semantic.VectorTypeAdmission{
						Package: owner, Module: "std::vec", Name: specialization.GeneratedName,
					})
				}
			}
			analysis, err = semantic.AnalyzeBoundNominalVectorCandidatePackages(prepared, bindings, vectors)
		} else {
			analysis, err = semantic.AnalyzeBoundNominalPackages(prepared, bindings)
		}
	} else {
		analysis, err = semantic.AnalyzePackages(prepared)
	}
	if err != nil {
		return semantic.Result{}, genericsource.PackageExpansion{}, &Error{Stage: StagePackageInput, Cause: err}
	}
	return analysis, expansion, nil
}

func checkStorageGenericPackages(input semantic.PackageInput, coordinates map[hir.PackageID]packagegraph.CoordinateID, limits genericplan.PlanLimits, policy admission.Policy[hir.PackageID]) (Result, genericplan.Plan, error) {
	analysis, expansion, err := analyzePackageGraph(input, coordinates, limits, policy)
	if err != nil {
		return Result{}, genericplan.Plan{}, err
	}
	result, err := lowerWithCapabilities(analysis, policy.Capabilities)
	if err == nil && result.Verified {
		owners := make(map[hir.PackageID]genericplan.PackageID, len(coordinates))
		for id, coordinate := range coordinates {
			owners[id] = genericPackageOwner(coordinate)
		}
		result.GenericProvenance, err = genericProvenance(input, owners, expansion, result.Semantic.NominalTypes)
		if err != nil {
			return Result{}, genericplan.Plan{}, err
		}
		result.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
	}
	return result, expansion.Plan, err
}
