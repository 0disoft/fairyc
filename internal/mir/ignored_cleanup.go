package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerIgnoredExpression(expression *hir.Expression) error {
	value, err := l.lowerExpression(expression)
	if err != nil || l.blockTerminated(l.current) || expression == nil {
		return err
	}
	if !l.properties.NeedsDrop(expression.Type) {
		return nil
	}
	// As in semantic resourcePlaceExpression, reading a named local (possibly
	// grouped) leaves its obligation with that local until lexical cleanup.
	place := expression
	for place.Kind == hir.ExpressionGroup && len(place.Children) == 1 {
		place = &place.Children[0]
	}
	if place.Kind == hir.ExpressionPath {
		return nil
	}
	if value.Kind == OperandInvalid {
		return fmt.Errorf("ignored resource expression has no value")
	}
	if path := l.liveResourcePath(expression); path != nil {
		value = l.materializeMove(path, value, "ignored-value", true)
	}
	// Give the temporary an explicit obligation so the existing MIR verifier,
	// interpreter and backends all use the normal typed structural drop glue.
	local := l.newTemporary()
	symbol := hir.SymbolID(local)
	l.result.Resources = append(l.result.Resources, ResourceLocal{
		Symbol: symbol, Type: expression.Type, Binding: expression.Span,
	})
	l.appendOperation(Operation{
		Kind: OperationInitialize, Span: expression.Span, Symbol: symbol,
		DestinationLocal: local, Operands: []Operand{value},
	})
	l.appendOperation(Operation{
		Kind: OperationDrop, Span: expression.Span, Symbol: symbol, FlagClearedFirst: true,
	})
	l.result.EliminatedForms = append(l.result.EliminatedForms, EliminatedFormFact{Form: "implicit-drop", RemainingCount: 0})
	return nil
}
