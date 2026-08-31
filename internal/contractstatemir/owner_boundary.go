package contractstatemir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func verifyHIRStateOwnerEntryBoundary(function hir.Function) error {
	if function.Receiver != "" {
		return fmt.Errorf("state transaction owner has receiver mode %q; receiver-bound owners are not admitted", function.Receiver)
	}
	if function.DropGlueFor != "" {
		return fmt.Errorf("state transaction owner is drop glue for %q; drop-glue owners are not admitted", function.DropGlueFor)
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
	if expression.Kind == "unsafe" || expression.UnsafeScope != nil {
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
	if value == "" || value == hir.TypeInvalid {
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
