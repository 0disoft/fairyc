package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (c *typeChecker) checkCall(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	if !c.analyzer.checkCallArgumentLabels(c.module, expression) {
		return hir.TypeInvalid
	}
	if len(expression.Children) == 0 {
		return hir.TypeInvalid
	}
	calleeExpression := expression.Children[0]
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindFieldExpr && !isRegionPlaceCall(expression) {
		return c.checkMethodCall(expression, result)
	}
	if calleeExpression != nil && calleeExpression.Kind == syntax.KindPath {
		if calleeExpression.Text == sourceLocationPath {
			return c.checkSourceLocation(expression, result)
		}
		if variant, ok := resultVariantPath(calleeExpression.Text); ok {
			return c.checkResultConstructor(expression, result, expected, variant)
		}
		if variant, ok := optionVariantPath(calleeExpression.Text); ok {
			return c.checkOptionConstructor(expression, result, expected, variant)
		}
		if constructor, ok := borrowedViewConstructorPath(calleeExpression.Text); ok {
			return c.checkBorrowedViewConstructor(expression, result, calleeExpression.Text, constructor)
		}
	}
	if isRegionPlaceCall(expression) {
		return c.checkRegionPlace(expression, result)
	}
	callee := c.analyzer.resolved[calleeExpression]
	if callee == nil || callee.kind != "function" && callee.kind != "foreign-function" {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target that is not a function")
		return hir.TypeInvalid
	}
	if callee.foreign != nil {
		c.checkCallEffects(calleeExpression, result, callee)
		return c.checkForeignCall(expression, result, callee)
	}
	if callee.function == nil {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "call target without a function declaration")
		return hir.TypeInvalid
	}
	result.Symbol = callee.id
	c.checkCallEffects(calleeExpression, result, callee)
	result.Children = append(result.Children, hir.Expression{
		Kind:   expressionKind(calleeExpression.Kind),
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
			expected = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[index].Type)
			expectedSpan = callee.function.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		child, actual = c.sharedCallArgument(&arguments[index], child, actual, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireDetailedType(
				span(c.module.file.Path, argument.Span),
				actual,
				expected,
				span(callee.source, expectedSpan),
			)
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.ResultType)
	if errorSpan, ok := resultFailureSpan(callee.function.ResultType); ok {
		callee.resultError = errorSpan
	}
	return callee.typ
}

func (c *typeChecker) checkOwnerReceiver(owner *symbol) string {
	for index := range c.function.Parameters {
		parameter := &c.function.Parameters[index]
		if parameter.Name != "this" {
			continue
		}
		if index != 0 {
			c.analyzer.addUnsupported(c.module.file.Path, parameter.NameSpan, "method receiver that is not the first parameter")
			return "invalid"
		}
		receiverType := c.canonicalType(parameter.Type)
		baseType := receiverType
		mode := "value"
		if element, mutable, borrow := hir.BorrowElement(receiverType); borrow {
			baseType = element
			mode = "shared"
			if mutable {
				mode = "exclusive"
			}
		}
		if baseType != owner.typ {
			c.analyzer.addUnsupported(c.module.file.Path, parameter.Type.Span, "method receiver type that does not match its owner")
			return "invalid"
		}
		return mode
	}
	return ""
}

