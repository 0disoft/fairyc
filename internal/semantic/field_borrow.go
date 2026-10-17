package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// sharedFieldBorrow projects an address, never a copied aggregate temporary.
// Every projection retains the complete root owner; fields do not split loans.
func (c *typeChecker) sharedFieldBorrow(source *ast.Expression, field hir.Expression) (hir.Expression, bool) {
	if root := borrowPlaceRoot(source); root != nil && root.Kind == syntax.KindIndex {
		if !c.indexedStructPlace(source) {
			return c.arrayFieldBorrow(source, field, false)
		}
		return projectFieldBorrow(field, false)
	}
	if result, ok := projectFieldBorrow(field, false); ok {
		return result, true
	}
	root := fieldAssignmentRoot(source)
	target := c.analyzer.resolved[root]
	if target == nil {
		return hir.Expression{}, false
	}
	element, mutable, borrowed := hir.BorrowElement(target.typ)
	if !borrowed || !mutable || !c.analyzer.plainViewStruct(element) {
		return hir.Expression{}, false
	}
	return projectFieldBorrow(shareFieldRoot(field), false)
}

// Shorten the named root before projecting shared addresses. Rebuild the HIR
// children so the original mutable place and its type facts stay unchanged.
func shareFieldRoot(field hir.Expression) hir.Expression {
	if field.Kind == hir.ExpressionPath {
		element, mutable, borrowed := hir.BorrowElement(field.Type)
		if borrowed && mutable {
			return hir.Expression{Kind: hir.ExpressionSharedReborrow, Type: hir.SharedBorrow(element),
				Span: field.Span, Symbol: hir.NoSymbol, Children: []hir.Expression{field}}
		}
	} else if (field.Kind == hir.ExpressionFieldExpression || field.Kind == hir.ExpressionGroup) && len(field.Children) == 1 {
		field.Children = []hir.Expression{shareFieldRoot(field.Children[0])}
	}
	return field
}

func projectFieldBorrow(field hir.Expression, mutable bool) (hir.Expression, bool) {
	if field.Kind == hir.ExpressionGroup && len(field.Children) == 1 {
		return projectFieldBorrow(field.Children[0], mutable)
	}
	if field.Kind != hir.ExpressionFieldExpression || len(field.Children) != 1 || field.Type == hir.TypeInvalid {
		return hir.Expression{}, false
	}
	if element, length, array := hir.FixedArrayParts(field.Type); array && !hir.ExecutableFixedArrayShape(element, length) {
		return hir.Expression{}, false
	}
	owner := field.Children[0]
	for owner.Kind == hir.ExpressionGroup && len(owner.Children) == 1 {
		owner = owner.Children[0]
	}
	if _, ownerMutable, borrowed := hir.BorrowElement(owner.Type); borrowed && ownerMutable != mutable {
		return hir.Expression{}, false
	} else if !borrowed {
		if owner.Kind == hir.ExpressionViewIndex {
			var ok bool
			owner, ok = indexedElementBorrow(owner, mutable)
			if !ok {
				return hir.Expression{}, false
			}
		} else if owner.Kind == hir.ExpressionPath && owner.Symbol != hir.NoSymbol {
			borrowType, operator := hir.SharedBorrow(owner.Type), "&"
			if mutable {
				borrowType, operator = hir.MutableBorrow(owner.Type), "&mut"
			}
			owner = hir.Expression{Kind: hir.ExpressionPrefix, Type: borrowType,
				Span: owner.Span, Symbol: hir.NoSymbol, Operator: operator, Children: []hir.Expression{owner}}
		} else {
			var ok bool
			owner, ok = projectFieldBorrow(owner, mutable)
			if !ok {
				return hir.Expression{}, false
			}
		}
	}
	borrowType := hir.SharedBorrow(field.Type)
	if mutable {
		borrowType = hir.MutableBorrow(field.Type)
	}
	return hir.Expression{Kind: hir.ExpressionFieldBorrow, Type: borrowType,
		Span: field.Span, Symbol: hir.NoSymbol, Operator: field.Operator, Children: []hir.Expression{owner}}, true
}

