package semantic

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkRepeatedArray(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed repeated fixed-array literal")
		return hir.TypeInvalid
	}
	count, countType := c.checkExpression(expression.Children[1], hir.TypeUSize)
	if countType == hir.TypeInvalid {
		return hir.TypeInvalid
	}
	length, validCount := parsePositiveArrayLength(count.Value)
	if count.Kind != hir.ExpressionLiteral || !isIntegerLiteral(count.Value) || !validCount || length > hir.MaxExecutableFixedArrayElements {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Children[1].Span,
			fmt.Sprintf("repeated fixed-array count outside constant integer range 1..=%d", hir.MaxExecutableFixedArrayElements))
		return hir.TypeInvalid
	}
	element, _, hasExpected := hir.FixedArrayParts(expected)
	if !hasExpected {
		element = hir.TypeInvalid
	}
	initializer, elementType := c.checkExpression(expression.Children[0], element)
	if elementType == hir.TypeInvalid {
		return hir.TypeInvalid
	}
	if hasExpected && elementType != element {
		c.requireType("E1006", initializer.Span, elementType, element, span(c.module.file.Path, expression.Span))
		return hir.TypeInvalid
	}
	if !hir.FixedArrayScalarElement(elementType) && !c.analyzer.plainViewStruct(elementType) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "fixed-array element type "+string(elementType))
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionArray
	result.Operator = "repeat"
	result.Children = appendExpression(result.Children, initializer)
	return hir.FixedArray(elementType, length)
}
