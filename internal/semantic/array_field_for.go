package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

// Keep the resolved field address and bounds in one explicit borrow. The common
// descriptor path holds its whole-owner loan through every written loop exit.
func (c *typeChecker) prepareArrayFieldFor(loop *ast.ByteForLoop, prepared, checked *ast.Expression, want hir.Type, mutable bool) string {
	const shape = "array-field iteration requires an exact scalar or plain struct item and a named plain struct owner; mutable items require mutable storage"
	if !sameVectorForView(loop.View, checked) {
		return shape
	}
	field := checked
	if field.Kind == syntax.KindSlice && len(field.Children) == 3 {
		field = field.Children[0]
	}
	root := arrayFieldRoot(field)
	owner := c.analyzer.resolved[root]
	if root == nil || owner == nil {
		return shape
	}
	value, typ := c.checkExpression(field, hir.TypeInvalid)
	if !c.analyzer.inlineArrayField(typ) {
		return shape
	}
	element, _, _ := hir.FixedArrayParts(typ)
	if element != want {
		c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, loop.View.Span))
		return shape
	}
	if _, ok := c.arrayFieldView(field, *value, mutable); !ok {
		if mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, loop.View.Span))
		}
		return shape
	}
	operator := "&"
	if mutable {
		operator = "&mut"
	}
	borrow := &ast.Expression{Kind: syntax.KindPrefix, Span: checked.Span, Text: operator, Children: []*ast.Expression{checked}}
	return c.prepareForDescriptor(loop, prepared, owner, borrow, mutable)
}
