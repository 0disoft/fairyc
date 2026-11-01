package syntax

import "github.com/0disoft/fairyc/internal/lexer"

// parseClosure retains typed explicit captures without guessing free variables.
func (p *parser) parseClosure() NodeID {
	start := p.currentFullIndex()
	p.expect("fn")
	p.expect("[")
	var children []NodeID
	for !p.at("]") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		s := p.currentFullIndex()
		typ := p.parseType()
		name, span := p.parseNameWithSpan()
		children = append(children, p.addNamedNode(KindCapture, s, []NodeID{typ}, name, "", span))
		if !p.match(",") {
			break
		}
	}
	p.expect("]")
	children = append(children, p.parseType())
	children = append(children, p.parseParameters(false, false)...)
	children = append(children, p.parseBlock())
	return p.addNode(KindClosure, start, children, "", "")
}
