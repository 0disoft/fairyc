package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyHIRStateOwnerEntryBoundary(function hir.Function) error {
	if function.Receiver != "" {
		return fmt.Errorf("state transaction owner has receiver mode %q; receiver-bound owners are not admitted", function.Receiver)
	}
	if function.DropGlueFor != hir.ParseType("") {
		return fmt.Errorf("state transaction owner is drop glue for %q; drop-glue owners are not admitted", function.DropGlueFor)
	}
	return nil
}

func verifyHIRStateOwnerTrapBoundary(function hir.Function) error {
	if hirBlockMayTrap(function.Body) {
		return fmt.Errorf("state transaction owner contains a potentially trapping operation; potentially trapping operations are not admitted")
	}
	return nil
}

func hirBlockMayTrap(block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementMayTrap(&block.Statements[index]) {
			return true
		}
	}
	return hirExpressionMayTrap(block.Tail)
}

func hirStatementMayTrap(statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if statement.Kind == hir.StatementAssert {
		return true
	}
	if statement.Kind == hir.StatementAssignment {
		if operator, ok := hir.CompoundAssignmentBinary(statement.Operator); ok && hirOperatorMayTrap(operator) {
			return true
		}
	}
	if hirExpressionMayTrap(statement.Target) || hirExpressionMayTrap(statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementMayTrap(statement.For.Initializer) ||
		hirExpressionMayTrap(statement.For.Condition) ||
		hirStatementMayTrap(statement.For.Post) || hirBlockMayTrap(statement.For.Body)
}

func hirExpressionMayTrap(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	switch expression.Kind {
	case hir.ExpressionArrayIndex, hir.ExpressionViewIndex, hir.ExpressionViewSlice:
		return true
	case hir.ExpressionPrefix, hir.ExpressionBinary:
		if hirOperatorMayTrap(expression.Operator) {
			return true
		}
	}
	for index := range expression.Children {
		if hirExpressionMayTrap(&expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockMayTrap(*expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionMayTrap(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func hirOperatorMayTrap(operator string) bool {
	switch operator {
	case "+", "-", "*", "/", "%", "<<", ">>":
		return true
	default:
		return false
	}
}

func verifyHIRStateOwnerLifetimeBoundary(input hir.Package, function hir.Function) error {
	if hirFunctionUsesOwnedResource(input, function) {
		return fmt.Errorf("state transaction owner carries an owned resource value; owned resource values are not admitted")
	}
	if hirBlockContainsLexicalRegion(function.Body) {
		return fmt.Errorf("state transaction owner contains a lexical region; lexical regions are not admitted")
	}
	return nil
}

func hirFunctionUsesOwnedResource(input hir.Package, function hir.Function) bool {
	types := make([]hir.Type, 0, 1+len(function.Parameters)+len(function.Locals))
	types = append(types, function.ResultType)
	for _, parameter := range function.Parameters {
		types = append(types, parameter.Type)
	}
	for _, local := range function.Locals {
		types = append(types, local.Type)
	}
	for _, value := range types {
		if hirTypeContainsOwnedResource(input, value, make(map[hir.Type]bool)) {
			return true
		}
	}
	return hirBlockUsesOwnedResource(input, function.Body)
}

func hirBlockUsesOwnedResource(input hir.Package, block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementUsesOwnedResource(input, &block.Statements[index]) {
			return true
		}
	}
	return hirExpressionUsesOwnedResource(input, block.Tail)
}

func hirStatementUsesOwnedResource(input hir.Package, statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if hirExpressionUsesOwnedResource(input, statement.Target) || hirExpressionUsesOwnedResource(input, statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementUsesOwnedResource(input, statement.For.Initializer) ||
		hirExpressionUsesOwnedResource(input, statement.For.Condition) ||
		hirStatementUsesOwnedResource(input, statement.For.Post) || hirBlockUsesOwnedResource(input, statement.For.Body)
}

func hirExpressionUsesOwnedResource(input hir.Package, expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if hirTypeContainsOwnedResource(input, expression.Type, make(map[hir.Type]bool)) ||
		hirTypeContainsOwnedResource(input, expression.TargetType, make(map[hir.Type]bool)) {
		return true
	}
	for index := range expression.Children {
		if hirExpressionUsesOwnedResource(input, &expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockUsesOwnedResource(input, *expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionUsesOwnedResource(input, &expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func hirTypeContainsOwnedResource(input hir.Package, value hir.Type, visiting map[hir.Type]bool) bool {
	if value == hir.ParseType("") || value == hir.TypeInvalid {
		return false
	}
	if _, _, borrow := hir.BorrowElement(value); borrow {
		return false
	}
	for _, module := range input.Modules {
		for _, resource := range module.Resources {
			if resource.Type == value {
				return true
			}
		}
	}
	if visiting[value] {
		return false
	}
	visiting[value] = true
	defer delete(visiting, value)

	if element, _, array := hir.FixedArrayParts(value); array {
		return hirTypeContainsOwnedResource(input, element, visiting)
	}
	if element, option := hir.OptionElement(value); option {
		return hirTypeContainsOwnedResource(input, element, visiting)
	}
	if success, failure, result := hir.ResultElements(value); result {
		return hirTypeContainsOwnedResource(input, success, visiting) || hirTypeContainsOwnedResource(input, failure, visiting)
	}
	for _, module := range input.Modules {
		for _, structure := range module.Structs {
			if structure.Type != value {
				continue
			}
			for _, field := range structure.Fields {
				if hirTypeContainsOwnedResource(input, field.Type, visiting) {
					return true
				}
			}
		}
		for _, enum := range module.Enums {
			if enum.Type != value {
				continue
			}
			for _, variant := range enum.Variants {
				for _, field := range variant.Fields {
					if hirTypeContainsOwnedResource(input, field.Type, visiting) {
						return true
					}
				}
			}
		}
	}
	return false
}

func hirBlockContainsLexicalRegion(block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementContainsLexicalRegion(&block.Statements[index]) {
			return true
		}
	}
	return hirExpressionContainsLexicalRegion(block.Tail)
}

func hirStatementContainsLexicalRegion(statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if hirExpressionContainsLexicalRegion(statement.Target) || hirExpressionContainsLexicalRegion(statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementContainsLexicalRegion(statement.For.Initializer) ||
		hirExpressionContainsLexicalRegion(statement.For.Condition) ||
		hirStatementContainsLexicalRegion(statement.For.Post) || hirBlockContainsLexicalRegion(statement.For.Body)
}

func hirExpressionContainsLexicalRegion(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == hir.ExpressionRegion {
		return true
	}
	for index := range expression.Children {
		if hirExpressionContainsLexicalRegion(&expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockContainsLexicalRegion(*expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionContainsLexicalRegion(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func verifyHIRStateOwnerClosedFormBoundary(function hir.Function) error {
	if err := verifyHIRStateOwnerBlockClosedForm(function.Body); err != nil {
		return fmt.Errorf("state transaction owner is outside the closed HIR form: %w", err)
	}
	return nil
}

func verifyHIRStateOwnerBlockClosedForm(block hir.Block) error {
	for index := range block.Statements {
		if err := verifyHIRStateOwnerStatementClosedForm(&block.Statements[index]); err != nil {
			return err
		}
	}
	return verifyHIRStateOwnerExpressionClosedForm(block.Tail)
}

func verifyHIRStateOwnerStatementClosedForm(statement *hir.Statement) error {
	if statement == nil {
		return nil
	}
	switch statement.Kind {
	case hir.StatementBinding, hir.StatementReturn, hir.StatementDiscard, hir.StatementExpressionStatement:
	case hir.StatementAssignment:
		operator := statement.Operator
		if operator == "" {
			operator = "="
		}
		switch operator {
		case "=", "&=", "|=", "^=":
		default:
			return fmt.Errorf("assignment operator %q is not admitted", statement.Operator)
		}
	default:
		return fmt.Errorf("statement kind %q is not admitted", statement.Kind)
	}
	if statement.For != nil {
		return fmt.Errorf("statement kind %q carries loop state", statement.Kind)
	}
	if err := verifyHIRStateOwnerExpressionClosedForm(statement.Target); err != nil {
		return err
	}
	return verifyHIRStateOwnerExpressionClosedForm(statement.Expression)
}

func verifyHIRStateOwnerExpressionClosedForm(expression *hir.Expression) error {
	if expression == nil {
		return nil
	}
	if len(expression.Effects) != 0 && expression.Kind != hir.ExpressionCall {
		return fmt.Errorf("expression kind %q carries expression-local effects", expression.Kind)
	}
	switch expression.Kind {
	case hir.ExpressionLiteral, hir.ExpressionPath, hir.ExpressionGroup, hir.ExpressionRecord, hir.ExpressionArray, hir.ExpressionArrayViewBorrow, hir.ExpressionArrayLength,
		hir.ExpressionFieldExpression, hir.ExpressionViewLength, hir.ExpressionViewEmpty, hir.ExpressionScalarCast, hir.ExpressionResultVariant,
		hir.ExpressionOptionVariant, hir.ExpressionEnumVariant, hir.ExpressionBlock, hir.ExpressionCall:
	case hir.ExpressionPrefix:
		switch expression.Operator {
		case "+", "!", "~", "&", "&mut", "*":
		default:
			return fmt.Errorf("prefix operator %q is not admitted", expression.Operator)
		}
	case hir.ExpressionBinary:
		switch expression.Operator {
		case "==", "!=", "<", "<=", ">", ">=", "&", "|", "^":
		default:
			return fmt.Errorf("binary operator %q is not admitted", expression.Operator)
		}
	default:
		return fmt.Errorf("expression kind %q is not admitted", expression.Kind)
	}
	for index := range expression.Children {
		if err := verifyHIRStateOwnerExpressionClosedForm(&expression.Children[index]); err != nil {
			return err
		}
	}
	if expression.Block != nil {
		if err := verifyHIRStateOwnerBlockClosedForm(*expression.Block); err != nil {
			return err
		}
	}
	for index := range expression.Arms {
		if err := verifyHIRStateOwnerExpressionClosedForm(&expression.Arms[index].Expression); err != nil {
			return err
		}
	}
	return nil
}

func verifyHIRStateOwnerMemoryBoundary(input hir.Package, function hir.Function) error {
	if function.Unsafe {
		return fmt.Errorf("state transaction owner is unsafe; unsafe owners are not admitted")
	}
	if hirBlockContainsUnsafeCode(function.Body) {
		return fmt.Errorf("state transaction owner contains unsafe code; unsafe code is not admitted")
	}
	if hirFunctionUsesRawPointer(input, function) {
		return fmt.Errorf("state transaction owner uses a raw-pointer type; raw-pointer types are not admitted")
	}
	return nil
}

func hirBlockContainsUnsafeCode(block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementContainsUnsafeCode(&block.Statements[index]) {
			return true
		}
	}
	return hirExpressionContainsUnsafeCode(block.Tail)
}

func hirStatementContainsUnsafeCode(statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if hirExpressionContainsUnsafeCode(statement.Target) || hirExpressionContainsUnsafeCode(statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementContainsUnsafeCode(statement.For.Initializer) ||
		hirExpressionContainsUnsafeCode(statement.For.Condition) ||
		hirStatementContainsUnsafeCode(statement.For.Post) || hirBlockContainsUnsafeCode(statement.For.Body)
}

func hirExpressionContainsUnsafeCode(expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if expression.Kind == hir.ExpressionUnsafe || expression.UnsafeScope != nil {
		return true
	}
	for index := range expression.Children {
		if hirExpressionContainsUnsafeCode(&expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockContainsUnsafeCode(*expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionContainsUnsafeCode(&expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func hirFunctionUsesRawPointer(input hir.Package, function hir.Function) bool {
	types := make([]hir.Type, 0, 3+len(function.Parameters)+len(function.Locals))
	types = append(types, function.OwnerType, function.ResultType, function.DropGlueFor)
	for _, parameter := range function.Parameters {
		types = append(types, parameter.Type)
	}
	for _, local := range function.Locals {
		types = append(types, local.Type)
	}
	for _, value := range types {
		if hirTypeContainsRawPointer(input, value, make(map[hir.Type]bool)) {
			return true
		}
	}
	return hirBlockUsesRawPointer(input, function.Body)
}

func hirBlockUsesRawPointer(input hir.Package, block hir.Block) bool {
	for index := range block.Statements {
		if hirStatementUsesRawPointer(input, &block.Statements[index]) {
			return true
		}
	}
	return hirExpressionUsesRawPointer(input, block.Tail)
}

func hirStatementUsesRawPointer(input hir.Package, statement *hir.Statement) bool {
	if statement == nil {
		return false
	}
	if hirExpressionUsesRawPointer(input, statement.Target) || hirExpressionUsesRawPointer(input, statement.Expression) {
		return true
	}
	if statement.For == nil {
		return false
	}
	return hirStatementUsesRawPointer(input, statement.For.Initializer) ||
		hirExpressionUsesRawPointer(input, statement.For.Condition) ||
		hirStatementUsesRawPointer(input, statement.For.Post) || hirBlockUsesRawPointer(input, statement.For.Body)
}

func hirExpressionUsesRawPointer(input hir.Package, expression *hir.Expression) bool {
	if expression == nil {
		return false
	}
	if hirTypeContainsRawPointer(input, expression.Type, make(map[hir.Type]bool)) ||
		hirTypeContainsRawPointer(input, expression.TargetType, make(map[hir.Type]bool)) {
		return true
	}
	for index := range expression.Children {
		if hirExpressionUsesRawPointer(input, &expression.Children[index]) {
			return true
		}
	}
	if expression.Block != nil && hirBlockUsesRawPointer(input, *expression.Block) {
		return true
	}
	for index := range expression.Arms {
		if hirExpressionUsesRawPointer(input, &expression.Arms[index].Expression) {
			return true
		}
	}
	return false
}

func hirTypeContainsRawPointer(input hir.Package, value hir.Type, visiting map[hir.Type]bool) bool {
	if value == hir.ParseType("") || value == hir.TypeInvalid {
		return false
	}
	if _, _, pointer := hir.RawPointerElement(value); pointer {
		return true
	}
	if visiting[value] {
		return false
	}
	visiting[value] = true
	defer delete(visiting, value)

	if element, _, borrow := hir.BorrowElement(value); borrow {
		return hirTypeContainsRawPointer(input, element, visiting)
	}
	if element, _, array := hir.FixedArrayParts(value); array {
		return hirTypeContainsRawPointer(input, element, visiting)
	}
	if element, option := hir.OptionElement(value); option {
		return hirTypeContainsRawPointer(input, element, visiting)
	}
	if success, failure, result := hir.ResultElements(value); result {
		return hirTypeContainsRawPointer(input, success, visiting) || hirTypeContainsRawPointer(input, failure, visiting)
	}
	for _, module := range input.Modules {
		for _, structure := range module.Structs {
			if structure.Type != value {
				continue
			}
			for _, field := range structure.Fields {
				if hirTypeContainsRawPointer(input, field.Type, visiting) {
					return true
				}
			}
		}
		for _, enum := range module.Enums {
			if enum.Type != value {
				continue
			}
			for _, variant := range enum.Variants {
				for _, field := range variant.Fields {
					if hirTypeContainsRawPointer(input, field.Type, visiting) {
						return true
					}
				}
			}
		}
		for _, resource := range module.Resources {
			if resource.Type != value {
				continue
			}
			for _, field := range resource.Fields {
				if hirTypeContainsRawPointer(input, field.Type, visiting) {
					return true
				}
			}
		}
	}
	return false
}
