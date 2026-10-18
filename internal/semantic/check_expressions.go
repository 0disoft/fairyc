package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
	"strings"
)

func (c *typeChecker) checkExpression(expression *ast.Expression, expected hir.Type) (*hir.Expression, hir.Type) {
	if c.analyzer.canceled() {
		return &hir.Expression{Type: hir.TypeInvalid}, hir.TypeInvalid
	}
	if expression == nil {
		return nil, hir.TypeVoid
	}
	result := &hir.Expression{
		Kind:   expressionKind(expression.Kind),
		Span:   span(c.module.file.Path, expression.Span),
		Value:  expression.Text,
		Cooked: expression.Cooked,
		Symbol: hir.NoSymbol,
	}
	var actual hir.Type
	switch expression.Kind {
	case ast.PrivateSharedReborrow:
		child, sourceType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
		result.Children = appendExpression(result.Children, child)
		element, mutable, borrowed := hir.BorrowElement(sourceType)
		if !borrowed || !mutable {
			actual = hir.TypeInvalid
		} else {
			actual = hir.SharedBorrow(element)
		}
	case ast.PrivateStorageLayout:
		actual = c.checkStorageLayout(expression, result)
	case ast.PrivateStorageTake:
		actual = c.checkStorageTake(expression, result)
	case ast.PrivateStoragePut:
		actual = c.checkStoragePut(expression, result)
	case ast.PrivateStorageBorrow:
		actual = c.checkStorageBorrow(expression, result)
	case ast.PrivateStorageElementAddress:
		actual = c.checkStorageElementAddress(expression, result)
	case syntax.KindLiteral:
		var supported bool
		actual, supported = literalType(expression.Text, expected)
		if !supported {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "literal "+literalClass(expression.Text))
		} else if isIntegerLiteral(expression.Text) && !integerLiteralFits(expression.Text, actual) {
			c.analyzer.addDiagnostic("E1013", span(c.module.file.Path, expression.Span))
			actual = hir.TypeInvalid
		}
	case syntax.KindPath:
		if expression.Text == sourceLocationPath {
			c.analyzer.addDiagnostic("E4401", span(c.module.file.Path, expression.Span))
			actual = hir.TypeInvalid
		} else if symbol := c.analyzer.resolved[expression]; symbol != nil {
			if value, constant := c.analyzer.constantValues[symbol]; constant {
				result.Kind = hir.ExpressionLiteral
				result.Value = value.canonical()
				result.Cooked = ""
				actual = value.typ
			} else {
				result.Symbol = symbol.id
				actual = symbol.typ
			}
		} else if variant, option := optionVariantPath(expression.Text); option {
			actual = c.checkOptionPath(expression, result, expected, variant)
		} else if strings.HasPrefix(expression.Text, ".") {
			actual = c.checkContextualEnumPath(expression, result, expected)
		} else if isResultVariantPath(expression.Text) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result variant used without construction or pattern")
			actual = hir.TypeInvalid
		} else {
			actual = hir.TypeInvalid
		}
	case syntax.KindGroup:
		if len(expression.Children) == 1 {
			child, childType := c.checkExpression(expression.Children[0], expected)
			result.Children = appendExpression(result.Children, child)
			actual = childType
		} else {
			actual = hir.TypeInvalid
		}
	case syntax.KindPrefix:
		actual = c.checkPrefix(expression, result, expected)
	case syntax.KindBinary:
		actual = c.checkBinary(expression, result, expected)
	case syntax.KindCast:
		actual = c.checkCast(expression, result)
	case syntax.KindCall:
		actual = c.checkCall(expression, result, expected)
	case syntax.KindArray:
		actual = c.checkArray(expression, result, expected)
	case syntax.KindIndex:
		actual = c.checkIndex(expression, result)
	case syntax.KindSlice:
		actual = c.checkSlice(expression, result)
	case syntax.KindIf:
		actual = c.checkIf(expression, result, expected)
	case syntax.KindSwitch:
		actual = c.checkMatch(expression, result, expected)
		result.Kind = hir.ExpressionMatch
	case syntax.KindWhile:
		actual = c.checkWhile(expression, result)
	case syntax.KindRecord:
		actual = c.checkRecord(expression, result, expected)
	case syntax.KindFieldExpr:
		actual = c.checkField(expression, result)
	case syntax.KindBlock:
		if expression.Block == nil {
			actual = hir.TypeInvalid
			break
		}
		block, blockType, _ := c.checkBlock(expression.Block, expected)
		result.Block = &block
		actual = blockType
	case syntax.KindRegion:
		var nestedReturn bool
		result, nestedReturn = c.checkRegion(expression, expected)
		_ = nestedReturn
		actual = hir.TypeVoid
	case syntax.KindPropagate:
		actual = c.checkPropagate(expression, result)
	case syntax.KindUnsafe:
		actual = c.checkUnsafe(expression, result, expected)
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "expression "+string(expression.Kind))
		actual = hir.TypeInvalid
	}
	result.Type = actual
	c.analyzer.expressionTypes[expression] = actual
	if actual != hir.TypeInvalid {
		c.addType(result.Span, actual)
	}
	return result, actual
}

func appendExpression(values []hir.Expression, expressions ...*hir.Expression) []hir.Expression {
	for _, expression := range expressions {
		if expression != nil {
			values = append(values, *expression)
		}
	}
	return values
}
