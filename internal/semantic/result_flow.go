package semantic

import (
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

type resultUseKind uint8

const (
	resultUseIgnore resultUseKind = iota
	resultUseStore
	resultUseTransfer
	resultUseUnhandled
	resultUseHandled
)

type resultUse struct {
	kind resultUseKind
	mode string
}

type resultObligation struct {
	pending   bool
	producers []hir.Span
}

type resultEnvironment map[*symbol]resultObligation

type resultBranch struct {
	environment resultEnvironment
	reachable   bool
}

type resultLoopEdge struct {
	kind        syntax.Kind
	environment resultEnvironment
}

type resultLoopFrame struct {
	edges []resultLoopEdge
}

func (a *analyzer) resultFlowFunction(module *moduleScope, function *ast.Function) {
	if function.Body == nil {
		return
	}
	environment := make(resultEnvironment)
	for index := range function.Parameters {
		parameter := &function.Parameters[index]
		value := a.parameter[parameter]
		if value == nil || !isResultType(value.typ) {
			continue
		}
		environment[value] = pendingResult(span(module.file.Path, parameter.NameSpan))
	}
	a.resultFlowBlock(module, function.Body, environment, resultUse{kind: resultUseHandled, mode: "return"}, true)
}

func (a *analyzer) resultFlowBlock(module *moduleScope, block *ast.Block, environment resultEnvironment, tailUse resultUse, functionBody bool) bool {
	if block == nil {
		return true
	}
	locals := make([]*symbol, 0)
	reachable := true
	for index := range block.Statements {
		if !reachable {
			break
		}
		statement := &block.Statements[index]
		switch statement.Kind {
		case syntax.KindBinding:
			binding := statement.Binding
			if binding == nil {
				continue
			}
			a.resultFlowExpression(module, binding.Expression, environment, resultUse{kind: resultUseStore})
			value := a.bindingSymbols[binding]
			if value != nil && isResultType(value.typ) {
				producer := span(module.file.Path, binding.NameSpan)
				if binding.Expression != nil {
					producer = span(module.file.Path, binding.Expression.Span)
				}
				environment[value] = pendingResult(producer)
				locals = append(locals, value)
			}
		case syntax.KindReturn:
			a.resultFlowExpression(module, statement.Expression, environment, resultUse{kind: resultUseHandled, mode: "return"})
			a.reportPendingResults(environment)
			reachable = false
		case syntax.KindBreak, syntax.KindContinue:
			a.captureResultLoopEdge(statement.Kind, environment)
			reachable = false
		case syntax.KindDiscard:
			a.resultFlowExpression(module, statement.Expression, environment, resultUse{kind: resultUseHandled, mode: "discard"})
		case syntax.KindExprStmt:
			a.resultFlowExpression(module, statement.Expression, environment, resultUse{kind: resultUseUnhandled})
		case syntax.KindAssignment:
			a.resultFlowExpression(module, statement.Expression, environment, resultUse{kind: resultUseStore})
			target := a.resolved[statement.Target]
			if target != nil && isResultType(target.typ) {
				a.reportResultObligation(environment[target])
				producer := span(module.file.Path, statement.Span)
				if statement.Expression != nil {
					producer = span(module.file.Path, statement.Expression.Span)
				}
				environment[target] = pendingResult(producer)
			}
		case syntax.KindWhile:
			reachable = a.resultFlowWhile(module, statement.Expression, environment)
		case syntax.KindFor:
			reachable = a.resultFlowFor(module, statement.For, environment)
		case syntax.KindRegion:
			reachable = a.resultFlowRegion(module, statement.Expression, environment)
		default:
			a.resultFlowExpression(module, statement.Expression, environment, resultUse{kind: resultUseIgnore})
		}
	}
	if reachable && block.Tail != nil {
		reachable = a.resultFlowExpression(module, block.Tail, environment, tailUse)
	}
	if functionBody {
		if reachable {
			a.reportPendingResults(environment)
			reachable = false
		}
	} else {
		for _, local := range locals {
			a.reportResultObligation(environment[local])
		}
	}
	for _, local := range locals {
		delete(environment, local)
	}
	return reachable
}

func (a *analyzer) resultFlowFor(module *moduleScope, loop *ast.ForLoop, environment resultEnvironment) bool {
	if loop == nil {
		return true
	}
	var locals []*symbol
	if initializer := loop.Initializer; initializer != nil {
		if initializer.Binding != nil {
			binding := initializer.Binding
			a.resultFlowExpression(module, binding.Expression, environment, resultUse{kind: resultUseStore})
			value := a.bindingSymbols[binding]
			if value != nil && isResultType(value.typ) {
				producer := span(module.file.Path, binding.NameSpan)
				if binding.Expression != nil {
					producer = span(module.file.Path, binding.Expression.Span)
				}
				environment[value] = pendingResult(producer)
				locals = append(locals, value)
			}
		} else if initializer.Kind == syntax.KindAssignment {
			a.resultFlowExpression(module, initializer.Expression, environment, resultUse{kind: resultUseStore})
		} else {
			a.resultFlowExpression(module, initializer.Expression, environment, resultUse{kind: resultUseUnhandled})
		}
	}
	reachable := a.resultFlowWhile(module, forAsWhile(loop), environment)
	for _, local := range locals {
		if reachable {
			a.reportResultObligation(environment[local])
		}
		delete(environment, local)
	}
	return reachable
}

func (a *analyzer) resultFlowExpression(module *moduleScope, expression *ast.Expression, environment resultEnvironment, use resultUse) bool {
	if expression == nil {
		return true
	}
	switch expression.Kind {
	case syntax.KindPropagate:
		a.recordPropagateHandling(module, expression)
		if len(expression.Children) == 1 {
			if !a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseTransfer}) {
				return false
			}
			// The error alternative returns before any later normal-path handling.
			// Transfer the operand first, then check other pending obligations.
			a.reportPendingResults(environment)
		}
		return true
	case syntax.KindIf:
		return a.resultFlowIf(module, expression, environment, use)
	case syntax.KindSwitch:
		return a.resultFlowMatch(module, expression, environment, use)
	case syntax.KindFieldExpr:
		if expression.Text == "is_ok" && len(expression.Children) == 1 && isResultType(a.expressionTypes[expression.Children[0]]) {
			a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseHandled, mode: "match"})
			return true
		}
		for _, child := range expression.Children {
			a.resultFlowExpression(module, child, environment, resultUse{kind: resultUseIgnore})
		}
		a.applyResultUse(module, expression, environment, use)
		return true
	case syntax.KindWhile:
		return a.resultFlowWhile(module, expression, environment)
	case syntax.KindRegion:
		return a.resultFlowRegion(module, expression, environment)
	case syntax.KindBlock:
		childUse := a.applyResultUse(module, expression, environment, use)
		return a.resultFlowBlock(module, expression.Block, environment, childUse, false)
	case syntax.KindGroup, syntax.KindUnsafe:
		childUse := a.applyResultUse(module, expression, environment, use)
		if len(expression.Children) == 1 {
			return a.resultFlowExpression(module, expression.Children[0], environment, childUse)
		}
		return true
	case syntax.KindCall:
		for _, argument := range expression.Children[1:] {
			a.resultFlowExpression(module, argument, environment, resultUse{kind: resultUseTransfer})
		}
		a.applyResultUse(module, expression, environment, use)
		return true
	case syntax.KindRecord:
		for _, field := range expression.Children[1:] {
			for _, value := range field.Children {
				a.resultFlowExpression(module, value, environment, resultUse{kind: resultUseTransfer})
			}
		}
		a.applyResultUse(module, expression, environment, use)
		return true
	case syntax.KindPath:
		a.applyResultUse(module, expression, environment, use)
		return true
	default:
		for _, child := range expression.Children {
			a.resultFlowExpression(module, child, environment, resultUse{kind: resultUseIgnore})
		}
		a.applyResultUse(module, expression, environment, use)
		return true
	}
}

