package hir

import (
	"fmt"
	"strconv"
)

func inlineStatementIf(e *Expression) bool {
	return e != nil && e.Kind == ExpressionIf && e.Type == TypeVoid
}

func (encoder *inlineFunctionEncoder) closeScope(depth int) {
	for _, symbol := range encoder.armLocals[depth:] {
		delete(encoder.bound, symbol)
	}
	encoder.armLocals = encoder.armLocals[:depth]
}

func (encoder *inlineFunctionEncoder) scopedBinding(statement Statement) (inlineEncodingNode, error) {
	// Keep lexical slots outside the legacy root's value-version range.
	node, err := encoder.statement(statement, inlineMaxSteps+len(encoder.armLocals))
	if err == nil {
		encoder.armLocals = append(encoder.armLocals, statement.Binding)
	}
	return node, err
}

func (encoder *inlineFunctionEncoder) statementBlock(block *Block, root bool) (inlineEncodingNode, bool, error) {
	defer encoder.closeScope(len(encoder.armLocals))
	node := inlineEncodingNode{Op: "block"}
	returns := false
	for _, statement := range block.Statements {
		encoder.count++
		if encoder.count > inlineMaxExpressionNodes {
			return node, false, fmt.Errorf("@inline statement bodies require at most 64 nodes")
		}
		child, exits, err := encoder.controlStatement(statement)
		if err != nil {
			return node, false, err
		}
		node.Children = append(node.Children, child)
		returns = returns || exits
	}
	if block.Tail == nil {
		return node, returns, nil
	}
	if !root {
		child, exits, err := encoder.branchStatement(block.Tail)
		node.Children = append(node.Children, child)
		return node, returns || exits, err
	}
	value, err := encoder.expression(block.Tail)
	node.Children = append(node.Children, inlineEncodingNode{Op: "return", Children: []inlineEncodingNode{value}})
	return node, true, err
}

func (encoder *inlineFunctionEncoder) controlStatement(statement Statement) (inlineEncodingNode, bool, error) {
	switch statement.Kind {
	case StatementWhile:
		node, err := encoder.whileStatement(statement.Expression)
		return node, false, err
	case StatementFor:
		node, err := encoder.forStatement(statement.For)
		return node, false, err
	case StatementBreak, StatementContinue:
		if encoder.loopDepth == 0 {
			return inlineEncodingNode{}, false, fmt.Errorf("@inline loop control requires an enclosing loop")
		}
		return inlineEncodingNode{Op: statement.Kind.String()}, false, nil
	case StatementExpressionStatement:
		return encoder.branchStatement(statement.Expression)
	case StatementReturn:
		value, err := encoder.expression(statement.Expression)
		return inlineEncodingNode{Op: "return", Children: []inlineEncodingNode{value}}, true, err
	case StatementBinding:
		initializer, err := encoder.scopedBinding(statement)
		op := "let"
		if encoder.locals[statement.Binding].Mutable {
			op = "var"
		}
		return inlineEncodingNode{Op: op, Children: []inlineEncodingNode{initializer}}, false, err
	case StatementAssignment:
		symbol, _, err := inlineStatementBinding(statement, encoder.locals, encoder.parameters, encoder.declared)
		if err != nil {
			return inlineEncodingNode{}, false, err
		}
		slot, visible := encoder.bound[symbol]
		if !visible {
			return inlineEncodingNode{}, false, fmt.Errorf("@inline assignment target is outside its lexical scope")
		}
		value, err := encoder.expression(statement.Expression)
		return inlineEncodingNode{Op: "assign", Value: strconv.Itoa(slot), Children: []inlineEncodingNode{value}}, false, err
	default:
		return inlineEncodingNode{}, false, fmt.Errorf("unsupported @inline statement")
	}
}

func (encoder *inlineFunctionEncoder) branchStatement(e *Expression) (inlineEncodingNode, bool, error) {
	node := inlineEncodingNode{Op: "if-statement"}
	encoder.count++
	if encoder.count > inlineMaxExpressionNodes || !inlineStatementIf(e) || e.Block != nil || len(e.Effects) != 0 || len(e.Arms) != 0 || e.UnsafeScope != nil || len(e.Children) < 2 || len(e.Children) > 3 {
		return node, false, fmt.Errorf("@inline requires a bounded void if statement")
	}
	condition, err := encoder.typedExpression(&e.Children[0], TypeBool, false)
	if err != nil {
		return node, false, err
	}
	thenArm, thenReturns, err := encoder.statementArm(&e.Children[1])
	if err != nil {
		return node, false, err
	}
	node.Children = []inlineEncodingNode{condition, thenArm}
	if len(e.Children) == 2 {
		return node, false, nil
	}
	elseArm, elseReturns, err := encoder.statementArm(&e.Children[2])
	node.Children = append(node.Children, elseArm)
	return node, thenReturns && elseReturns, err
}

func (encoder *inlineFunctionEncoder) statementArm(e *Expression) (inlineEncodingNode, bool, error) {
	if inlineStatementIf(e) {
		return encoder.branchStatement(e)
	}
	encoder.count++
	if encoder.count > inlineMaxExpressionNodes || e.Kind != ExpressionBlock || e.Type != TypeVoid || e.Block == nil || len(e.Children) != 0 || len(e.Arms) != 0 || len(e.Effects) != 0 || e.UnsafeScope != nil {
		return inlineEncodingNode{}, false, fmt.Errorf("@inline if statement requires void arms")
	}
	return encoder.statementBlock(e.Block, false)
}
