package mir

import (
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func fieldStoreScalar(typ hir.Type) bool {
	return typ == hir.TypeI32 || typ == hir.TypeI64 || typ == hir.TypeU8 || typ == hir.TypeU32 || typ == hir.TypeUSize || typ == hir.TypeBool
}

func (l *functionLowerer) lowerFieldStore(statement hir.Statement) error {
	target := statement.Target
	unsupported := &Unsupported{Feature: "field assignment without executable field storage", Span: target.Span}
	if len(target.Children) != 1 || target.Operator == "" ||
		statement.Expression == nil || statement.Expression.Type != target.Type {
		return unsupported
	}
	owner := &target.Children[0]
	var fields []string
	for {
		if owner.Kind == hir.ExpressionGroup && len(owner.Children) == 1 {
			owner = &owner.Children[0]
			continue
		}
		if owner.Kind != hir.ExpressionFieldExpression {
			break
		}
		if len(owner.Children) != 1 || owner.Operator == "" {
			return unsupported
		}
		fields = append(fields, owner.Operator)
		owner = &owner.Children[0]
	}
	slices.Reverse(fields)
	indexed := owner.Kind == hir.ExpressionViewIndex || owner.Kind == hir.ExpressionArrayIndex
	if indexed {
		borrowed, ok := mutableIndexedOwner(*owner)
		if !ok {
			return unsupported
		}
		owner = &borrowed
	} else if owner.Kind != hir.ExpressionPath {
		return unsupported
	}
	if _, mutable, borrowed := hir.BorrowElement(owner.Type); borrowed && !mutable {
		return unsupported
	}
	if len(fields) > 0 && ((!hir.PlainViewField(target.Type) && !l.properties.inlineArrayField(target.Type) && !plainStoredStruct(target.Type, l.properties.Struct)) || !plainStoredStruct(fieldAggregateType(owner.Type), l.properties.Struct)) {
		return unsupported
	}
	receiver, err := l.lowerExpression(owner)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if len(fields) > 0 || hir.IntegerTransportArray(target.Type) || l.properties.inlineArrayField(target.Type) || plainStoredStruct(target.Type, l.properties.Struct) {
		if _, _, borrowed := hir.BorrowElement(owner.Type); borrowed {
			saved := l.newTemporary()
			l.appendOperation(Operation{Kind: OperationAssign, Span: target.Span, DestinationLocal: saved, Operands: []Operand{receiver}})
			receiver = Operand{Kind: OperandLocal, Local: saved}
		}
	}
	operator := statement.Operator
	var prior Operand
	if operator != "" && operator != "=" {
		if !hir.CompoundAssignmentType(operator, target.Type) {
			return unsupported
		}
		// Read the original scalar before RHS effects, without copying a place
		// for the eventual store. Ordinary field reads keep their existing lowering.
		if indexed {
			prior, err = l.lowerSelectedFieldRead(receiver, owner.Type, append(slices.Clone(fields), target.Operator), target.Span)
		} else {
			prior, err = l.lowerExpression(target)
		}
		if err != nil {
			return err
		}
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if operator != "" && operator != "=" {
		binary, _ := hir.CompoundAssignmentBinary(operator)
		combined := &hir.Expression{Kind: hir.ExpressionBinary, Type: target.Type, Span: statement.Span,
			Operator: binary, Children: []hir.Expression{*target, *statement.Expression}}
		value, err = l.lowerBinaryOperands(combined, prior, value)
		if err != nil {
			return err
		}
	}
	op := Operation{Kind: OperationFieldStore, Span: target.Span, Symbol: hir.NoSymbol,
		DestinationLocal: NoLocal, OwnerType: owner.Type, Type: target.Type, Field: target.Operator, Fields: fields,
		Operands: []Operand{receiver, value}}
	if !ValidFieldStoreShape(op) && !ValidOneWordStructFieldStoreShape(op, l.properties.Struct(target.Type)) &&
		!ValidPlainStructFieldStoreShape(op, l.properties.Struct) && !ValidCEnumFieldStoreShape(op, l.properties.Enum(target.Type)) {
		return unsupported
	}
	l.appendOperation(op)
	return nil
}

func validFieldStoreCommonShape(op Operation) bool {
	_, mutable, borrowed := hir.BorrowElement(op.OwnerType)
	return op.OwnerType != hir.ParseType("") && op.OwnerType != hir.TypeInvalid && (!borrowed || mutable) &&
		op.Field != "" && op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 &&
		len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == hir.ParseType("") && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}

// ValidFieldStoreShape checks scalar store representation. Nested declarations
// and path types are checked separately by ResolveNestedFieldStore.
func ValidFieldStoreShape(op Operation) bool {
	return fieldStoreScalar(op.Type) && validFieldStoreCommonShape(op) && scalarOperandMatches(op.Operands[1], op.Type)
}

// ValidCEnumFieldStoreShape admits a typed local into one direct enum field.
// The verifier separately checks the exact owner field and nominal operand types.
func ValidCEnumFieldStoreShape(op Operation, definition *EnumType) bool {
	return len(op.Fields) == 0 && definition != nil && definition.Type == op.Type &&
		hir.CEnumVariantCapacity(definition.CBase) > 0 && validFieldStoreCommonShape(op) &&
		op.Operands[1].Kind == OperandLocal && op.Operands[1].Local >= 0
}

// ValidOneWordStructFieldStoreShape admits only an inert, non-FFI usize wrapper
// supplied by a typed local. The verifier still checks the owner field and both
// local types against program declarations before executable use.
func ValidOneWordStructFieldStoreShape(op Operation, definition *StructType) bool {
	return len(op.Fields) == 0 && definition != nil && definition.Type == op.Type && !definition.ReprC &&
		len(definition.Fields) == 1 && definition.Fields[0].Type == hir.TypeUSize &&
		validFieldStoreCommonShape(op) && op.Operands[1].Kind == OperandLocal && op.Operands[1].Local >= 0
}

func mutableIndexedOwner(index hir.Expression) (hir.Expression, bool) {
	if len(index.Children) != 2 {
		return hir.Expression{}, false
	}
	if index.Kind == hir.ExpressionArrayIndex {
		element, length, array := hir.FixedArrayParts(index.Children[0].Type)
		if !array || element != index.Type || !hir.ExecutableFixedArrayShape(element, length) {
			return hir.Expression{}, false
		}
		view := hir.MutableBorrow(hir.Slice(element))
		if !hir.ArrayViewBorrow(index.Children[0].Type, view, "&mut") {
			return hir.Expression{}, false
		}
		index.Children = []hir.Expression{
			{Kind: hir.ExpressionArrayViewBorrow, Type: view, Span: index.Children[0].Span,
				Symbol: hir.NoSymbol, Operator: "&mut", Children: []hir.Expression{index.Children[0]}},
			index.Children[1],
		}
	}
	element, nominal := hir.StructViewElement(index.Children[0].Type)
	if !nominal || !hir.MutableIndexedView(index.Children[0].Type) || element != index.Type {
		return hir.Expression{}, false
	}
	return hir.Expression{Kind: hir.ExpressionViewElementBorrow, Type: hir.MutableBorrow(element), Span: index.Span, Symbol: hir.NoSymbol, Operator: "[]", Children: index.Children}, true
}

// Compound assignment reads through the already selected receiver. Index
// effects run once and the prior scalar is captured before RHS effects.
func (l *functionLowerer) lowerSelectedFieldRead(receiver Operand, typ hir.Type, fields []string, at hir.Span) (Operand, error) {
	for _, name := range fields {
		definition := l.properties.Struct(fieldAggregateType(typ))
		if definition == nil {
			return Operand{}, &Unsupported{Feature: "indexed field owner", Span: at}
		}
		field, found := structField(definition.Fields, name)
		if !found {
			return Operand{}, &Unsupported{Feature: "indexed field name", Span: at}
		}
		local := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationFieldRead, Span: at, DestinationLocal: local, Operands: []Operand{receiver}, OwnerType: typ, Type: field.Type, Field: name})
		receiver = Operand{Kind: OperandLocal, Local: local}
		typ = field.Type
	}
	return receiver, nil
}
