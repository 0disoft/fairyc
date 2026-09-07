package compiler

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// AnalyzeExplicitFiles compiles a closed source set without a manifest, SDK
// injection, package coordinates, interface publication, or cache admission.
// Its generic owner is a build-local namespace, never a fabricated package identity.
func AnalyzeExplicitFiles(files []semantic.File) (Analysis, error) {
	const owner genericplan.PackageID = "explicit_files_v0"
	hasGeneric := false
	bindings := map[string]genericplan.PackageID{}
	for _, file := range files {
		if file.Package != hir.DefaultPackage || file.Module == nil {
			return Analysis{}, fmt.Errorf("explicit files require syntax-clean default-package modules")
		}
		bindings[file.Module.Path] = owner
		for _, function := range file.Module.Functions {
			hasGeneric = hasGeneric || len(function.TypeParameters) != 0
		}
	}
	if !hasGeneric {
		return Analysis{Semantic: semantic.Analyze(files)}, nil
	}
	modules := make([]genericsource.PackageModule, len(files))
	for i, file := range files {
		modules[i] = genericsource.PackageModule{Module: file.Module, Owner: owner, Imports: bindings}
	}
	expansion, err := genericsource.PreparePackages(modules, genericplan.DefaultPlanLimits())
	if err != nil {
		if rejection, located := genericSourceDiagnosticOwned(err, semantic.PackageInput{Files: files}, map[hir.PackageID]genericplan.PackageID{hir.DefaultPackage: owner}); located {
			return Analysis{Semantic: semantic.Result{FailedPhase: semantic.PhaseResolve, Diagnostics: []semantic.Diagnostic{rejection}}}, nil
		}
		// Duplicate modules and absent imports belong to ordinary resolution,
		// not an invented package-coordinate or generic diagnostic.
		ordinary := semantic.Analyze(files)
		if ordinary.FailedPhase == semantic.PhaseResolve && len(ordinary.Diagnostics) != 0 {
			return Analysis{Semantic: ordinary}, nil
		}
		return Analysis{}, err
	}
	prepared := slices.Clone(files)
	for i := range prepared {
		prepared[i].Module = expansion.Modules[i]
	}
	analysis := Analysis{Semantic: semantic.Analyze(prepared)}
	checked := analysis.Semantic
	if checked.CompletedThrough == semantic.PhaseFlow && checked.FailedPhase == semantic.PhaseNone && len(checked.Diagnostics) == 0 && len(checked.Unsupported) == 0 {
		analysis.Semantic.RequiresGenericProvenance = len(expansion.Declarations) != 0
		analysis.GenericProvenance, err = genericProvenance(semantic.PackageInput{Files: files}, map[hir.PackageID]genericplan.PackageID{hir.DefaultPackage: owner}, expansion, checked.NominalTypes)
	}
	return analysis, err
}

func CheckExplicitFiles(files []semantic.File) (Result, error) {
	analysis, err := AnalyzeExplicitFiles(files)
	if err != nil {
		return Result{}, err
	}
	return LowerAnalyzed(analysis)
}
