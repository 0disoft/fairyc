package semantic

import (
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type typeChecker struct {
	analyzer *analyzer
	module   *moduleScope
	function *ast.Function
	hir      *hir.Function
}

func (a *analyzer) typeCheck() Result {
	result := Result{}
	a.prepareAggregateIdentities()
	a.prepareResourceTypes()
	a.prepareStructTypes()
	for _, module := range a.moduleOrder {
		hirModule := hir.Module{Path: module.file.Module.Path, SourcePath: module.file.Path}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if a.functions[function] == nil {
				continue
			}
			checker := typeChecker{analyzer: a, module: module, function: function}
			hirModule.Functions = append(hirModule.Functions, checker.checkFunction())
		}
		result.HIR.Modules = append(result.HIR.Modules, hirModule)
	}
	return result
}

func (c *typeChecker) checkFunction() hir.Function {
	functionSymbol := c.analyzer.functions[c.function]
	resultType := c.canonicalType(c.function.ResultType)
	functionSymbol.typ = resultType
	if errorSpan, ok := resultFailureSpan(c.function.ResultType); ok {
		functionSymbol.resultError = errorSpan
	}
	result := hir.Function{
		Symbol:     functionSymbol.id,
		Name:       c.function.Name,
		Span:       span(c.module.file.Path, c.function.Span),
		ResultType: resultType,
	}
	c.hir = &result
	for index := range c.function.Parameters {
		parameter := &c.function.Parameters[index]
		symbol := c.analyzer.parameter[parameter]
		if symbol == nil {
			continue
		}
		symbol.typ = c.canonicalType(parameter.Type)
		if errorSpan, ok := resultFailureSpan(parameter.Type); ok {
			symbol.resultError = errorSpan
		}
		parameterSpan := span(c.module.file.Path, parameter.NameSpan)
		result.Parameters = append(result.Parameters, hir.Parameter{
			Symbol: symbol.id,
			Name:   parameter.Name,
			Type:   symbol.typ,
			Span:   parameterSpan,
		})
		c.addType(parameterSpan, symbol.typ)
	}
	if c.function.Body == nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "function without a body")
		return result
	}
	body, bodyType, hasReturn := c.checkBlock(c.function.Body, resultType)
	result.Body = body
	if c.function.Body.Tail != nil {
		c.requireType(
			"E1009",
			span(c.module.file.Path, c.function.Body.Tail.Span),
			bodyType,
			resultType,
			span(c.module.file.Path, c.function.ResultType.Span),
		)
	} else if resultType != hir.TypeVoid && !hasReturn {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Body.Span, "non-void function without a checked return value")
	}
	return result
}

func (c *typeChecker) checkBlock(block *ast.Block, expectedTail hir.Type) (hir.Block, hir.Type, bool) {
	result := hir.Block{Span: span(c.module.file.Path, block.Span)}
	hasReturn := false
	for index := range block.Statements {
		statement := &block.Statements[index]
		hirStatement := hir.Statement{Kind: string(statement.Kind), Span: span(c.module.file.Path, statement.Span), Binding: hir.NoSymbol}
		switch {
		case statement.Binding != nil:
			binding := statement.Binding
			symbol := c.analyzer.bindingSymbols[binding]
			if symbol == nil {
				continue
			}
			expected := hir.TypeInvalid
			if binding.Type != nil {
				expected = c.canonicalType(*binding.Type)
			}
			expression, actual := c.checkExpression(binding.Expression, expected)
			if expected != hir.TypeInvalid {
				c.requireType(
					"E1006",
					span(c.module.file.Path, binding.Expression.Span),
					actual,
					expected,
					span(c.module.file.Path, binding.Type.Span),
				)
				symbol.typ = expected
			} else {
				symbol.typ = actual
			}
			hirStatement.Binding = symbol.id
			hirStatement.Expression = expression
			localSpan := span(c.module.file.Path, binding.NameSpan)
			c.hir.Locals = append(c.hir.Locals, hir.Local{
				Symbol:  symbol.id,
				Name:    binding.Name,
				Type:    symbol.typ,
				Mutable: binding.Mutable,
				Span:    localSpan,
			})
			c.addType(localSpan, symbol.typ)
		case statement.Kind == syntax.KindReturn:
			hasReturn = true
			expression, actual := c.checkExpression(statement.Expression, expectedTail)
			hirStatement.Expression = expression
			if statement.Expression == nil {
				actual = hir.TypeVoid
			}
			primary := statement.Span
			if statement.Expression != nil {
				primary = statement.Expression.Span
			}
			c.requireType(
				"E1009",
				span(c.module.file.Path, primary),
				actual,
				expectedTail,
				span(c.module.file.Path, c.function.ResultType.Span),
			)
		case statement.Kind == syntax.KindAssignment:
			target, targetType := c.checkExpression(statement.Target, hir.TypeInvalid)
			value, actual := c.checkExpression(statement.Expression, targetType)
			hirStatement.Target = target
			hirStatement.Expression = value
			if statement.Target != nil {
				resolved := c.analyzer.resolved[statement.Target]
				if resolved == nil || !resolved.mutable {
					c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, statement.Target.Span))
				}
			}
			if statement.Expression != nil {
				c.requireType(
					"E1006",
					span(c.module.file.Path, statement.Expression.Span),
					actual,
					targetType,
					span(c.module.file.Path, statement.Target.Span),
				)
			}
		case statement.Kind == syntax.KindExprStmt || statement.Kind == syntax.KindDiscard:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeInvalid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindWhile:
			expression, _ := c.checkExpression(statement.Expression, hir.TypeVoid)
			hirStatement.Expression = expression
		case statement.Kind == syntax.KindRegion:
			expression, nestedReturn := c.checkRegion(statement.Expression, expectedTail)
			hirStatement.Expression = expression
			hasReturn = hasReturn || nestedReturn
		default:
			c.analyzer.addUnsupported(c.module.file.Path, statement.Span, "statement "+string(statement.Kind))
		}
		result.Statements = append(result.Statements, hirStatement)
	}
	resultType := hir.TypeVoid
	if block.Tail != nil {
		result.Tail, resultType = c.checkExpression(block.Tail, expectedTail)
	}
	return result, resultType, hasReturn
}

