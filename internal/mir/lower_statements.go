package mir

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/hir"
)

func (l *functionLowerer) lowerBlock(block hir.Block) (bool, error) {
	previousScope := l.currentScope
	l.currentScope = block.Span
	defer func() { l.currentScope = previousScope }()
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return terminated, err
	}
	if block.Tail != nil {
		if err := l.lowerReturn(block.Tail, block.Tail.Span); err != nil {
			return false, err
		}
		return true, nil
	}
	return false, nil
}

func (l *functionLowerer) lowerStatements(statements []hir.Statement) (bool, error) {
	for _, statement := range statements {
		if l.blockTerminated(l.current) {
			return true, nil
		}
		switch statement.Kind {
		case hir.StatementBinding:
			value, err := l.lowerExpression(statement.Expression)
			if err != nil {
				return false, err
			}
			binding, found := l.bindings[statement.Binding]
			if found && l.properties.NeedsDrop(binding.typ) {
				if value.Kind == OperandInvalid {
					return false, fmt.Errorf("resource binding %d has no value", statement.Binding)
				}
				if path := l.liveResourcePath(statement.Expression); path != nil {
					value = l.materializeMove(path, value, "binding-value", true)
				}
				l.appendOperation(Operation{
					Kind: OperationInitialize, Span: binding.span, Symbol: statement.Binding,
					DestinationLocal: LocalID(statement.Binding), Operands: []Operand{value},
				})
				l.live[statement.Binding] = true
			} else if value.Kind != OperandInvalid {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Binding), Operands: []Operand{value}})
			}
		case hir.StatementReturn:
			if err := l.lowerReturn(statement.Expression, expressionSpan(statement.Expression, statement.Span)); err != nil {
				return false, err
			}
			return true, nil
		case hir.StatementDiscard, hir.StatementExpressionStatement:
			if err := l.lowerIgnoredExpression(statement.Expression); err != nil {
				return false, err
			}
		case hir.StatementBurn:
			if statement.Expression == nil || statement.Expression.Kind != hir.ExpressionPath {
				return false, fmt.Errorf("burn requires one direct resource path")
			}
			binding, found := l.bindings[statement.Expression.Symbol]
			if !found || !l.properties.NeedsDrop(binding.typ) {
				return false, fmt.Errorf("burn names non-resource local %d", statement.Expression.Symbol)
			}
			if !l.live[statement.Expression.Symbol] {
				return false, fmt.Errorf("burn names discharged resource local %d", statement.Expression.Symbol)
			}
			l.appendOperation(Operation{
				Kind: OperationBurn, Span: statement.Span, Symbol: statement.Expression.Symbol,
				DestinationLocal: NoLocal, Type: binding.typ,
			})
			l.live[statement.Expression.Symbol] = false
		case hir.StatementAssignment:
			if statement.Target != nil && statement.Target.Kind == hir.ExpressionPrefix && statement.Target.Operator == "*" {
				if err := l.lowerDerefStore(statement); err != nil {
					return false, err
				}
				continue
			}
			if statement.Target != nil && statement.Target.Kind == hir.ExpressionFieldExpression {
				if err := l.lowerFieldStore(statement); err != nil {
					return false, err
				}
				continue
			}
			if statement.Target != nil && (statement.Target.Kind == hir.ExpressionArrayIndex || statement.Target.Kind == hir.ExpressionViewIndex) {
				if err := l.lowerIndexedAssignment(statement); err != nil {
					return false, err
				}
				continue
			}
			operator := statement.Operator
			if operator == "" {
				operator = "="
			}
			var value Operand
			var err error
			if operator != "=" && statement.Target != nil && statement.Expression != nil {
				binary, ok := hir.CompoundAssignmentBinary(operator)
				if !ok || statement.Target.Kind != hir.ExpressionPath || !hir.CompoundAssignmentType(operator, statement.Target.Type) {
					return false, fmt.Errorf("malformed compound assignment %q", operator)
				}
				prior, lowerErr := l.lowerExpression(statement.Target)
				if lowerErr != nil {
					return false, lowerErr
				}
				prior = l.freezeScalarOperand(prior, statement.Target.Span)
				right, lowerErr := l.lowerExpression(statement.Expression)
				if lowerErr != nil {
					return false, lowerErr
				}
				combined := &hir.Expression{
					Kind: hir.ExpressionBinary, Type: statement.Target.Type, Span: statement.Span, Operator: binary,
					Children: []hir.Expression{*statement.Target, *statement.Expression},
				}
				value, err = l.lowerBinaryOperands(combined, prior, right)
			} else {
				value, err = l.lowerExpression(statement.Expression)
			}
			if err != nil {
				return false, err
			}
			if statement.Target == nil || statement.Target.Kind != hir.ExpressionPath || value.Kind == OperandInvalid {
				continue
			}
			binding, found := l.bindings[statement.Target.Symbol]
			if found && l.properties.NeedsDrop(binding.typ) {
				if path := l.liveResourcePath(statement.Expression); path != nil {
					value = l.materializeMove(path, value, "reassignment-value", true)
				}
				droppedOld := l.live[statement.Target.Symbol]
				if droppedOld {
					l.appendOperation(Operation{Kind: OperationDrop, Span: binding.span, Symbol: statement.Target.Symbol, FlagClearedFirst: true})
					l.live[statement.Target.Symbol] = false
				}
				l.appendOperation(Operation{
					Kind: OperationInitialize, Span: statement.Target.Span, Symbol: statement.Target.Symbol,
					DestinationLocal: LocalID(statement.Target.Symbol), Operands: []Operand{value},
				})
				l.live[statement.Target.Symbol] = true
				l.result.Reassignments = append(l.result.Reassignments, ReassignmentFact{
					Binding: binding.span, Target: statement.Target.Span, Value: statement.Expression.Span,
					ValueBeforeDrop: true, DroppedOld: droppedOld, InstalledNew: true,
				})
			} else {
				l.appendOperation(Operation{Kind: OperationAssign, Span: statement.Span, DestinationLocal: LocalID(statement.Target.Symbol), Operands: []Operand{value}})
			}
		case hir.StatementAssert:
			if err := l.lowerAssert(statement.Expression, statement.Span); err != nil {
				return false, err
			}
		case hir.StatementDefer:
			if err := l.lowerDefer(statement); err != nil {
				return false, err
			}
		case hir.StatementWhile:
			if _, err := l.lowerWhile(statement.Expression); err != nil {
				return false, err
			}
		case hir.StatementFor:
			if err := l.lowerFor(statement.For); err != nil {
				return false, err
			}
		case hir.StatementBreak, hir.StatementContinue:
			if err := l.lowerLoopControl(statement.Kind, statement.Span); err != nil {
				return false, err
			}
			return true, nil
		case hir.StatementRegion:
			if _, err := l.lowerExpression(statement.Expression); err != nil {
				return false, err
			}
		default:
			return false, fmt.Errorf("unsupported HIR statement %q", statement.Kind)
		}
	}
	return l.blockTerminated(l.current), nil
}

func (l *functionLowerer) lowerNestedBlock(block hir.Block, ownsCleanup bool) (Operand, bool, error) {
	previousScope := l.currentScope
	l.currentScope = block.Span
	defer func() { l.currentScope = previousScope }()
	terminated, err := l.lowerStatements(block.Statements)
	if err != nil || terminated {
		return Operand{}, terminated, err
	}
	if block.Tail == nil {
		l.emitDeferredCallsForScope(block.Span)
		if ownsCleanup {
			l.finishValueBlockCleanup(block.Span)
		}
		return Operand{}, false, nil
	}
	value, err := l.lowerExpression(block.Tail)
	if err == nil && !l.blockTerminated(l.current) {
		if ownsCleanup && l.hasValueBlockCleanup(block.Span) {
			if path := l.liveResourcePath(block.Tail); path != nil {
				value = l.materializeMove(path, value, "block-result", true)
			} else {
				value = l.freezeScalarOperand(value, block.Tail.Span)
			}
		}
		l.emitDeferredCallsForScope(block.Span)
		if ownsCleanup {
			l.finishValueBlockCleanup(block.Span)
		}
	}
	return value, l.blockTerminated(l.current), err
}
