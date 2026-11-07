package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/closureprep"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func closureValueSourceType(reference ast.TypeReference) bool {
	return closureprep.ClosureValueType(reference)
}

// Generic preparation copies this source-owned fact before clearing the written
// type parameters. Generated names are not evidence of monomorphic admission.
func (c *typeChecker) checkClosureMonomorphic(value hir.Type, at diagnostic.Span) {
	if (c.function.SourceGeneric || len(c.function.TypeParameters) != 0) && c.analyzer.closureBearing(value) {
		c.analyzer.addUnsupported(c.module.file.Path, at, "closure-bearing values require a monomorphic source function")
	}
}

func (c *typeChecker) checkClosureTarget(result *hir.Function) {
	if c.function.LiftedClosure == nil {
		return
	}
	if err := closureprep.ValidateValueTarget(*c.function); err != nil {
		c.analyzer.addUnsupported(c.module.file.Path, c.function.Span, err.Error())
		return
	}
	result.ClosureTarget = &hir.ClosureTarget{CaptureCount: c.function.LiftedClosure.CaptureCount}
}

func (c *typeChecker) checkClosureValue(expression *ast.Expression, result *hir.Expression, expected hir.Type) hir.Type {
	reject := func(detail string) hir.Type {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, detail)
		return hir.TypeInvalid
	}
	if expression.ClosureValue == nil || !hir.IsClosureI32(expected) {
		return reject("closure value requires prepared explicitly typed outer immutable binding")
	}
	target := c.module.functions[expression.ClosureValue.Target]
	if target == nil || target.function == nil || closureprep.ValidateValueTarget(*target.function) != nil {
		return reject("closure constructor target is not a safe pure lifted function")
	}
	if len(expression.Children) != target.function.LiftedClosure.CaptureCount {
		return reject("closure constructor capture count does not match the actual lifted target")
	}
	result.Kind, result.Symbol = hir.ExpressionClosureValue, target.id
	for _, capture := range expression.Children {
		if capture == nil || capture.Kind != syntax.KindPath {
			return reject("closure constructor capture must be an explicit named copied value")
		}
		value := c.analyzer.resolved[capture]
		if value == nil || value.kind != "local" && value.kind != "parameter" {
			return reject("closure constructor capture must resolve to a local or parameter")
		}
		child, actual := c.checkExpression(capture, hir.TypeI32)
		c.requireType("E1006", span(c.module.file.Path, capture.Span), actual, hir.TypeI32, span(target.source, target.declaration))
		result.Children = appendExpression(result.Children, child)
	}
	return hir.TypeClosureI32
}

func (c *typeChecker) checkClosureCall(expression *ast.Expression, result *hir.Expression) hir.Type {
	if len(expression.TypeArguments) != 0 || expression.CallLabels != nil {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "closure value invocation requires one positional nongeneric argument")
		return hir.TypeInvalid
	}
	result.Kind, result.Symbol = hir.ExpressionClosureCall, hir.NoSymbol
	callee, actual := c.checkExpression(expression.Children[0], hir.TypeClosureI32)
	if !hir.IsClosureI32(actual) {
		return hir.TypeInvalid
	}
	result.Children = appendExpression(result.Children, callee)
	if len(expression.Children) != 2 {
		c.analyzer.addDiagnostic("E1010", span(c.module.file.Path, expression.Span))
	}
	for _, argument := range expression.Children[1:] {
		child, actual := c.checkExpression(argument, hir.TypeI32)
		c.requireType("E1006", span(c.module.file.Path, argument.Span), actual, hir.TypeI32, span(c.module.file.Path, expression.Children[0].Span))
		result.Children = appendExpression(result.Children, child)
	}
	return hir.TypeI32
}

