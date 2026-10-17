package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerResultVariant(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || (expression.Variant != "ok" && expression.Variant != "err") {
		return Operand{}, fmt.Errorf("malformed Result variant %q with %d payloads", expression.Variant, len(expression.Children))
	}
	payload, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if payload.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("Result::%s payload has no value", expression.Variant)
	}
	if path := l.liveResourcePath(&expression.Children[0]); path != nil {
		payload = l.materializeMove(path, payload, "result-payload", true)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationResultConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{payload}, Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerOptionVariant(expression *hir.Expression) (Operand, error) {
	if expression.Variant != "some" && expression.Variant != "none" {
		return Operand{}, fmt.Errorf("malformed Option variant %q", expression.Variant)
	}
	operands := make([]Operand, 0, 1)
	if expression.Variant == "some" {
		if len(expression.Children) != 1 {
			return Operand{}, fmt.Errorf("Option::Some has %d payloads", len(expression.Children))
		}
		payload, err := l.lowerExpression(&expression.Children[0])
		if err != nil {
			return Operand{}, err
		}
		if payload.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("Option::Some payload has no value")
		}
		if path := l.liveResourcePath(&expression.Children[0]); path != nil {
			payload = l.materializeMove(path, payload, "option-payload", true)
		}
		operands = append(operands, payload)
	} else if len(expression.Children) != 0 {
		return Operand{}, fmt.Errorf("Option::None has %d payloads", len(expression.Children))
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationOptionConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerEnumVariant(expression *hir.Expression) (Operand, error) {
	definition := l.properties.Enum(expression.Type)
	if definition == nil || expression.Variant == "" || len(expression.Children) != len(expression.Fields) {
		return Operand{}, fmt.Errorf("malformed enum variant %q with %d values and %d fields", expression.Variant, len(expression.Children), len(expression.Fields))
	}
	variant, found := enumVariantDefinition(definition, expression.Variant)
	if !found || !sameNames(expression.Fields, enumFieldNames(variant.Fields)) {
		return Operand{}, fmt.Errorf("enum variant %q fields do not match type %q", expression.Variant, expression.Type)
	}
	operands := make([]Operand, 0, len(expression.Children))
	for index := range expression.Children {
		value, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if value.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("enum payload field %q has no value", expression.Fields[index])
		}
		operands = append(operands, value)
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationEnumConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Fields: slices.Clone(expression.Fields), Variant: expression.Variant, Type: expression.Type,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
