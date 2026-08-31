package contractstatemir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

// PackageLayout binds one build-local package to its exact compiler-private
// persistent-state layout.
type PackageLayout struct {
	Package hir.PackageID
	Layout  Layout
}

// LoweredTransaction binds one package-owned HIR function to a verified
// compiler-private transaction program.
type LoweredTransaction struct {
	Package  hir.PackageID
	Function hir.SymbolID
	Program  Program
}

// LowerHIRLayoutForPackage lowers only one package-owned layout. Malformed
// layouts owned by other packages remain the responsibility of their own
// interface extraction.
func LowerHIRLayoutForPackage(input hir.Package, packageID hir.PackageID) (Layout, bool, error) {
	selected := hir.Package{}
	for _, definition := range input.StateLayouts {
		if definition.Package == packageID {
			selected.StateLayouts = append(selected.StateLayouts, definition)
		}
	}
	lowered, err := LowerHIRLayouts(selected)
	if err != nil {
		return Layout{}, false, err
	}
	if len(lowered) == 0 {
		return Layout{}, false, nil
	}
	return lowered[0].Layout, true, nil
}

// LowerHIRLayouts validates and lowers every source-independent HIR layout in
// package order. At most one contract layout may belong to each package.
func LowerHIRLayouts(input hir.Package) ([]PackageLayout, error) {
	definitions := slices.Clone(input.StateLayouts)
	slices.SortFunc(definitions, func(left, right hir.StateLayout) int {
		if left.Package < right.Package {
			return -1
		}
		if left.Package > right.Package {
			return 1
		}
		return 0
	})
	result := make([]PackageLayout, 0, len(definitions))
	for index, definition := range definitions {
		if index > 0 && definitions[index-1].Package == definition.Package {
			return nil, fmt.Errorf("package %d declares multiple contract state layouts", definition.Package)
		}
		declarations := make([]Declaration, 0, len(definition.Fields))
		for _, field := range definition.Fields {
			kind, err := lowerFieldKind(field.Kind)
			if err != nil {
				return nil, fmt.Errorf("package %d state field %q: %w", definition.Package, field.Name, err)
			}
			declarations = append(declarations, Declaration{
				Name: field.Name, Kind: kind, KeyTypeID: field.KeyTypeID, ValueTypeID: field.ValueTypeID,
			})
		}
		layout, err := BuildLayout(definition.ContractUID, declarations)
		if err != nil {
			return nil, fmt.Errorf("package %d contract state layout: %w", definition.Package, err)
		}
		result = append(result, PackageLayout{Package: definition.Package, Layout: layout})
	}
	return result, nil
}

// LowerHIRTransactions lowers every source-independent HIR transaction in
// package/function order. The owning HIR function supplies the effect set, and
// every result passes the transaction-MIR verifier before publication.
func LowerHIRTransactions(input hir.Package) ([]LoweredTransaction, error) {
	layouts, err := LowerHIRLayouts(input)
	if err != nil {
		return nil, err
	}
	layoutByPackage := make(map[hir.PackageID]Layout, len(layouts))
	for _, item := range layouts {
		layoutByPackage[item.Package] = item.Layout
	}

	transactions := slices.Clone(input.StateTransactions)
	slices.SortFunc(transactions, compareHIRTransactions)
	result := make([]LoweredTransaction, 0, len(transactions))
	for index, transaction := range transactions {
		if transaction.Function == hir.NoSymbol {
			return nil, fmt.Errorf("package %d state transaction has no function symbol", transaction.Package)
		}
		if index > 0 && transactions[index-1].Package == transaction.Package && transactions[index-1].Function == transaction.Function {
			return nil, fmt.Errorf("package %d function %d declares multiple state transactions", transaction.Package, transaction.Function)
		}
		layout, found := layoutByPackage[transaction.Package]
		if !found {
			return nil, fmt.Errorf("package %d state transaction has no owned layout", transaction.Package)
		}
		function, err := findHIRFunction(input, transaction.Package, transaction.Function)
		if err != nil {
			return nil, err
		}
		if hirBlockContainsCall(function.Body) {
			return nil, fmt.Errorf("package %d function %d state transaction owner contains a call; calls are not admitted", transaction.Package, transaction.Function)
		}
		if hirBlockContainsDynamicControlFlow(function.Body) {
			return nil, fmt.Errorf("package %d function %d state transaction owner contains dynamic control flow; dynamic control flow is not admitted", transaction.Package, transaction.Function)
		}
		if err := verifyHIRStateOwnerMemoryBoundary(input, function); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		if err := verifyHIRStateOwnerEntryBoundary(function); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		program, err := lowerHIRTransaction(layout, function.Effects, transaction)
		if err != nil {
			return nil, fmt.Errorf("package %d function %d state transaction: %w", transaction.Package, transaction.Function, err)
		}
		result = append(result, LoweredTransaction{Package: transaction.Package, Function: transaction.Function, Program: program})
	}
	return result, nil
}

func compareHIRTransactions(left, right hir.StateTransaction) int {
	if left.Package < right.Package {
		return -1
	}
	if left.Package > right.Package {
		return 1
	}
	if left.Function < right.Function {
		return -1
	}
	if left.Function > right.Function {
		return 1
	}
	return 0
}

