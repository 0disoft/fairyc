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
		actual = c.checkCall(expression, result)
	case syntax.KindIf:
		actual = c.checkIf(expression, result, expected)
	case syntax.KindBlock:
		if expression.Block == nil {
			actual = hir.TypeInvalid
			break
		}
		block, blockType, _ := c.checkBlock(expression.Block, expected)
		result.Block = &block
		actual = blockType
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "expression "+string(expression.Kind))
		actual = hir.TypeInvalid
	}
	result.Type = actual
	if actual != hir.TypeInvalid {
		c.addType(result.Span, actual)
	}
	return result, actual
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
	case "+", "-", "*", "/", "%", "<<", ">>", "&", "^", "|":
		c.requireType("E1006", span(c.module.file.Path, expression.Children[0].Span), leftType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		c.requireType("E1006", span(c.module.file.Path, expression.Children[1].Span), rightType, hir.TypeI32, span(c.module.file.Path, expression.Span))
		return hir.TypeI32
	default:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "binary operator "+expression.Text)
		return hir.TypeInvalid
	}
}

func (c *typeChecker) checkCall(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	calleeExpression := expression.Children[0]
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
			expected = c.canonicalType(callee.function.Parameters[index].Type)
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
	callee.typ = c.canonicalType(callee.function.ResultType)
	return callee.typ
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
	switch strings.TrimSpace(reference.Text) {
	case "bool":
		return hir.TypeBool
	case "i32":
		return hir.TypeI32
	default:
		c.analyzer.addUnsupported(c.module.file.Path, reference.Span, "type "+strings.TrimSpace(reference.Text))
		return hir.TypeInvalid
	}
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
