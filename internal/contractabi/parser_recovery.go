package contractabi

import "fmt"

func (p *interfaceParser) recoverMember() {
	for !p.atEOF() && !p.at("}") {
		if p.match(";") {
			return
		}
		p.advance()
	}
}

func (p *interfaceParser) recoverTopLevel() {
	braceDepth := 0
	for !p.atEOF() {
		switch {
		case p.match("{"):
			braceDepth++
		case p.match("}"):
			if braceDepth == 0 {
				return
			}
			braceDepth--
			if braceDepth == 0 {
				return
			}
		case braceDepth == 0 && p.match(";"):
			return
		default:
			p.advance()
		}
	}
}

func (p *interfaceParser) require(text string) bool {
	if p.match(text) {
		return true
	}
	p.addDiagnostic("CIDL1002", fmt.Sprintf("expected %q", text), p.current().span)
	return false
}

func (p *interfaceParser) match(text string) bool {
	if !p.at(text) {
		return false
	}
	p.advance()
	return true
}

func (p *interfaceParser) at(text string) bool {
	return p.current().text == text
}

func (p *interfaceParser) atEOF() bool {
	return p.current().kind == tokenEOF
}

func (p *interfaceParser) current() token {
	if p.position < 0 || p.position >= len(p.tokens) {
		return token{kind: tokenEOF, span: sourceSpan{start: len(p.source), end: len(p.source)}}
	}
	return p.tokens[p.position]
}

func (p *interfaceParser) previous() token {
	if p.position <= 0 || p.position-1 >= len(p.tokens) {
		return token{kind: tokenEOF}
	}
	return p.tokens[p.position-1]
}

func (p *interfaceParser) previousEnd() int {
	if p.position <= 0 {
		return p.current().span.start
	}
	return p.previous().span.end
}

func (p *interfaceParser) advance() {
	if !p.atEOF() {
		p.position++
	}
}

func (p *interfaceParser) addDuplicate(kind, name string, current, previous sourceSpan) {
	p.addDiagnostic(
		"CIDL1005",
		fmt.Sprintf("duplicate %s %q; first declaration starts at byte %d", kind, name, previous.start),
		current,
	)
}

func (p *interfaceParser) addDiagnostic(code, message string, span sourceSpan) {
	if span.start < 0 {
		span.start = 0
	}
	if span.start > len(p.source) {
		span.start = len(p.source)
	}
	if span.end < span.start {
		span.end = span.start
	}
	if span.end > len(p.source) {
		span.end = len(p.source)
	}
	line, column := lineColumn(p.source, span.start)
	p.diagnostics = append(p.diagnostics, Diagnostic{
		Code: code, Message: message, StartByte: span.start, EndByte: span.end, Line: line, Column: column,
	})
}

func lineColumn(source []byte, position int) (int, int) {
	line, column := 1, 1
	if position > len(source) {
		position = len(source)
	}
	for index := 0; index < position; index++ {
		if source[index] == '\n' {
			line++
			column = 1
		} else {
			column++
		}
	}
	return line, column
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isIdentifierContinue(value byte) bool {
	return isIdentifierStart(value) || value >= '0' && value <= '9'
}

func validDecimalInteger(value string) bool {
	if value == "" || value[0] < '0' || value[0] > '9' || value[len(value)-1] < '0' || value[len(value)-1] > '9' {
		return false
	}
	for index := 1; index < len(value)-1; index++ {
		if value[index] == '_' && (value[index-1] < '0' || value[index-1] > '9' || value[index+1] < '0' || value[index+1] > '9') {
			return false
		}
	}
	return true
}
