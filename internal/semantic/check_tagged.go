package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
	"strings"
)

func (c *typeChecker) checkOptionPath(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantNone {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".some used without one payload")
		return hir.TypeInvalid
	}
	_, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".none without contextual Option type")
		return hir.TypeInvalid
	}
	result.Kind = hir.ExpressionOptionVariant
	result.Variant = optionVariantNone
	return expected
}

func (c *typeChecker) checkOptionConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantSome {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".none is a bare variant, not a call")
		return hir.TypeInvalid
	}
	payloadType, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option constructor without contextual Option type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, ".some arity other than one")
		return hir.TypeInvalid
	}
	payload, actual := c.checkExpression(expression.Children[1], payloadType)
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), actual, payloadType, span(c.module.file.Path, expression.Span))
	result.Kind = hir.ExpressionOptionVariant
	result.Variant = optionVariantSome
	result.Children = appendExpression(result.Children, payload)
	return expected
}

func (c *typeChecker) checkResultConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	success, failure, ok := hir.ResultElements(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result constructor without contextual Result type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Result constructor arity other than one")
		return hir.TypeInvalid
	}
	payloadType := success
	if variant == resultVariantErr {
		payloadType = failure
	}
	payload, actual := c.checkExpression(expression.Children[1], payloadType)
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), actual, payloadType, span(c.module.file.Path, expression.Span))
	result.Kind = hir.ExpressionResultVariant
	result.Variant = variant
	result.Children = appendExpression(result.Children, payload)
	return expected
}

func (c *typeChecker) checkMatch(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) < 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "match without arms")
		return hir.TypeInvalid
	}
	scrutinee, scrutineeType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, scrutinee)
	if _, _, ok := hir.ResultElements(scrutineeType); ok {
		if len(expression.Children) != 3 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "built-in Result match without exactly two arms")
			return hir.TypeInvalid
		}
		return c.checkResultMatch(expression, result, expected, scrutineeType)
	}
	if payloadType, ok := hir.OptionElement(scrutineeType); ok {
		if len(expression.Children) != 3 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "built-in Option match without exactly two arms")
			return hir.TypeInvalid
		}
		return c.checkOptionMatch(expression, result, expected, payloadType)
	}
	if info := c.analyzer.enumTypes[scrutineeType]; info != nil {
		return c.checkEnumMatch(expression, result, expected, info)
	}
	c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "match scrutinee outside the implemented Result, Option, and restricted enum subset")
	return hir.TypeInvalid
}

func (c *typeChecker) checkEnumMatch(expression *ast.Expression, result *hir.Expression, expected hir.Type, info *enumInfo) hir.Type {
	seen := make(map[string]struct{}, len(info.variants))
	actual := expected
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed enum match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		variantName := strings.TrimPrefix(pattern.Text, ".")
		variant, found := info.variants[variantName]
		if !found {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern from a different or unresolved enum")
			return hir.TypeInvalid
		}
		if _, duplicate := seen[variant.name]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate enum match variant")
			return hir.TypeInvalid
		}
		seen[variant.name] = struct{}{}
		arm := hir.MatchArm{Variant: variant.name, Binding: hir.NoSymbol}
		boundFields := make(map[string]struct{}, len(pattern.Children)-1)
		for _, fieldExpression := range pattern.Children[1:] {
			field, declared := variant.fields[fieldExpression.Text]
			binding := c.analyzer.resolved[fieldExpression]
			if !declared || binding == nil || fieldExpression.Cooked == "" {
				c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "unknown or malformed enum pattern field "+fieldExpression.Text)
				return hir.TypeInvalid
			}
			if _, duplicate := boundFields[fieldExpression.Text]; duplicate {
				c.analyzer.addUnsupported(c.module.file.Path, fieldExpression.Span, "duplicate enum pattern field "+fieldExpression.Text)
				return hir.TypeInvalid
			}
			boundFields[fieldExpression.Text] = struct{}{}
			binding.typ = field.typ
			bindingSpan := span(c.module.file.Path, fieldExpression.Span)
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: field.typ, Span: bindingSpan})
			c.addType(bindingSpan, field.typ)
			arm.Bindings = append(arm.Bindings, hir.MatchBinding{Field: field.name, Symbol: binding.id, Type: field.typ, Span: bindingSpan})
		}
		if len(boundFields) != len(variant.fields) {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "enum pattern with missing payload fields")
			return hir.TypeInvalid
		}
		body, bodyType := c.checkExpression(armExpression.Children[1], actual)
		if actual == hir.TypeInvalid {
			actual = bodyType
		} else {
			c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
		}
		arm.Expression = *body
		result.Arms = append(result.Arms, arm)
	}
	if len(seen) != len(info.variants) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive enum match")
		return hir.TypeInvalid
	}
	return actual
}

