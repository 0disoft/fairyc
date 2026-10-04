package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// sharedFieldBorrow projects an address, never a copied aggregate temporary.
// Every projection retains the complete root owner; fields do not split loans.
func sharedFieldBorrow(field hir.Expression) (hir.Expression, bool) {
	return projectFieldBorrow(field, false)
}

func projectFieldBorrow(field hir.Expression, mutable bool) (hir.Expression, bool) {
	if field.Kind == "group" && len(field.Children) == 1 {
		return projectFieldBorrow(field.Children[0], mutable)
	}
	if field.Kind != "field-expression" || len(field.Children) != 1 || field.Type == hir.TypeInvalid {
		return hir.Expression{}, false
	}
	owner := field.Children[0]
	for owner.Kind == "group" && len(owner.Children) == 1 {
		owner = owner.Children[0]
	}
	if _, ownerMutable, borrowed := hir.BorrowElement(owner.Type); borrowed && ownerMutable != mutable {
		return hir.Expression{}, false
	} else if !borrowed {
		if owner.Kind == "path" && owner.Symbol != hir.NoSymbol {
			borrowType, operator := hir.SharedBorrow(owner.Type), "&"
			if mutable {
				borrowType, operator = hir.MutableBorrow(owner.Type), "&mut"
			}
			owner = hir.Expression{Kind: "prefix", Type: borrowType,
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
	return hir.Expression{Kind: "field-borrow", Type: borrowType,
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

// Reuse exclusive-subview generations for field children. The generation freezes
// the whole parent and its aliases; field names never prove disjointness.
func (f *borrowFlowAnalyzer) evaluateFieldBorrowRoot(expression, root *ast.Expression, environment borrowEnvironment) (borrowValueState, bool) {
	value, reachable := f.evaluateExpression(root, environment)
	if expression.Text != "&mut" {
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
