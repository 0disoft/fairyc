package syntax

import (
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
)

func (p *parser) parseBlock() NodeID {
	return p.parseBlockWithTail(true)
}

func (p *parser) parseBlockWithTail(allowTail bool) NodeID {
	if !p.enter() {
		return NoNode
	}
	defer p.leave()
	start := p.currentFullIndex()
	p.expect("{")
	children := make([]NodeID, 0, 8)
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		switch p.currentText() {
		case "let", "var":
			children = append(children, p.parseBinding())
		case "return":
			children = append(children, p.parseReturn())
		case "assert":
			children = append(children, p.parseAssert())
		case "discard":
			children = append(children, p.parseDiscard())
		case "while":
			children = append(children, p.parseWhile())
		case "break":
			statementStart := p.currentFullIndex()
			p.advance()
			p.expectStatementTerminator()
			children = append(children, p.addNode(KindBreak, statementStart, nil, "", ""))
		case "continue":
			statementStart := p.currentFullIndex()
			p.advance()
			p.expectStatementTerminator()
			children = append(children, p.addNode(KindContinue, statementStart, nil, "", ""))
		case "region":
			children = append(children, p.parseRegion())
		default:
			statementStart := p.currentFullIndex()
			expression := p.parseExpression(0, true)
			if p.match("=") {
				right := p.parseExpression(0, true)
				p.expectStatementTerminator()
				children = append(children, p.addNode(KindAssignment, statementStart, []NodeID{expression, right}, "", ""))
				continue
			}
			if p.match(";") {
				children = append(children, p.addNode(KindExprStmt, statementStart, []NodeID{expression}, "", ""))
				continue
			}
			expressionNode, expressionFound := p.tree.Node(expression)
			if expressionFound && isBlockStatement(expressionNode.Kind) && (!p.at("}") || !allowTail) {
				children = append(children, p.addNode(KindExprStmt, statementStart, []NodeID{expression}, "", ""))
				if p.at("}") {
					break
				}
				continue
			}
			if p.at("}") {
				if allowTail {
					children = append(children, p.addNode(KindTail, statementStart, []NodeID{expression}, "", ""))
					break
				}
				p.expectStatementTerminator()
				children = append(children, p.addNode(KindExprStmt, statementStart, []NodeID{expression}, "", ""))
				break
			}
			p.expectStatementTerminator()
			children = append(children, p.addNode(KindExprStmt, statementStart, []NodeID{expression}, "", ""))
		}
	}
	p.expect("}")
	return p.addNode(KindBlock, start, children, "", "")
}

func isBlockStatement(kind Kind) bool {
	switch kind {
	case KindBlock, KindIf, KindUnsafe:
		return true
	default:
		return false
	}
}

func (p *parser) parseAssert() NodeID {
	start := p.currentFullIndex()
	p.advance()
	p.expect("(")
	condition := p.parseExpression(0, true)
	p.expect(")")
	p.expectStatementTerminator()
	return p.addNode(KindAssert, start, []NodeID{condition}, "", "")
}

func (p *parser) parseBinding() NodeID {
	start := p.currentFullIndex()
	bindingKind := p.currentText()
	p.advance()
	var name string
	var nameSpan diagnostic.Span
	children := make([]NodeID, 0, 2)
	if p.atKind(lexer.Identifier) && p.peekText(1) == "=" {
		name, nameSpan = p.parseNameWithSpan()
	} else if p.acceptsPreviousRevision() && p.atKind(lexer.Identifier) && p.peekText(1) == ":" {
		name, nameSpan = p.parseNameWithSpan()
		p.expect(":")
		children = append(children, p.parseType())
	} else {
		children = append(children, p.parseType())
		name, nameSpan = p.parseNameWithSpan()
	}
	p.expect("=")
	children = append(children, p.parseExpression(0, true))
	p.expectStatementTerminator()
	return p.addNamedNode(KindBinding, start, children, name, bindingKind, nameSpan)
}

func (p *parser) parseReturn() NodeID {
	start := p.currentFullIndex()
	p.advance()
	children := make([]NodeID, 0, 1)
	if !p.at(";") {
		children = append(children, p.parseExpression(0, true))
	}
	p.expectStatementTerminator()
	return p.addNode(KindReturn, start, children, "", "")
}

func (p *parser) parseDiscard() NodeID {
	start := p.currentFullIndex()
	p.advance()
	expression := p.parseExpression(0, true)
	p.expectStatementTerminator()
	return p.addNode(KindDiscard, start, []NodeID{expression}, "", "")
}

func (p *parser) parseWhile() NodeID {
	start := p.currentFullIndex()
	p.advance()
	condition := p.parseControlExpression()
	body := p.parseBlockForStatement()
	return p.addNode(KindWhile, start, []NodeID{condition, body}, "", "")
}

func (p *parser) parseControlExpression() NodeID {
	if !p.acceptsPreviousRevision() {
		p.expect("(")
		condition := p.parseExpression(0, true)
		p.expect(")")
		return condition
	}
	if p.parenthesizedControlHeader() {
		p.expect("(")
		condition := p.parseExpression(0, true)
		p.expect(")")
		return condition
	}
	return p.parseExpression(0, false)
}

func (p *parser) parenthesizedControlHeader() bool {
	if !p.at("(") {
		return false
	}
	depth := 0
	for offset := 0; ; offset++ {
		token := p.peek(offset)
		if token.Kind == lexer.EOF {
			return false
		}
		switch token.Text(p.tree.Source) {
		case "(":
			depth++
		case ")":
			depth--
			if depth == 0 {
				return p.peekText(offset+1) == "{"
			}
			if depth < 0 {
				return false
			}
		}
	}
}

func (p *parser) parseRegion() NodeID {
	start := p.currentFullIndex()
	p.advance()
	name := p.parseName()
	p.expect("in")
	arena := p.parseExpression(0, false)
	body := p.parseBlockForStatement()
	return p.addNode(KindRegion, start, []NodeID{arena, body}, name, "")
}

func (p *parser) parseBlockForStatement() NodeID {
	if p.acceptsPreviousRevision() {
		return p.parseBlock()
	}
	return p.parseBlockWithTail(false)
}
