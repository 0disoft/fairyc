package mir

import "github.com/0disoft/fairyc/internal/hir"

func (l *functionLowerer) lowerDerefStore(statement hir.Statement) error {
	target := statement.Target
	unsupported := &Unsupported{Feature: "dereference assignment without executable mutable reference storage", Span: target.Span}
	operator := statement.Operator
	compound := operator != "" && operator != "="
	if len(target.Children) != 1 || target.Operator != "*" || (!fieldStoreScalar(target.Type) && !l.properties.plainViewStruct(target.Type)) ||
		statement.Expression == nil || statement.Expression.Type != target.Type || compound && !hir.CompoundAssignmentType(operator, target.Type) {
		return unsupported
	}
	reference := &target.Children[0]
	for reference.Kind == hir.ExpressionGroup && len(reference.Children) == 1 {
		reference = &reference.Children[0]
	}
	if reference.Kind != hir.ExpressionPath || reference.Type != hir.MutableBorrow(target.Type) {
		return unsupported
	}
	receiver, err := l.lowerExpression(reference)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	// Select the referent before RHS effects, including rebinding the named reference.
	saved := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: target.Span, DestinationLocal: saved, Operands: []Operand{receiver}})
	receiver = Operand{Kind: OperandLocal, Local: saved}
	var prior Operand
	if compound {
		// Load from the selected address, not from a reference that RHS effects
		// may rebind. The temporary also snapshots the scalar before the RHS.
		old := l.newTemporary()
		l.appendOperation(Operation{Kind: OperationUnary, Span: target.Span, DestinationLocal: old,
			Operator: "*", OwnerType: reference.Type, Type: target.Type, Operands: []Operand{receiver}})
		prior = Operand{Kind: OperandLocal, Local: old}
	}
	value, err := l.lowerExpression(statement.Expression)
	if err != nil {
		return err
	}
	if l.blockTerminated(l.current) {
		return nil
	}
	if compound {
		binary, _ := hir.CompoundAssignmentBinary(operator)
		combined := &hir.Expression{Kind: hir.ExpressionBinary, Type: target.Type, Span: statement.Span,
			Operator: binary, Children: []hir.Expression{*target, *statement.Expression}}
		value, err = l.lowerBinaryOperands(combined, prior, value)
		if err != nil {
			return err
		}
	}
	l.appendOperation(Operation{Kind: OperationDerefStore, Span: target.Span, Symbol: hir.NoSymbol,
		DestinationLocal: NoLocal, OwnerType: reference.Type, Type: target.Type,
		Operands: []Operand{receiver, value}})
	return nil
}

// ValidDerefStoreShape excludes implicit moves, aggregate replacement and
// shared references. The verifier checks the concrete local types separately.
func ValidDerefStoreShape(op Operation) bool {
	return fieldStoreScalar(op.Type) && validDerefStoreCommonShape(op) && scalarOperandMatches(op.Operands[1], op.Type)
}

// ValidPlainStructDerefStoreShape admits whole copies of the same plain struct graph
// used by indexed replacement. Concrete receiver/value types stay verifier-owned.
func ValidPlainStructDerefStoreShape(op Operation, lookup func(hir.Type) *StructType) bool {
	return validDerefStoreCommonShape(op) && op.Operands[1].Kind == OperandLocal && op.Operands[1].Local >= 0 && plainFieldStoreStruct(op.Type, lookup)
}

func validDerefStoreCommonShape(op Operation) bool {
	return op.OwnerType == hir.MutableBorrow(op.Type) &&
		op.DestinationLocal == NoLocal && op.Symbol == hir.NoSymbol && op.Field == "" &&
		len(op.Operands) == 2 && op.Operands[0].Kind == OperandLocal && op.Operands[0].Local >= 0 &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" && op.Destination == "" &&
		op.ProjectionEntity == "" && op.DropFlagEntity == "" && op.IndexType == hir.ParseType("") && op.Variant == "" &&
		!op.MovePayload && !op.FlagClearedFirst && op.UnsafeScope == (hir.Span{}) && op.SourceForm == "" && op.ExitKind == ""
}
