package mir

import (
	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) movePath(expression *hir.Expression, destination string, record bool) {
	binding := l.bindings[expression.Symbol]
	l.appendOperation(Operation{Kind: OperationMove, Span: expression.Span, Symbol: expression.Symbol, Destination: destination})
	l.live[expression.Symbol] = false
	if record {
		l.result.Moves = append(l.result.Moves, MoveFact{
			Binding: binding.span, Operand: expression.Span, Destination: destination, CleanupObligationTransferred: true,
		})
	}
}

func (l *functionLowerer) materializeMove(expression *hir.Expression, value Operand, destination string, record bool) Operand {
	temporary := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Span, DestinationLocal: temporary, Operands: []Operand{value}})
	l.movePath(expression, destination, record)
	return Operand{Kind: OperandLocal, Local: temporary}
}

func spanContains(container, nested hir.Span) bool {
	return container.Package == nested.Package && container.Source == nested.Source &&
		container.StartByte <= nested.StartByte && container.EndByte >= nested.EndByte
}

func (l *functionLowerer) emitCleanup(live map[hir.SymbolID]bool) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.result.Resources) - 1; index >= 0; index-- {
		resource := l.result.Resources[index]
		if !live[resource.Symbol] {
			continue
		}
		l.appendOperation(Operation{Kind: OperationDrop, Span: resource.Binding, Symbol: resource.Symbol, FlagClearedFirst: true})
		live[resource.Symbol] = false
		actions = append(actions, CleanupAction{Kind: "drop", Binding: resource.Binding})
	}
	return actions
}

func (l *functionLowerer) emitCleanupDeclaredWithin(live map[hir.SymbolID]bool, scope hir.Span) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.result.Resources) - 1; index >= 0; index-- {
		resource := l.result.Resources[index]
		if !sameSpanSource(resource.Binding, scope) || resource.Binding.StartByte < scope.StartByte || resource.Binding.EndByte > scope.EndByte || !live[resource.Symbol] {
			continue
		}
		l.appendOperation(Operation{Kind: OperationDrop, Span: resource.Binding, Symbol: resource.Symbol, FlagClearedFirst: true})
		live[resource.Symbol] = false
		actions = append(actions, CleanupAction{Kind: "drop", Binding: resource.Binding})
	}
	return actions
}

func (l *functionLowerer) emitFunctionExitCleanup(live map[hir.SymbolID]bool, exit hir.Span, exitKind string) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.regionStack) - 1; index >= 0; index-- {
		region := l.regionStack[index]
		cleanup := l.emitCleanupDeclaredWithin(live, region.span)
		actions = append(actions, cleanup...)
		l.appendOperation(Operation{Kind: OperationRegionReset, Span: region.span, Symbol: region.symbol, DestinationLocal: NoLocal, ExitKind: exitKind})
		actions = append(actions, CleanupAction{Kind: "region-reset", Binding: region.span})
		fact := &l.result.RegionScopes[region.factIndex]
		fact.EarlyResets = append(fact.EarlyResets, RegionResetFact{
			Exit: exit, ExitKind: exitKind, CleanupBeforeReset: cleanup, Reset: region.span,
		})
	}
	actions = append(actions, l.emitCleanup(live)...)
	return actions
}

func (l *functionLowerer) emitLoopControlCleanup(live map[hir.SymbolID]bool, loop activeLoop, exit hir.Span, exitKind string) []CleanupAction {
	actions := make([]CleanupAction, 0)
	for index := len(l.regionStack) - 1; index >= loop.regionDepth; index-- {
		region := l.regionStack[index]
		cleanup := l.emitCleanupDeclaredWithin(live, region.span)
		actions = append(actions, cleanup...)
		l.appendOperation(Operation{Kind: OperationRegionReset, Span: region.span, Symbol: region.symbol, DestinationLocal: NoLocal, ExitKind: exitKind})
		actions = append(actions, CleanupAction{Kind: "region-reset", Binding: region.span})
		fact := &l.result.RegionScopes[region.factIndex]
		fact.EarlyResets = append(fact.EarlyResets, RegionResetFact{
			Exit: exit, ExitKind: exitKind, CleanupBeforeReset: cleanup, Reset: region.span,
		})
	}
	actions = append(actions, l.emitCleanupDeclaredWithin(live, loop.body)...)
	return actions
}

func cleanupContainsDrop(actions []CleanupAction) bool {
	for _, action := range actions {
		if action.Kind == "drop" {
			return true
		}
	}
	return false
}

func (l *functionLowerer) liveResourcePath(expression *hir.Expression) *hir.Expression {
	if expression == nil {
		return nil
	}
	switch expression.Kind {
	case hir.ExpressionGroup, hir.ExpressionUnsafe:
		if len(expression.Children) == 1 {
			return l.liveResourcePath(&expression.Children[0])
		}
		return nil
	case hir.ExpressionBlock:
		if expression.Block != nil && expression.Block.Tail != nil {
			return l.liveResourcePath(expression.Block.Tail)
		}
		return nil
	case hir.ExpressionPath:
		if expression.Symbol == hir.NoSymbol || !l.live[expression.Symbol] {
			return nil
		}
		binding, found := l.bindings[expression.Symbol]
		if found && l.properties.NeedsDrop(binding.typ) {
			return expression
		}
	}
	return nil
}
