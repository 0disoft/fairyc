package contractstatemir

import (
	"bytes"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

// ReboundStateBody binds one verified stable payload to a fresh build-local
// package and function-symbol namespace.
type ReboundStateBody struct {
	Package      hir.PackageID
	Hash         StateBodyHash
	Layout       Layout
	Transactions []LoweredTransaction
}

// RebindStableStateBody checks the caller's expected body identity and current
// HIR layout/signatures before reconstructing each transaction with fresh
// private capability authority.
func RebindStableStateBody(body StableStateBodyArtifact, expected StateBodyHash, input hir.Package, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID) (ReboundStateBody, error) {
	return RebindStableStateBodyWithSymbols(body, expected, input, packageID, coordinates, nil)
}

// RebindStableStateBodyWithSymbols performs the checked fresh-owner rebind and,
// when functionSymbols is nonnil, requires every transaction to use the symbol
// assigned by the separately reassembled ordinary MIR program.
func RebindStableStateBodyWithSymbols(body StableStateBodyArtifact, expected StateBodyHash, input hir.Package, packageID hir.PackageID, coordinates map[hir.PackageID]packagegraph.CoordinateID, functionSymbols map[string]hir.SymbolID) (ReboundStateBody, error) {
	if expected == (StateBodyHash{}) {
		return ReboundStateBody{}, fmt.Errorf("stable contract-state body rebind requires an expected body hash")
	}
	if body.Hash != expected {
		return ReboundStateBody{}, fmt.Errorf("stable contract-state body does not match the expected body hash")
	}
	if err := ValidateStableStateBody(body); err != nil {
		return ReboundStateBody{}, fmt.Errorf("validate stable contract-state body before rebind: %w", err)
	}
	if err := validateStateBodyCoordinateMap(coordinates); err != nil {
		return ReboundStateBody{}, err
	}
	coordinate, found := coordinates[packageID]
	if !found || coordinate == (packagegraph.CoordinateID{}) {
		return ReboundStateBody{}, fmt.Errorf("package %d has no PackageCoordinateId for stable body rebind", packageID)
	}
	if coordinate != body.Package {
		return ReboundStateBody{}, fmt.Errorf("package %d coordinate does not own the stable contract-state body", packageID)
	}

	stableLayout, err := layoutFromStableState(body.Layout)
	if err != nil {
		return ReboundStateBody{}, err
	}
	currentLayout, hasCurrentLayout, err := LowerHIRLayoutForPackage(input, packageID)
	if err != nil {
		return ReboundStateBody{}, fmt.Errorf("lower current package layout before stable body rebind: %w", err)
	}
	if hasCurrentLayout != (body.Layout != nil) || !equalStateBodyLayouts(currentLayout, stableLayout) {
		return ReboundStateBody{}, fmt.Errorf("current package layout does not match the stable contract-state body")
	}

	result := ReboundStateBody{
		Package: packageID, Hash: body.Hash, Layout: currentLayout,
		Transactions: make([]LoweredTransaction, 0, len(body.Transactions)),
	}
	boundSymbols := make(map[hir.SymbolID]string, len(body.Transactions))
	for index, transaction := range body.Transactions {
		descriptor, err := decodeStableStateFunctionDescriptor(transaction.Function)
		if err != nil {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d function: %w", index, err)
		}
		function, err := bindStableStateFunction(input, packageID, descriptor, transaction.Function, coordinates)
		if err != nil {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d: %w", index, err)
		}
		if function.Symbol == hir.NoSymbol {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d function has no fresh symbol", index)
		}
		if hirBlockContainsCall(function.Body) {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d function %q contains a call; calls are not admitted", index, descriptor.name)
		}
		if hirBlockContainsDynamicControlFlow(function.Body) {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d function %q contains dynamic control flow; dynamic control flow is not admitted", index, descriptor.name)
		}
		if err := verifyHIRStateOwnerMemoryBoundary(input, function); err != nil {
			return ReboundStateBody{}, fmt.Errorf("stable transaction %d function %q: %w", index, descriptor.name, err)
		}
		freshSymbol := function.Symbol
		if functionSymbols != nil {
			var found bool
			freshSymbol, found = functionSymbols[descriptor.name]
			if !found || freshSymbol == hir.NoSymbol {
				return ReboundStateBody{}, fmt.Errorf("stable transaction %d function %q has no aligned MIR symbol", index, descriptor.name)
			}
		}
		if previous, duplicate := boundSymbols[freshSymbol]; duplicate {
			return ReboundStateBody{}, fmt.Errorf("stable functions %q and %q share fresh symbol %d", previous, descriptor.name, freshSymbol)
		}
		boundSymbols[freshSymbol] = descriptor.name
		program, err := programFromStableStateTransaction(currentLayout, descriptor, transaction)
		if err != nil {
			return ReboundStateBody{}, fmt.Errorf("rebind stable transaction %d: %w", index, err)
		}
		result.Transactions = append(result.Transactions, LoweredTransaction{
			Package: packageID, Function: freshSymbol, Program: program,
		})
	}
	return result, nil
}

func validateStateBodyCoordinateMap(coordinates map[hir.PackageID]packagegraph.CoordinateID) error {
	owners := make(map[packagegraph.CoordinateID]hir.PackageID, len(coordinates))
	for packageID, coordinate := range coordinates {
		if coordinate == (packagegraph.CoordinateID{}) {
			return fmt.Errorf("package %d has an empty PackageCoordinateId", packageID)
		}
		if previous, duplicate := owners[coordinate]; duplicate {
			return fmt.Errorf("packages %d and %d share one PackageCoordinateId", previous, packageID)
		}
		owners[coordinate] = packageID
	}
	return nil
}

func equalStateBodyLayouts(left, right Layout) bool {
	return left.ContractUID == right.ContractUID && left.Hash == right.Hash && slices.Equal(left.Fields, right.Fields)
}

func bindStableStateFunction(input hir.Package, packageID hir.PackageID, descriptor stableStateFunctionDescriptor, encoded []byte, coordinates map[hir.PackageID]packagegraph.CoordinateID) (hir.Function, error) {
	separator := strings.LastIndex(descriptor.name, "::")
	if separator <= 0 || separator+2 >= len(descriptor.name) {
		return hir.Function{}, fmt.Errorf("stable function has an invalid logical name")
	}
	modulePath, functionName := descriptor.name[:separator], descriptor.name[separator+2:]
	var owner hir.Module
	var function hir.Function
	found := false
	for _, module := range input.Modules {
		if module.Package != packageID || module.Path != modulePath {
			continue
		}
		for _, candidate := range module.Functions {
			if candidate.Name != functionName {
				continue
			}
			if found {
				return hir.Function{}, fmt.Errorf("current package has multiple functions named %q", descriptor.name)
			}
			owner, function, found = module, candidate, true
		}
	}
	if !found {
		return hir.Function{}, fmt.Errorf("current package is missing stable function %q", descriptor.name)
	}
	current, err := describeStateBodyFunction(owner, function, coordinates)
	if err != nil {
		return hir.Function{}, fmt.Errorf("describe current function %q: %w", descriptor.name, err)
	}
	if !bytes.Equal(current, encoded) {
		return hir.Function{}, fmt.Errorf("current function %q signature does not match the stable body", descriptor.name)
	}
	return function, nil
}
