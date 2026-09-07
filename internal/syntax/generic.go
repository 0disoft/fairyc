package syntax

import "github.com/0disoft/fairyc/internal/lexer"

func (p *parser) parseTypeParameters() NodeID {
	if !p.at("<") {
		return NoNode
	}
	start := p.currentFullIndex()
	p.advance()
	var names []NodeID
	if p.at(">") {
		p.parseName()
	}
	for !p.at(">") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		parameterStart := p.currentFullIndex()
		name, span := p.parseNameWithSpan()
		names = append(names, p.addNamedNode(KindPath, parameterStart, nil, name, "", span))
		if !p.match(",") {
			break
		}
	}
	p.expectTypeClose()
	return p.addNode(KindTypeParameters, start, names, "", "")
}

// Only a closed angle list with the requested continuation is specialization.
// The caller separately decides whether records are legal in this expression context.
func (p *parser) explicitTypeArgumentsAhead(left NodeID, continuation string) bool {
	node, found := p.tree.Node(left)
	if !found || node.Kind != KindPath {
		return false
	}
	depth := 0
	for offset := 0; offset < 256; offset++ {
		switch p.peekText(offset) {
		case "<":
			depth++
		case ">":
			depth--
		case ">>":
			depth -= 2
		case "", ";", "{", "}", "=", "+", "-", "/", "%", "&&", "||":
			return false
		}
		if depth <= 0 {
			return depth == 0 && offset > 1 && p.peekText(offset+1) == continuation
		}
	}
	return false
}

func (p *parser) parseTypeArguments() []NodeID {
	p.expect("<")
	var arguments []NodeID
	for !p.at(">") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		arguments = append(arguments, p.parseType())
		if !p.match(",") {
			break
		}
	}
	p.expectTypeClose()
	return arguments
}
