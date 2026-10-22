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
	if reference.Kind == ast.TypeKindForeignFunction && reference.ABI == "C" && reference.Result != nil && len(reference.Parameters) == 2 {
		result, resultOK := a.canonicalTypeReference(module, *reference.Result, false)
		value, valueOK := a.canonicalTypeReference(module, reference.Parameters[0], false)
		data, dataOK := a.canonicalTypeReference(module, reference.Parameters[1], false)
		if resultOK && valueOK && dataOK && result == hir.TypeI32 && value == hir.TypeI32 && data == hir.RawPointer(hir.TypeVoid, true) {
			return hir.TypeCCallbackI32UserData
		}
	}
	return a.canonicalType(module, reference)
}

func (c *typeChecker) checkCallbackArgument(expression *ast.Expression) (*hir.Expression, hir.Type) {
	target := c.analyzer.resolved[expression]
	valid := expression.Kind == syntax.KindPath && target != nil && target.function != nil &&
		target.function.Unsafe && target.function.Owner == "" && len(target.function.Parameters) == 2 && len(target.effects) == 0
	if valid {
		module := c.analyzer.symbolModule(target)
		function := target.function
		valid = c.analyzer.canonicalType(module, function.ResultType) == hir.TypeI32 &&
			c.analyzer.canonicalType(module, function.Parameters[0].Type) == hir.TypeI32 &&
			c.analyzer.canonicalType(module, function.Parameters[1].Type) == hir.RawPointer(hir.TypeVoid, true)
	}
	if !valid {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "callback requires a named non-capturing unsafe fn i32(i32, ptr mut void) without declared effects")
		return &hir.Expression{Type: hir.TypeInvalid}, hir.TypeInvalid
	}
	result := &hir.Expression{Kind: hir.ExpressionCCallback, Type: hir.TypeCCallbackI32UserData,
		Symbol: target.id, Span: span(c.module.file.Path, expression.Span)}
	c.analyzer.expressionTypes[expression] = result.Type
	return result, result.Type
}
