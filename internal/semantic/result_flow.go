package semantic

import (
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func (a *analyzer) resultFlowFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil {
		return
	}
	a.resultFlowBlock(module, function.Body, true)
}

func (a *analyzer) resultFlowBlock(module *moduleScope, block *ast.Block, functionBody bool) {
	if block == nil {
		return
	}
	for index := range block.Statements {
		statement := &block.Statements[index]
		switch statement.Kind {
		case syntax.KindReturn:
			a.recordResultHandling(module, statement.Expression, "return")
		case syntax.KindDiscard:
			a.recordResultHandling(module, statement.Expression, "discard")
		case syntax.KindExprStmt:
			if _, _, ok := hir.ResultElements(a.expressionTypes[statement.Expression]); ok {
				a.addDiagnostic("E4101", span(module.file.Path, statement.Expression.Span))
			}
		}
		if statement.Binding != nil {
			a.scanResultPropagates(module, statement.Binding.Expression)
		}
		a.scanResultPropagates(module, statement.Target)
		a.scanResultPropagates(module, statement.Expression)
	}
	if block.Tail != nil {
		if functionBody {
			a.recordResultHandling(module, block.Tail, "return")
		}
		a.scanResultPropagates(module, block.Tail)
	}
}

func (a *analyzer) scanResultPropagates(module *moduleScope, expression *ast.Expression) {
	if expression == nil {
		return
	}
	if expression.Kind == syntax.KindPropagate {
		a.recordPropagateHandling(module, expression)
	}
	if expression.Kind == syntax.KindMatch && len(expression.Children) > 0 {
		a.recordResultHandling(module, expression.Children[0], "match")
	}
	if expression.Block != nil {
		a.resultFlowBlock(module, expression.Block, false)
	}
	for _, child := range expression.Children {
		a.scanResultPropagates(module, child)
	}
}

func (a *analyzer) recordResultHandling(module *moduleScope, expression *ast.Expression, mode string) {
	if expression == nil {
		return
	}
	value := a.expressionTypes[expression]
	success, failure, ok := hir.ResultElements(value)
	if !ok {
		return
	}
	a.resultHandling = append(a.resultHandling, ResultHandlingFact{
		Expression:      span(module.file.Path, expression.Span),
		Mode:            mode,
		ResultType:      displayResultType(module.file.Module.Path, value),
		SuccessType:     displayResultType(module.file.Module.Path, success),
		ErrorType:       displayResultType(module.file.Module.Path, failure),
		EvaluationCount: 1,
	})
}

func (a *analyzer) recordPropagateHandling(module *moduleScope, expression *ast.Expression) {
	if expression == nil || len(expression.Children) != 1 {
		return
	}
	value := a.expressionTypes[expression.Children[0]]
	success, failure, ok := hir.ResultElements(value)
	if !ok {
		return
	}
	a.resultHandling = append(a.resultHandling, ResultHandlingFact{
		Expression:      span(module.file.Path, expression.Span),
		Mode:            "propagate",
		ResultType:      displayResultType(module.file.Module.Path, value),
		SuccessType:     displayResultType(module.file.Module.Path, success),
		ErrorType:       displayResultType(module.file.Module.Path, failure),
		EvaluationCount: 1,
	})
}
