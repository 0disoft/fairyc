package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// sharedFieldBorrow projects an address, never a copied aggregate temporary.
// Every projection retains the complete root owner; fields do not split loans.
func sharedFieldBorrow(field hir.Expression) (hir.Expression, bool) {
	if field.Kind == "group" && len(field.Children) == 1 {
		return sharedFieldBorrow(field.Children[0])
	}
	if field.Kind != "field-expression" || len(field.Children) != 1 || field.Type == hir.TypeInvalid {
		return hir.Expression{}, false
	}
	owner := field.Children[0]
	for owner.Kind == "group" && len(owner.Children) == 1 {
		owner = owner.Children[0]
	}
	if _, mutable, borrowed := hir.BorrowElement(owner.Type); mutable {
		return hir.Expression{}, false
	} else if !borrowed {
		if owner.Kind == "path" && owner.Symbol != hir.NoSymbol {
			owner = hir.Expression{Kind: "prefix", Type: hir.SharedBorrow(owner.Type),
				Span: owner.Span, Symbol: hir.NoSymbol, Operator: "&", Children: []hir.Expression{owner}}
		} else {
			var ok bool
			owner, ok = sharedFieldBorrow(owner)
			if !ok {
				return hir.Expression{}, false
			}
		}
	}
	return hir.Expression{Kind: "field-borrow", Type: hir.SharedBorrow(field.Type),
		Span: field.Span, Symbol: hir.NoSymbol, Operator: field.Operator, Children: []hir.Expression{owner}}, true
}

func borrowPlaceRoot(expression *ast.Expression) *ast.Expression {
	for expression != nil && len(expression.Children) == 1 &&
		(expression.Kind == syntax.KindFieldExpr || expression.Kind == syntax.KindGroup) {
		expression = expression.Children[0]
	}
	return expression
}
