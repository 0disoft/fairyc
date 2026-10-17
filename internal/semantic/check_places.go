package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) mutableAssignmentTarget(source *ast.Expression, target *hir.Expression) bool {
	if source == nil || target == nil {
		return false
	}
	switch target.Kind {
	case hir.ExpressionPath:
		resolved := c.analyzer.resolved[source]
		return resolved != nil && resolved.mutable
	case hir.ExpressionArrayIndex:
		if len(source.Children) != 2 || len(target.Children) != 2 {
			return false
		}
		if view, ok := c.arrayFieldView(source.Children[0], target.Children[0], true); ok {
			target.Kind = hir.ExpressionViewIndex
			target.Children[0] = view
			return true
		}
		receiver := assignmentRootPath(source.Children[0])
		resolved := c.analyzer.resolved[receiver]
		_, _, fixedArray := hir.FixedArrayParts(target.Children[0].Type)
		return fixedArray && resolved != nil && resolved.mutable
	case hir.ExpressionViewIndex:
		if len(source.Children) == 2 && arrayFieldRoot(source.Children[0]) != nil {
			field, _ := c.checkExpression(source.Children[0], hir.TypeInvalid)
			view, ok := c.arrayFieldView(source.Children[0], *field, true)
			if !ok {
				return false
			}
			target.Children[0] = view
			return true
		}
		return len(target.Children) == 2 && hir.MutableIndexedView(target.Children[0].Type)
	case hir.ExpressionPrefix:
		if !derefAssignmentTarget(source) || target.Operator != "*" || len(target.Children) != 1 ||
			(!hir.FixedArrayScalarElement(target.Type) && !c.analyzer.plainViewStruct(target.Type)) || !directDerefBorrowPath(source.Children[0]) {
			return false
		}
		element, mutable, borrowed := hir.BorrowElement(target.Children[0].Type)
		return borrowed && mutable && element == target.Type
	case hir.ExpressionFieldExpression:
		if len(source.Children) != 1 || len(target.Children) != 1 {
			return false
		}
		root := target
		depth := 0
		for root.Kind == hir.ExpressionFieldExpression || root.Kind == hir.ExpressionGroup {
			if len(root.Children) != 1 {
				return false
			}
			if root.Kind == hir.ExpressionFieldExpression {
				depth++
			}
			root = &root.Children[0]
		}
		if root.Kind == hir.ExpressionViewIndex {
			if index := indexedPlace(source); index != nil && arrayFieldRoot(index.Children[0]) != nil {
				_, stored := c.indexedArrayBorrow(index, root, true)
				return stored && c.analyzer.plainViewStruct(root.Type) &&
					(hir.FixedArrayScalarElement(target.Type) || c.analyzer.plainViewStruct(target.Type))
			}
			return c.indexedStructPlace(source) && hir.MutableIndexedView(root.Children[0].Type) && (hir.FixedArrayScalarElement(target.Type) || c.analyzer.plainViewStruct(target.Type))
		}
		if root.Kind == hir.ExpressionArrayIndex {
			_, stored := c.indexedArrayBorrow(borrowPlaceRoot(source), root, true)
			return stored && c.analyzer.plainViewStruct(root.Type) &&
				(hir.FixedArrayScalarElement(target.Type) || c.analyzer.plainViewStruct(target.Type))
		}
		if root.Kind != hir.ExpressionPath {
			return false
		}
		rootType := root.Type
		_, mutable, borrowed := hir.BorrowElement(rootType)
		if borrowed {
			rootType, _, _ = hir.BorrowElement(rootType)
		}
		if depth > 1 && ((!hir.PlainViewField(target.Type) && !c.analyzer.inlineArrayField(target.Type) && !hir.PlainStoredStruct(target.Type, c.analyzer.plainStructFields)) || !hir.PlainStoredStruct(rootType, c.analyzer.plainStructFields)) {
			return false
		}
		if borrowed {
			return mutable
		}
		resolved := c.analyzer.resolved[fieldAssignmentRoot(source)]
		return resolved != nil && resolved.mutable
	default:
		return false
	}
}

func derefAssignmentTarget(expression *ast.Expression) bool {
	return expression != nil && expression.Kind == syntax.KindPrefix && expression.Text == "*" && len(expression.Children) == 1
}

func directDerefBorrowPath(expression *ast.Expression) bool {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	return expression != nil && expression.Kind == syntax.KindPath
}

func assignmentRootPath(expression *ast.Expression) *ast.Expression {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	if expression == nil || expression.Kind != syntax.KindPath {
		return nil
	}
	return expression
}
