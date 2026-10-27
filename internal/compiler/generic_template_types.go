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

type genericTemplateScope struct {
	packageID hir.PackageID
	module    string
}

func genericTemplateTypes(input semantic.PackageInput, declarations []GenericDeclaration) []GenericTemplateType {
	// Keep only positions in the call-local indexes, not copies of signatures.
	local := make(map[genericTemplateScope][]int)
	public := make(map[genericTemplateScope][]int)
	for position, declaration := range declarations {
		if declaration.Kind != genericsource.OriginStruct && declaration.Kind != genericsource.OriginResource {
			continue
		}
		scope := genericTemplateScope{declaration.Package, declaration.Module}
		local[scope] = append(local[scope], position)
		if declaration.Public {
			public[scope] = append(public[scope], position)
		}
	}
	if len(local) == 0 {
		return nil
	}
	providers := make(map[genericTemplateScope][]hir.PackageID)
	for _, binding := range input.Imports {
		scope := genericTemplateScope{binding.FromPackage, binding.ImportPath}
		providers[scope] = append(providers[scope], binding.ToPackage)
	}
	var result []GenericTemplateType
	for _, file := range input.Files {
		for _, position := range local[genericTemplateScope{file.Package, file.Module.Path}] {
			declaration := declarations[position]
			result = append(result, GenericTemplateType{file.Package, file.Module.Path, declaration.Name, declaration.ID})
		}
		for _, imported := range file.Module.Imports {
			for _, provider := range providers[genericTemplateScope{file.Package, imported.Path}] {
				for _, position := range public[genericTemplateScope{provider, imported.Path}] {
					declaration := declarations[position]
					result = append(result, GenericTemplateType{file.Package, file.Module.Path, imported.Alias + "::" + declaration.Name, declaration.ID})
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
