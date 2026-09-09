package mir

import "github.com/0disoft/fairyc/internal/hir"

func fieldStoreScalar(typ hir.Type) bool {
	return typ == hir.TypeI32 || typ == hir.TypeU8 || typ == hir.TypeU32 || typ == hir.TypeUSize || typ == hir.TypeBool
}

func (l *functionLowerer) lowerFieldStore(statement hir.Statement) error {
	target := statement.Target
	unsupported := &Unsupported{Feature: "field assignment without executable field storage", Span: target.Span}
	if len(target.Children) != 1 || target.Operator == "" || !fieldStoreScalar(target.Type) ||
		statement.Expression == nil || statement.Expression.Type != target.Type || statement.Operator != "" && statement.Operator != "=" {
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
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	l.appendOperation(Operation{Kind: OperationFieldStore, Span: target.Span, Symbol: hir.NoSymbol,
		DestinationLocal: NoLocal, OwnerType: owner.Type, Type: target.Type, Field: target.Operator,
		Operands: []Operand{receiver, value}})
	return nil
}

// ValidFieldStoreShape excludes ownership transfer and all non-scalar stores.
// The verifier checks declared fields, operand liveness and concrete types;
// the native consumer independently checks the field and operand types.
func ValidFieldStoreShape(op Operation) bool {
	_, mutable, borrowed := hir.BorrowElement(op.OwnerType)
	return op.OwnerType != "" && op.OwnerType != hir.TypeInvalid && (!borrowed || mutable) &&
		fieldStoreScalar(op.Type) && op.Field != "" && op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 &&
		scalarOperandMatches(op.Operands[1], op.Type) &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}