func (c *typeChecker) checkExpression(expression *ast.Expression, expected hir.Type) (*hir.Expression, hir.Type) {
	if expression == nil {
		return nil, hir.TypeVoid
	}
	result := &hir.Expression{
		Kind:   string(expression.Kind),
		Span:   span(c.module.file.Path, expression.Span),
		Value:  expression.Text,
		Symbol: hir.NoSymbol,
	}
	var actual hir.Type
	switch expression.Kind {
	case syntax.KindLiteral:
		var supported bool
		actual, supported = literalType(expression.Text)
		if !supported {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "literal "+literalClass(expression.Text))
		}
	case syntax.KindPath:
		if symbol := c.analyzer.resolved[expression]; symbol != nil {
			result.Symbol = symbol.id
			actual = symbol.typ
		} else if variant, option := optionVariantPath(expression.Text); option {
			actual = c.checkOptionPath(expression, result, expected, variant)
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
		actual = c.checkPrefix(expression, result)
	case syntax.KindBinary:
		actual = c.checkBinary(expression, result)
	case syntax.KindCall:
		actual = c.checkCall(expression, result, expected)
	case syntax.KindIf:
		actual = c.checkIf(expression, result, expected)
	case syntax.KindMatch:
		actual = c.checkMatch(expression, result, expected)
	case syntax.KindWhile:
		actual = c.checkWhile(expression, result)
	case syntax.KindRecord:
		actual = c.checkRecord(expression, result)
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

func (c *typeChecker) checkWhile(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	condition, conditionType := c.checkExpression(expression.Children[0], hir.TypeBool)
	result.Children = appendExpression(result.Children, condition)
	if conditionType != hir.TypeInvalid && conditionType != hir.TypeBool {
		c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
	}
	body, _ := c.checkExpression(expression.Children[1], hir.TypeVoid)
	result.Children = appendExpression(result.Children, body)
	return hir.TypeVoid
}

func (c *typeChecker) checkRecord(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	constructor := c.analyzer.resolved[expression.Children[0]]
	if constructor != nil && constructor.typ == hir.TypeRegion {
		c.analyzer.addDiagnostic("E3104", span(c.module.file.Path, expression.Span))
		result.Symbol = constructor.id
		return hir.TypeInvalid
	}
	if constructor == nil || constructor.resource == nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "record constructor that is not a resource")
		return hir.TypeInvalid
	}
	info := c.analyzer.resourceTypes[constructor.typ]
	if info == nil {
		return hir.TypeInvalid
	}
	result.Symbol = constructor.id
	seen := make(map[string]struct{}, len(expression.Children)-1)
	for _, fieldExpression := range expression.Children[1:] {
		field, found := info.fields[fieldExpression.Text]
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
	if len(seen) != len(info.fields) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "resource construction with missing fields")
	}
	return constructor.typ
}

