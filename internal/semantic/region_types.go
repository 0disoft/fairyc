package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func isRegionPlaceCall(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return false
	}
	callee := expression.Children[0]
	return callee.Kind == syntax.KindFieldExpr && callee.Text == "place" && len(callee.Children) == 1
}

func (c *typeChecker) checkRegionPlace(expression *ast.Expression, result *hir.Expression) hir.Type {
	callee := expression.Children[0]
	receiver, receiverType := c.checkExpression(callee.Children[0], hir.TypeRegion)
	result.Operator = "place"
	result.Children = appendExpression(result.Children, receiver)
	if receiverType != hir.TypeInvalid && receiverType != hir.TypeRegion {
		c.analyzer.addUnsupported(c.module.file.Path, callee.Span, "place on a non-Region receiver")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Region.place argument count")
		return hir.TypeInvalid
	}
	value, valueType := c.checkExpression(expression.Children[1], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, value)
	if valueType == hir.TypeInvalid {
		return hir.TypeInvalid
	}
	return hir.SharedBorrow(valueType)
}

func (c *typeChecker) checkRegion(expression *ast.Expression, expectedTail hir.Type) (*hir.Expression, bool) {
	result := &hir.Expression{Kind: string(syntax.KindRegion), Type: hir.TypeVoid, Span: span(c.module.file.Path, expression.Span), Symbol: hir.NoSymbol, Operator: expression.Text}
	if len(expression.Children) != 2 {
		return result, false
	}
	if region := c.analyzer.regions[expression]; region != nil {
		region.typ = hir.TypeRegion
		result.Symbol = region.id
	}
	arena, arenaType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, arena)
	if !c.analyzer.isArenaBorrow(arenaType) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "region source that is not &mut std::alloc::Arena")
	}
	body := expression.Children[1]
	if body.Block == nil {
		return result, false
	}
	block, _, hasReturn := c.checkBlock(body.Block, expectedTail)
	result.Block = &block
	return result, hasReturn
}

func (a *analyzer) isArenaBorrow(value hir.Type) bool {
	element, mutable, ok := hir.BorrowElement(value)
	if !ok || !mutable {
		return false
	}
	info := a.resourceTypes[element]
	return info != nil && info.symbol != nil && info.symbol.module == "std::alloc" && info.symbol.name == "Arena"
}
