package mir

import "github.com/0disoft/fairyc/internal/hir"

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
	for owner.Kind == "group" && len(owner.Children) == 1 {
		owner = &owner.Children[0]
	}
	if owner.Kind != "path" {
		return unsupported
	}
	if _, mutable, borrowed := hir.BorrowElement(owner.Type); borrowed && !mutable {
		return unsupported
	}
	receiver, err := l.lowerExpression(owner)
	if err != nil {
		return err
	}
	operator := statement.Operator
	var prior Operand
	if operator != "" && operator != "=" {
		if !hir.CompoundAssignmentType(operator, target.Type) {
			return unsupported
		}
		// Capture the original field before RHS effects, using the already selected receiver.
		destination := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationFieldRead, Span: target.Span,
			DestinationLocal: destination, Operands: []Operand{receiver}, OwnerType: owner.Type,
			Type: target.Type, Field: target.Operator})
		prior = Operand{Kind: OperandLocal, Local: destination}
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	if operator != "" && operator != "=" {
		binary, _ := hir.CompoundAssignmentBinary(operator)
		combined := &hir.Expression{Kind: "binary", Type: target.Type, Span: statement.Span,
			Operator: binary, Children: []hir.Expression{*target, *statement.Expression}}
		value, err = l.lowerBinaryOperands(combined, prior, value)
		if err != nil {
			return err
		}
	}
	op := Operation{Kind: OperationFieldStore, Span: target.Span, Symbol: hir.NoSymbol,
		DestinationLocal: NoLocal, OwnerType: owner.Type, Type: target.Type, Field: target.Operator,
		Operands: []Operand{receiver, value}}
	if !ValidFieldStoreShape(op) && !ValidOneWordStructFieldStoreShape(op, l.properties.Struct(target.Type)) {
		return unsupported
	}
	l.appendOperation(op)
	return nil
}

func validFieldStoreCommonShape(op Operation) bool {
	_, mutable, borrowed := hir.BorrowElement(op.OwnerType)
	return op.OwnerType != "" && op.OwnerType != hir.TypeInvalid && (!borrowed || mutable) &&
		op.Field != "" && op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}

// ValidFieldStoreShape keeps ordinary scalar stores within their existing shape.
func ValidFieldStoreShape(op Operation) bool {
	return fieldStoreScalar(op.Type) && validFieldStoreCommonShape(op) && scalarOperandMatches(op.Operands[1], op.Type)
}

// ValidOneWordStructFieldStoreShape admits only an inert, non-FFI usize wrapper
// supplied by a typed local. The verifier still checks the owner field and both
// local types against program declarations before executable use.
func ValidOneWordStructFieldStoreShape(op Operation, definition *StructType) bool {
	return definition != nil && definition.Type == op.Type && !definition.ReprC &&
		len(definition.Fields) == 1 && definition.Fields[0].Type == hir.TypeUSize &&
		validFieldStoreCommonShape(op) && op.Operands[1].Kind == OperandLocal && op.Operands[1].Local >= 0
}
