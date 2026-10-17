package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerDefer(statement hir.Statement) error {
	expression := statement.Expression
	if expression == nil || expression.Kind != hir.ExpressionCall || expression.Type != hir.TypeVoid ||
		expression.Symbol == hir.NoSymbol || expression.ForeignABI != "" || len(expression.Children) == 0 || l.currentScope.Source == "" {
		return fmt.Errorf("malformed defer statement")
	}
	arguments := make([]Operand, 0, len(expression.Children)-1)
	argumentSpans := make([]hir.Span, 0, len(expression.Children)-1)
	for index := 1; index < len(expression.Children); index++ {
		argument := &expression.Children[index]
		if !hir.FixedArrayScalarElement(argument.Type) {
			return fmt.Errorf("defer argument %d has unsupported type %q", index-1, argument.Type)
		}
		value, err := l.lowerExpression(argument)
		if err != nil {
			return err
		}
		if l.blockTerminated(l.current) {
			return nil
		}
		if value.Kind == OperandInvalid {
			return fmt.Errorf("defer argument %d has no executable value", index-1)
		}
		arguments = append(arguments, l.freezeScalarOperand(value, argument.Span))
		argumentSpans = append(argumentSpans, argument.Span)
	}
	l.deferred = append(l.deferred, deferredCall{
		scope: l.currentScope, span: expression.Span, callee: expression.Symbol,
		arguments: arguments, argumentSpans: argumentSpans,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "lexical-defer", RemainingCount: 0})
	return nil
}

func (l *functionLowerer) emitDeferredCallsForScope(scope hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return sameHIRSpan(call.scope, scope)
	})
}

func (l *functionLowerer) emitDeferredCallsForFunctionExit(exit hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return spanContains(call.scope, exit) && call.span.StartByte < exit.EndByte
	})
}

func (l *functionLowerer) emitDeferredCallsForLoopExit(loop activeLoop, exit hir.Span) []CleanupAction {
	return l.emitDeferredCalls(func(call deferredCall) bool {
		return spanContains(loop.body, call.scope) && spanContains(call.scope, exit) && call.span.StartByte < exit.EndByte
	})
}

func (l *functionLowerer) emitDeferredCalls(selectCall func(deferredCall) bool) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.deferred) - 1; index >= 0; index-- {
		call := l.deferred[index]
		if !selectCall(call) {
			continue
		}
		l.result.Calls = append(l.result.Calls, CallFact{
			Call: call.span, ArgumentOrder: slices.Clone(call.argumentSpans), NormalSuccessor: true,
		})
		continuation := l.newBlock()
		l.terminate(Terminator{
			Kind: TerminatorCall, Span: call.span, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock,
			Callee: call.callee, Arguments: slices.Clone(call.arguments), DestinationLocal: NoLocal,
		})
		l.current = continuation
		actions = append(actions, CleanupAction{Kind: "defer-call", Binding: call.span})
	}
	return actions
}

func sameHIRSpan(left, right hir.Span) bool {
	return left.Package == right.Package && left.Source == right.Source &&
		left.StartByte == right.StartByte && left.EndByte == right.EndByte
}