func borrowPlaceRoot(expression *ast.Expression) *ast.Expression {
	for expression != nil && len(expression.Children) == 1 &&
		(expression.Kind == syntax.KindFieldExpr || expression.Kind == syntax.KindGroup) {
		expression = expression.Children[0]
	}
	return expression
}

// fieldAssignmentRoot accepts only named places through field/group projections.
// It does not promote calls, index expressions or pointer dereferences to places.
func fieldAssignmentRoot(expression *ast.Expression) *ast.Expression {
	return assignmentRootPath(borrowPlaceRoot(expression))
}

// Mutable projections stay within the plain graph and named-place boundary.
func (c *typeChecker) mutableFieldBorrow(source *ast.Expression, child *hir.Expression) (hir.Expression, bool) {
	if child == nil {
		return hir.Expression{}, false
	}
	if c.indexedStructPlace(source) {
		return projectFieldBorrow(*child, true)
	}
	if field, ok := c.arrayFieldBorrow(source, *child, true); ok {
		return field, true
	}
	root := fieldAssignmentRoot(source)
	target := c.analyzer.resolved[root]
	if target == nil {
		return hir.Expression{}, false
	}
	rootType := target.typ
	if element, mutable, borrowed := hir.BorrowElement(rootType); borrowed {
		if !mutable {
			return hir.Expression{}, false
		}
		rootType = element
	} else if !target.mutable {
		return hir.Expression{}, false
	}
	if !c.analyzer.plainViewStruct(rootType) {
		return hir.Expression{}, false
	}
	return projectFieldBorrow(*child, true)
}

// Rebase each field projection on an address in the original fixed array.
// Never borrow a copied array-index value. Rebuild children without mutating
// the ordinary field-read HIR; every projected reference retains one root loan.
func (c *typeChecker) arrayFieldBorrow(source *ast.Expression, field hir.Expression, mutable bool) (hir.Expression, bool) {
	for field.Kind == hir.ExpressionGroup && len(field.Children) == 1 {
		field = field.Children[0]
	}
	if field.Kind != hir.ExpressionFieldExpression || len(field.Children) != 1 {
		return hir.Expression{}, false
	}
	owner := field.Children[0]
	for owner.Kind == hir.ExpressionGroup && len(owner.Children) == 1 {
		owner = owner.Children[0]
	}
	var borrowed hir.Expression
	var ok bool
	if (owner.Kind == hir.ExpressionArrayIndex || owner.Kind == hir.ExpressionViewIndex) && c.analyzer.plainViewStruct(owner.Type) {
		borrowed, ok = c.indexedArrayBorrow(source, &owner, mutable)
	} else {
		borrowed, ok = c.arrayFieldBorrow(source, owner, mutable)
	}
	if !ok {
		return hir.Expression{}, false
	}
	field.Children = []hir.Expression{borrowed}
	return projectFieldBorrow(field, mutable)
}

// Reuse exclusive-subview generations for field children. The generation freezes
// the whole parent and its aliases; field names never prove disjointness.
func (f *borrowFlowAnalyzer) evaluateFieldBorrowRoot(expression, root *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	value, reachable := f.evaluateExpression(root, environment)
	if expression.Text != "&mut" {
		if _, mutable, borrowed := hir.BorrowElement(f.analyzer.expressionTypes[root]); borrowed && mutable {
			f.checkReferenceAccess(root, value, false, environment, span(f.module.file.Path, expression.Span))
			value = f.retainSharedParent(root, span(f.module.file.Path, expression.Span), value)
		}
		return value, reachable
	}
	f.checkReferenceAccess(root, value, true, environment, span(f.module.file.Path, expression.Span))
	value = cloneBorrowValue(value)
	for _, origin := range value.origins {
		value.pending = append(value.pending, pendingBorrowLoan{subview: true,
			owner: f.analyzer.resolved[root], borrow: span(f.module.file.Path, expression.Span), kind: "mutable", origin: origin})
	}
	return value, reachable
}
