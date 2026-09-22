package syntax

import "github.com/0disoft/fairyc/internal/lexer"

// Only '=' distinguishes a label from existing contextual variant expressions.
func (p *parser) parseCallArgument() NodeID {
	if !p.isPreviousRevision() && p.at(".") && p.peekText(2) == "=" {
		start := p.currentFullIndex()
		p.advance()
		if p.current().Kind == lexer.Identifier {
			name := p.parseName()
			p.expect("=")
			value := p.parseExpression(0, true)
			return p.addNode(KindArgumentLabel, start, []NodeID{value}, name, "")
		}
	}
	return p.parseExpression(0, true)
}