func (a *analyzer) resultFlowIf(module *moduleScope, expression *ast.Expression, environment resultEnvironment, use resultUse) bool {
	if len(expression.Children) < 2 {
		return true
	}
	a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseIgnore})
	branchUse := a.applyResultUse(module, expression, environment, use)
	baseline := cloneResultEnvironment(environment)
	branches := make([]resultBranch, 0, 2)
	thenEnvironment := cloneResultEnvironment(baseline)
	branches = append(branches, resultBranch{
		environment: thenEnvironment,
		reachable:   a.resultFlowExpression(module, expression.Children[1], thenEnvironment, branchUse),
	})
	if len(expression.Children) == 2 {
		branches = append(branches, resultBranch{environment: cloneResultEnvironment(baseline), reachable: true})
	} else {
		elseEnvironment := cloneResultEnvironment(baseline)
		branches = append(branches, resultBranch{
			environment: elseEnvironment,
			reachable:   a.resultFlowExpression(module, expression.Children[2], elseEnvironment, branchUse),
		})
	}
	return mergeResultBranches(environment, baseline, branches)
}

func (a *analyzer) resultFlowMatch(module *moduleScope, expression *ast.Expression, environment resultEnvironment, use resultUse) bool {
	if len(expression.Children) < 2 {
		return true
	}
	a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseHandled, mode: "match"})
	branchUse := a.applyResultUse(module, expression, environment, use)
	baseline := cloneResultEnvironment(environment)
	branches := make([]resultBranch, 0, len(expression.Children)-1)
	for _, arm := range expression.Children[1:] {
		branchEnvironment := cloneResultEnvironment(baseline)
		var binding *symbol
		var body *ast.Expression
		if arm != nil && arm.Kind == syntax.KindPattern && len(arm.Children) == 2 {
			pattern := arm.Children[0]
			body = arm.Children[1]
			if pattern != nil && pattern.Kind == syntax.KindPattern && len(pattern.Children) == 2 {
				binding = a.resolved[pattern.Children[1]]
				if binding != nil && isResultType(binding.typ) {
					branchEnvironment[binding] = pendingResult(span(module.file.Path, pattern.Children[1].Span))
				}
			}
		}
		reachable := a.resultFlowExpression(module, body, branchEnvironment, branchUse)
		if binding != nil {
			if reachable {
				a.reportResultObligation(branchEnvironment[binding])
			}
			delete(branchEnvironment, binding)
		}
		branches = append(branches, resultBranch{environment: branchEnvironment, reachable: reachable})
	}
	return mergeResultBranches(environment, baseline, branches)
}

