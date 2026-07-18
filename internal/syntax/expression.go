package syntax

import (
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
)

type infixInfo struct {
	leftBindingPower  int
	rightBindingPower int
	comparison        bool
}

var infixOperators = map[string]infixInfo{
	"*":  {110, 111, false},
	"/":  {110, 111, false},
	"%":  {110, 111, false},
	"+":  {100, 101, false},
	"-":  {100, 101, false},
	"<<": {90, 91, false},
	">>": {90, 91, false},
	"&":  {80, 81, false},
	"^":  {70, 71, false},
	"|":  {60, 61, false},
	"==": {50, 51, true},
	"!=": {50, 51, true},
	"<":  {50, 51, true},
	"<=": {50, 51, true},
	">":  {50, 51, true},
	">=": {50, 51, true},
	"&&": {40, 41, false},
	"||": {30, 31, false},
}

func (p *parser) parseExpression(minBindingPower int, allowRecord bool) NodeID {
	if !p.enter() {
		return NoNode
	}
	defer p.leave()
	left := p.parsePrefix(allowRecord)
	comparisonSeen := false
	for p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		start := p.nodeStart(left)
		if p.at("{") && !allowRecord {
			return left
		}
		switch p.currentText() {
		case "(":
			if 140 < minBindingPower {
				return left
			}
			p.advance()
			children := []NodeID{left}
			for !p.at(")") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
				children = append(children, p.parseExpression(0, true))
				if !p.match(",") {
					break
				}
			}
			p.expect(")")
			left = p.addNode(KindCall, start, children, "", "")
			continue
		case "[":
			if 140 < minBindingPower {
				return left
			}
			p.advance()
			index := p.parseExpression(0, true)
			p.expect("]")
			left = p.addNode(KindIndex, start, []NodeID{left, index}, "", "")
			continue
		case ".":
			if 140 < minBindingPower {
				return left
			}
			p.advance()
			name := p.parseName()
			left = p.addNode(KindFieldExpr, start, []NodeID{left}, name, "")
			continue
		case "?":
			if 140 < minBindingPower {
				return left
			}
			p.advance()
			left = p.addNode(KindPropagate, start, []NodeID{left}, "?", "")
			continue
		case "as":
			if 120 < minBindingPower {
				return left
			}
			p.advance()
			typeID := p.parseType()
			left = p.addNode(KindCast, start, []NodeID{left, typeID}, "as", "")
			continue
		}

		operator := p.currentText()
		info, found := infixOperators[operator]
		if found {
			if info.leftBindingPower < minBindingPower {
				return left
			}
			operatorToken := p.current()
			p.advance()
			if info.comparison && comparisonSeen {
				p.emit("E0009", operatorToken.Span, diagnostic.RepairNone, "")
			}
			right := p.parseExpression(info.rightBindingPower, allowRecord)
			left = p.addNode(KindBinary, start, []NodeID{left, right}, operator, "")
			comparisonSeen = comparisonSeen || info.comparison
			continue
		}

		if isExpressionTerminator(p.currentText()) {
			return left
		}
		unexpected := p.current()
		if unexpected.Kind == lexer.Invalid {
			p.advance()
		} else {
			p.emit("E0008", unexpected.Span, diagnostic.RepairSkip, unexpected.Text(p.tree.Source))
			p.advance()
		}
		children := []NodeID{left}
		if canStartExpression(p.current(), p.currentText()) {
			children = append(children, p.parseExpression(0, allowRecord))
		}
		left = p.addNode(KindError, start, children, "expression", "")
	}
	return left
}

func (p *parser) parsePrefix(allowRecord bool) NodeID {
	operator := p.currentText()
	if isPrefixOperator(operator) {
		start := p.currentFullIndex()
		p.advance()
		if operator == "&" && p.match("mut") {
			operator = "&mut"
		}
		operand := p.parseExpression(130, allowRecord)
		return p.addNode(KindPrefix, start, []NodeID{operand}, operator, "")
	}
	return p.parsePrimary(allowRecord)
}

