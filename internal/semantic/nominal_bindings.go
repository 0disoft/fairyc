package semantic

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

// NominalTypeBinding identifies an existing concrete declaration in the exact
// input closure. It is compiler-private expansion metadata, not import access
// or permission to inspect fields, construct resources, copy or destroy them.
type NominalTypeBinding struct {
	Key     string
	Package hir.PackageID
	Module  string
	Name    string
}

// AnalyzeBoundNominalPackages consumes a checked expander's nominal references.
// The caller owns source argument visibility; this boundary validates exact
// targets and preserves their declaration owners for ordinary access checking.
func AnalyzeBoundNominalPackages(input PackageInput, nominals []NominalTypeBinding) (Result, error) {
	imports, err := validatePackageInput(input)
	if err != nil {
		return Result{}, err
	}
	type target struct {
		packageID    hir.PackageID
		module, name string
	}
	known := map[target]bool{}
	for _, file := range input.Files {
		for _, declaration := range file.Module.Structs {
			if len(declaration.TypeParameters) == 0 {
				known[target{file.Package, file.Module.Path, declaration.Name}] = true
			}
		}
		for _, declaration := range file.Module.Resources {
			if len(declaration.TypeParameters) == 0 {
				known[target{file.Package, file.Module.Path, declaration.Name}] = true
			}
		}
	}
	bindings := make(map[string]NominalTypeBinding, len(nominals))
	for _, binding := range nominals {
		if binding.Key == "" || !known[target{binding.Package, binding.Module, binding.Name}] {
			return Result{}, fmt.Errorf("nominal binding requires an exact concrete declaration")
		}
		if _, exists := bindings[binding.Key]; exists {
			return Result{}, fmt.Errorf("duplicate nominal binding key")
		}
		bindings[binding.Key] = binding
	}
	return analyzeBoundNominals(input.Files, imports, bindings), nil
}

func (a *analyzer) canonicalBoundNominal(reference ast.TypeReference) (hir.Type, bool) {
	if reference.Mutable || reference.Element != nil || reference.Result != nil || reference.Length != nil ||
		len(reference.Arguments) != 0 || len(reference.Parameters) != 0 || reference.ABI != "" {
		return hir.TypeInvalid, false
	}
	binding, found := a.nominalBindings[reference.Path]
	if !found {
		return hir.TypeInvalid, false
	}
	module := a.modules[moduleKey{packageID: binding.Package, path: binding.Module}]
	if module == nil {
		return hir.TypeInvalid, false
	}
	declaration := module.item(binding.Name)
	if declaration == nil || declaration.structure == nil && declaration.resource == nil {
		return hir.TypeInvalid, false
	}
	return declaration.typ, true
}
