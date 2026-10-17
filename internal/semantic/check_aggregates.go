package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"strings"
)

func (c *typeChecker) checkRecord(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.TypeArguments) != 0 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "generic-aggregate-specialization")
		return hir.TypeInvalid
	}
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	if path := expression.Children[0]; path != nil && strings.HasPrefix(path.Text, ".") {
		return c.checkContextualEnumRecord(expression, result, expected)
	}
	constructor := c.analyzer.resolved[expression.Children[0]]
	if constructor != nil && constructor.typ == hir.TypeRegion {
		c.analyzer.addDiagnostic("E3104", span(c.module.file.Path, expression.Span))
		result.Symbol = constructor.id
		return hir.TypeInvalid
	}
	if constructor == nil || constructor.resource == nil && constructor.structure == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "record constructor that is not an aggregate")
		return hir.TypeInvalid
	}
	if constructor.resource != nil && (constructor.packageID != c.module.key.packageID || constructor.module != c.module.file.Module.Path) {
		c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, expression.Children[0].Span), RelatedSpan{
			Role: "declaration",
			Span: span(constructor.source, constructor.declaration),
		})
		return hir.TypeInvalid
	}
	fields := map[string]resourceField(nil)
	if info := c.analyzer.resourceTypes[constructor.typ]; info != nil {
		fields = info.fields
	} else if info := c.analyzer.structTypes[constructor.typ]; info != nil {
		fields = info.fields
	}
	if fields == nil {
		return hir.TypeInvalid
	}
	result.Symbol = constructor.id
	seen := make(map[string]struct{}, len(expression.Children)-1)
	for _, fieldExpression := range expression.Children[1:] {
		field, found := fields[fieldExpression.Text]
		if !found || len(fieldExpression.Children) != 1 {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed resource field "+fieldExpression.Text)
			continue
		}
		if _, duplicate := seen[fieldExpression.Text]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate resource field "+fieldExpression.Text)
			continue
		}
		seen[fieldExpression.Text] = struct{}{}
		value, actual := c.checkExpression(fieldExpression.Children[0], field.typ)
		result.Children = appendExpression(result.Children, value)
		result.Fields = append(result.Fields, fieldExpression.Text)
		c.requireType("E1006", span(c.module.file.Path, fieldExpression.Children[0].Span), actual, field.typ, field.declaration)
	}
	if len(seen) != len(fields) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "resource construction with missing fields")
	}
	return constructor.typ
}

func (c *typeChecker) checkContextualEnumPath(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	info := c.analyzer.enumTypes[expected]
	name := strings.TrimPrefix(expression.Text, ".")
	if info == nil || name == "" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "contextual enum variant without an expected enum type")
		return hir.TypeInvalid
	}
	variant, found := info.variants[name]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown contextual enum variant "+name)
		return hir.TypeInvalid
	}
	if len(variant.orderedFields) != 0 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "enum payload variant used without record construction")
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionEnumVariant
	result.Symbol = info.symbol.id
	result.Variant = name
	return expected
}

func (c *typeChecker) checkContextualEnumRecord(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	info := c.analyzer.enumTypes[expected]
	name := strings.TrimPrefix(expression.Children[0].Text, ".")
	if info == nil || name == "" {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "contextual enum construction without an expected enum type")
		return hir.TypeInvalid
	}
	variant, found := info.variants[name]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown contextual enum variant "+name)
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionEnumVariant
	result.Symbol = info.symbol.id
	result.Variant = name
	seen := make(map[string]struct{}, len(expression.Children)-1)
	for _, fieldExpression := range expression.Children[1:] {
		field, declared := variant.fields[fieldExpression.Text]
		if !declared || len(fieldExpression.Children) != 1 {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed enum payload field "+fieldExpression.Text)
			continue
		}
		if _, duplicate := seen[fieldExpression.Text]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate enum payload field "+fieldExpression.Text)
			continue
		}
		seen[fieldExpression.Text] = struct{}{}
		value, actual := c.checkExpression(fieldExpression.Children[0], field.typ)
		result.Children = appendExpression(result.Children, value)
		result.Fields = append(result.Fields, fieldExpression.Text)
		c.requireType("E1006", span(c.module.file.Path, fieldExpression.Children[0].Span), actual, field.typ, field.declaration)
	}
	if len(seen) != len(variant.fields) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "enum payload construction with missing fields")
	}
	return expected
}

func (c *typeChecker) checkField(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, receiver)
	if successType, errorType, isResult := hir.ResultElements(receiverType); isResult && expression.Text == "is_ok" {
		result.Kind = hir.ExpressionMatch
		for _, arm := range []struct {
			variant string
			value   string
		}{{"ok", "true"}, {"err", "false"}} {
			binding := c.analyzer.allocateSymbol()
			bindingSpan := span(c.module.file.Path, expression.Span)
			payloadType := successType
			if arm.variant == "err" {
				payloadType = errorType
			}
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding, Name: "_", Type: payloadType, Span: bindingSpan})
			result.Arms = append(result.Arms, hir.MatchArm{
				Variant: arm.variant, Binding: binding, BindingSpan: bindingSpan,
				Expression: hir.Expression{Kind: hir.ExpressionLiteral, Type: hir.TypeBool, Span: bindingSpan, Value: arm.value, Symbol: hir.NoSymbol},
			})
		}
		return hir.TypeBool
	}
	if element, _, borrow := hir.BorrowElement(receiverType); borrow && (element == hir.TypeStr || element == hir.TypeByteSlice || hir.NonByteIndexedView(receiverType)) {
		switch expression.Text {
		case "len":
			result.Kind = hir.ExpressionViewLength
			result.Operator = "len"
			return hir.TypeUSize
		case "is_empty":
			result.Kind = hir.ExpressionViewEmpty
			result.Operator = "is_empty"
			return hir.TypeBool
		default:
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown borrowed-view field "+expression.Text)
			return hir.TypeInvalid
		}
	}
	if element, length, array := hir.FixedArrayParts(receiverType); array && hir.ExecutableFixedArrayShape(element, length) {
		if expression.Text == "len" {
			result.Kind = hir.ExpressionArrayLength
			result.Operator = "len"
			return hir.TypeUSize
		}
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown fixed-array field "+expression.Text)
		return hir.TypeInvalid
	}
	if element, _, borrow := hir.BorrowElement(receiverType); borrow {
		receiverType = element
	}
	fields := map[string]resourceField(nil)
	if info, found := c.analyzer.resourceType(receiverType); found {
		owner := info.symbol
		if owner.packageID != c.module.key.packageID || owner.module != c.module.file.Module.Path {
			c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, expression.Span), RelatedSpan{
				Role: "declaration", Span: span(owner.source, owner.declaration),
			})
			return hir.TypeInvalid
		}
		fields = info.fields
	} else if info, found := c.analyzer.structType(receiverType); found {
		fields = info.fields
	}
	if fields == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "field access on a non-aggregate value")
		return hir.TypeInvalid
	}
	field, found := fields[expression.Text]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown resource field "+expression.Text)
		return hir.TypeInvalid
	}
	result.Operator = expression.Text
	return field.typ
}
