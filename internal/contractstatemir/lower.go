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
	type node struct {
		transaction hir.StateTransaction
		function    hir.Function
		name        string
		program     Program
	}
	names := make(map[hir.SymbolID]string, len(transactions))
	nameOwners := make(map[string]hir.SymbolID, len(transactions))
	functions := make(map[hir.SymbolID]hir.Function, len(transactions))
	packages := make(map[hir.SymbolID]hir.PackageID, len(transactions))
	for index, transaction := range transactions {
		if transaction.Function == hir.NoSymbol {
			return nil, fmt.Errorf("package %d state transaction has no function symbol", transaction.Package)
		}
		if index > 0 && transactions[index-1].Package == transaction.Package && transactions[index-1].Function == transaction.Function {
			return nil, fmt.Errorf("package %d function %d declares multiple state transactions", transaction.Package, transaction.Function)
		}
		module, function, err := findHIRFunctionOwner(input, transaction.Package, transaction.Function)
		if err != nil {
			return nil, err
		}
		if function.Name != "" && module.Path != "" {
			name := module.Path + "::" + function.Name
			if previous, duplicate := nameOwners[name]; duplicate && previous != transaction.Function {
				return nil, fmt.Errorf("package %d stable transaction name %q has ambiguous owners %d and %d", transaction.Package, name, previous, transaction.Function)
			}
			names[transaction.Function] = name
			nameOwners[name] = transaction.Function
		}
		if previous, duplicate := packages[transaction.Function]; duplicate && previous != transaction.Package {
			return nil, fmt.Errorf("state transaction function symbol %d is shared by packages %d and %d", transaction.Function, previous, transaction.Package)
		}
		functions[transaction.Function] = function
		packages[transaction.Function] = transaction.Package
	}

	nodes := make(map[hir.SymbolID]node, len(transactions))
	called := make(map[hir.SymbolID]bool, len(transactions))
	for _, transaction := range transactions {
		layout, found := layoutByPackage[transaction.Package]
		if !found {
			return nil, fmt.Errorf("package %d state transaction has no owned layout", transaction.Package)
		}
		function := functions[transaction.Function]
		hirCalls, err := hirStaticStateCalls(function.Body)
		if err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		stateCalls := hirStateTransactionCalls(transaction)
		if len(transaction.Blocks) != 0 && len(stateCalls) != 0 {
			return nil, fmt.Errorf("package %d function %d graph helper calls are not admitted", transaction.Package, transaction.Function)
		}
		if !slices.Equal(hirCalls, stateCalls) {
			return nil, fmt.Errorf("package %d function %d helper calls do not match the transaction body", transaction.Package, transaction.Function)
		}
		for _, callee := range hirCalls {
			calleeFunction, found := functions[callee]
			if !found {
				return nil, fmt.Errorf("package %d function %d calls a missing or non-transaction helper %d", transaction.Package, transaction.Function, callee)
			}
			if packages[callee] != transaction.Package {
				return nil, fmt.Errorf("package %d function %d calls cross-package transaction helper %d", transaction.Package, transaction.Function, callee)
			}
			if len(calleeFunction.Parameters) != 0 || calleeFunction.ResultType != hir.TypeVoid {
				return nil, fmt.Errorf("package %d helper %d must be void and have no parameters", transaction.Package, callee)
			}
			if names[callee] == "" {
				return nil, fmt.Errorf("package %d helper %d has no stable module-qualified name", transaction.Package, callee)
			}
			called[callee] = true
		}
		if err := verifyHIRStateCallEffects(function.Body, functions); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
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
		if err := verifyHIRStateOwnerTrapBoundary(function); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		if err := verifyHIRStateOwnerLifetimeBoundary(input, function); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		if err := verifyHIRStateOwnerClosedFormBoundary(function); err != nil {
			return nil, fmt.Errorf("package %d function %d: %w", transaction.Package, transaction.Function, err)
		}
		program, err := lowerHIRTransaction(layout, function.Effects, transaction, names)
		if err != nil {
			return nil, fmt.Errorf("package %d function %d state transaction: %w", transaction.Package, transaction.Function, err)
		}
		nodes[transaction.Function] = node{transaction: transaction, function: function, name: names[transaction.Function], program: program}
	}

	result := make([]LoweredTransaction, 0, len(nodes))
	for _, transaction := range transactions {
		if called[transaction.Function] {
			continue
		}
		root := nodes[transaction.Function]
		program := root.program
		program.Operations = cloneStateOperations(root.program.Operations)
		program.Blocks = cloneStateBlocks(root.program.Blocks)
		reachable := make(map[string]bool)
		var collect func(Program)
		collect = func(body Program) {
			for _, name := range bodyCalls(body.Operations, body.Blocks) {
				if reachable[name] {
					continue
				}
				reachable[name] = true
				for _, candidate := range nodes {
					if candidate.name == name {
						collect(candidate.program)
						break
					}
				}
			}
		}
		collect(program)
		helperNames := make([]string, 0, len(reachable))
		for name := range reachable {
			helperNames = append(helperNames, name)
		}
		slices.Sort(helperNames)
		for _, name := range helperNames {
			for _, candidate := range nodes {
				if candidate.name != name {
					continue
				}
				body := candidate.program
				program.Helpers = append(program.Helpers, Helper{
					Name: name, Effects: slices.Clone(body.Effects), Operations: cloneStateOperations(body.Operations),
					Exit: body.Exit, TrapCategory: body.TrapCategory, Entry: body.Entry, Blocks: cloneStateBlocks(body.Blocks),
				})
				break
			}
		}
		program.Capability = newTransactionCapability()
		shareHelperCapability(&program)
		if failures := Verify(program); len(failures) != 0 {
			return nil, fmt.Errorf("package %d function %d state transaction: %w", transaction.Package, transaction.Function, VerificationErrors{Failures: failures})
		}
		result = append(result, LoweredTransaction{Package: transaction.Package, Function: transaction.Function, Program: program})
	}
	if len(nodes) != 0 && len(result) == 0 {
		return nil, fmt.Errorf("contract-state helper call graph has no root transaction")
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

func hirStaticStateCalls(block hir.Block) ([]hir.SymbolID, error) {
	result := make([]hir.SymbolID, 0)
	var visitExpression func(*hir.Expression) error
	var visitBlock func(hir.Block) error
	visitExpression = func(expression *hir.Expression) error {
		if expression == nil {
			return nil
		}
		if expression.Kind == "call" {
			if expression.Symbol == hir.NoSymbol || expression.ForeignABI != "" || expression.Type != hir.TypeVoid || len(expression.Children) != 1 {
				return fmt.Errorf("state transaction owner contains a call; calls are not admitted unless resolved to an internal void transaction helper with no arguments")
			}
			result = append(result, expression.Symbol)
		}
		for index := range expression.Children {
			if expression.Kind == "call" && index == 0 {
				continue
			}
			if err := visitExpression(&expression.Children[index]); err != nil {
				return err
			}
		}
		if expression.Block != nil {
			if err := visitBlock(*expression.Block); err != nil {
				return err
			}
		}
		for index := range expression.Arms {
			if err := visitExpression(&expression.Arms[index].Expression); err != nil {
				return err
			}
		}
		return nil
	}
	visitBlock = func(value hir.Block) error {
		for index := range value.Statements {
			statement := &value.Statements[index]
			if err := visitExpression(statement.Target); err != nil {
				return err
			}
			if err := visitExpression(statement.Expression); err != nil {
				return err
			}
			if statement.For != nil {
				return fmt.Errorf("state transaction owner contains a call; calls are not admitted inside loop state")
			}
		}
		return visitExpression(value.Tail)
	}
	return result, visitBlock(block)
}

func hirStateTransactionCalls(transaction hir.StateTransaction) []hir.SymbolID {
	result := make([]hir.SymbolID, 0)
	for _, access := range transaction.Accesses {
		if access.Kind == hir.StateAccessCall {
			result = append(result, access.Callee)
		}
	}
	for _, block := range transaction.Blocks {
		for _, access := range block.Accesses {
			if access.Kind == hir.StateAccessCall {
				result = append(result, access.Callee)
			}
		}
	}
	return result
}

func verifyHIRStateCallEffects(block hir.Block, functions map[hir.SymbolID]hir.Function) error {
	var visitExpression func(*hir.Expression) error
	var visitBlock func(hir.Block) error
	visitExpression = func(expression *hir.Expression) error {
		if expression == nil {
			return nil
		}
		if expression.Kind == "call" {
			callee, found := functions[expression.Symbol]
			if !found || !slices.Equal(expression.Effects, callee.Effects) {
				return fmt.Errorf("state helper call effects do not match the resolved callee")
			}
		}
		for index := range expression.Children {
			if err := visitExpression(&expression.Children[index]); err != nil {
				return err
			}
		}
		if expression.Block != nil {
			if err := visitBlock(*expression.Block); err != nil {
				return err
			}
		}
		for index := range expression.Arms {
			if err := visitExpression(&expression.Arms[index].Expression); err != nil {
				return err
			}
		}
		return nil
	}
	visitBlock = func(value hir.Block) error {
		for index := range value.Statements {
			if err := visitExpression(value.Statements[index].Target); err != nil {
				return err
			}
			if err := visitExpression(value.Statements[index].Expression); err != nil {
				return err
			}
		}
		return visitExpression(value.Tail)
	}
	return visitBlock(block)
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

func lowerHIRTransaction(layout Layout, effects []hir.Effect, transaction hir.StateTransaction, helperNames map[hir.SymbolID]string) (Program, error) {
	fieldIndices := make(map[string]uint32, len(layout.Fields))
	for _, field := range layout.Fields {
		fieldIndices[field.Name] = field.Index
	}
	capability := newTransactionCapability()
	if len(transaction.Blocks) != 0 {
		blocks := make([]Block, 0, len(transaction.Blocks))
		for blockIndex, sourceBlock := range transaction.Blocks {
			parameters := make([]BlockParameter, 0, len(sourceBlock.Parameters))
			for _, parameter := range sourceBlock.Parameters {
				parameters = append(parameters, BlockParameter{ID: ValueID(parameter.ID), TypeID: parameter.TypeID})
			}
			operations := make([]Operation, 0, len(sourceBlock.Accesses))
			for accessIndex, access := range sourceBlock.Accesses {
				if access.Kind == hir.StateAccessCall {
					name, found := helperNames[access.Callee]
					if !found {
						return Program{}, fmt.Errorf("block %d access %d calls unknown helper %d", blockIndex, accessIndex, access.Callee)
					}
					operations = append(operations, Operation{Kind: OperationCall, Capability: capability, Callee: name, Cost: access.Cost})
					continue
				}
				fieldIndex, found := fieldIndices[access.Field]
				if !found {
					return Program{}, fmt.Errorf("block %d access %d names unknown field %q", blockIndex, accessIndex, access.Field)
				}
				kind, err := lowerStateAccessKind(access.Kind)
				if err != nil {
					return Program{}, fmt.Errorf("block %d access %d: %w", blockIndex, accessIndex, err)
				}
				operations = append(operations, Operation{
					Kind: kind, Capability: capability, FieldIndex: fieldIndex,
					Key: lowerStateValue(access.Key), Value: lowerStateValue(access.Value),
					KeyInput: ValueID(access.KeyInput), ValueInput: ValueID(access.ValueInput),
					Result: ValueID(access.Result), Found: ValueID(access.FoundResult), Cost: access.Cost,
				})
			}
			blocks = append(blocks, Block{
				ID: BlockID(sourceBlock.ID), Parameters: parameters, Operations: operations,
				Terminator: lowerStateTerminator(sourceBlock.Terminator),
			})
		}
		program := Program{
			Layout: layout, Capability: capability, Effects: slices.Clone(effects),
			Entry: BlockID(transaction.Entry), Blocks: blocks,
		}
		return program, nil
	}
	operations := make([]Operation, 0, len(transaction.Accesses))
	for index, access := range transaction.Accesses {
		if access.Kind == hir.StateAccessCall {
			name, found := helperNames[access.Callee]
			if !found {
				return Program{}, fmt.Errorf("access %d calls unknown helper %d", index, access.Callee)
			}
			operations = append(operations, Operation{Kind: OperationCall, Capability: capability, Callee: name, Cost: access.Cost})
			continue
		}
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
	return program, nil
}

func lowerStateTerminator(value hir.StateTerminator) Terminator {
	return Terminator{
		Kind: TerminatorKind(value.Kind), Condition: ValueID(value.Condition),
		Target: BlockID(value.Target), Arguments: lowerStateValueIDs(value.Arguments),
		TrueTarget: BlockID(value.TrueTarget), TrueArguments: lowerStateValueIDs(value.TrueArguments),
		FalseTarget: BlockID(value.FalseTarget), FalseArguments: lowerStateValueIDs(value.FalseArguments),
		TrapCategory: value.TrapCategory, Cost: value.Cost,
	}
}

func lowerStateValueIDs(values []hir.StateValueID) []ValueID {
	result := make([]ValueID, 0, len(values))
	for _, value := range values {
		result = append(result, ValueID(value))
	}
	return result
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
	case hir.StateTransactionReturn:
		return ExitReturn, nil
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
