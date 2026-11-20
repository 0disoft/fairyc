package hir

import "fmt"

func (encoder *inlineFunctionEncoder) whileStatement(e *Expression) (inlineEncodingNode, error) {
	if e == nil || e.Kind != ExpressionWhile || e.Type != TypeVoid || e.Block != nil || len(e.Children) != 2 || len(e.Arms) != 0 || len(e.Effects) != 0 || e.UnsafeScope != nil {
		return inlineEncodingNode{}, fmt.Errorf("@inline requires a typed while statement")
	}
	condition, err := encoder.typedExpression(&e.Children[0], TypeBool, false)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	encoder.enterLoop()
	defer encoder.leaveLoop()
	body, _, err := encoder.statementArm(&e.Children[1])
	return inlineEncodingNode{Op: "while", Children: []inlineEncodingNode{condition, body}}, err
}

func (encoder *inlineFunctionEncoder) forStatement(loop *ForLoop) (inlineEncodingNode, error) {
	if loop == nil {
		return inlineEncodingNode{}, fmt.Errorf("@inline requires a typed classic for statement")
	}
	defer encoder.closeScope(len(encoder.armLocals))
	initial, err := encoder.forClause(loop.Initializer, true)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	condition := inlineEncodingNode{Op: "omitted"}
	if loop.Condition != nil {
		condition, err = encoder.typedExpression(loop.Condition, TypeBool, false)
		if err != nil {
			return inlineEncodingNode{}, err
		}
	}
	post, err := encoder.forClause(loop.Post, false)
	if err != nil {
		return inlineEncodingNode{}, err
	}
	encoder.enterLoop()
	defer encoder.leaveLoop()
	encoder.count++
	if encoder.count > inlineMaxExpressionNodes {
		return inlineEncodingNode{}, fmt.Errorf("@inline statement bodies require at most 64 nodes")
	}
	body, _, err := encoder.statementBlock(&loop.Body, false)
	return inlineEncodingNode{Op: "for", Children: []inlineEncodingNode{initial, condition, post, body}}, err
}

func (encoder *inlineFunctionEncoder) forClause(statement *Statement, declaration bool) (inlineEncodingNode, error) {
	if statement == nil {
		return inlineEncodingNode{Op: "omitted"}, nil
	}
	encoder.count++
	if encoder.count > inlineMaxExpressionNodes || !(statement.Kind == StatementAssignment || declaration && statement.Kind == StatementBinding) {
		return inlineEncodingNode{}, fmt.Errorf("@inline requires a bounded scalar for clause")
	}
	node, _, err := encoder.controlStatement(*statement)
	return node, err
}

func (encoder *inlineFunctionEncoder) enterLoop() {
	encoder.hasLoop = true
	encoder.loopDepth++
}

func (encoder *inlineFunctionEncoder) leaveLoop() { encoder.loopDepth-- }