func (p *parser) parsePrimary(allowRecord bool) NodeID {
	start := p.currentFullIndex()
	token := p.current()
	text := p.currentText()
	if isLiteralToken(token, text) {
		return p.parseLiteral()
	}
	if token.Kind == lexer.Invalid {
		p.advance()
		return p.addNode(KindError, start, nil, text, "")
	}
	if text == "self" {
		p.advance()
		return p.addNode(KindPath, start, nil, text, "")
	}
	if token.Kind == lexer.Identifier {
		pathID, value, _ := p.pathNode()
		if allowRecord && p.match("{") {
			children := []NodeID{pathID}
			for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
				fieldStart := p.currentFullIndex()
				name := p.parseName()
				p.expect(":")
				valueID := p.parseExpression(0, true)
				children = append(children, p.addNode(KindField, fieldStart, []NodeID{valueID}, name, ""))
				if !p.match(",") {
					break
				}
			}
			p.expect("}")
			return p.addNode(KindRecord, start, children, value, "")
		}
		return pathID
	}
	switch text {
	case "(":
		p.advance()
		expression := p.parseExpression(0, true)
		p.expect(")")
		return p.addNode(KindGroup, start, []NodeID{expression}, "", "")
	case "[":
		return p.parseArray()
	case "{":
		return p.parseBlock()
	case "if":
		return p.parseIf()
	case "match":
		return p.parseMatch()
	case "unsafe":
		p.advance()
		block := p.parseBlock()
		return p.addNode(KindUnsafe, start, []NodeID{block}, "", "")
	default:
		return p.unexpected()
	}
}

func (p *parser) parseLiteral() NodeID {
	start := p.currentFullIndex()
	token := p.advance()
	return p.addNode(KindLiteral, start, nil, token.Text(p.tree.Source), token.Cooked)
}

func (p *parser) parseArray() NodeID {
	start := p.currentFullIndex()
	p.advance()
	children := make([]NodeID, 0, 4)
	if !p.at("]") {
		children = append(children, p.parseExpression(0, true))
		if p.match(";") {
			children = append(children, p.parseConstExpression())
		} else {
			for !p.budget.Exhausted() && p.match(",") && !p.at("]") {
				children = append(children, p.parseExpression(0, true))
			}
		}
	}
	p.expect("]")
	return p.addNode(KindArray, start, children, "", "")
}

func (p *parser) parseIf() NodeID {
	start := p.currentFullIndex()
	p.advance()
	condition := p.parseExpression(0, false)
	thenBlock := p.parseBlock()
	children := []NodeID{condition, thenBlock}
	if p.match("else") {
		if p.at("if") {
			children = append(children, p.parseIf())
		} else {
			children = append(children, p.parseBlock())
		}
	}
	return p.addNode(KindIf, start, children, "", "")
}

func (p *parser) parseMatch() NodeID {
	start := p.currentFullIndex()
	p.advance()
	scrutinee := p.parseExpression(0, false)
	p.expect("{")
	children := []NodeID{scrutinee}
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		armStart := p.currentFullIndex()
		pattern := p.parsePattern()
		p.expect("=>")
		expression := p.parseExpression(0, true)
		p.expect(",")
		children = append(children, p.addNode(KindPattern, armStart, []NodeID{pattern, expression}, "arm", ""))
	}
	p.expect("}")
	return p.addNode(KindMatch, start, children, "", "")
}

func (p *parser) parsePattern() NodeID {
	start := p.currentFullIndex()
	pathID, value, _ := p.pathNode()
	children := []NodeID{pathID}
	if p.match("(") {
		nameStart := p.currentFullIndex()
		name := p.parseName()
		children = append(children, p.addNode(KindPath, nameStart, nil, name, ""))
		p.expect(")")
	} else if p.match("{") {
		for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
			fieldStart := p.currentFullIndex()
			field := p.parseName()
			p.expect(":")
			binding := p.parseName()
			children = append(children, p.addNode(KindField, fieldStart, nil, field, binding))
			if !p.match(",") {
				break
			}
		}
		p.expect("}")
	}
	return p.addNode(KindPattern, start, children, value, "")
}

func (p *parser) nodeStart(id NodeID) int {
	if node, found := p.tree.Node(id); found {
		return node.TokenStart
	}
	return p.currentFullIndex()
}

func isPrefixOperator(value string) bool {
	switch value {
	case "-", "+", "!", "~", "&", "*":
		return true
	default:
		return false
	}
}

func isExpressionTerminator(value string) bool {
	switch value {
	case "", ";", ",", ")", "]", "}", "=", "=>", "let", "var", "return", "discard", "while", "break", "continue", "region":
		return true
	default:
		return false
	}
}

func canStartExpression(token lexer.Token, text string) bool {
	if token.Kind == lexer.Identifier || token.Kind == lexer.Integer || token.Kind == lexer.Float || token.Kind == lexer.String || token.Kind == lexer.ByteString || token.Kind == lexer.CString {
		return true
	}
	switch text {
	case "true", "false", "null", "self", "(", "[", "{", "if", "match", "unsafe", "-", "+", "!", "~", "&", "*":
		return true
	default:
		return false
	}
}

func isLiteralToken(token lexer.Token, text string) bool {
	switch token.Kind {
	case lexer.Integer, lexer.Float, lexer.String, lexer.ByteString, lexer.CString:
		return true
	case lexer.Keyword:
		return text == "true" || text == "false" || text == "null"
	default:
		return false
	}
}
