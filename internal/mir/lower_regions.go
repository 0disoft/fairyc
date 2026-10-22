package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerRegion(expression *hir.Expression) (Operand, error) {
	if expression.Symbol == hir.NoSymbol || len(expression.Children) != 1 || expression.Block == nil {
		return Operand{}, fmt.Errorf("malformed lexical region")
	}
	arena := &expression.Children[0]
	if arena.Kind != hir.ExpressionPrefix || arena.Operator != "&mut" || len(arena.Children) != 1 || arena.Children[0].Kind != hir.ExpressionPath {
		return Operand{}, fmt.Errorf("executable lexical region source must be a direct &mut local")
	}
	if _, active := l.regions[expression.Symbol]; active {
		return Operand{}, fmt.Errorf("region %d is entered twice", expression.Symbol)
	}
	arenaOwner, err := l.lowerExpression(&arena.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if arenaOwner.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("executable lexical region source must name one Arena local")
	}
	factIndex := len(l.result.RegionScopes)
	l.result.RegionScopes = append(l.result.RegionScopes, RegionScopeFact{Region: expression.Span})
	l.regions[expression.Symbol] = factIndex
	l.regionStack = append(l.regionStack, activeRegion{symbol: expression.Symbol, span: expression.Span, factIndex: factIndex})
	l.appendOperation(Operation{
		Kind: OperationRegionEnter, Span: expression.Span, Symbol: expression.Symbol,
		DestinationLocal: NoLocal, Operands: []Operand{arenaOwner},
		OwnerType: arena.Children[0].Type, Type: arena.Type, Operator: "&mut",
	})
	_, terminated, err := l.lowerNestedBlock(*expression.Block, false)
	if err != nil {
		return Operand{}, err
	}
	if err := l.leaveLexicalRegion(expression.Symbol); err != nil {
		return Operand{}, err
	}
	if terminated {
		return Operand{}, nil
	}
	cleanup := l.emitCleanupDeclaredWithin(l.live, expression.Span)
	l.appendOperation(Operation{Kind: OperationRegionReset, Span: expression.Span, Symbol: expression.Symbol, DestinationLocal: NoLocal, ExitKind: "fallthrough"})
	fact := &l.result.RegionScopes[factIndex]
	fact.CleanupBeforeReset = cleanup
	fact.Reset = expression.Span
	return Operand{}, nil
}

func (l *functionLowerer) lowerRegionPlace(expression *hir.Expression) (Operand, error) {
	receiver := &expression.Children[0]
	if receiver.Kind != hir.ExpressionPath || receiver.Symbol == hir.NoSymbol {
		return Operand{}, fmt.Errorf("Region.place receiver is not one region capability")
	}
	factIndex, active := l.regions[receiver.Symbol]
	if !active {
		return Operand{}, fmt.Errorf("Region.place uses inactive region %d", receiver.Symbol)
	}
	valueExpression := &expression.Children[1]
	value, err := l.lowerExpression(valueExpression)
	if err != nil {
		return Operand{}, err
	}
	if value.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Region.place value has no executable operand")
	}
	operator := ""
	if expression.Operator != "place" {
		operator = expression.Operator
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Operator: operator,
		Kind:     OperationRegionPlace, Span: expression.Span, Symbol: receiver.Symbol,
		DestinationLocal: destination, Operands: []Operand{value}, Type: expression.Type, OwnerType: valueExpression.Type,
	})
	l.result.RegionScopes[factIndex].Placements = append(l.result.RegionScopes[factIndex].Placements, expression.Span)
	regionEntity := fmt.Sprintf("region:%d", receiver.Symbol)
	operationEntity := fmt.Sprintf("operation:region-place:%d:%d", expression.Span.StartByte, expression.Span.EndByte)
	l.result.RegionValues = append(l.result.RegionValues, RegionValue{
		Entity: fmt.Sprintf("local:region-value:%d", destination), OperationEntity: operationEntity,
		RegionEntity: regionEntity, NeedsDrop: l.properties.NeedsDrop(valueExpression.Type),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) leaveLexicalRegion(symbol hir.SymbolID) error {
	if len(l.regionStack) == 0 || l.regionStack[len(l.regionStack)-1].symbol != symbol {
		return fmt.Errorf("region %d does not close in lexical stack order", symbol)
	}
	l.regionStack = l.regionStack[:len(l.regionStack)-1]
	delete(l.regions, symbol)
	return nil
}
