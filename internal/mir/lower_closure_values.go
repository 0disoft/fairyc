package mir

import (
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerClosureValue(expression *hir.Expression) (Operand, error) {
	if expression.Type != hir.TypeClosureI32 || expression.Symbol == hir.NoSymbol || len(expression.Children) > 4 {
		return Operand{}, fmt.Errorf("invalid copied closure construction")
	}
	operands := make([]Operand, 0, len(expression.Children))
	spans := make([]hir.Span, 0, len(expression.Children))
	for i := range expression.Children {
		child := &expression.Children[i]
		if child.Type != hir.TypeI32 {
			return Operand{}, fmt.Errorf("closure capture is not i32")
		}
		operand, err := l.lowerExpression(child)
		if err != nil || l.blockTerminated(l.current) {
			return Operand{}, err
		}
		operands = append(operands, l.freezeScalarOperand(operand, child.Span))
		spans = append(spans, child.Span)
	}
	if len(spans) > 1 {
		l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{Expression: expression.Span, Sequence: spans, MaterializedBeforeParent: true})
	}
	destination := l.newTemporary()
	l.appendOperation(Operation{Kind: OperationClosureCreate, Span: expression.Span, Symbol: expression.Symbol, Type: hir.TypeClosureI32, DestinationLocal: destination, Operands: operands})
	return Operand{Kind: OperandLocal, Local: destination}, nil
}

func (l *functionLowerer) lowerClosureCall(expression *hir.Expression) (Operand, error) {
	if expression.Type != hir.TypeI32 || expression.Symbol != hir.NoSymbol || expression.ForeignABI != "" || expression.UnsafeScope != nil || len(expression.Children) != 2 || expression.Children[0].Type != hir.TypeClosureI32 || expression.Children[1].Type != hir.TypeI32 {
		return Operand{}, fmt.Errorf("invalid copied closure value call")
	}
	operands := make([]Operand, 0, 2)
	spans := make([]hir.Span, 0, 2)
	for i := range expression.Children {
		child := &expression.Children[i]
		operand, err := l.lowerExpression(child)
		if err != nil || l.blockTerminated(l.current) {
			return Operand{}, err
		}
		if i == 0 {
			// Freeze the callable value before evaluating its argument.
			operand = l.freezeScalarOperand(operand, child.Span)
		}
		operands = append(operands, operand)
		spans = append(spans, child.Span)
	}
	l.result.OrderedEvaluations = append(l.result.OrderedEvaluations, OrderedEvaluationFact{Expression: expression.Span, Sequence: slices.Clone(spans), MaterializedBeforeParent: true})
	l.result.Calls = append(l.result.Calls, CallFact{Call: expression.Span, ArgumentOrder: spans, NormalSuccessor: true})
	destination := l.newTemporary()
	continuation := l.newBlock()
	l.terminate(Terminator{Kind: TerminatorClosureCall, Span: expression.Span, Arguments: operands, DestinationLocal: destination, Successors: []BlockID{continuation}, UnwindSuccessor: NoBlock})
	l.current = continuation
	return Operand{Kind: OperandLocal, Local: destination}, nil
}
