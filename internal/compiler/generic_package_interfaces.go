package compiler

import (
	"cmp"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// GenericPackageInterface binds source-interface facts to their exact semantic
// package and stable coordinate. Ordinary excludes generated specializations;
// Generic additionally includes exported template closures, including unused ones.
type GenericPackageInterface struct {
	Package    hir.PackageID
	Coordinate packagegraph.CoordinateID
	Ordinary   moduleinterface.Hash
	Generic    moduleinterface.GenericHash
}

// PackageInterfaces returns detached source-interface facts in semantic package
// order. Explicit-file analysis has no package coordinates and publishes none.
// These are not stable cache-admission records or a serialized result field.
func (value GenericProvenance) PackageInterfaces() []GenericPackageInterface {
	return slices.Clone(value.interfaces)
}

func retainGenericPackageInterfaces(ordinary map[hir.PackageID]moduleinterface.Hash, generic map[hir.PackageID]moduleinterface.GenericHash, coordinates map[hir.PackageID]packagegraph.CoordinateID) []GenericPackageInterface {
	result := make([]GenericPackageInterface, 0, len(generic))
	for id, hash := range generic {
		result = append(result, GenericPackageInterface{Package: id, Coordinate: coordinates[id], Ordinary: ordinary[id], Generic: hash})
	}
	slices.SortFunc(result, func(a, b GenericPackageInterface) int { return cmp.Compare(a.Package, b.Package) })
	return result
}
