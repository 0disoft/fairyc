package syntax

import (
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
)

func (p *parser) parseSource() NodeID {
	if !p.enter() {
		return NoNode
	}
	defer p.leave()
	start := p.currentFullIndex()
	children := []NodeID{p.parseModule()}
	for !p.budget.Exhausted() && p.at("import") {
		children = append(children, p.parseImport())
	}
	for p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		children = append(children, p.parseItem())
	}
	return p.addNode(KindSource, start, children, "", "")
}

func (p *parser) parseModule() NodeID {
	start := p.currentFullIndex()
	p.expect("module")
	pathID, modulePath, ok := p.pathNode()
	p.expect(";")
	p.tree.Module = modulePath
	if ok {
		expected, err := p.file.ExpectedModule()
		if err != nil || expected != modulePath {
			span := diagnostic.Span{}
			if node, found := p.tree.Node(pathID); found {
				span = node.Span
			}
			p.emit("E0002", span, diagnostic.RepairNone, "")
		}
	}
	return p.addNode(KindModule, start, []NodeID{pathID}, modulePath, "")
}

func (p *parser) parseImport() NodeID {
	start := p.currentFullIndex()
	startByte := p.current().Span.StartByte
	p.advance()
	pathID, importedPath, _ := p.pathNode()
	children := []NodeID{pathID}
	alias := ""
	if !p.match("as") {
		p.synchronize(";", "}")
		endByte := p.current().Span.EndByte
		if p.current().Kind == lexer.EOF {
			endByte = p.current().Span.StartByte
		}
		p.emit("E0003", diagnostic.Span{StartByte: startByte, EndByte: endByte}, diagnostic.RepairSynchronize, ";")
		p.match(";")
		return p.addNode(KindImport, start, children, importedPath, "")
	}
	if p.atKind(lexer.Identifier) {
		alias = p.currentText()
		p.advance()
	} else {
		children = append(children, p.unexpected())
	}
	p.expect(";")
	return p.addNode(KindImport, start, children, importedPath, alias)
}

func (p *parser) parseItem() NodeID {
	start := p.currentFullIndex()
	children := make([]NodeID, 0, 4)
	for !p.budget.Exhausted() && p.at("@") {
		children = append(children, p.parseAttribute())
	}
	p.match("pub")
	unsafeItem := p.match("unsafe")
	switch p.currentText() {
	case "const":
		return p.parseConst(start, children)
	case "struct":
		return p.parseStruct(start, children)
	case "resource":
		return p.parseResource(start, children)
	case "enum":
		return p.parseEnum(start, children)
	case "impl":
		return p.parseImpl(start, children)
	case "fn":
		return p.parseFunction(start, children, false)
	case "extern":
		if unsafeItem {
			return p.parseForeignFunction(start, children)
		}
	}
	children = append(children, p.unexpected())
	p.synchronize(";", "}")
	p.match(";")
	return p.addNode(KindError, start, children, "item", "")
}

func (p *parser) parseAttribute() NodeID {
	start := p.currentFullIndex()
	p.advance()
	pathID, value, _ := p.pathNode()
	children := []NodeID{pathID}
	if p.match("(") {
		for !p.at(")") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
			if isLiteralToken(p.current(), p.currentText()) {
				children = append(children, p.parseLiteral())
			} else {
				argument, _, _ := p.pathNode()
				children = append(children, argument)
			}
			if !p.match(",") {
				break
			}
		}
		p.expect(")")
	}
	return p.addNode(KindAttribute, start, children, value, "")
}

func (p *parser) parseConst(start int, leading []NodeID) NodeID {
	p.advance()
	var name string
	var nameSpan diagnostic.Span
	var typeID NodeID
	if p.atKind(lexer.Identifier) && p.peekText(1) == ":" {
		name, nameSpan = p.parseNameWithSpan()
		p.expect(":")
		typeID = p.parseType()
	} else {
		typeID = p.parseType()
		name, nameSpan = p.parseNameWithSpan()
	}
	p.expect("=")
	expression := p.parseExpression(0, true)
	p.expect(";")
	children := append(append([]NodeID(nil), leading...), typeID, expression)
	return p.addNamedNode(KindConst, start, children, name, "", nameSpan)
}

