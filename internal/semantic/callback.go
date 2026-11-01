package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (a *analyzer) foreignParameterType(module *moduleScope, reference ast.TypeReference) hir.Type {
	if reference.Kind == ast.TypeKindGrouped && reference.Element != nil {
		return a.foreignParameterType(module, *reference.Element)
	}
	if reference.Kind == ast.TypeKindForeignFunction && reference.ABI == "C" && reference.Result != nil {
		result, resultOK := a.canonicalTypeReference(module, *reference.Result, false)
		parameters := make([]hir.Type, len(reference.Parameters))
		valid := resultOK
		for i, parameter := range reference.Parameters {
			var ok bool
			parameters[i], ok = a.canonicalTypeReference(module, parameter, false)
			valid = valid && ok
		}
		if valid && hir.ValidCallbackSignature(result, parameters) {
			return hir.CCallback(result, parameters)
		}
	}
	return a.canonicalType(module, reference)
}

func (c *typeChecker) checkCallbackArgument(expression *ast.Expression, expected hir.Type) (*hir.Expression, hir.Type) {
	wantResult, wantParameters, signatureOK := hir.CallbackSignature(expected)
	target := c.analyzer.resolved[expression]
	valid := signatureOK && expression.Kind == syntax.KindPath && target != nil && target.function != nil &&
		target.function.Unsafe && target.function.Owner == "" && len(target.function.Parameters) == len(wantParameters) && len(target.effects) == 0
	if valid {
		module := c.analyzer.symbolModule(target)
		function := target.function
		valid = c.analyzer.canonicalType(module, function.ResultType) == wantResult
		for i, parameter := range function.Parameters {
			valid = valid && c.analyzer.canonicalType(module, parameter.Type) == wantParameters[i]
		}

	}
	if !valid {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "callback requires a named non-capturing unsafe function with the exact signature and no declared effects")
		return &hir.Expression{Type: hir.TypeInvalid}, hir.TypeInvalid
	}
	result := &hir.Expression{Kind: hir.ExpressionCCallback, Type: expected,
		Symbol: target.id, Span: span(c.module.file.Path, expression.Span)}
	c.analyzer.expressionTypes[expression] = result.Type
	return result, result.Type
}