func (c *typeChecker) checkField(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiver, receiverType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, receiver)
	if element, _, borrow := hir.BorrowElement(receiverType); borrow {
		receiverType = element
	}
	info, found := c.analyzer.resourceType(receiverType)
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "field access on a non-resource value")
		return hir.TypeInvalid
	}
	field, found := info.fields[expression.Text]
	if !found {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "unknown resource field "+expression.Text)
		return hir.TypeInvalid
	}
	result.Operator = expression.Text
	return field.typ
}

func (c *typeChecker) checkPrefix(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 1 {
		return hir.TypeInvalid
	}
	child, childType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, child)
	switch expression.Text {
	case "!":
		if childType != hir.TypeInvalid && childType != hir.TypeBool {
			c.analyzer.addDiagnostic("E1007", span(c.module.file.Path, expression.Children[0].Span))
		}
		return hir.TypeBool
	case "+", "-", "~":
		c.requireType(
			"E1006",
			span(c.module.file.Path, expression.Children[0].Span),
			childType,
			hir.TypeI32,
			span(c.module.file.Path, expression.Span),
		)
		return hir.TypeI32
	case "&":
		if c.analyzer.resolved[expression.Children[0]] == nil {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "borrow of a non-place expression")
			return hir.TypeInvalid
		}
		return hir.SharedBorrow(childType)
	case "&mut":
		target := c.analyzer.resolved[expression.Children[0]]
		if target == nil || !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, expression.Children[0].Span))
			return hir.TypeInvalid
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

func (c *typeChecker) checkBinary(expression *ast.Expression, result *hir.Expression) hir.Type {
	result.Operator = expression.Text
	if len(expression.Children) != 2 {
		return hir.TypeInvalid
	}
	left, leftType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	right, rightType := c.checkExpression(expression.Children[1], leftType)
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
		}
		return hir.TypeBool
	case "<", "<=", ">", ">=":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		return hir.TypeBool
	case "<<":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.analyzer.addDiagnostic("E1011", span(c.module.file.Path, expression.Span))
		return hir.TypeI32
	case "+", "-", "*", "/", "%", ">>", "&", "^", "|":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		return hir.TypeI32
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "binary operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkCall(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	calleeExpression := expression.Children[0]
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindPath {
		if variant, ok := resultVariantPath(calleeExpression.Text); ok {
			return c.checkResultConstructor(expression, result, expected, variant)
		}
		if variant, ok := optionVariantPath(calleeExpression.Text); ok {
			return c.checkOptionConstructor(expression, result, expected, variant)
		}
	}
	if isRegionPlaceCall(expression) {
		return c.checkRegionPlace(expression, result)
	}
	callee := c.analyzer.resolved[calleeExpression]
	if callee == nil || callee.kind != "function" || callee.function == nil {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target that is not a function")
		return hir.TypeInvalid
	}
	result.Symbol = callee.id
	result.Children = append(result.Children, hir.Expression{
		Kind:   string(calleeExpression.Kind),
		Type:   hir.TypeInvalid,
		Span:   span(c.module.file.Path, calleeExpression.Span),
		Symbol: callee.id,
	})
	arguments := expression.Children[1:]
	if len(arguments) != len(callee.function.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration",
			Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		expected := hir.TypeInvalid
		var expectedSpan = callee.typeSpan
		if index < len(callee.function.Parameters) {
			expected = c.analyzer.canonicalType(c.analyzer.modules[callee.module], callee.function.Parameters[index].Type)
			expectedSpan = callee.function.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireType(
				"E1006",
				span(c.module.file.Path, argument.Span),
				actual,
				expected,
				span(callee.source, expectedSpan),
			)
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.modules[callee.module], callee.function.ResultType)
	if errorSpan, ok := resultFailureSpan(callee.function.ResultType); ok {
		callee.resultError = errorSpan
	}
	return callee.typ
}

func (c *typeChecker) checkOptionPath(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantNone {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::Some used without one payload")
		return hir.TypeInvalid
	}
	_, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::None without contextual Option type")
		return hir.TypeInvalid
	}
	result.Kind = "option-variant"
	result.Variant = optionVariantNone
	return expected
}