func (a *analyzer) resultFlowWhile(module *moduleScope, expression *ast.Expression, environment resultEnvironment) bool {
	if expression == nil || len(expression.Children) != 2 {
		return true
	}
	a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseIgnore})
	baseline := cloneResultEnvironment(environment)
	bodyEnvironment := cloneResultEnvironment(baseline)
	a.resultLoops = append(a.resultLoops, resultLoopFrame{})
	bodyReachable := a.resultFlowExpression(module, expression.Children[1], bodyEnvironment, resultUse{kind: resultUseIgnore})
	firstFrame := a.resultLoops[len(a.resultLoops)-1]
	a.resultLoops = a.resultLoops[:len(a.resultLoops)-1]
	exitBranches := []resultBranch{{environment: cloneResultEnvironment(baseline), reachable: true}}
	backBranches := make([]resultBranch, 0, len(firstFrame.edges)+1)
	if bodyReachable {
		backBranches = append(backBranches, resultBranch{environment: bodyEnvironment, reachable: true})
	}
	for _, edge := range firstFrame.edges {
		branch := resultBranch{environment: edge.environment, reachable: true}
		if edge.kind == syntax.KindBreak {
			exitBranches = append(exitBranches, branch)
		} else {
			backBranches = append(backBranches, branch)
		}
	}
	for _, branch := range backBranches {
		exitBranches = append(exitBranches, branch)
	}
	if len(backBranches) > 0 && !a.canSkipScalarResultBackEdge(expression, baseline) {
		backEnvironment := cloneResultEnvironment(baseline)
		mergeResultBranches(backEnvironment, baseline, backBranches)
		secondEnvironment := cloneResultEnvironment(backEnvironment)
		a.resultLoops = append(a.resultLoops, resultLoopFrame{})
		secondReachable := a.resultFlowExpression(module, expression.Children[1], secondEnvironment, resultUse{kind: resultUseIgnore})
		secondFrame := a.resultLoops[len(a.resultLoops)-1]
		a.resultLoops = a.resultLoops[:len(a.resultLoops)-1]
		if secondReachable {
			exitBranches = append(exitBranches, resultBranch{environment: secondEnvironment, reachable: true})
		}
		for _, edge := range secondFrame.edges {
			exitBranches = append(exitBranches, resultBranch{environment: edge.environment, reachable: true})
		}
	}
	mergeResultBranches(environment, baseline, exitBranches)
	return true
}