func findHIRFunction(input hir.Package, packageID hir.PackageID, symbol hir.SymbolID) (hir.Function, error) {
	var result hir.Function
	found := false
	for _, module := range input.Modules {
		if module.Package != packageID {
			continue
		}
		for _, function := range module.Functions {
			if function.Symbol != symbol {
				continue
			}
			if found {
				return hir.Function{}, fmt.Errorf("package %d function symbol %d is duplicated", packageID, symbol)
			}
			result = function
			found = true
		}
	}
	if !found {
		return hir.Function{}, fmt.Errorf("package %d state transaction names missing function %d", packageID, symbol)
	}
	return result, nil
}

func hirBlockContainsCall(block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementContainsCall(&block.Statements[index]) {
			return true
		}
	}
	return hirExpressionContainsCall(block.Tail)
}

func hirStatementContainsCall(statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if hirExpressionContainsCall(statement.Target) || hirExpressionContainsCall(statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementContainsCall(statement.For.Initializer) ||
		hirExpressionContainsCall(statement.For.Condition) ||
		hirStatementContainsCall(statement.For.Post) || hirBlockContainsCall(statement.For.Body)
}

func hirExpressionContainsCall(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == "call" {
		return true
	}
	for index := range expression.Children {
		if hirExpressionContainsCall(&expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockContainsCall(*expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionContainsCall(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func hirBlockContainsDynamicControlFlow(block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementContainsDynamicControlFlow(&block.Statements[index]) {
			return true
		}
	}
	return hirExpressionContainsDynamicControlFlow(block.Tail)
}

func hirStatementContainsDynamicControlFlow(statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	switch statement.Kind {
	case "assert", "while", "for", "break", "continue":
		return true
	}
	if hirExpressionContainsDynamicControlFlow(statement.Target) || hirExpressionContainsDynamicControlFlow(statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementContainsDynamicControlFlow(statement.For.Initializer) ||
		hirExpressionContainsDynamicControlFlow(statement.For.Condition) ||
		hirStatementContainsDynamicControlFlow(statement.For.Post) || hirBlockContainsDynamicControlFlow(statement.For.Body)
}

func hirExpressionContainsDynamicControlFlow(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case "if", "match", "while", "propagate":
		return true
	case "binary":
		if expression.Operator == "&&" || expression.Operator == "||" {
			return true
		}
	}
	for index := range expression.Children {
		if hirExpressionContainsDynamicControlFlow(&expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockContainsDynamicControlFlow(*expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionContainsDynamicControlFlow(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func lowerHIRTransaction(layout Layout, effects []hir.Effect, transaction hir.StateTransaction) (Program, error) {
	fieldIndices := make(map[string]uint32, len(layout.Fields))
	for _, field := range layout.Fields {
		fieldIndices[field.Name] = field.Index
	}
	capability := newTransactionCapability()
	operations := make([]Operation, 0, len(transaction.Accesses))
	for index, access := range transaction.Accesses {
		fieldIndex, found := fieldIndices[access.Field]
		if !found {
			return Program{}, fmt.Errorf("access %d names unknown field %q", index, access.Field)
		}
		kind, err := lowerStateAccessKind(access.Kind)
		if err != nil {
			return Program{}, fmt.Errorf("access %d: %w", index, err)
		}
		operations = append(operations, Operation{
			Kind: kind, Capability: capability, FieldIndex: fieldIndex,
			Key: lowerStateValue(access.Key), Value: lowerStateValue(access.Value),
		})
	}
	exit, err := lowerStateTransactionExit(transaction.Exit)
	if err != nil {
		return Program{}, err
	}
	program := Program{
		Layout: layout, Capability: capability, Effects: slices.Clone(effects), Operations: operations,
		Exit: exit, TrapCategory: transaction.TrapCategory,
	}
	if failures := Verify(program); len(failures) != 0 {
		return Program{}, VerificationErrors{Failures: failures}
	}
	return program, nil
}

func lowerStateAccessKind(kind hir.StateAccessKind) (OperationKind, error) {
	switch kind {
	case hir.StateAccessCellRead:
		return OperationCellRead, nil
	case hir.StateAccessCellWrite:
		return OperationCellWrite, nil
	case hir.StateAccessCellDelete:
		return OperationCellDelete, nil
	case hir.StateAccessMapRead:
		return OperationMapRead, nil
	case hir.StateAccessMapWrite:
		return OperationMapWrite, nil
	case hir.StateAccessMapDelete:
		return OperationMapDelete, nil
	default:
		return "", fmt.Errorf("unsupported state access kind %q", kind)
	}
}

func lowerStateTransactionExit(exit hir.StateTransactionExit) (ExitKind, error) {
	switch exit {
	case hir.StateTransactionCommit:
		return ExitCommit, nil
	case hir.StateTransactionReject:
		return ExitReject, nil
	case hir.StateTransactionTrap:
		return ExitTrap, nil
	default:
		return "", fmt.Errorf("unsupported state transaction exit %q", exit)
	}
}

func lowerStateValue(value hir.StateValue) Value {
	return Value{TypeID: value.TypeID, Encoded: slices.Clone(value.Encoded)}
}

func lowerFieldKind(kind hir.StateFieldKind) (contractstate.FieldKind, error) {
	switch kind {
	case hir.StateFieldCell:
		return contractstate.FieldCell, nil
	case hir.StateFieldMap:
		return contractstate.FieldMap, nil
	default:
		return 0, fmt.Errorf("unsupported field kind %q", kind)
	}
}
