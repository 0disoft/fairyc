package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
	"slices"
)

func (l *functionLowerer) lowerArrayConstruction(expression *hir.Expression) (Operand, error) {
	elementType, length, array := hir.FixedArrayParts(expression.Type)
	validForm := expression.Operator == "list" && uint64(len(expression.Children)) == length ||
		expression.Operator == "repeat" && len(expression.Children) == 1
	if !array || !validForm || !hir.ExecutableFixedArrayShape(elementType, length) {
		return Operand{}, fmt.Errorf("malformed fixed-array construction")
	}
	operands := make([]Operand, 0, len(expression.Children))
	sequence := make([]hir.Span, 0, len(expression.Children))
	for index := range expression.Children {
		if expression.Children[index].Type != elementType {
			return Operand{}, fmt.Errorf("fixed-array element %d has type %q, want %q", index, expression.Children[index].Type, elementType)
		}
		operand, err := l.lowerExpression(&expression.Children[index])
		if err != nil {
			return Operand{}, err
		}
		if l.blockTerminated(l.current) {
			return Operand{}, nil
		}
		if operand.Kind == OperandInvalid {
			return Operand{}, fmt.Errorf("fixed-array element %d has no executable value", index)
		}
		if operand.Kind == OperandLocal {
			operand = l.freezeScalarOperand(operand, expression.Children[index].Span)
		}
		operands = append(operands, operand)
		sequence = append(sequence, expression.Children[index].Span)
	}
	if expression.Operator == "repeat" {
		value := operands[0]
		operands = make([]Operand, int(length))
		for index := range operands {
			operands[index] = value
		}
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayConstruct, Span: expression.Span, DestinationLocal: destination,
		Operands: operands, Type: expression.Type, OwnerType: elementType, Operator: "list",
	})
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: sequence, MaterializedBeforeParent: true,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayBorrowView(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || !hir.ArrayViewBorrow(expression.Children[0].Type, expression.Type, expression.Operator) {
		return Operand{}, fmt.Errorf("malformed fixed-array view borrow")
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind != OperandLocal {
		return Operand{}, fmt.Errorf("fixed-array byte-view borrow requires one local owner")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayBorrowView, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayLength(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 1 || expression.Operator != "len" || expression.Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed fixed-array length")
	}
	elementType, length, array := hir.FixedArrayParts(expression.Children[0].Type)
	if !array || !hir.ExecutableFixedArrayShape(elementType, length) {
		return Operand{}, fmt.Errorf("fixed-array length has unsupported owner type %q", expression.Children[0].Type)
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("fixed-array length has no receiver value")
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayLength, Span: expression.Span, DestinationLocal: destination,
		Operands: []Operand{receiver}, OwnerType: expression.Children[0].Type,
		Type: expression.Type, Field: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerArrayIndex(expression *hir.Expression) (Operand, error) {
	if len(expression.Children) != 2 || expression.Operator != "[]" || expression.Children[1].Type != hir.TypeUSize {
		return Operand{}, fmt.Errorf("malformed fixed-array index")
	}
	elementType, _, array := hir.FixedArrayParts(expression.Children[0].Type)
	if !array || expression.Type != elementType {
		return Operand{}, fmt.Errorf("fixed-array index result type %q does not match owner %q", expression.Type, expression.Children[0].Type)
	}
	receiver, err := l.lowerExpression(&expression.Children[0])
	if err != nil {
		return Operand{}, err
	}
	index, err := l.lowerExpression(&expression.Children[1])
	if err != nil {
		return Operand{}, err
	}
	if receiver.Kind == OperandInvalid || index.Kind == OperandInvalid {
		return Operand{}, fmt.Errorf("fixed-array index has an invalid operand")
	}
	operands := []Operand{receiver, index}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{
		Expression: expression.Span, Sequence: []hir.Span{expression.Children[0].Span, expression.Children[1].Span}, MaterializedBeforeParent: true,
	})
	l.appendOperation(Operation{
		Kind: OperationArrayIndexCheck, Span: expression.Span, DestinationLocal: NoLocal, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: hir.TypeUSize, Type: elementType, Operator: expression.Operator,
		TrapCategories: []string{"bounds"},
	})
	continuation := l.newBlock()
	trap := l.newBlock()
	l.terminate(Terminator{
		Kind: TerminatorSwitch, Span: expression.Span, Successors: []BlockID{continuation, trap}, UnwindSuccessor: NoBlock,
		SwitchKind: "checked-array-index", SwitchOperands: slices.Clone(operands), Operator: expression.Operator,
	})
	l.current = trap
	l.terminate(Terminator{Kind: TerminatorTrap, Span: expression.Span, UnwindSuccessor: NoBlock, TrapCategories: []string{"bounds"}})
	l.result.TrapPaths = append(l.result.TrapPaths, TrapPathFact{
		Operation: expression.Span, Categories: []string{"bounds"}, CleanupActions: []CleanupAction{}, HasSuccessor: false,
	})
	l.current = continuation
	destination := l.newTemporary()
	l.appendOperation(Operation{
		Kind: OperationArrayIndex, Span: expression.Span, DestinationLocal: destination, Operands: slices.Clone(operands),
		OwnerType: expression.Children[0].Type, IndexType: hir.TypeUSize, Type: elementType, Operator: expression.Operator,
	})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