func (p *parser) parseStruct(start int, leading []NodeID) NodeID {
	p.advance()
	name, nameSpan := p.parseNameWithSpan()
	p.expect("{")
	children := append([]NodeID(nil), leading...)
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		children = append(children, p.parseField())
	}
	p.expect("}")
	return p.addNamedNode(KindStruct, start, children, name, "", nameSpan)
}

func (p *parser) parseResource(start int, leading []NodeID) NodeID {
	p.advance()
	name, nameSpan := p.parseNameWithSpan()
	p.expect("{")
	children := append([]NodeID(nil), leading...)
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		if p.at("cleanup") {
			children = append(children, p.parseCleanup(nil))
			break
		}
		if p.at("@") {
			attributes := make([]NodeID, 0, 2)
			for p.at("@") && !p.budget.Exhausted() {
				attributes = append(attributes, p.parseAttribute())
			}
			if p.at("cleanup") {
				children = append(children, p.parseCleanup(attributes))
				break
			}
			children = append(children, attributes...)
			continue
		}
		if p.at("drop") {
			children = append(children, p.parseDrop())
			break
		}
		children = append(children, p.parseField())
	}
	p.expect("}")
	return p.addNamedNode(KindResource, start, children, name, "", nameSpan)
}

func (p *parser) parseCleanup(leading []NodeID) NodeID {
	start := p.currentFullIndex()
	p.advance()
	block := p.parseBlock()
	children := append(append([]NodeID(nil), leading...), block)
	return p.addNode(KindCleanup, start, children, "cleanup", "")
}

func (p *parser) parseDrop() NodeID {
	start := p.currentFullIndex()
	p.advance()
	p.expect("fn")
	p.expect("(")
	p.expect("self")
	p.expect(":")
	p.expect("&")
	p.expect("mut")
	if !p.at("Self") {
		p.expect("Self")
	} else {
		p.advance()
	}
	p.expect(")")
	block := p.parseBlock()
	return p.addNode(KindFunction, start, []NodeID{block}, "drop", "")
}

func (p *parser) parseEnum(start int, leading []NodeID) NodeID {
	p.advance()
	name, nameSpan := p.parseNameWithSpan()
	p.expect("{")
	children := append([]NodeID(nil), leading...)
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		variantStart := p.currentFullIndex()
		variantName, variantNameSpan := p.parseNameWithSpan()
		variantChildren := make([]NodeID, 0, 2)
		if p.match("{") {
			for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
				variantChildren = append(variantChildren, p.parseEnumField())
			}
			p.expect("}")
		}
		p.expect(";")
		children = append(children, p.addNamedNode(KindVariant, variantStart, variantChildren, variantName, "", variantNameSpan))
	}
	p.expect("}")
	return p.addNamedNode(KindEnum, start, children, name, "", nameSpan)
}

func (p *parser) parseEnumField() NodeID {
	start := p.currentFullIndex()
	name, nameSpan, typeID := p.parseFieldHead()
	p.expect(";")
	return p.addNamedNode(KindField, start, []NodeID{typeID}, name, "", nameSpan)
}

func (p *parser) parseImpl(start int, leading []NodeID) NodeID {
	p.advance()
	typeID := p.parseType()
	p.expect("{")
	children := append(append([]NodeID(nil), leading...), typeID)
	for !p.at("}") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		methodStart := p.currentFullIndex()
		methodLeading := make([]NodeID, 0, 2)
		for !p.budget.Exhausted() && p.at("@") {
			methodLeading = append(methodLeading, p.parseAttribute())
		}
		p.match("pub")
		p.match("unsafe")
		children = append(children, p.parseFunction(methodStart, methodLeading, true))
	}
	p.expect("}")
	return p.addNode(KindImpl, start, children, "", "")
}