func (a *analyzer) captureResultLoopEdge(kind syntax.Kind, environment resultEnvironment) {
	if len(a.resultLoops) == 0 {
		return
	}
	index := len(a.resultLoops) - 1
	a.resultLoops[index].edges = append(a.resultLoops[index].edges, resultLoopEdge{
		kind: kind, environment: cloneResultEnvironment(environment),
	})
}

func (a *analyzer) resultFlowRegion(module *moduleScope, expression *ast.Expression, environment resultEnvironment) bool {
	if expression == nil || len(expression.Children) != 2 {
		return true
	}
	a.resultFlowExpression(module, expression.Children[0], environment, resultUse{kind: resultUseIgnore})
	return a.resultFlowExpression(module, expression.Children[1], environment, resultUse{kind: resultUseIgnore})
}

func (a *analyzer) applyResultUse(module *moduleScope, expression *ast.Expression, environment resultEnvironment, use resultUse) resultUse {
	if expression == nil || !isResultType(a.expressionTypes[expression]) {
		return resultUse{kind: resultUseIgnore}
	}
	if expression.Kind == syntax.KindPath {
		if value := a.resolved[expression]; value != nil {
			obligation := environment[value]
			obligation.pending = false
			obligation.producers = nil
			environment[value] = obligation
		}
	}
	switch use.kind {
	case resultUseHandled:
		a.recordResultHandling(module, expression, use.mode)
	case resultUseUnhandled, resultUseIgnore:
		a.addDiagnostic("E4101", span(module.file.Path, expression.Span))
	}
	return resultUse{kind: resultUseTransfer}
}

func (a *analyzer) reportPendingResults(environment resultEnvironment) {
	for _, obligation := range environment {
		a.reportResultObligation(obligation)
	}
}

func (a *analyzer) reportResultObligation(obligation resultObligation) {
	if !obligation.pending {
		return
	}
	for _, producer := range obligation.producers {
		a.addDiagnostic("E4101", producer)
	}
}

func pendingResult(producer hir.Span) resultObligation {
	return resultObligation{pending: true, producers: []hir.Span{producer}}
}

func cloneResultEnvironment(input resultEnvironment) resultEnvironment {
	result := make(resultEnvironment, len(input))
	for value, obligation := range input {
		obligation.producers = slices.Clone(obligation.producers)
		result[value] = obligation
	}
	return result
}

func mergeResultBranches(target, baseline resultEnvironment, branches []resultBranch) bool {
	reachable := false
	for _, branch := range branches {
		reachable = reachable || branch.reachable
	}
	for value, before := range baseline {
		merged := resultObligation{}
		for _, branch := range branches {
			if !branch.reachable {
				continue
			}
			actual, found := branch.environment[value]
			if !found {
				actual = before
			}
			if actual.pending {
				merged.pending = true
				merged.producers = appendUniqueResultProducers(merged.producers, actual.producers...)
			}
		}
		target[value] = merged
	}
	return reachable
}

func appendUniqueResultProducers(target []hir.Span, values ...hir.Span) []hir.Span {
	for _, value := range values {
		duplicate := false
		for _, existing := range target {
			if compareSpan(existing, value) == 0 {
				duplicate = true
				break
			}
		}
		if !duplicate {
			target = append(target, value)
		}
	}
	return target
}

func isResultType(value hir.Type) bool {
	_, _, ok := hir.ResultElements(value)
	return ok
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
		ResultType:      displayResultType(module.identityPath(), value),
		SuccessType:     displayResultType(module.identityPath(), success),
		ErrorType:       displayResultType(module.identityPath(), failure),
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
		ResultType:      displayResultType(module.identityPath(), value),
		SuccessType:     displayResultType(module.identityPath(), success),
		ErrorType:       displayResultType(module.identityPath(), failure),
		EvaluationCount: 1,
	})
}