func (c *typeChecker) checkMethodCall(expression *ast.Expression, result *hir.Expression) hir.Type {
	calleeExpression := expression.Children[0]
	if len(calleeExpression.Children) != 1 {
		return hir.TypeInvalid
	}
	receiverExpression := calleeExpression.Children[0]
	receiver, receiverType := c.checkExpression(receiverExpression, hir.TypeInvalid)
	ownerType := receiverType
	if element, _, borrow := hir.BorrowElement(ownerType); borrow {
		ownerType = element
	}
	callee := c.analyzer.methodSymbol(ownerType, calleeExpression.Text)
	if callee == nil || callee.function == nil || len(callee.function.Parameters) == 0 || callee.function.Parameters[0].Name != "this" {
		c.analyzer.addUnsupported(c.module.file.Path, calleeExpression.Span, "unknown method "+calleeExpression.Text)
		return hir.TypeInvalid
	}
	if callee.module != c.module.file.Module.Path && !callee.public {
		c.analyzer.addDiagnostic("E0103", span(c.module.file.Path, calleeExpression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
		return hir.TypeInvalid
	}
	expectedReceiver := c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[0].Type)
	if element, mutable, borrow := hir.BorrowElement(expectedReceiver); borrow && receiverType == element {
		target := c.analyzer.resolved[receiverExpression]
		if target == nil || mutable && !target.mutable {
			c.analyzer.addDiagnostic("E1008", span(c.module.file.Path, receiverExpression.Span))
			return hir.TypeInvalid
		}
		operator := "&"
		if mutable {
			operator = "&mut"
		}
		borrowExpression := &ast.Expression{
			Kind: syntax.KindPrefix, Span: receiverExpression.Span, Text: operator, Children: []*ast.Expression{receiverExpression},
		}
		calleeExpression.Children[0] = borrowExpression
		receiver = &hir.Expression{
			Kind: hir.ExpressionPrefix, Span: span(c.module.file.Path, receiverExpression.Span), Operator: operator,
			Type: expectedReceiver, Symbol: hir.NoSymbol, Children: []hir.Expression{*receiver},
		}
		receiverType = expectedReceiver
	}
	receiver, receiverType = c.sharedCallArgument(&calleeExpression.Children[0], receiver, receiverType, expectedReceiver)
	c.requireType("E1006", span(c.module.file.Path, receiverExpression.Span), receiverType, expectedReceiver, span(callee.source, callee.function.Parameters[0].Type.Span))
	c.analyzer.methodCalls[expression] = callee
	result.Symbol = callee.id
	c.checkCallEffects(calleeExpression, result, callee)
	result.Children = append(result.Children, hir.Expression{
		Kind: hir.ExpressionPath, Type: hir.TypeInvalid, Span: span(c.module.file.Path, calleeExpression.Span), Symbol: callee.id,
	})
	result.Children = appendExpression(result.Children, receiver)
	arguments := expression.Children[1:]
	if len(arguments)+1 != len(callee.function.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		parameterIndex := index + 1
		expectedType := hir.TypeInvalid
		expectedSpan := callee.typeSpan
		if parameterIndex < len(callee.function.Parameters) {
			expectedType = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.Parameters[parameterIndex].Type)
			expectedSpan = callee.function.Parameters[parameterIndex].Type.Span
		}
		child, actual := c.checkExpression(argument, expectedType)
		child, actual = c.sharedCallArgument(&arguments[index], child, actual, expectedType)
		result.Children = appendExpression(result.Children, child)
		if expectedType != hir.TypeInvalid {
			c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, expectedType, span(callee.source, expectedSpan))
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), callee.function.ResultType)
	return callee.typ
}

func (a *analyzer) methodSymbol(ownerType hir.Type, name string) *symbol {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if function.Owner == "" || function.Name != name || len(function.Parameters) == 0 || function.Parameters[0].Name != "this" {
				continue
			}
			owner := a.ownerSymbol(module, function.Owner)
			if owner != nil && owner.typ == ownerType {
				return a.functions[function]
			}
		}
	}
	return nil
}

func (a *analyzer) callableForCall(expression *ast.Expression) *symbol {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return nil
	}
	if method := a.methodCalls[expression]; method != nil {
		return method
	}
	return a.resolved[expression.Children[0]]
}

func (a *analyzer) callArguments(expression *ast.Expression) []*ast.Expression {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return nil
	}
	arguments := expression.Children[1:]
	if a.methodCalls[expression] == nil {
		return arguments
	}
	callee := expression.Children[0]
	if callee == nil || callee.Kind != syntax.KindFieldExpr || len(callee.Children) != 1 {
		return arguments
	}
	result := make([]*ast.Expression, 0, len(arguments)+1)
	result = append(result, callee.Children[0])
	return append(result, arguments...)
}

func (c *typeChecker) checkForeignCall(expression *ast.Expression, result *hir.Expression, callee *symbol) hir.Type {
	declaration := callee.foreign
	result.Symbol = callee.id
	result.ForeignABI = declaration.ABI
	result.Children = append(result.Children, hir.Expression{
		Kind: hir.ExpressionPath, Type: hir.TypeInvalid,
		Span: span(c.module.file.Path, expression.Children[0].Span), Symbol: callee.id,
	})
	if len(c.unsafeScopes) == 0 {
		c.analyzer.addDiagnostic("E5101", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	} else {
		scope := c.unsafeScopes[len(c.unsafeScopes)-1]
		result.UnsafeScope = &scope
	}
	arguments := expression.Children[1:]
	if len(arguments) != len(declaration.Parameters) {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span), RelatedSpan{
			Role: "declaration", Span: span(callee.source, callee.declaration),
		})
	}
	for index, argument := range arguments {
		expected := hir.TypeInvalid
		expectedSpan := declaration.ResultType.Span
		if index < len(declaration.Parameters) {
			expected = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), declaration.Parameters[index].Type)
			expectedSpan = declaration.Parameters[index].Type.Span
		}
		child, actual := c.checkExpression(argument, expected)
		result.Children = appendExpression(result.Children, child)
		if expected != hir.TypeInvalid {
			c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, expected, span(callee.source, expectedSpan))
		}
	}
	callee.typ = c.analyzer.canonicalType(c.analyzer.symbolModule(callee), declaration.ResultType)
	return callee.typ
}
