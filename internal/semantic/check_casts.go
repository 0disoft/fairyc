package semantic

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) checkCast(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = "as"
	if len(expression.Children) != 1 || expression.Type == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed cast")
		return hir.TypeInvalid
	}
	targetType := c.analyzer.canonicalType(c.module, *expression.Type)
	result.TargetType = targetType
	child, sourceType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, child)
	if hir.ValuePreservingIntegerCast(sourceType, targetType) {
		result.Kind = hir.ExpressionScalarCast
		return targetType
	}
	sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(sourceType)
	targetElement, targetMutable, targetPointer := hir.RawPointerElement(targetType)
	sameElement := sourceElement == targetElement && (sourceMutable || !targetMutable)
	directLocalBorrow := child != nil && child.Kind == hir.ExpressionPrefix && (child.Operator == "&" || child.Operator == "&mut") &&
		len(child.Children) == 1 && child.Children[0].Kind == hir.ExpressionPath
	borrowValue := child != nil && child.Kind == hir.ExpressionPath
	opaqueElement := targetElement == hir.TypeVoid && (sourceMutable || !targetMutable) &&
		(directLocalBorrow || borrowValue)
	byteSliceBytes := sourceElement == hir.TypeByteSlice && targetElement == hir.TypeU8 && (sourceMutable || !targetMutable)
	stringBytes := sourceElement == hir.TypeStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	cStringBytes := sourceElement == hir.TypeCStr && !sourceMutable && targetElement == hir.TypeU8 && !targetMutable
	if !sourceBorrow || !targetPointer || !sameElement && !opaqueElement && !byteSliceBytes && !stringBytes && !cStringBytes {
		c.analyzer.addUnsupported(
			c.module.file.Path,
			expression.Span,
			fmt.Sprintf("cast from %s to %s", sourceType, targetType),
		)
		return hir.TypeInvalid
	}
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span))
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	return targetType
}

func (c *typeChecker) checkUnsafe(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) != 1 || expression.Children[0] == nil || expression.Children[0].Kind != syntax.KindBlock {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed unsafe block")
		return hir.TypeInvalid
	}
	scope := span(c.module.file.Path, expression.Span)
	c.unsafeScopes = append(c.unsafeScopes, scope)
	child, actual := c.checkExpression(expression.Children[0], expected)
	c.unsafeScopes = c.unsafeScopes[:len(c.unsafeScopes)-1]
	result.Children = appendExpression(result.Children, child)
	return actual
}
