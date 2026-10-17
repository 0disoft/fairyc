package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerViewLength(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "len" || expression.Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed borrowed-view length")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("borrowed-view length has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewLength, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerViewEmpty(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "is_empty" || expression.Type != hir.TypeBool {
		return Operand{}, fmt.Errorf("malformed borrowed-view emptiness")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("borrowed-view emptiness has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewEmpty, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerViewIndex(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 {
		return Operand{}, fmt.Errorf("malformed indexed-view operands")
	}
	element, indexed := hir.IndexedViewElement(expression.Children[0].Type)
	resultType := element
	kind := OperationViewIndex
	if expression.Kind == hir.ExpressionViewElementBorrow {
		referent, mutable, borrowed := hir.BorrowElement(expression.Type)
		if !borrowed || referent != element || mutable && !hir.MutableIndexedView(expression.Children[0].Type) {
			return Operand{}, fmt.Errorf("invalid indexed element reference")
		}
		resultType = expression.Type
		kind = OperationViewElementBorrow
	}
	if !indexed || expression.Operator != "[]" || expression.Type != resultType || expression.Children[1].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed indexed-view read")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	// Select the descriptor before index-side effects can reassign its binding.
	// This copies the descriptor, not the underlying borrowed storage.
	if kind == OperationViewElementBorrow || receiver.Kind == OperandLocal {
		receiver = l.freezeScalarOperand(receiver, expression.Children[0].Span)
	}
	index, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("byte-view index has an invalid operand")
	}
	operands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               expression.Span,
		Sequence:                 []hir.Span{expression.Children[0].Span, expression.Children[1].Span},
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationViewIndexCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: expression.Children[1].Type, Type: element, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-view-index", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: kind, Symbol: hir.NoSymbol, Span: expression.Span, DestinationLocal: destination,
		Operands: slices.Clone(operands), OwnerType: expression.Children[0].Type,
		IndexType: expression.Children[1].Type, Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func isIndexedViewType(value hir.Type) bool {
	_, indexed := hir.IndexedViewElement(value)
	return indexed
}

func (l *functionLowerer) lowerViewSlice(expression *hir.Expression) (Operand, error) {
	wantView := expression.Type
	if len(expression.Children) != 3 || expression.Operator != ".." ||
		!isIndexedViewType(wantView) || expression.Children[0].Type != wantView || expression.Children[1].Type != hir.TypeUSize || expression.Children[2].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed immutable byte-view slice")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if l.blockTerminated(l.current) {
		return Operand{}, nil
	}
	// Bounds may reassign a descriptor binding; keep the receiver selected first.
	if receiver.Kind == OperandLocal {
		receiver = l.freezeScalarOperand(receiver, expression.Children[0].Span)
	}
	start := Operand{Kind: OperandUSize, U64: 0}
	if expression.Children[1].Kind == hir.ExpressionOmittedRangeBound {
		if expression.Children[1].Operator != "start" {
			return Operand{}, fmt.Errorf("malformed omitted byte-view slice start")
		}
	} else {
		start, err = l.lowerExpression(&expression.Children[1])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if start.Kind == OperandLocal {
			start = l.freezeScalarOperand(start, expression.Children[1].Span)
		}
	}
	var end Operand
	if expression.Children[2].Kind == hir.ExpressionOmittedRangeBound {
		if expression.Children[2].Operator != "end" {
			return Operand{}, fmt.Errorf("malformed omitted byte-view slice end")
		}
		endLocal := l.newTemporary()
		l.appendOperation(Operation{
			Kind: OperationViewLength, Span: expression.Children[2].Span, DestinationLocal: endLocal,
			Operands: []Operand{receiver}, OwnerType: wantView, Type: hir.TypeUSize, Field: "len",
		})
		end = Operand{Kind: OperandLocal, Local: endLocal}
	} else {
		end, err = l.lowerExpression(&expression.Children[2])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
	}
	if receiver.Kind == OperandInvalid || start.Kind == OperandInvalid || end.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("immutable byte-view slice has an invalid operand")
	}
	operands := []Operand{receiver, start, end}
	sequence := []hir.Span{expression.Children[0].Span}
	if expression.Children[1].Kind != hir.ExpressionOmittedRangeBound {
		sequence = append(sequence, expression.Children[1].Span)
	}
	if expression.Children[2].Kind != hir.ExpressionOmittedRangeBound {
		sequence = append(sequence, expression.Children[2].Span)
	}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression:               expression.Span,
		Sequence:                 sequence,
		MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationViewSliceCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: wantView, IndexType: hir.TypeUSize, Type: wantView, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-view-slice", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationViewSlice, Span: expression.Span, DestinationLocal: destination,
		Operands: slices.Clone(operands), OwnerType: wantView,
		IndexType: hir.TypeUSize, Type: wantView, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
