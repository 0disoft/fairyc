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
		if owner.Kind == "group" && len(owner.Children) == 1 {
			owner = &owner.Children[0]
			continue
		}
		if owner.Kind != "field-expression" {
			break
		}
		if len(owner.Children) != 1 || owner.Operator == "" {
			return unsupported
		}
		fields = append(fields, owner.Operator)
		owner = &owner.Children[0]
	}
	slices.Reverse(fields)
	if owner.Kind != "path" {
		return unsupported
	}
	if _, mutable, borrowed := hir.BorrowElement(owner.Type); borrowed && !mutable {
		return unsupported
	}
	if len(fields) > 0 && ((!fieldStoreScalar(target.Type) && !l.properties.plainViewStruct(target.Type)) || !l.properties.plainViewStruct(fieldAggregateType(owner.Type))) {
		return unsupported
	}
	receiver, err := l.lowerExpression(owner)
	if err != nil {
		return err
	}
	if len(fields) > 0 || l.properties.plainViewStruct(target.Type) {
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
		prior, err = l.lowerExpression(target)
		if err != nil {
			return err
		}
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
		DestinationLocal: NoLocal, OwnerType: owner.Type, Type: target.Type, Field: target.Operator, Fields: fields,
		Operands: []Operand{receiver, value}}
	if !ValidFieldStoreShape(op) && !ValidOneWordStructFieldStoreShape(op, l.properties.Struct(target.Type)) &&
		!ValidPlainStructFieldStoreShape(op, l.properties.Struct) {
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
		len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}

// ValidFieldStoreShape checks scalar store representation. Nested declarations
// and path types are checked separately by ResolveNestedFieldStore.
func ValidFieldStoreShape(op Operation) bool {
	return fieldStoreScalar(op.Type) && validFieldStoreCommonShape(op) && scalarOperandMatches(op.Operands[1], op.Type)
}

// ValidOneWordStructFieldStoreShape admits only an inert, non-FFI usize wrapper
// supplied by a typed local. The verifier still checks the owner field and both
// local types against program declarations before executable use.
func ValidOneWordStructFieldStoreShape(op Operation, definition *StructType) bool {
	return len(op.Fields) == 0 && definition != nil && definition.Type == op.Type && !definition.ReprC &&
		len(definition.Fields) == 1 && definition.Fields[0].Type == hir.TypeUSize &&
		validFieldStoreCommonShape(op) && op.Operands[1].Kind == OperandLocal && op.Operands[1].Local >= 0
}
