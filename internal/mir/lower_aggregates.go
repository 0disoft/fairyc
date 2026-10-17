package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerAggregateConstruction(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != len(expression.Fields) || expression.Type == hir.TypeInvalid {
		return Operand{}, fmt.Errorf("malformed aggregate construction with %d values and %d fields", len(expression.Children), len(expression.Fields))
	}
	isResource := l.properties.IsResource(expression.Type)
	isStruct := l.properties.IsStruct(expression.Type)
	if isResource && len(expression.Children) == 0 {
		return Operand{}, fmt.Errorf("empty resource construction has no MIR support")
	}
	if !isResource && !isStruct {
		return Operand{}, fmt.Errorf("aggregate construction type %q has no MIR definition", expression.Type)
	}
	fields := l.properties.constructionFields[expression.Type]
	if len(fields) != len(expression.Fields) {
		return Operand{}, fmt.Errorf("aggregate construction fields do not match definition %q", expression.Type)
	}
	positions := make(map[string]int, len(fields))
	for index, name := range fields {
		positions[name] = index
	}
	operands := make([]Operand, len(fields))
	for index := range expression.Children {
		value, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if value.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("aggregate field %q has no value", expression.Fields[index])
		}
		if path := l.liveResourcePath(&expression.Children[index]); isResource && path != nil {
			value = l.materializeMove(path, value, "resource-field", true)
		}
		// Evaluate written initializers in order; snapshot copyable values before
		// later initializers can mutate their source locals. Only the final
		// construction operand list follows declaration order.
		if _, sourceLocal := l.bindings[hir.SymbolID(value.Local)]; value.Kind == OperandLocal && sourceLocal && !l.properties.NeedsDrop(expression.Children[index].Type) {
			value = l.freezeScalarOperand(value, expression.Children[index].Span)
		}
		position, found := positions[expression.Fields[index]]
		if !found || operands[position].Kind != OperandInvalid {
			return Operand{}, fmt.Errorf("aggregate construction has unknown or duplicate field %q", expression.Fields[index])
		}
		operands[position] = value
	}
	destination := l.newTemporary()
	kind := OperationStructConstruct
	if isResource {
		kind = OperationResourceConstruct
	}
	l.appendOperation(Operation{
		Kind: kind, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Fields: slices.Clone(fields), Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerFieldRead(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator == "" {
		return Operand{}, fmt.Errorf("malformed field read %q with %d receivers", expression.Operator, len(expression.Children))
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("field read %q has no receiver value", expression.Operator)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationFieldRead, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