// The same leaf can be stored in ordinary structs, but it never grants wrapper,
// resource, foreign or address-taking permission to a containing nominal type.
func (a *analyzer) closureBearing(value hir.Type) bool {
	if hir.IsClosureI32(value) {
		return true
	}
	switch value.Descriptor().Kind {
	case hir.TypeKindBuiltin, hir.TypeKindUnset, hir.TypeKindOpaque:
		return false
	}
	seen := map[hir.Type]bool{}
	var visit func(hir.Type) bool
	visit = func(value hir.Type) bool {
		if hir.IsClosureI32(value) {
			return true
		}
		if seen[value] {
			return false
		}
		seen[value] = true
		d := value.Descriptor()
		if visitChild := d.Element != (hir.Type{}) && visit(d.Element); visitChild {
			return true
		}
		if d.Failure != (hir.Type{}) && visit(d.Failure) {
			return true
		}
		if info := a.structTypes[value]; info != nil {
			for _, field := range info.orderedFields {
				if visit(field.typ) {
					return true
				}
			}
		}
		if info := a.resourceTypes[value]; info != nil {
			for _, field := range info.orderedFields {
				if visit(field.typ) {
					return true
				}
			}
		}
		return false
	}
	return visit(value)
}

func (a *analyzer) validClosureValueType(value hir.Type) bool {
	if !a.closureBearing(value) || hir.IsClosureI32(value) {
		return true
	}
	info := a.structTypes[value]
	return info != nil && !info.reprC
}

func (a *analyzer) validateClosureValueTypes() {
	for _, module := range a.moduleOrder {
		for i := range module.file.Module.Structs {
			structure := &module.file.Module.Structs[i]
			owner := a.structs[structure]
			if owner != nil {
				for _, field := range a.structTypes[owner.typ].orderedFields {
					if !a.validClosureValueType(field.typ) {
						a.addUnsupported(module.file.Path, structure.Span, "wrapped closure-bearing struct field")
					}
				}
			}
			if owner != nil && a.structTypes[owner.typ].reprC && a.closureBearing(owner.typ) {
				a.addUnsupported(module.file.Path, structure.Span, "repr(C) struct containing a closure value")
			}
		}
		for i := range module.file.Module.Resources {
			resource := &module.file.Module.Resources[i]
			if owner := a.resources[resource]; owner != nil && a.closureBearing(owner.typ) {
				a.addUnsupported(module.file.Path, resource.Span, "resource containing a closure value")
			}
		}
		for _, foreign := range module.file.Module.ForeignFunctions {
			for _, parameter := range foreign.Parameters {
				value, _ := a.canonicalTypeReference(module, parameter.Type, true)
				if a.closureBearing(value) {
					a.addUnsupported(module.file.Path, parameter.Span, "foreign closure-bearing parameter")
				}
			}
			value, _ := a.canonicalTypeReference(module, foreign.ResultType, true)
			if a.closureBearing(value) {
				a.addUnsupported(module.file.Path, foreign.ResultType.Span, "foreign closure-bearing result")
			}
		}
	}
}

func (c *typeChecker) checkClosureExpressionUse(expression *ast.Expression, result *hir.Expression, actual hir.Type) {
	c.checkClosureMonomorphic(actual, expression.Span)
	if c.analyzer.closureBearing(actual) && expression.Kind == syntax.KindPath {
		if symbol := c.analyzer.resolved[expression]; symbol != nil && symbol.kind != "local" && symbol.kind != "parameter" {
			c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "closure-bearing value requires a constructed local or parameter, not a function path")
		}
	}
	if !c.analyzer.validClosureValueType(actual) {
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "wrapped closure-bearing value")
	}
	if len(result.Children) == 0 {
		return
	}
	bearing := false
	for _, child := range result.Children {
		bearing = bearing || c.analyzer.closureBearing(child.Type)
	}
	if !bearing {
		return
	}
	switch result.Kind {
	case hir.ExpressionCast, hir.ExpressionScalarCast, hir.ExpressionBinary, hir.ExpressionPrefix,
		hir.ExpressionFieldBorrow, hir.ExpressionSharedReborrow, hir.ExpressionArrayViewBorrow:
		c.analyzer.addUnsupported(c.module.file.Path, expression.Span, "closure-bearing operator, cast or address access")
	}
}
