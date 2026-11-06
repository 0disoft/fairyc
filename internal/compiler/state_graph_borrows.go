package compiler

import (
	"fmt"
	"strconv"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func stateGraphBorrowAlias(expression *ast.Expression, environment []stateGraphBinding) (stateGraphBinding, bool) {
	expression = stateGraphUngroup(expression)
	if expression == nil || expression.Kind != syntax.KindPath {
		return stateGraphBinding{}, false
	}
	binding, found := stateGraphLookup(expression.Text, environment)
	return binding, found && binding.borrow != 0
}

func (lower *stateGraphSource) nextBorrow() hir.StateBorrowID {
	lower.borrows++
	return lower.borrows
}

func (lower *stateGraphSource) borrowBinding(name string, expression *ast.Expression, current int, environment []stateGraphBinding) (stateGraphBinding, error) {
	if lower.helper {
		return stateGraphBinding{}, lower.fail(expression.Span, "state borrows belong to graph roots, not helpers")
	}
	operation, args, _ := stateGraphCall(expression)
	access := hir.StateAccess{Cost: 1, Span: lower.span(expression.Span)}
	binding := stateGraphBinding{name: name}
	if operation == "storage::borrow" {
		if len(args) == 0 {
			return stateGraphBinding{}, lower.fail(expression.Span, "borrow requires a field literal")
		}
		fieldName, ok := stateGraphString(args[0])
		field, found := lower.fields[fieldName]
		if !ok || !found {
			return stateGraphBinding{}, lower.fail(expression.Span, "borrow requires a declared field literal")
		}
		count := 1
		if field.Kind == hir.StateFieldMap {
			count++
		}
		if len(args) != count {
			return stateGraphBinding{}, lower.fail(expression.Span, "borrow operand count differs from field kind")
		}
		access.Field, access.Kind = fieldName, hir.StateAccessCellBorrow
		if field.Kind == hir.StateFieldMap {
			access.Kind = hir.StateAccessMapBorrow
			var err error
			access.Key, access.KeyInput, err = lower.operand(args[1], field.KeyTypeID, environment)
			if err != nil || access.KeyInput != 0 {
				return stateGraphBinding{}, lower.fail(expression.Span, "map borrow requires an exact literal key")
			}
		}
		access.BorrowResult, access.FoundResult = lower.nextBorrow(), lower.value()
		binding.borrow, binding.found, binding.typeID = access.BorrowResult, access.FoundResult, field.ValueTypeID
	} else {
		count := 1
		if operation == "storage::view" {
			count = 3
		}
		if len(args) != count {
			return stateGraphBinding{}, lower.fail(expression.Span, "view/copy requires a borrowed local and literal view bounds")
		}
		argument := stateGraphUngroup(args[0])
		if argument == nil || argument.Kind != syntax.KindPath {
			return stateGraphBinding{}, lower.fail(expression.Span, "view/copy requires an available borrowed local")
		}
		borrowed, found := stateGraphLookup(argument.Text, environment)
		if !found || borrowed.borrow == 0 {
			return stateGraphBinding{}, lower.fail(expression.Span, "borrow cannot be used outside its defining block")
		}
		binding.typeID = borrowed.typeID
		access.BorrowInput = borrowed.borrow
		if operation == "storage::copy" {
			access.Kind, access.Result = hir.StateAccessBorrowCopy, lower.value()
			binding.value = access.Result
		} else {
			offset, err := stateGraphViewBound(args[1])
			if err != nil {
				return stateGraphBinding{}, err
			}
			length, err := stateGraphViewBound(args[2])
			if err != nil || length == 0 {
				return stateGraphBinding{}, lower.fail(expression.Span, "borrow view requires a positive u32 length")
			}
			access.Kind, access.ViewOffset, access.ViewLength, access.BorrowResult = hir.StateAccessBorrowView, offset, length, lower.nextBorrow()
			binding.borrow, binding.found = access.BorrowResult, borrowed.found
		}
	}
	if err := lower.appendAccess(current, access); err != nil {
		return stateGraphBinding{}, err
	}
	return binding, nil
}

func stateGraphViewBound(expression *ast.Expression) (uint32, error) {
	if expression == nil || expression.Kind != syntax.KindLiteral {
		return 0, fmt.Errorf("borrow view bound requires a decimal u32 literal")
	}
	number, err := strconv.ParseUint(expression.Text, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid borrow view bound: %w", err)
	}
	return uint32(number), nil
}
