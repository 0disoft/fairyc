package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerWhile(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 {
		return Operand{}, fmt.Errorf("while expression has %d children", len(expression.Children))
	}
	headerLive := cloneLive(l.live)
	headerBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = headerBlock
	l.live = cloneLive(headerLive)
	condition, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	if !sameLiveState(headerLive, l.live) {
		return Operand{}, fmt.Errorf("while condition changes resource state after Flow validation")
	}
	bodyBlock := l.newBlock()
	exitBlock := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Children[0].Span,
		Successors: []BlockID{bodyBlock, exitBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchOperands: []Operand{condition},
	})

	l.current = bodyBlock
	l.live = cloneLive(headerLive)
	l.loopStack = append(l.loopStack, activeLoop{
		header: headerBlock, continueTarget: headerBlock, exit: exitBlock, body: expression.Children[1].Span,
		regionDepth: len(l.regionStack), headerLive: cloneLive(headerLive),
	})
	if _, err := l.lowerExpression(&expression.Children[1]); err != nil {
		l.loopStack = l.loopStack[:len(l.loopStack)-1]
		return Operand{}, err
	}
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
	if !l.blockTerminated(l.current) {
		cleanup := l.emitCleanupDeclaredWithin(l.live, expression.Children[1].Span)
		if cleanupContainsDrop(cleanup) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		if !matchesLoopHeader(headerLive, l.live) {
			return Operand{}, fmt.Errorf("while back-edge changes resource state after Flow validation")
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: expression.Children[1].Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = exitBlock
	l.live = cloneLive(headerLive)
	return Operand{}, nil
}

func (l *functionLowerer) lowerLoopControl(kind hir.StatementKind, exit hir.Span) error {
	if len(l.loopStack) == 0 {
		return fmt.Errorf("%s has no enclosing loop after type validation", kind)
	}
	loop := l.loopStack[len(l.loopStack)-1]
	actions := l.emitDeferredCallsForLoopExit(loop, exit)
	actions = append(actions, l.emitLoopControlCleanup(l.live, loop, exit, kind.String())...)
	if !matchesLoopHeader(loop.headerLive, l.live) {
		return fmt.Errorf("%s edge changes resource state after Flow validation", kind)
	}
	l.result.CleanupPaths = append(l.result.CleanupPaths, CleanupPathFact{Exit: exit, ExitKind: kind.String(), Actions: actions})
	if cleanupContainsDrop(actions) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	target := loop.continueTarget
	if kind == hir.StatementBreak {
		target = loop.exit
	}
	l.terminate(Terminator{Kind: TerminatorGoto, Span: exit, Successors: []BlockID{target}, UnwindSuccessor: NoBlock})
	return nil
}

func (l *functionLowerer) lowerFor(loop *hir.ForLoop) error {
	if loop == nil {
		return fmt.Errorf("for statement has no typed loop")
	}
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "classic-for", RemainingCount: 0})
	if loop.Initializer != nil {
		terminated, err := l.lowerStatements([]hir.Statement{*loop.Initializer})
		if err != nil {
			return err
		}
		if terminated {
			return fmt.Errorf("for initializer terminates control flow")
		}
	}

	headerLive := cloneLive(l.live)
	headerBlock := l.newBlock()
	bodyBlock := l.newBlock()
	postBlock := l.newBlock()
	exitBlock := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = headerBlock
	l.live = cloneLive(headerLive)
	conditionExpression := loop.Condition
	if conditionExpression == nil {
		conditionExpression = &hir.Expression{Kind: hir.ExpressionLiteral, Type: hir.TypeBool, Span: loop.Span, Value: "true", Cooked: "true", Symbol: hir.NoSymbol}
	}
	condition, err := l.lowerExpression(conditionExpression)
	if err != nil {
		return err
	}
	if !sameLiveState(headerLive, l.live) {
		return fmt.Errorf("for condition changes resource state after Flow validation")
	}
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: conditionExpression.Span,
		Successors: []BlockID{bodyBlock, exitBlock}, UnwindSuccessor: NoBlock,
		SwitchKind: "bool", SwitchType: hir.TypeBool, SwitchOperands: []Operand{condition},
	})

	l.current = bodyBlock
	l.live = cloneLive(headerLive)
	l.loopStack = append(l.loopStack, activeLoop{
		header: headerBlock, continueTarget: postBlock, exit: exitBlock, body: loop.Body.Span,
		regionDepth: len(l.regionStack), headerLive: cloneLive(headerLive),
	})
	_, bodyTerminated, err := l.lowerNestedBlock(loop.Body, false)
	l.loopStack = l.loopStack[:len(l.loopStack)-1]
	if err != nil {
		return err
	}
	if !bodyTerminated && !l.blockTerminated(l.current) {
		cleanup := l.emitCleanupDeclaredWithin(l.live, loop.Body.Span)
		if cleanupContainsDrop(cleanup) {
			l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
		}
		if !matchesLoopHeader(headerLive, l.live) {
			return fmt.Errorf("for body back-edge changes resource state after Flow validation")
		}
		l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Body.Span, Successors: []BlockID{postBlock}, UnwindSuccessor: NoBlock})
	}

	l.current = postBlock
	l.live = cloneLive(headerLive)
	if loop.Post != nil {
		terminated, lowerErr := l.lowerStatements([]hir.Statement{*loop.Post})
		if lowerErr != nil {
			return lowerErr
		}
		if terminated {
			return fmt.Errorf("for post clause terminates control flow")
		}
	}
	if !sameLiveState(headerLive, l.live) {
		return fmt.Errorf("for post clause changes resource state after Flow validation")
	}
	l.terminate(Terminator{Kind: TerminatorGoto, Span: loop.Span, Successors: []BlockID{headerBlock}, UnwindSuccessor: NoBlock})

	l.current = exitBlock
	l.live = cloneLive(headerLive)
	cleanup := l.emitCleanupDeclaredWithin(l.live, loop.Span)
	if cleanupContainsDrop(cleanup) {
		l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	}
	return nil
}

func cloneLive(input map[hir.SymbolID]bool) map[hir.SymbolID]bool {
	result := make(map[hir.SymbolID]bool, len(input))
	for symbol, live := range input {
		result[symbol] = live
	}
	return result
}

func matchesLoopHeader(header, actual map[hir.SymbolID]bool) bool {
	for symbol, live := range header {
		if actual[symbol] != live {
			return false
		}
	}
	for symbol, live := range actual {
		if _, existed := header[symbol]; !existed && live {
			return false
		}
	}
	return true
}