func (p *parser) parseFunction(start int, leading []NodeID, method bool) NodeID {
	p.expect("fn")
	var name string
	var nameSpan diagnostic.Span
	var resultType NodeID
	var parameters []NodeID
	cShaped := false
	if p.atKind(lexer.Identifier) && p.peekText(1) == "(" {
		name, nameSpan = p.parseNameWithSpan()
		parameters = p.parseParameters(method, true)
		p.expect("->")
		resultType = p.parseType()
	} else {
		cShaped = true
		resultType = p.parseType()
		name, nameSpan = p.parseNameWithSpan()
		parameters = p.parseParameters(method, true)
	}
	body := p.parseBlockWithTail(!cShaped)
	children := append(append([]NodeID(nil), leading...), parameters...)
	children = append(children, resultType, body)
	return p.addNamedNode(KindFunction, start, children, name, "", nameSpan)
}

func (p *parser) parseForeignFunction(start int, leading []NodeID) NodeID {
	p.expect("extern")
	if !p.at("C") {
		p.expect("C")
	} else {
		p.advance()
	}
	p.expect("fn")
	var name string
	var nameSpan diagnostic.Span
	var resultType NodeID
	var parameters []NodeID
	if p.atKind(lexer.Identifier) && p.peekText(1) == "(" {
		name, nameSpan = p.parseNameWithSpan()
		parameters = p.parseParameters(false, false)
		p.expect("->")
		resultType = p.parseType()
	} else {
		resultType = p.parseType()
		name, nameSpan = p.parseNameWithSpan()
		parameters = p.parseParameters(false, false)
	}
	p.expect(";")
	children := append(append([]NodeID(nil), leading...), parameters...)
	children = append(children, resultType)
	return p.addNamedNode(KindForeign, start, children, name, "", nameSpan)
}

func (p *parser) parseParameters(method bool, allowMutable bool) []NodeID {
	p.expect("(")
	parameters := make([]NodeID, 0, 4)
	if method && p.at("self") {
		parameters = append(parameters, p.parseReceiver())
		if !p.match(",") {
			p.expect(")")
			return parameters
		}
	}
	for !p.at(")") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
		start := p.currentFullIndex()
		modifier := ""
		var name string
		var nameSpan diagnostic.Span
		var typeID NodeID
		legacy := allowMutable && p.at("mut") || p.atKind(lexer.Identifier) && p.peekText(1) == ":"
		if legacy {
			if allowMutable && p.match("mut") {
				modifier = "mut"
			}
			name, nameSpan = p.parseNameWithSpan()
			p.expect(":")
			typeID = p.parseType()
		} else {
			typeID = p.parseType()
			name, nameSpan = p.parseNameWithSpan()
		}
		parameters = append(parameters, p.addNamedNode(KindParameter, start, []NodeID{typeID}, name, modifier, nameSpan))
		if !p.match(",") {
			break
		}
	}
	p.expect(")")
	return parameters
}

func (p *parser) parseReceiver() NodeID {
	start := p.currentFullIndex()
	p.advance()
	p.expect(":")
	if p.match("&") {
		p.match("mut")
	}
	if !p.at("Self") {
		p.expect("Self")
	} else {
		p.advance()
	}
	return p.addNode(KindParameter, start, nil, "self", "")
}

func (p *parser) parseField() NodeID {
	start := p.currentFullIndex()
	p.match("pub")
	name, nameSpan, typeID := p.parseFieldHead()
	p.expect(";")
	return p.addNamedNode(KindField, start, []NodeID{typeID}, name, "", nameSpan)
}

func (p *parser) parseFieldHead() (string, diagnostic.Span, NodeID) {
	if p.atKind(lexer.Identifier) && p.peekText(1) == ":" {
		name, nameSpan := p.parseNameWithSpan()
		p.expect(":")
		return name, nameSpan, p.parseType()
	}
	typeID := p.parseType()
	name, nameSpan := p.parseNameWithSpan()
	return name, nameSpan, typeID
}

