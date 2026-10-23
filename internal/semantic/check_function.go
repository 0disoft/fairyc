package semantic

import (
	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (c *typeChecker) checkFunction() hir.Function {
	functionSymbol := c.analyzer.functions[c.function]
	for _, attribute := range c.function.Attributes {
		if attribute.Name == testAttribute && c.module.file.Test {
			continue
		}
		if attribute.Name == effectsAttribute || attribute.Name == mustUseAttribute || attribute.Name == constEvalAttribute {
			continue
		}
		c.analyzer.addUnsupported(c.module.file.Path, attribute.Span, "function attribute @"+attribute.Name)
	}
	resultType := hir.TypeVoid
	if c.analyzer.dropOwners[c.function] == nil {
		resultType = c.canonicalType(c.function.ResultType)
	}
	c.resultType = resultType
	functionSymbol.typ = resultType
	if resultType == hir.TypeRegion {
		c.analyzer.addDiagnostic("E3103", span(c.module.file.Path, c.function.ResultType.Span))
	}
	if _, _, borrowLeaves := safeBorrowLeaf(resultType); borrowLeaves > 1 {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.ResultType.Span, "function result with more than one safe-borrow leaf")
	}
	functionSymbol.mustUse = c.analyzer.finalizeMustUse(c.module.file.Path, c.function.ResultType.Span, functionSymbol, resultType)
	if errorSpan, ok := resultFailureSpan(c.function.ResultType); ok {
		functionSymbol.resultError = errorSpan
	}
	result := hir.Function{
		Symbol:            functionSymbol.id,
		Name:              functionSymbol.name,
		Public:            c.function.Public,
		Unsafe:            c.function.Unsafe,
		Span:              span(c.module.file.Path, c.function.Span),
		DocumentationSpan: optionalSpan(c.module.file.Path, c.function.DocumentationSpan),
		Effects:           slices.Clone(functionSymbol.effects),
		MustUse:           functionSymbol.mustUse,
		ResultType:        resultType,
	}
	if c.function.Owner != "" {
		if owner := c.analyzer.ownerSymbol(c.module, c.function.Owner); owner != nil {
			result.OwnerType = owner.typ
			result.Receiver = c.checkOwnerReceiver(owner)
		}
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
			Symbol:  symbol.id,
			Name:    parameter.Name,
			Type:    symbol.typ,
			Mutable: parameter.Mutable,
			Span:    parameterSpan,
		})
		c.addType(parameterSpan, symbol.typ)
	}
	if c.function.Body == nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, "function without a body")
		return result
	}
	if c.function.Unsafe {
		c.unsafeScopes = append(c.unsafeScopes, span(c.module.file.Path, c.function.Span))
	}
	body, bodyType, _ := c.checkBlock(c.function.Body, resultType)
	if c.function.Unsafe {
		c.unsafeScopes = c.unsafeScopes[:len(c.unsafeScopes)-1]
	}
	result.Body = body
	if c.function.Body.Tail != nil {
		c.requireType(
			"E1009",
			span(c.module.file.Path, c.function.Body.Tail.Span),
			bodyType,
			resultType,
			span(c.module.file.Path, c.function.ResultType.Span),
		)
	} else if resultType != hir.TypeVoid && !blockAlwaysTerminates(c.function.Body) {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Body.Span, "non-void function without a checked return value")
	}
	return result
}
