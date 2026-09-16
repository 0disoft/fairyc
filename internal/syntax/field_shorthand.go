package syntax

// A shorthand field reuses the ordinary path expression and its exact identifier
// span. Only a field terminator may omit '='; malformed expressions keep the
// existing recovery path. No source tokens are invented or rewritten.
func (p *parser) parseFieldValue(nameStart int, name string) NodeID {
	if !p.isPreviousRevision() && name != "" && (p.at(",") || p.at("}")) {
		return p.addNode(KindPath, nameStart, nil, name, "")
	}
	p.expect("=")
	return p.parseExpression(0, true)
}