func (p *parser) parseType() NodeID {
	if !p.enter() {
		return NoNode
	}
	defer p.leave()
	start := p.currentFullIndex()
	var result NodeID
	switch {
	case p.match("ref"):
		form := TypeFormReference
		if p.match("mut") {
			form = TypeFormReferenceMut
		}
		child := p.parseType()
		result = p.addTypeNode(start, []NodeID{child}, form)
	case p.match("ptr"):
		form := TypeFormPointer
		if p.match("mut") {
			form = TypeFormPointerMut
		}
		child := p.parseType()
		result = p.addTypeNode(start, []NodeID{child}, form)
	case p.match("&"):
		form := TypeFormReference
		if p.match("mut") {
			form = TypeFormReferenceMut
		}
		child := p.parseType()
		result = p.addTypeNode(start, []NodeID{child}, form)
	case p.match("*"):
		form := TypeFormPointer
		if p.match("mut") {
			form = TypeFormPointerMut
		}
		child := p.parseType()
		result = p.addTypeNode(start, []NodeID{child}, form)
	case p.match("["):
		children := []NodeID{p.parseType()}
		form := TypeFormSlice
		if p.match(";") {
			form = TypeFormArray
			children = append(children, p.parseConstExpression())
		}
		p.expect("]")
		result = p.addTypeNode(start, children, form)
	case p.match("("):
		child := p.parseType()
		p.expect(")")
		result = p.addTypeNode(start, []NodeID{child}, TypeFormGrouped)
	case p.match("unsafe"):
		p.expect("extern")
		if p.at("C") {
			p.advance()
		} else {
			p.expect("C")
		}
		p.expect("fn")
		p.expect("(")
		children := make([]NodeID, 0, 4)
		for !p.at(")") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
			children = append(children, p.parseType())
			if !p.match(",") {
				break
			}
		}
		p.expect(")")
		p.expect("->")
		children = append(children, p.parseType())
		result = p.addTypeNode(start, children, TypeFormForeignFunction)
	default:
		pathID, _, _ := p.pathNode()
		children := []NodeID{pathID}
		if p.match("<") {
			for !p.at(">") && p.current().Kind != lexer.EOF && !p.budget.Exhausted() {
				children = append(children, p.parseType())
				if !p.match(",") {
					break
				}
			}
			p.expectTypeClose()
		}
		result = p.addTypeNode(start, children, TypeFormNamed)
	}
	return p.parseTypePostfix(result)
}

func (p *parser) parseTypePostfix(base NodeID) NodeID {
	for !p.budget.Exhausted() && p.match("[") {
		start := p.nodeStart(base)
		children := []NodeID{base}
		form := TypeFormSlice
		if !p.at("]") {
			form = TypeFormArray
			children = append(children, p.parseConstExpression())
		}
		p.expect("]")
		base = p.addTypeNode(start, children, form)
	}
	return base
}

func (p *parser) parseConstExpression() NodeID {
	if p.atKind(lexer.Integer) {
		return p.parseLiteral()
	}
	pathID, _, _ := p.pathNode()
	return pathID
}

func (p *parser) addTypeNode(start int, children []NodeID, form TypeForm) NodeID {
	return p.addNode(KindType, start, children, p.rawFrom(start), string(form))
}

func (p *parser) rawFrom(startFull int) string {
	if startFull < 0 || startFull >= len(p.tree.Tokens) || p.lastFullIndex < startFull || p.lastFullIndex >= len(p.tree.Tokens) {
		return ""
	}
	start := p.tree.Tokens[startFull].Span.StartByte
	end := p.lastEndByte
	return string(p.tree.Source[start:end])
}

func (p *parser) parseName() string {
	name, _ := p.parseNameWithSpan()
	return name
}

func (p *parser) parseNameWithSpan() (string, diagnostic.Span) {
	if p.atKind(lexer.Identifier) {
		name := p.currentText()
		span := p.current().Span
		p.advance()
		return name, span
	}
	p.unexpected()
	return "", diagnostic.Span{}
}
