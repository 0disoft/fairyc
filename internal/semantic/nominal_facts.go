package semantic

import (
	"cmp"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

// NominalTypeFact records a visible written type name in one successfully checked
// module scope. CanonicalType is build-local and must be rebound before transport.
// It proves name resolution only, not validity of an uninstantiated generic body.
type NominalTypeFact struct {
	Package       hir.PackageID
	Module        string
	Name          string
	CanonicalType hir.Type
}

func (a *analyzer) visibleNominalTypes() []NominalTypeFact {
	var facts []NominalTypeFact
	for _, module := range a.moduleOrder {
		appendScope := func(target *moduleScope, prefix string) {
			appendType := func(value *symbol) {
				if target != module && !value.public {
					return
				}
				facts = append(facts, NominalTypeFact{Package: module.key.packageID, Module: module.file.Module.Path,
					Name: prefix + value.name, CanonicalType: value.typ})
			}
			for _, value := range target.structs {
				appendType(value)
			}
			for _, value := range target.enums {
				appendType(value)
			}
			for _, value := range target.resources {
				appendType(value)
			}
		}
		appendScope(module, "")
		for alias, target := range module.imports {
			appendScope(target, alias+"::")
		}
	}
	slices.SortFunc(facts, func(a, b NominalTypeFact) int {
		if c := cmp.Compare(a.Package, b.Package); c != 0 {
			return c
		}
		if c := cmp.Compare(a.Module, b.Module); c != 0 {
			return c
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return facts
}
