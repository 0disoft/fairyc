package compiler

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

type GenericDeclaration struct {
	ID         genericplan.DeclarationID
	Package    hir.PackageID
	Module     string
	Name       string
	Span       hir.Span
	NameSpan   hir.Span
	Parameters []string
	Public     bool
}

// GenericProvenance is an immutable current-source sidecar. It is rebuilt during
// analysis, never loaded from an artifact cache, and asserts no semantic facts
// about uninstantiated template bodies.
type GenericProvenance struct {
	declarations []GenericDeclaration
	origins      []genericsource.SpecializationOrigin
}

func (value GenericProvenance) Declarations() []GenericDeclaration {
	result := slices.Clone(value.declarations)
	for i := range result {
		result[i].Parameters = slices.Clone(result[i].Parameters)
	}
	return result
}

func (value GenericProvenance) Specializations() []genericsource.SpecializationOrigin {
	result := slices.Clone(value.origins)
	for i := range result {
		result[i].Bindings = slices.Clone(result[i].Bindings)
	}
	return result
}

func genericProvenance(input semantic.PackageInput, owners map[hir.PackageID]genericplan.PackageID, expansion genericsource.PackageExpansion) (GenericProvenance, error) {
	result := GenericProvenance{}
	for _, declaration := range expansion.Declarations {
		var file *semantic.File
		for i := range input.Files {
			candidate := &input.Files[i]
			if owners[candidate.Package] == declaration.Owner && candidate.Module != nil && candidate.Module.Path == declaration.Module {
				if file != nil {
					return GenericProvenance{}, fmt.Errorf("ambiguous generic declaration source")
				}
				file = candidate
			}
		}
		if file == nil || file.Path == "" || !declaration.Span.Valid(file.Module.SourceSize) || !declaration.NameSpan.Valid(file.Module.SourceSize) {
			return GenericProvenance{}, fmt.Errorf("generic declaration has no exact source")
		}
		result.declarations = append(result.declarations, GenericDeclaration{ID: declaration.ID,
			Package: file.Package, Module: declaration.Module, Name: declaration.Name, Public: declaration.Public,
			Span:       hir.Span{Package: file.Package, Source: file.Path, StartByte: declaration.Span.StartByte, EndByte: declaration.Span.EndByte},
			NameSpan:   hir.Span{Package: file.Package, Source: file.Path, StartByte: declaration.NameSpan.StartByte, EndByte: declaration.NameSpan.EndByte},
			Parameters: slices.Clone(declaration.Parameters)})
	}
	result.origins = slices.Clone(expansion.Origins)
	for i := range result.origins {
		result.origins[i].Bindings = slices.Clone(result.origins[i].Bindings)
	}
	return result, nil
}
