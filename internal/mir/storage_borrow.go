package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

// OperationStorageBorrow retains a shared owner alongside an unsafe element
// address. It creates no owned value and transfers no destruction obligation.
const OperationStorageBorrow OperationKind = "storage-borrow"

// OperationStorageBorrowMut retains an exclusive owner while borrowing an
// initialized element mutably. It creates no owned value or cleanup duty.
const OperationStorageBorrowMut OperationKind = "storage-borrow-mut"

func validStorageBorrowShape(op Operation, properties typeProperties) bool {
	if hir.I64View(op.Type) {
		return validI64StorageViewShape(op, properties)
	}
	if op.Kind != OperationStorageBorrow && op.Kind != OperationStorageBorrowMut {
		return false
	}
	wantsMutable := op.Kind == OperationStorageBorrowMut
	element, mutable, borrowed := hir.BorrowElement(op.Type)
	_, ownerMutable, ownerBorrowed := hir.BorrowElement(op.OwnerType)
	pointerElement, pointerMutable, pointer := hir.RawPointerElement(op.IndexType)
	return borrowed && mutable == wantsMutable && storageQueryType(element, properties) &&
		ownerBorrowed && ownerMutable == wantsMutable && pointer && pointerElement == hir.TypeVoid &&
		(!wantsMutable || pointerMutable) &&
		op.DestinationLocal >= 0 && op.Symbol == hir.NoSymbol && len(op.Operands) == 2 &&
		op.Operands[0].Kind == OperandLocal && op.Operands[1].Kind == OperandLocal &&
		op.Operands[0].Local >= 0 && op.Operands[1].Local >= 0 &&
		op.Operands[0].Local != op.Operands[1].Local &&
		op.DestinationLocal != op.Operands[0].Local && op.DestinationLocal != op.Operands[1].Local &&
		len(op.Fields) == 0 && len(op.TrapCategories) == 0 && op.Operator == "" &&
		op.Destination == "" && op.ProjectionEntity == "" && op.DropFlagEntity == "" &&
		op.Field == "" && op.Variant == "" && !op.MovePayload && !op.FlagClearedFirst &&
		op.SourceForm == "" && op.ExitKind == ""
}

func (l *functionLowerer) lowerStorageBorrow(expression *hir.Expression) (Operand, error) {
	if expression.TargetType == hir.TypeI64Slice {
		return l.lowerI64StorageView(expression)
	}
	mutable := expression.Kind == string(OperationStorageBorrowMut)
	wantType := hir.SharedBorrow(expression.TargetType)
	if mutable {
		wantType = hir.MutableBorrow(expression.TargetType)
	}
	if len(expression.Children) != 2 || expression.Type != wantType ||
		expression.UnsafeScope == nil || expression.Operator != "" || len(expression.Effects) != 0 {
		return Operand{}, fmt.Errorf("malformed private storage borrow")
	}
	owner, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if owner.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("private storage borrow requires a materialized owner")
	}
	// Freeze the first argument before evaluating an address expression with calls.
	retained := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationAssign, Span: expression.Children[0].Span,
		DestinationLocal: retained, Type: expression.Children[0].Type, Operands: []Operand{owner}})
	address, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	kind := OperationStorageBorrow
	if mutable {
		kind = OperationStorageBorrowMut
	}
	op := Operation{Kind: kind, Entity: sourceEntity(string(kind), expression.Span),
		Span: expression.Span, Symbol: hir.NoSymbol, DestinationLocal: l.newTemporary(),
		Type: expression.Type, OwnerType: expression.Children[0].Type, IndexType: expression.Children[1].Type,
		Operands: []Operand{{Kind: OperandLocal, Local: retained}, address}, UnsafeScope: *expression.UnsafeScope}
	if !validStorageBorrowShape(op, l.properties) {
		return Operand{}, fmt.Errorf("malformed private storage borrow operands")
	}
	l.appendOperation(op)
	l.result.UnsafeOperations = append(l.result.UnsafeOperations, UnsafeOperation{Entity: op.Entity,
		SourceSpanEntity: sourceEntity("span", op.Span), ScopeEntity: sourceEntity("unsafe-scope", op.UnsafeScope)})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: []hir.Span{expression.Children[0].Span, expression.Children[1].Span}, MaterializedBeforeParent: true})
	return Operand{Kind: OperandLocal, Local: op.DestinationLocal}, nil
}
