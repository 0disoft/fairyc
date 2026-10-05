package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Grouping preserves named array and array-field storage identity. Calls and
// temporary arrays cannot become places through slicing.
func arraySubviewSelection(expression *ast.Expression) *ast.Expression {
	for expression != nil && expression.Kind == syntax.KindGroup && len(expression.Children) == 1 {
		expression = expression.Children[0]
	}
	if expression != nil && expression.Kind == syntax.KindSlice && len(expression.Children) == 3 && (assignmentRootPath(expression.Children[0]) != nil || arrayFieldRoot(expression.Children[0]) != nil) {
		return expression
	}
	return nil
}

// Explicit ref/ref mut of an array range produces an existing view-slice HIR
// over an array-view-borrow, never a copied array or an address of a temporary.
func (c *typeChecker) checkArraySubviewBorrow(expression *ast.Expression, result *hir.Expression) (hir.Type, bool) {
	if !isBorrowExpression(expression) {
		return hir.TypeInvalid, false
	}
	selected := arraySubviewSelection(expression.Children[0])
	if selected == nil {
		return hir.TypeInvalid, false
	}
	if arrayFieldRoot(selected.Children[0]) != nil {
		field, typ := c.checkExpression(selected.Children[0], hir.TypeInvalid)
		if !c.analyzer.inlineArrayField(typ) {
			return hir.TypeInvalid, false
		}
		view, ok := c.arrayFieldView(selected.Children[0], *field, expression.Text == "&mut")
		if !ok {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, selected.Children[0].Span))
			return hir.TypeInvalid, true
		}
		actual := c.checkSliceRange(selected, result, &view, view.Type)
		c.analyzer.expressionTypes[selected] = actual
		return actual, true
	}
	owner := c.analyzer.resolved[assignmentRootPath(selected.Children[0])]
	if owner == nil || (owner.kind != "local" && owner.kind != "parameter") {
		return hir.TypeInvalid, false
	}
	element, length, array := hir.FixedArrayParts(owner.typ)
	if !array {
		return hir.TypeInvalid, false
	}
	if !hir.ExecutableFixedArrayShape(element, length) || (!hir.FixedArrayScalarElement(element) && !c.analyzer.plainViewStruct(element)) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "array subview borrow requires an admitted scalar or plain struct array")
		return hir.TypeInvalid, true
	}
	mutable := expression.Text == "&mut"
	if mutable && !owner.mutable {
		c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, selected.Children[0].Span))
		return hir.TypeInvalid, true
	}
	root, _ := c.checkExpression(selected.Children[0], hir.TypeInvalid)
	view := hir.SharedBorrow(hir.Type("[" + string(element) + "]"))
	if mutable {
		view = hir.MutableBorrow(hir.Type("[" + string(element) + "]"))
	}
	borrow := &hir.Expression{Kind: "array-view-borrow", Type: view, Span: root.Span, Symbol: hir.NoSymbol,
		Operator: expression.Text, Children: []hir.Expression{*root}}
	actual := c.checkSliceRange(selected, result, borrow, view)
	c.analyzer.expressionTypes[selected] = actual
	return actual, true
}
