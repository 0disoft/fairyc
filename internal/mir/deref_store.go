package mir

import "github.com/0disoft/fairyc/internal/hir"

func (l *functionLowerer) lowerDerefStore(statement hir.Statement) error {
	target := statement.Target
	unsupported := &Unsupported{Feature: "scalar dereference assignment without executable mutable reference storage", Span: target.Span}
	if len(target.Children) != 1 || target.Operator != "*" || !fieldStoreScalar(target.Type) ||
		statement.Expression == nil || statement.Expression.Type != target.Type || statement.Operator != "=" {
		return unsupported
	}
	reference := &target.Children[0]
	for reference.Kind == "group" && len(reference.Children) == 1 {
		reference = &reference.Children[0]
	}
	if reference.Kind != "path" || reference.Type != hir.MutableBorrow(target.Type) {
		return unsupported
	}
	receiver, err := l.lowerExpression(reference)
	if err != nil {
		return err
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	l.appendOperation(Operation{Kind: OperationDerefStore, Span: target.Span, Symbol: hir.NoSymbol,
		DestinationLocal: NoLocal, OwnerType: reference.Type, Type: target.Type,
		Operands: []Operand{receiver, value}})
	return nil
}

// ValidDerefStoreShape excludes implicit moves, aggregate replacement and
// shared references. The verifier checks the concrete local types separately.
func ValidDerefStoreShape(op Operation) bool {
	return fieldStoreScalar(op.Type) && op.OwnerType == hir.MutableBorrow(op.Type) &&
		op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol && op.Field == "" &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 &&
		scalarOperandMatches(op.Operands[1], op.Type) &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == "" && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}