func (c *typeChecker) checkResultMatch(expression *ast.Expression, result *hir.Expression, expected, scrutineeType hir.Type) hir.Type {
	success, failure, ok := hir.ResultElements(scrutineeType)
	if !ok {
		return hir.TypeInvalid
	}
	seen := make(map[string]struct{}, 2)
	actual := expected
	continuing := 0
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed Result match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		if pattern == nil || pattern.Kind != syntax.KindPattern || len(pattern.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, armExpression.Span, "Result arm without one payload binding")
			return hir.TypeInvalid
		}
		variant, variantOK := resultVariantPath(pattern.Text)
		if !variantOK {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a contextual Result variant")
			return hir.TypeInvalid
		}
		if _, duplicate := seen[variant]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate Result match variant")
			return hir.TypeInvalid
		}
		seen[variant] = struct{}{}
		bindingExpression := pattern.Children[1]
		binding := c.analyzer.resolved[bindingExpression]
		if binding == nil {
			return hir.TypeInvalid
		}
		bindingType := success
		if variant == resultVariantErr {
			bindingType = failure
		}
		binding.typ = bindingType
		bindingSpan := span(c.module.file.Path, bindingExpression.Span)
		c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: bindingType, Span: bindingSpan})
		c.addType(bindingSpan, bindingType)
		armExpected := actual
		body, bodyType := c.checkExpression(armExpression.Children[1], armExpected)
		if !returnOnlyMatchArm(armExpression.Children[1]) && !terminalFalseAssertMatchArm(armExpression.Children[1]) {
			continuing++
			if actual == hir.TypeInvalid {
				actual = bodyType
			} else {
				c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
			}
		}
		result.Arms = append(result.Arms, hir.MatchArm{Variant: variant, Binding: binding.id, BindingSpan: bindingSpan, Expression: *body})
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Result match")
		return hir.TypeInvalid
	}
	if continuing == 0 {
		return hir.TypeVoid
	}
	return actual
}

func returnOnlyMatchArm(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindBlock || expression.Block == nil || expression.Block.Tail != nil {
		return false
	}
	statements := expression.Block.Statements
	return len(statements) > 0 && statements[len(statements)-1].Kind == syntax.KindReturn
}

func terminalFalseAssertMatchArm(expression *ast.Expression) bool {
	if expression == nil || expression.Kind != syntax.KindBlock || expression.Block == nil || expression.Block.Tail != nil {
		return false
	}
	statements := expression.Block.Statements
	if len(statements) == 0 {
		return false
	}
	last := &statements[len(statements)-1]
	return last.Kind == syntax.KindAssert && last.Expression != nil &&
		last.Expression.Kind == syntax.KindLiteral && last.Expression.Text == "false"
}

func (c *typeChecker) checkOptionMatch(expression *ast.Expression, result *hir.Expression, expected, payloadType hir.Type) hir.Type {
	seen := make(map[string]struct{}, 2)
	actual := expected
	continuing := 0
	for _, armExpression := range expression.Children[1:] {
		if armExpression == nil || armExpression.Kind != syntax.KindPattern || armExpression.Text != "arm" || len(armExpression.Children) != 2 {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "malformed Option match arm")
			return hir.TypeInvalid
		}
		pattern := armExpression.Children[0]
		if pattern == nil || pattern.Kind != syntax.KindPattern {
			c.analyzer.addUnsupported(c.module.file.Path, armExpression.Span, "malformed Option pattern")
			return hir.TypeInvalid
		}
		variant, variantOK := optionVariantPath(pattern.Text)
		if !variantOK {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a contextual Option variant")
			return hir.TypeInvalid
		}
		if _, duplicate := seen[variant]; duplicate {
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "duplicate Option match variant")
			return hir.TypeInvalid
		}
		seen[variant] = struct{}{}
		arm := hir.MatchArm{Variant: variant, Binding: hir.NoSymbol}
		switch variant {
		case optionVariantSome:
			if len(pattern.Children) != 2 {
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, ".some pattern without one payload binding")
				return hir.TypeInvalid
			}
			bindingExpression := pattern.Children[1]
			binding := c.analyzer.resolved[bindingExpression]
			if binding == nil {
				return hir.TypeInvalid
			}
			binding.typ = payloadType
			bindingSpan := span(c.module.file.Path, bindingExpression.Span)
			c.hir.Locals = append(c.hir.Locals, hir.Local{Symbol: binding.id, Name: binding.name, Type: payloadType, Span: bindingSpan})
			c.addType(bindingSpan, payloadType)
			arm.Binding = binding.id
			arm.BindingSpan = bindingSpan
		case optionVariantNone:
			if len(pattern.Children) != 1 {
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, ".none pattern with a payload binding")
				return hir.TypeInvalid
			}
		}
		body, bodyType := c.checkExpression(armExpression.Children[1], actual)
		if !terminalFalseAssertMatchArm(armExpression.Children[1]) {
			continuing++
			if actual == hir.TypeInvalid {
				actual = bodyType
			} else {
				c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
			}
		}
		arm.Expression = *body
		result.Arms = append(result.Arms, arm)
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Option match")
		return hir.TypeInvalid
	}
	if continuing == 0 {
		return hir.TypeVoid
	}
	return actual
}
