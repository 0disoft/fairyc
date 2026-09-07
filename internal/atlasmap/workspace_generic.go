package atlasmap

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/moduleinterface"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func workspaceInterfaceFacts(result compiler.Result, packages []hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID, schema string) (map[hir.PackageID]moduleinterface.Hash, map[hir.PackageID]moduleinterface.GenericHash, error) {
	facts := result.GenericProvenance.PackageInterfaces()
	if schema != LogicalSchemaV2 || !result.Semantic.RequiresGenericProvenance && len(facts) == 0 {
		ordinary, err := moduleinterface.ComputePackages(result.Semantic.HIR, packages, coordinates)
		return ordinary, nil, err
	}
	if !result.Verified || len(facts) != len(packages) {
		return nil, nil, fmt.Errorf("generic map requires complete verified source interface facts")
	}
	ordinary := map[hir.PackageID]moduleinterface.Hash{}
	generic := map[hir.PackageID]moduleinterface.GenericHash{}
	for _, fact := range facts {
		if fact.Coordinate == (packagegraph.CoordinateID{}) || fact.Coordinate != coordinates[fact.Package] ||
			fact.Ordinary == (moduleinterface.Hash{}) || fact.Generic == (moduleinterface.GenericHash{}) {
			return nil, nil, fmt.Errorf("generic source interface contradicts package coordinates or lacks a hash")
		}
		if _, duplicate := ordinary[fact.Package]; duplicate {
			return nil, nil, fmt.Errorf("duplicate generic package interface")
		}
		ordinary[fact.Package], generic[fact.Package] = fact.Ordinary, fact.Generic
	}
	for _, id := range packages {
		if _, found := ordinary[id]; !found {
			return nil, nil, fmt.Errorf("generic source interface package coverage differs from workspace")
		}
	}
	return ordinary, generic, nil
}
