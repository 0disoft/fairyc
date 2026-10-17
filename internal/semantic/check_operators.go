package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func (c *typeChecker) checkPrefix(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	if typ, handled := c.checkArraySubviewBorrow(expression, result); handled {
		return typ
	}
	if expression.Text == "-" && expected == hir.TypeI64 && directSignedMinimumLiteral(expression.Children[0], 64) {
		result.Kind = hir.ExpressionLiteral
		result.Value = "-9223372036854775808"
		result.Operator = ""
		return hir.TypeI64
	}
	if expression.Text == "-" && expected != hir.TypeI64 && directI32MinimumLiteral(expression.Children[0]) {
		result.Kind = hir.ExpressionLiteral
		result.Value = "-2147483648"
		result.Operator = ""
		return hir.TypeI32
	}
	childExpected := hir.TypeInvalid
	if (expression.Text == "+" || expression.Text == "-" || expression.Text == "~") && expected == hir.TypeI64 {
		childExpected = expected
	}
	if expression.Text == "~" && hir.BitwiseNotType(expected) {
		childExpected = expected
	}
	child, childType := c.checkExpression(expression.Children[0], childExpected)
	result.Children = appendExpression(result.Children, child)
	switch expression.Text {
	case "!":
		if childType != hir.TypeInvalid && childType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		return hir.TypeBool
	case "+", "-":
		if childType == hir.TypeI64 {
			return childType
		}
		c.requireType(
			"E1006",
			span(c.module.file.Path, expression.Children[0].Span),
			childType,
			hir.TypeI32,
			span(c.module.file.Path, expression.Span),
		)
		return hir.TypeI32
	case "~":
		if childType != hir.TypeInvalid && !hir.BitwiseNotType(childType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "bitwise not on "+childType.String())
		}
		return childType
	case "&":
		if borrowed, ok := c.indexedViewBorrow(expression.Children[0], child, false); ok {
			*result = borrowed
			result.Span = span(c.module.file.Path, expression.Span)
			return borrowed.Type
		}
		if child != nil {
			if field, ok := c.sharedFieldBorrow(expression.Children[0], *child); ok {
				if _, indexed := hir.IndexedViewElement(expected); indexed && c.analyzer.inlineArrayField(childType) {
					view, ok := c.arrayFieldView(expression.Children[0], *child, false)
					if !ok {
						c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "array field view requires a named plain struct owner")
						return hir.TypeInvalid
					}
					view.Span = result.Span
					*result = view
					return view.Type
				}
				if _, _, array := hir.FixedArrayParts(childType); array && !hir.ScalarArrayField(childType) {
					c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "nominal array field references require an unsized view")
					return hir.TypeInvalid
				}
				*result = field
				result.Span = span(c.module.file.Path, expression.Span)
				return field.Type
			}
		}
		if c.analyzer.resolved[expression.Children[0]] == nil {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "borrow of a non-place expression")
			return hir.TypeInvalid
		}
		if hir.ArrayViewBorrow(childType, expected, "&") {
			result.Kind = hir.ExpressionArrayViewBorrow
			return expected
		}
		return hir.SharedBorrow(childType)
	case "&mut":
		if borrowed, ok := c.indexedViewBorrow(expression.Children[0], child, true); ok {
			*result = borrowed
			result.Span = span(c.module.file.Path, expression.Span)
			return borrowed.Type
		}
		if field, ok := c.mutableFieldBorrow(expression.Children[0], child); ok {
			if _, indexed := hir.IndexedViewElement(expected); indexed && c.analyzer.inlineArrayField(childType) {
				view, ok := c.arrayFieldView(expression.Children[0], *child, true)
				if !ok {
					c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "array field view requires a named plain struct owner")
					return hir.TypeInvalid
				}
				view.Span = result.Span
				*result = view
				return view.Type
			}
			if _, _, array := hir.FixedArrayParts(childType); array && !hir.ScalarArrayField(childType) {
				c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "nominal array field references require an unsized view")
				return hir.TypeInvalid
			}
			*result = field
			result.Span = span(c.module.file.Path, expression.Span)
			return field.Type
		}
		target := c.analyzer.resolved[expression.Children[0]]
		if target == nil || !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, expression.Children[0].Span))
			return hir.TypeInvalid
		}
		if hir.ArrayViewBorrow(childType, expected, "&mut") {
			result.Kind = hir.ExpressionArrayViewBorrow
			return expected
		}
		return hir.MutableBorrow(childType)
	case "*":
		element, _, ok := hir.BorrowElement(childType)
		if !ok {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "dereference of a non-borrow value")
			return hir.TypeInvalid
		}
		return element
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "prefix operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkBinary(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	context := integerBinaryContext(expression.Text, expected)
	var left, right *hir.Expression
	var leftType, rightType hir.Type
	if contextualIntegerTree(expression.Children[0]) && !contextualIntegerTree(expression.Children[1]) && contextualIntegerOperator(expression.Text) {
		// Select a fixed peer type before defaulting the literal tree. HIR children
		// stay in source order, so lowering still evaluates left before right.
		right, rightType = c.checkExpression(expression.Children[1], context)
		left, leftType = c.checkExpression(expression.Children[0], integerPeerContext(rightType, context))
	} else {
		left, leftType = c.checkExpression(expression.Children[0], context)
		right, rightType = c.checkExpression(expression.Children[1], leftType)
	}
	result.Children = appendExpression(result.Children, left, right)
	switch expression.Text {
	case "&&", "||":
		if leftType != hir.TypeInvalid && leftType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		if rightType != hir.TypeInvalid && rightType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[1].Span))
		}
		return hir.TypeBool
	case "==", "!=":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
		} else if leftType != hir.TypeInvalid && !hir.ScalarEqualityType(leftType) {
			if _, _, pointer := hir.RawPointerElement(leftType); !pointer {
				c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "equality comparison on "+leftType.String())
			}
		}
		return hir.TypeBool
	case "<", "<=", ">", ">=":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
		} else if leftType != hir.TypeInvalid && !hir.ScalarOrderedType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "ordered comparison on "+leftType.String())
		}
		return hir.TypeBool
	case "<<":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedShiftType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked left shift on "+leftType.String())
		}
		return leftType
	case "+", "-", "*":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedArithmeticType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked arithmetic on "+leftType.String())
		}
		return leftType
	case "/", "%":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedDivisionType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked division on "+leftType.String())
		}
		return leftType
	case ">>":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.CheckedShiftType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "checked right shift on "+leftType.String())
		}
		return leftType
	case "&", "^", "|":
		if leftType != hir.TypeInvalid && rightType != hir.TypeInvalid && leftType != rightType {
			c.analyzer.addDiagnostic("E1006", span(c.module.file.Path, expression.Children[1].Span))
			return leftType
		}
		if leftType != hir.TypeInvalid && !hir.BitwiseIntegerType(leftType) {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "bitwise operation on "+leftType.String())
		}
		return leftType
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "binary operator "+expression.Text)
		return hir.TypeInvalid
	}
}