func (c *typeChecker) checkOptionConstructor(expression *ast.Expression, result *hir.Expression, expected hir.Type, variant string) hir.Type {
	if variant != optionVariantSome {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::None is a bare variant, not a call")
		return hir.TypeInvalid
	}
	payloadType, ok := hir.OptionElement(expected)
	if !ok {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option constructor without contextual Option type")
		return hir.TypeInvalid
	}
	if len(expression.Children) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "Option::Some arity other than one")
		return hir.TypeInvalid
	}
	payload, actual := c.checkExpression(expression.Children[1], payloadType)
	c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), actual, payloadType, span(c.module.file.Path, expression.Span))
	result.Kind = "option-variant"
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
	result.Kind = "result-variant"
	result.Variant = variant
	result.Children = appendExpression(result.Children, payload)
	return expected
}

func (c *typeChecker) checkMatch(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if len(expression.Children) != 3 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "built-in match without exactly two arms")
		return hir.TypeInvalid
	}
	scrutinee, scrutineeType := c.checkExpression(expression.Children[0], hir.TypeInvalid)
	result.Children = appendExpression(result.Children, scrutinee)
	if _, _, ok := hir.ResultElements(scrutineeType); ok {
		return c.checkResultMatch(expression, result, expected, scrutineeType)
	}
	if payloadType, ok := hir.OptionElement(scrutineeType); ok {
		return c.checkOptionMatch(expression, result, expected, payloadType)
	}
	c.analyzer.addUnsupported(c.module.file.Path, expression.Children[0].Span, "match scrutinee outside the implemented Result and Option subset")
	return hir.TypeInvalid
}

func (c *typeChecker) checkResultMatch(expression *ast.Expression, result *hir.Expression, expected, scrutineeType hir.Type) hir.Type {
	success, failure, ok := hir.ResultElements(scrutineeType)
	if !ok {
		return hir.TypeInvalid
	}
	seen := make(map[string]struct{}, 2)
	actual := expected
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
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a qualified Result variant")
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
		if actual == hir.TypeInvalid {
			actual = bodyType
		} else {
			c.requireType("E1006", span(c.module.file.Path, armExpression.Children[1].Span), bodyType, actual, span(c.module.file.Path, expression.Span))
		}
		result.Arms = append(result.Arms, hir.MatchArm{Variant: variant, Binding: binding.id, BindingSpan: bindingSpan, Expression: *body})
	}
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Result match")
		return hir.TypeInvalid
	}
	return actual
}

func (c *typeChecker) checkOptionMatch(expression *ast.Expression, result *hir.Expression, expected, payloadType hir.Type) hir.Type {
	seen := make(map[string]struct{}, 2)
	actual := expected
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
			c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "pattern that is not a qualified Option variant")
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
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "Option::Some pattern without one payload binding")
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
				c.analyzer.addUnsupported(c.module.file.Path, pattern.Span, "Option::None pattern with a payload binding")
				return hir.TypeInvalid
			}
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
	if len(seen) != 2 {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "non-exhaustive Option match")
		return hir.TypeInvalid
	}
	return actual
}

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

func (c *typeChecker) canonicalType(reference ast.TypeReference) hir.Type {
	return c.analyzer.canonicalType(c.module, reference)
}

func (c *typeChecker) requireType(code string, primary hir.Span, actual, expected hir.Type, expectedSpan hir.Span) {
	if actual == hir.TypeInvalid || expected == hir.TypeInvalid || actual == expected {
		return
	}
	c.analyzer.addDiagnostic(code, primary, RelatedSpan{
		Role: "expected-type",
		Span: expectedSpan,
	})
}

func (c *typeChecker) addType(expression hir.Span, value hir.Type) {
	if value == hir.TypeInvalid {
		return
	}
	c.analyzer.types = append(c.analyzer.types, TypeFact{Expression: expression, CanonicalType: value})
}

func literalType(raw string) (hir.Type, bool) {
	if raw == "true" || raw == "false" {
		return hir.TypeBool, true
	}
	if raw != "" && raw[0] >= '0' && raw[0] <= '9' && !strings.ContainsAny(raw, ".eE") {
		return hir.TypeI32, true
	}
	return hir.TypeInvalid, false
}

func literalClass(raw string) string {
	switch {
	case strings.HasPrefix(raw, "\""):
		return "string"
	case strings.HasPrefix(raw, "b\""):
		return "byte-string"
	case strings.HasPrefix(raw, "c\""):
		return "c-string"
	case raw == "null":
		return "null"
	case strings.ContainsAny(raw, ".eE"):
		return "float"
	default:
		return "unknown"
	}
}

func appendExpression(values []hir.Expression, expressions ...*hir.Expression) []hir.Expression {
	for _, expression := range expressions {
		if expression != nil {
			values = append(values, *expression)
		}
	}
	return values
}
