package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerPropagate(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("propagate expression has %d children", len(expression.Children))
	}
	operand := &expression.Children[0]
	resultValue, err := l.lowerExpression(operand)
	if err != nil {
		return Operand{}, err
	}
	if resultValue.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("propagate operand has no value")
	}
	if path := l.liveResourcePath(operand); path != nil {
		resultValue = l.materializeMove(path, resultValue, "result-branch", false)
	}
	l.appendOperation(Operation{Kind: OperationResultSwitch, Span: expression.Span, Operands: []Operand{resultValue}})
	success := l.newBlock()
	errorBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{success, errorBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "result", SwitchOperands: []Operand{resultValue},
	})

	errorLive := cloneLive(l.live)
	l.current = errorBlock
	_, errorType, operandIsResult := hir.ResultElements(operand.Type)
	_, returnErrorType, returnIsResult := hir.ResultElements(l.source.ResultType)
	if !operandIsResult || !returnIsResult || errorType != returnErrorType {
		return Operand{}, fmt.Errorf("propagate error type %q does not match function result %q", errorType, l.source.ResultType)
	}
	errorPayload := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultPayload, Span: expression.Span, DestinationLocal: errorPayload,
		Operands: []Operand{resultValue}, Variant: "err", MovePayload: l.properties.NeedsDrop(errorType), Type: errorType,
	})
	propagatedResult := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultConstruct, Span: expression.Span, DestinationLocal: propagatedResult,
		Operands: []Operand{{Kind: OperandLocal, Local: errorPayload}}, Variant: "err", Type: l.source.ResultType,
	})
	errorActions := l.emitDeferredCallsForFunctionExit(expression.Span)
	errorActions = append(errorActions, l.emitFunctionExitCleanup(errorLive, expression.Span, "propagate")...)
	l.terminate(Terminator{
		Kind: TerminatorReturn, Span: expression.Span, UnwindSuccessor: NoBlock,
		ReturnValue: Operand{Kind: OperandLocal, Local: propagatedResult},
	})
	question := hir.Span{Package: expression.Span.Package, Source: expression.Span.Source, StartByte: max(expression.Span.StartByte, expression.Span.EndByte-1), EndByte: expression.Span.EndByte}
	l.result.ResultBranches = append(l.result.ResultBranches, ResultBranchFact{
		Question: question, ResultExpression: operand.Span, EvaluationCount: 1,
		SuccessPayload: "Move", ErrorPayload: "Move", ResultTemporaryDeadOn: []string{"success", "error"},
		ErrorCleanupActions: errorActions,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "question", RemainingCount: 0})
	l.current = success
	payload := l.newTemporary()
	movePayload := false
	successType, _, resultType := hir.ResultElements(operand.Type)
	if resultType {
		movePayload = l.properties.NeedsDrop(successType)
	}
	l.appendOperation(Operation{
		Kind: OperationResultPayload, Span: expression.Span, DestinationLocal: payload,
		Operands: []Operand{resultValue}, Variant: "ok", MovePayload: movePayload, Type: successType,
	})
	return Operand{Kind: OperandLocal, Local: payload}, nil
}

func (l *functionLowerer) lowerReturn(expression *hir.Expression, exit hir.Span) error {
	value, err := l.lowerExpression(expression)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if path := l.liveResourcePath(expression); path != nil {
		value = l.materializeMove(path, value, "return-value", true)
	}
	actions := l.emitDeferredCallsForFunctionExit(exit)
	actions = append(actions, l.emitFunctionExitCleanup(l.live, exit, "return")...)
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: "return", Actions: actions})
	if cleanupContainsDrop(actions) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	l.terminate(Terminator{Kind: TerminatorReturn, Span: exit, UnwindSuccessor: NoBlock, ReturnValue: value})
	return nil
}
