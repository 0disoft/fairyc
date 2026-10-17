package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkIf(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) < 2 {
		return hir.TypeInvalid
	}
	condition, conditionType := c.checkExpression(expression.Children[0], hir.TypeBool)
	result.Children = appendExpression(result.Children, condition)
	if conditionType != hir.TypeInvalid && conditionType != hir.TypeBool {
		c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
	}
	thenExpression, thenType := c.checkExpression(expression.Children[1], expected)
	result.Children = appendExpression(result.Children, thenExpression)
	if len(expression.Children) == 2 {
		return hir.TypeVoid
	}
	elseExpression, elseType := c.checkExpression(expression.Children[2], expected)
	result.Children = appendExpression(result.Children, elseExpression)
	if thenType == hir.TypeInvalid || elseType == hir.TypeInvalid {
		return hir.TypeInvalid
	}
	if thenType != elseType {
		c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[2].Span), RelatedSpan{
			Role: "expected-type",
			Span: span(c.module.file.Path, expression.Children[1].Span),
		})
		return hir.TypeInvalid
	}
	return thenType
}
