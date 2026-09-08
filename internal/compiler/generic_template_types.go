package compiler

import (
	"cmp"
	"slices"

	"github.com/0disoft/fairyc/internal/genericplan"
	"github.com/0disoft/fairyc/internal/genericsource"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
)

// GenericTemplateType binds a written aggregate name in one source scope to an
// original declaration. It describes visibility, not an instantiated nominal type.
type GenericTemplateType struct {
	Package     hir.PackageID
	Module      string
	Name        string
	Declaration genericplan.DeclarationID
}

func (value GenericProvenance) TemplateTypes() []GenericTemplateType {
	return slices.Clone(value.templateTypes)
}

func genericTemplateTypes(input semantic.PackageInput, declarations []GenericDeclaration) []GenericTemplateType {
	var result []GenericTemplateType
	for _, file := range input.Files {
		for _, declaration := range declarations {
			if declaration.Kind != genericsource.OriginStruct && declaration.Kind != genericsource.OriginResource {
				continue
			}
			if declaration.Package == file.Package && declaration.Module == file.Module.Path {
				result = append(result, GenericTemplateType{file.Package, file.Module.Path, declaration.Name, declaration.ID})
			}
			if !declaration.Public {
				continue
			}
			for _, imported := range file.Module.Imports {
				for _, binding := range input.Imports {
					if binding.FromPackage == file.Package && binding.ImportPath == imported.Path && binding.ToPackage == declaration.Package && imported.Path == declaration.Module {
						result = append(result, GenericTemplateType{file.Package, file.Module.Path, imported.Alias + "::" + declaration.Name, declaration.ID})
					}
				}
			}
		}
	}
	slices.SortFunc(result, func(a, b GenericTemplateType) int {
		if n := cmp.Compare(a.Package, b.Package); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Module, b.Module); n != 0 {
			return n
		}
		if n := cmp.Compare(a.Name, b.Name); n != 0 {
			return n
		}
		return cmp.Compare(a.Declaration, b.Declaration)
	})
	return slices.Compact(result)
}
