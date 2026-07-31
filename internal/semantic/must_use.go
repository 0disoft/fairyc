package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

const mustUseAttribute = "must_use"

type mustUseDisposition uint8

const (
	mustUseIgnored mustUseDisposition = iota
	mustUseHandled
)

func (a *analyzer) prepareCallableMustUse() {
	for _, module := range a.moduleOrder {
		for index := range module.file.Module.ForeignFunctions {
			function := &module.file.Module.ForeignFunctions[index]
			if value := a.foreignFunctions[function]; value != nil {
				value.mustUse, value.mustUseSpan = a.parseMustUse(module.file.Path, function.NameSpan, function.Attributes)
			}
		}
		for index := range module.file.Module.Functions {
			function := &module.file.Module.Functions[index]
			if value := a.functions[function]; value != nil {
				value.mustUse, value.mustUseSpan = a.parseMustUse(module.file.Path, function.NameSpan, function.Attributes)
			}
		}
	}
}

func (a *analyzer) parseMustUse(source string, declaration diagnostic.Span, attributes []ast.Attribute) (bool, diagnostic.Span) {
	var first diagnostic.Span
	seen := false
	valid := true
	for _, attribute := range attributes {
		if attribute.Name != mustUseAttribute {
			continue
		}
		if seen {
			valid = false
			a.addDiagnostic("E4201", span(source, attribute.Span), RelatedSpan{Role: "must-use-declaration", Span: span(source, first)})
			continue
		}
		seen = true
		first = attribute.Span
		if len(attribute.Arguments) != 0 {
			valid = false
			a.addDiagnostic("E4201", span(source, attribute.Span), RelatedSpan{Role: "declaration", Span: span(source, declaration)})
		}
	}
	return seen && valid, first
}

func (a *analyzer) finalizeMustUse(source string, typeSpan diagnostic.Span, value *symbol, resultType hir.Type) bool {
	if value == nil || !value.mustUse {
		return false
	}
	eligible := resultType != hir.TypeInvalid && resultType != hir.TypeVoid && resultType != hir.TypeRegion && !a.isResource(resultType)
	if containsSafeBorrow(resultType) {
		eligible = false
	}
	if _, _, result := hir.ResultElements(resultType); result {
		eligible = false
	}
	if eligible {
		return true
	}
	a.addDiagnostic("E4201", span(source, value.mustUseSpan), RelatedSpan{Role: "result-type", Span: span(source, typeSpan)})
	return false
}

func (a *analyzer) mustUseFlowFunction(module *moduleScope, function *ast.Function) {
	if function == nil || function.Body == nil {
		return
	}
	a.mustUseFlowBlock(module, function.Body, mustUseHandled)
}

func (a *analyzer) mustUseFlowBlock(module *moduleScope, block *ast.Block, tail mustUseDisposition) {
	if block == nil {
		return
	}
	for index := range block.Statements {
		a.mustUseFlowStatement(module, &block.Statements[index])
	}
	a.mustUseFlowExpression(module, block.Tail, tail)
}

func (a *analyzer) mustUseFlowStatement(module *moduleScope, statement *ast.Statement) {
	if statement == nil {
		return
	}
	switch statement.Kind {
	case syntax.KindBinding:
		if statement.Binding != nil {
			a.mustUseFlowExpression(module, statement.Binding.Expression, mustUseHandled)
		}
	case syntax.KindExprStmt:
		a.mustUseFlowExpression(module, statement.Expression, mustUseIgnored)
	case syntax.KindReturn, syntax.KindDiscard, syntax.KindAssert, syntax.KindDefer:
		a.mustUseFlowExpression(module, statement.Expression, mustUseHandled)
	case syntax.KindAssignment:
		a.mustUseFlowExpression(module, statement.Target, mustUseHandled)
		a.mustUseFlowExpression(module, statement.Expression, mustUseHandled)
	case syntax.KindWhile, syntax.KindRegion:
		a.mustUseFlowExpression(module, statement.Expression, mustUseIgnored)
	case syntax.KindFor:
		a.mustUseFlowFor(module, statement.For)
	default:
		a.mustUseFlowExpression(module, statement.Expression, mustUseHandled)
	}
}

func (a *analyzer) mustUseFlowFor(module *moduleScope, loop *ast.ForLoop) {
	if loop == nil {
		return
	}
	a.mustUseFlowStatement(module, loop.Initializer)
	a.mustUseFlowExpression(module, loop.Condition, mustUseHandled)
	a.mustUseFlowStatement(module, loop.Post)
	a.mustUseFlowBlock(module, loop.Body, mustUseIgnored)
}

func (a *analyzer) mustUseFlowExpression(module *moduleScope, expression *ast.Expression, use mustUseDisposition) {
	if expression == nil {
		return
	}
	switch expression.Kind {
	case syntax.KindCall:
		if callee := a.mustUseCallTarget(expression); callee != nil && callee.mustUse && use == mustUseIgnored {
			a.addDiagnostic("E4202", span(module.file.Path, expression.Span), RelatedSpan{
				Role: "must-use-declaration", Span: span(callee.source, callee.mustUseSpan),
			})
		}
		for _, child := range expression.Children {
			a.mustUseFlowExpression(module, child, mustUseHandled)
		}
	case syntax.KindGroup, syntax.KindUnsafe:
		if expression.Block != nil {
			a.mustUseFlowBlock(module, expression.Block, use)
			return
		}
		for _, child := range expression.Children {
			a.mustUseFlowExpression(module, child, use)
		}
	case syntax.KindBlock:
		a.mustUseFlowBlock(module, expression.Block, use)
	case syntax.KindIf:
		if len(expression.Children) > 0 {
			a.mustUseFlowExpression(module, expression.Children[0], mustUseHandled)
		}
		for _, child := range expression.Children[1:] {
			a.mustUseFlowExpression(module, child, use)
		}
	case syntax.KindSwitch:
		if len(expression.Children) > 0 {
			a.mustUseFlowExpression(module, expression.Children[0], mustUseHandled)
		}
		for _, child := range expression.Children[1:] {
			a.mustUseFlowExpression(module, child, use)
		}
	case syntax.KindPattern:
		for index, child := range expression.Children {
			childUse := mustUseHandled
			if index == len(expression.Children)-1 {
				childUse = use
			}
			a.mustUseFlowExpression(module, child, childUse)
		}
	case syntax.KindWhile:
		if len(expression.Children) > 0 {
			a.mustUseFlowExpression(module, expression.Children[0], mustUseHandled)
		}
		for _, child := range expression.Children[1:] {
			a.mustUseFlowExpression(module, child, mustUseIgnored)
		}
	case syntax.KindRegion:
		for _, child := range expression.Children {
			a.mustUseFlowExpression(module, child, mustUseIgnored)
		}
	default:
		for _, child := range expression.Children {
			a.mustUseFlowExpression(module, child, mustUseHandled)
		}
	}
}

func (a *analyzer) mustUseCallTarget(expression *ast.Expression) *symbol {
	if expression == nil || expression.Kind != syntax.KindCall || len(expression.Children) == 0 {
		return nil
	}
	if method := a.methodCalls[expression]; method != nil {
		return method
	}
	return a.resolved[expression.Children[0]]
}
