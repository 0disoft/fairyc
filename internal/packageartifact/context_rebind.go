package packageartifact

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// RebindContextDescriptor reconstructs process-local MIR type and foreign declarations solely
// from a stable descriptor. Internal drop functions and foreign declarations receive explicit,
// collision-free caller-owned symbols; no source span or entity is invented as semantic truth.
func RebindContextDescriptor(descriptor ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID, functionSymbols map[FunctionID]hir.SymbolID, firstForeignSymbol hir.SymbolID) (mir.Program, error) {
	return rebindContextDescriptor(descriptor, coordinates, functionSymbols, firstForeignSymbol, indexedPackageResolver(coordinates))
}

func rebindContextDescriptor(descriptor ContextDescriptor, coordinates map[hir.PackageID]packagegraph.CoordinateID, functionSymbols map[FunctionID]hir.SymbolID, firstForeignSymbol hir.SymbolID, resolvePackage packageResolver) (mir.Program, error) {
	if err := ValidateContextDescriptor(descriptor); err != nil {
		return mir.Program{}, err
	}
	boundSymbols, err := contextRebindFunctionSymbols(functionSymbols, firstForeignSymbol)
	if err != nil {
		return mir.Program{}, err
	}
	program, bindings, err := rebindContextDeclarations(descriptor, resolvePackage, functionSymbols, boundSymbols, firstForeignSymbol)
	if err != nil {
		return mir.Program{}, err
	}
	roundTrip, err := DescribeContext(program, coordinates, bindings)
	if err != nil {
		return mir.Program{}, fmt.Errorf("describe rebound stable type context: %w", err)
	}
	if !reflect.DeepEqual(roundTrip, descriptor) {
		return mir.Program{}, fmt.Errorf("rebound stable type context does not round-trip")
	}
	return program, nil
}

func contextRebindFunctionSymbols(functionSymbols map[FunctionID]hir.SymbolID, firstForeignSymbol hir.SymbolID) (map[hir.SymbolID]string, error) {
	if firstForeignSymbol == hir.NoSymbol {
		return nil, fmt.Errorf("stable type-context rebind requires a foreign symbol base")
	}
	boundSymbols := make(map[hir.SymbolID]string, len(functionSymbols))
	for id, symbol := range functionSymbols {
		if id == (FunctionID{}) || symbol == hir.NoSymbol {
			return nil, fmt.Errorf("stable type-context rebind has an invalid function symbol binding")
		}
		if previous, duplicate := boundSymbols[symbol]; duplicate {
			return nil, fmt.Errorf("function symbol %d is duplicated by %s", symbol, previous)
		}
		boundSymbols[symbol] = id.String()
	}
	return boundSymbols, nil
}

func packageForStableNominal(value string, resolve packageResolver) (hir.PackageID, error) {
	const prefix = "pkg["
	separator := strings.Index(value, "]::")
	if !strings.HasPrefix(value, prefix) || separator <= len(prefix) {
		return 0, fmt.Errorf("stable nominal type %q is invalid", value)
	}
	coordinate, err := packagegraph.ParseCoordinateID(value[len(prefix):separator])
	if err != nil {
		return 0, fmt.Errorf("parse stable nominal type %q: %w", value, err)
	}
	return packageForCoordinateWithResolver(coordinate, resolve)
}

func packageForCoordinate(coordinate packagegraph.CoordinateID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (hir.PackageID, error) {
	return packageForCoordinateWithResolver(coordinate, func(coordinate packagegraph.CoordinateID) (packageCoordinateBinding, bool) {
		return scanPackageCoordinate(coordinate, coordinates)
	})
}

func packageForCoordinateWithResolver(coordinate packagegraph.CoordinateID, resolve packageResolver) (hir.PackageID, error) {
	binding, found := resolve(coordinate)
	if binding.ambiguous {
		return 0, fmt.Errorf("PackageCoordinateId %s maps to multiple local packages", coordinate)
	}
	if !found {
		return 0, fmt.Errorf("PackageCoordinateId %s has no local package", coordinate)
	}
	return binding.id, nil
}
