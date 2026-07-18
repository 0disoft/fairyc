package syntax

import "github.com/0disoft/fairyc/internal/lexer"

func (p *parser) parseBlock() NodeID {
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
			if p.at("}") {
				children = append(children, p.addNode(KindTail, statementStart, []NodeID{expression}, "", ""))
				break
			}
			p.expectStatementTerminator()
			children = append(children, p.addNode(KindExprStmt, statementStart, []NodeID{expression}, "", ""))
		}
	}
	p.expect("}")
	return p.addNode(KindBlock, start, children, "", "")
}

func (p *parser) parseBinding() NodeID {
	start := p.currentFullIndex()
	bindingKind := p.currentText()
	p.advance()
	name := p.parseName()
	children := make([]NodeID, 0, 2)
	if p.match(":") {
		children = append(children, p.parseType())
	}
	p.expect("=")
	children = append(children, p.parseExpression(0, true))
	p.expectStatementTerminator()
	return p.addNode(KindBinding, start, children, name, bindingKind)
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
	condition := p.parseExpression(0, false)
	body := p.parseBlock()
	return p.addNode(KindWhile, start, []NodeID{condition, body}, "", "")
}

func (p *parser) parseRegion() NodeID {
	start := p.currentFullIndex()
	p.advance()
	name := p.parseName()
	p.expect("in")
	arena := p.parseExpression(0, false)
	body := p.parseBlock()
	return p.addNode(KindRegion, start, []NodeID{arena, body}, name, "")
}
