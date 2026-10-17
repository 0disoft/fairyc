package semantic

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkArray(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if expression.Text == "repeat" {
		return c.checkRepeatedArray(expression, result, expected)
	}
	if expression.Text != "list" || len(expression.Children) == 0 {
		feature := "empty fixed-array literal"
		if expression.Text == "repeat" {
			feature = "repeated fixed-array literal"
		} else if expression.Text != "empty" {
			feature = "malformed fixed-array literal"
		}
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, feature)
		return hir.TypeInvalid
	}
	expectedElement, _, hasExpectedArray := hir.FixedArrayParts(expected)
	elementType := expectedElement
	valid := true
	for index := range expression.Children {
		childExpected := elementType
		if index == 0 && !hasExpectedArray {
			childExpected = hir.TypeInvalid
		}
		child, childType := c.checkExpression(expression.Children[index], childExpected)
		result.Children = appendExpression(result.Children, child)
		if index == 0 && !hasExpectedArray {
			elementType = childType
		}
		if childType == hir.TypeInvalid || elementType == hir.TypeInvalid {
			valid = false
			continue
		}
		if childType != elementType {
			c.requireType(
				"E1006",
				span(c.module.file.Path, expression.Children[index].Span),
				childType,
				elementType,
				span(c.module.file.Path, expression.Span),
			)
			valid = false
		}
	}
	if !valid {
		return hir.TypeInvalid
	}
	if !hir.FixedArrayScalarElement(elementType) && !c.analyzer.plainViewStruct(elementType) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "fixed-array element type "+elementType.String())
		return hir.TypeInvalid
	}
	length := uint64(len(expression.Children))
	if !hir.ExecutableFixedArrayShape(elementType, length) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, fmt.Sprintf("fixed-array literal with %d elements", length))
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionArray
	result.Operator = "list"
	return hir.FixedArray(elementType, length)
}

func (c *typeChecker) checkIndex(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed index expression")
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	index, indexType := c.checkExpression(expression.Children[1], hir.TypeUSize)
	result.Children = appendExpression(result.Children, receiver, index)
	arrayElement, _, fixedArray := hir.FixedArrayParts(receiverType)
	viewElement, indexedView := hir.IndexedViewElement(receiverType)
	if !indexedView && !fixedArray {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "indexing outside executable array views")
		return hir.TypeInvalid
	}
	c.requireType(
		"E1006",
		span(c.module.file.Path, expression.Children[1].Span),
		indexType,
		hir.TypeUSize,
		span(c.module.file.Path, expression.Span),
	)
	if indexType == hir.TypeInvalid || indexType != hir.TypeUSize {
		return hir.TypeInvalid
	}
	if fixedArray {
		if view, ok := c.arrayFieldView(expression.Children[0], *receiver, false); ok {
			result.Kind, result.Operator = hir.ExpressionViewIndex, "[]"
			result.Children[0] = view
			return arrayElement
		}
		result.Kind = hir.ExpressionArrayIndex
		result.Operator = "[]"
		return arrayElement
	}
	result.Kind = hir.ExpressionViewIndex
	result.Operator = "[]"
	return viewElement
}
