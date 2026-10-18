package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerScalarCast(expression *hir.Expression) (Operand, error) {
	if expression.Operator != "as" || expression.TargetType != expression.Type || len(expression.Children) != 1 ||
		!hir.ValuePreservingIntegerCast(expression.Children[0].Type, expression.Type) {
		return Operand{}, fmt.Errorf("malformed value-preserving integer cast")
	}
	operand, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if operand.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("value-preserving integer cast has no operand value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationScalarCast, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{operand}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerRawAddressCast(expression *hir.Expression) (Operand, error) {
	if expression.Operator != "as" || expression.TargetType != expression.Type || expression.UnsafeScope == nil || len(expression.Children) != 1 {
		return Operand{}, fmt.Errorf("malformed raw-address cast")
	}
	borrow := &expression.Children[0]
	if borrow.Type == hir.SharedBorrow(hir.TypeStr) && expression.Type == hir.RawPointer(hir.TypeU8, false) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("string data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("str-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationStrDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	byteSliceElement, byteSliceMutable, byteSliceBorrow := hir.BorrowElement(borrow.Type)
	bytePointerElement, bytePointerMutable, bytePointer := hir.RawPointerElement(expression.Type)
	if byteSliceBorrow && byteSliceElement == hir.TypeByteSlice &&
		bytePointer && bytePointerElement == hir.TypeU8 && (byteSliceMutable || !bytePointerMutable) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandByteString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("byte-slice data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("byte-slice-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationByteSliceDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	if borrow.Type == hir.SharedBorrow(hir.TypeCStr) && expression.Type == hir.RawPointer(hir.TypeU8, false) {
		value, err := l.lowerExpression(borrow)
		if err != nil {
			return Operand{}, err
		}
		if value.Kind != OperandCString && value.Kind != OperandLocal {
			return Operand{}, fmt.Errorf("C string data address source has invalid operand kind %q", value.Kind)
		}
		destination := l.newTemporary()
		entity := sourceEntity("cstr-data-address", expression.Span)
		l.appendOperation(Operation{
			Kind: OperationCStrDataAddress, Entity: entity, Span: expression.Span,
			DestinationLocal: destination,
			Operands:         []Operand{value},
			Type:             expression.Type,
			OwnerType:        borrow.Type,
			Operator:         expression.Operator,
			UnsafeScope:      *expression.UnsafeScope,
		})
		l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
			Entity:           entity,
			SourceSpanEntity: sourceEntity("span", expression.Span),
			ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
		})
		return Operand{Kind: OperandLocal, Local: destination}, nil
	}
	if borrow.Kind == hir.ExpressionPath && borrow.Symbol != hir.NoSymbol {
		sourceElement, sourceMutable, sourceBorrow := hir.BorrowElement(borrow.Type)
		targetElement, targetMutable, targetPointer := hir.RawPointerElement(expression.Type)
		if sourceBorrow && targetPointer && (targetElement == sourceElement || targetElement == hir.TypeVoid) &&
			(sourceMutable || !targetMutable) {
			value, err := l.lowerExpression(borrow)
			if err != nil {
				return Operand{}, err
			}
			if value.Kind != OperandLocal {
				return Operand{}, fmt.Errorf("raw borrow pointer source has invalid operand kind %q", value.Kind)
			}
			destination := l.newTemporary()
			entity := sourceEntity("raw-borrow-pointer", expression.Span)
			l.appendOperation(Operation{
				Kind: OperationRawBorrowPointer, Entity: entity, Span: expression.Span,
				DestinationLocal: destination,
				Operands:         []Operand{value},
				Type:             expression.Type,
				OwnerType:        borrow.Type,
				Operator:         expression.Operator,
				UnsafeScope:      *expression.UnsafeScope,
			})
			l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
				Entity:           entity,
				SourceSpanEntity: sourceEntity("span", expression.Span),
				ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
			})
			return Operand{Kind: OperandLocal, Local: destination}, nil
		}
	}
	if borrow.Kind != hir.ExpressionPrefix || (borrow.Operator != "&" && borrow.Operator != "&mut") || len(borrow.Children) != 1 {
		return Operand{}, fmt.Errorf("raw-address cast source is not a direct borrow")
	}
	place := &borrow.Children[0]
	if place.Kind != hir.ExpressionPath || place.Symbol == hir.NoSymbol {
		return Operand{}, fmt.Errorf("raw-address cast source is not a direct local place")
	}
	element, _, pointer := hir.RawPointerElement(expression.Type)
	if !pointer || element != place.Type && element != hir.TypeVoid {
		return Operand{}, fmt.Errorf("raw-address cast type %q does not match place type %q", expression.Type, place.Type)
	}
	destination := l.newTemporary()
	entity := sourceEntity("raw-address", expression.Span)
	l.appendOperation(Operation{
		Kind: OperationRawAddress, Entity: entity, Span: expression.Span,
		DestinationLocal: destination,
		Operands:         []Operand{{Kind: OperandLocal, Local: LocalID(place.Symbol)}},
		Type:             expression.Type,
		OwnerType:        place.Type,
		Operator:         borrow.Operator,
		UnsafeScope:      *expression.UnsafeScope,
	})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
		Entity:           entity,
		SourceSpanEntity: sourceEntity("span", expression.Span),
		ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerBorrowedViewFromRawParts(expression *hir.Expression) (Operand, error) {
	if expression.UnsafeScope == nil || len(expression.Children) != 3 {
		return Operand{}, fmt.Errorf("malformed borrowed-view raw-parts construction")
	}
	operationKind := OperationInvalid
	entityKind := ""
	switch {
	case expression.Operator == "str::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeStr):
		operationKind = OperationStrFromRawParts
		entityKind = "str-from-raw-parts"
	case expression.Operator == "CStr::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeCStr):
		operationKind = OperationCStrFromRawParts
		entityKind = "cstr-from-raw-parts"
	case expression.Operator == "byte_slice::from_raw_parts" && expression.Type == hir.SharedBorrow(hir.TypeByteSlice):
		operationKind = OperationByteSliceFromRawParts
		entityKind = "byte-slice-from-raw-parts"
	case expression.Operator == "byte_slice::from_raw_parts_mut" && expression.Type == hir.MutableBorrow(hir.TypeByteSlice):
		operationKind = OperationMutByteSliceFromRawParts
		entityKind = "mut-byte-slice-from-raw-parts"
	default:
		return Operand{}, fmt.Errorf("unsupported borrowed-view raw-parts constructor %q with result %q", expression.Operator, expression.Type)
	}

	operands := make([]Operand, 0, len(expression.Children))
	for index := range expression.Children {
		operand, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if operand.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("borrowed-view raw-parts argument %d has no value", index)
		}
		operands = append(operands, operand)
	}

	destination := l.newTemporary()
	entity := sourceEntity(entityKind, expression.Span)
	l.appendOperation(Operation{
		Kind:             operationKind,
		Entity:           entity,
		Span:             expression.Span,
		DestinationLocal: destination,
		Operands:         operands,
		Type:             expression.Type,
		OwnerType:        expression.Children[0].Type,
		Operator:         expression.Operator,
		UnsafeScope:      *expression.UnsafeScope,
	})
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{
		Entity:           entity,
		SourceSpanEntity: sourceEntity("span", expression.Span),
		ScopeEntity:      sourceEntity("unsafe-scope", *expression.UnsafeScope),
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
