package syntax

// A closed angle list followed immediately by a call is the explicit type-argument form.
// Chained comparisons are already rejected by the ordinary expression grammar.
func (p *parser) explicitTypeCallAhead(left NodeID) bool {
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
			return depth == 0 && offset > 1 && p.peekText(offset+1) == "("
		}
	}
	return false
}
