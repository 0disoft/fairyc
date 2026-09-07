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
	Kind              string
	ID                genericplan.DeclarationID
	Package           hir.PackageID
	Module            string
	Name              string
	Span              hir.Span
	NameSpan          hir.Span
	DocumentationSpan hir.Span
	Parameters        []string
	Public            bool
	Signature         GenericSignature
}

// GenericProvenance is an immutable current-source sidecar. It is rebuilt during
// analysis, never loaded from an artifact cache, and asserts no semantic facts
// about uninstantiated template bodies.
type GenericProvenance struct {
	declarations []GenericDeclaration
	origins      []genericsource.SpecializationOrigin
	nominalTypes []semantic.NominalTypeFact
	interfaces   []GenericPackageInterface
}

func (value GenericProvenance) NominalTypes() []semantic.NominalTypeFact {
	return slices.Clone(value.nominalTypes)
}

func (value GenericProvenance) Declarations() []GenericDeclaration {
	result := slices.Clone(value.declarations)
	for i := range result {
		result[i].Parameters = slices.Clone(result[i].Parameters)
		result[i].Signature = cloneGenericSignature(result[i].Signature)
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

func genericProvenance(input semantic.PackageInput, owners map[hir.PackageID]genericplan.PackageID, expansion genericsource.PackageExpansion, nominalTypes []semantic.NominalTypeFact) (GenericProvenance, error) {
	result := GenericProvenance{nominalTypes: slices.Clone(nominalTypes)}
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
		original, err := originalGenericHeader(file.Module, declaration)
		if err != nil {
			return GenericProvenance{}, err
		}
		var documentation hir.Span
		if original.documentation.EndByte > original.documentation.StartByte {
			documentation = hir.Span{Package: file.Package, Source: file.Path, StartByte: original.documentation.StartByte, EndByte: original.documentation.EndByte}
		}
		result.declarations = append(result.declarations, GenericDeclaration{Kind: declaration.Kind, ID: declaration.ID,
			Package: file.Package, Module: declaration.Module, Name: declaration.Name, Public: declaration.Public,
			Span:              hir.Span{Package: file.Package, Source: file.Path, StartByte: declaration.Span.StartByte, EndByte: declaration.Span.EndByte},
			NameSpan:          hir.Span{Package: file.Package, Source: file.Path, StartByte: declaration.NameSpan.StartByte, EndByte: declaration.NameSpan.EndByte},
			Parameters:        slices.Clone(declaration.Parameters),
			DocumentationSpan: documentation,
			Signature:         cloneGenericSignature(original.signature)})
	}
	result.origins = slices.Clone(expansion.Origins)
	for i := range result.origins {
		result.origins[i].Bindings = slices.Clone(result.origins[i].Bindings)
	}
	return result, nil
}
