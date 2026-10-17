package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) checkSlice(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 3 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed slice expression")
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	return c.checkSliceRange(expression, result, receiver, receiverType)
}

// Array-subview borrows supply a view of original storage; ordinary slices
// supply their already checked descriptor. Both share the same bounds rules.
func (c *typeChecker) checkSliceRange(expression *ast.Expression, result *hir.Expression, receiver *hir.Expression, receiverType hir.Type) hir.Type {
	start, startType := c.checkSliceBound(expression.Children[1], "start")
	end, endType := c.checkSliceBound(expression.Children[2], "end")
	result.Children = appendExpression(result.Children, receiver, start, end)
	if _, indexed := hir.IndexedViewElement(receiverType); !indexed {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "slicing outside executable array views")
		return hir.TypeInvalid
	}
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), startType, hir.TypeUSize, span(c.module.file.Path, expression.Span))
	c.requireType("E1006", span(c.module.file.Path, expression.Children[2].Span), endType, hir.TypeUSize, span(c.module.file.Path, expression.Span))
	if startType != hir.TypeUSize || endType != hir.TypeUSize {
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionViewSlice
	result.Operator = ".."
	return receiverType
}

func (c *typeChecker) checkSliceBound(expression *ast.Expression, position string) (*hir.Expression, hir.Type) {
	if expression != nil && expression.Kind == syntax.KindOmitBound {
		if expression.Text != position {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed omitted slice bound")
			return nil, hir.TypeInvalid
		}
		return &hir.Expression{
			Kind: hir.ExpressionOmittedRangeBound, Type: hir.TypeUSize, Span: span(c.module.file.Path, expression.Span),
			Operator: position, Symbol: hir.NoSymbol,
		}, hir.TypeUSize
	}
	return c.checkExpression(expression, hir.TypeUSize)
}
