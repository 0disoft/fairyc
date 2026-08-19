package contractabi

import "fmt"

func (p *interfaceParser) parseDocument() Document {
	document := Document{Schema: SchemaVersion, Contracts: []Contract{}, Events: []Event{}}
	p.require("module")
	module, moduleSpan, ok := p.parsePath()
	if ok {
		document.Module = module
	}
	p.require(";")

	contractNames := make(map[string]sourceSpan)
	eventNames := make(map[string]sourceSpan)
	for !p.atEOF() {
		startPosition := p.position
		attributes := p.parseAttributes()
		switch {
		case p.at("extern"):
			contract, ok := p.parseContract(attributes)
			if ok {
				if previous, duplicate := contractNames[contract.Name]; duplicate {
					p.addDuplicate("contract", contract.Name, contract.nameSpan, previous)
				} else if previous, collision := eventNames[contract.Name]; collision {
					p.addDuplicate("top-level declaration", contract.Name, contract.nameSpan, previous)
				} else {
					contractNames[contract.Name] = contract.nameSpan
					document.Contracts = append(document.Contracts, contract.Contract)
				}
			}
		case p.at("struct"):
			event, ok := p.parseEvent(attributes)
			if ok {
				if previous, duplicate := eventNames[event.Name]; duplicate {
					p.addDuplicate("event", event.Name, event.nameSpan, previous)
				} else if previous, collision := contractNames[event.Name]; collision {
					p.addDuplicate("top-level declaration", event.Name, event.nameSpan, previous)
				} else {
					eventNames[event.Name] = event.nameSpan
					document.Events = append(document.Events, event.Event)
				}
			}
		default:
			span := p.current().span
			if len(attributes) > 0 {
				span = attributes[0].span
			}
			p.addDiagnostic("CIDL1003", "expected extern contract or @event struct declaration", span)
			p.recoverTopLevel()
		}
		if p.position == startPosition && !p.atEOF() {
			p.advance()
		}
	}
	if document.Module == "" && moduleSpan.end > moduleSpan.start {
		document.Module = string(p.source[moduleSpan.start:moduleSpan.end])
	}
	if len(document.Contracts) == 0 && len(document.Events) == 0 && len(p.diagnostics) == 0 {
		p.addDiagnostic("CIDL1003", "contract interface declares no contracts or events", p.current().span)
	}
	if len(p.diagnostics) == 0 {
		if err := validateDocument(document); err != nil {
			p.addDiagnostic("CIDL1007", err.Error(), p.current().span)
		}
	}
	return document
}

func (p *interfaceParser) parseContract(attributes []parsedAttribute) (parsedContract, bool) {
	start := p.current().span.start
	for _, attribute := range attributes {
		p.addDiagnostic("CIDL1006", fmt.Sprintf("attribute @%s is not valid on a contract declaration", attribute.name), attribute.span)
	}
	p.require("extern")
	if !p.require("contract") {
		p.recoverTopLevel()
		return parsedContract{}, false
	}
	name, nameSpan, ok := p.parseIdentifier("contract name")
	if !ok {
		p.recoverTopLevel()
		return parsedContract{}, false
	}
	if !p.require("{") {
		p.recoverTopLevel()
		return parsedContract{}, false
	}
	contract := parsedContract{Contract: Contract{Name: name, Functions: []Function{}}, nameSpan: nameSpan}
	seenMethods := make(map[string]sourceSpan)
	for !p.at("}") && !p.atEOF() {
		before := p.position
		methodAttributes := p.parseAttributes()
		if !p.at("fn") {
			p.addDiagnostic("CIDL1003", "expected contract function declaration", p.current().span)
			p.recoverMember()
		} else {
			method, parsed := p.parseFunction(methodAttributes)
			if parsed {
				if previous, duplicate := seenMethods[method.Name]; duplicate {
					p.addDuplicate("contract function", method.Name, method.nameSpan, previous)
				} else {
					seenMethods[method.Name] = method.nameSpan
					contract.methods = append(contract.methods, method)
					contract.Functions = append(contract.Functions, method.Function)
				}
			}
		}
		if p.position == before && !p.atEOF() {
			p.advance()
		}
	}
	p.require("}")
	contract.span = sourceSpan{start: start, end: p.previousEnd()}
	return contract, true
}

func (p *interfaceParser) parseFunction(attributes []parsedAttribute) (parsedFunction, bool) {
	start := p.current().span.start
	p.require("fn")
	result, ok := p.parseType()
	if !ok {
		p.recoverMember()
		return parsedFunction{}, false
	}
	name, nameSpan, ok := p.parseIdentifier("function name")
	if !ok {
		p.recoverMember()
		return parsedFunction{}, false
	}
	if !p.require("(") {
		p.recoverMember()
		return parsedFunction{}, false
	}
	parameters := make([]Parameter, 0, 4)
	parameterNames := make(map[string]sourceSpan)
	for !p.at(")") && !p.atEOF() {
		parameterType, parsed := p.parseType()
		if !parsed {
			p.recoverMember()
			return parsedFunction{}, false
		}
		parameterName, parameterSpan, parsed := p.parseIdentifier("parameter name")
		if !parsed {
			p.recoverMember()
			return parsedFunction{}, false
		}
		if previous, duplicate := parameterNames[parameterName]; duplicate {
			p.addDuplicate("parameter", parameterName, parameterSpan, previous)
		} else {
			parameterNames[parameterName] = parameterSpan
			parameters = append(parameters, Parameter{Name: parameterName, Type: parameterType})
		}
		if !p.match(",") {
			break
		}
	}
	p.require(")")
	p.require(";")

	view, payable := false, false
	seenAttributes := make(map[string]sourceSpan)
	for _, attribute := range attributes {
		if previous, duplicate := seenAttributes[attribute.name]; duplicate {
			p.addDuplicate("attribute", "@"+attribute.name, attribute.span, previous)
			continue
		}
		seenAttributes[attribute.name] = attribute.span
		switch attribute.name {
		case "view":
			view = true
		case "payable":
			payable = true
		default:
			p.addDiagnostic("CIDL1006", fmt.Sprintf("unknown contract function attribute @%s", attribute.name), attribute.span)
		}
	}
	if view && payable {
		p.addDiagnostic("CIDL1008", "@view and @payable cannot appear on the same contract function", nameSpan)
	}
	p.validateType(result, true)
	for _, parameter := range parameters {
		p.validateType(parameter.Type, false)
	}
	function := parsedFunction{
		Function: Function{
			Name:       name,
			Parameters: parameters,
			Result:     result,
			View:       view,
			Payable:    payable,
			Effects:    []string{externalCallEffect},
		},
		span:     sourceSpan{start: start, end: p.previousEnd()},
		nameSpan: nameSpan,
		attrs:    attributes,
	}
	return function, true
}

func (p *interfaceParser) parseEvent(attributes []parsedAttribute) (parsedEvent, bool) {
	start := p.current().span.start
	eventAttributeCount := 0
	for _, attribute := range attributes {
		if attribute.name == "event" {
			eventAttributeCount++
			continue
		}
		p.addDiagnostic("CIDL1006", fmt.Sprintf("unknown event attribute @%s", attribute.name), attribute.span)
	}
	if eventAttributeCount != 1 {
		span := p.current().span
		if len(attributes) > 0 {
			span = attributes[0].span
		}
		p.addDiagnostic("CIDL1006", "event structs require exactly one @event attribute", span)
	}
	p.require("struct")
	name, nameSpan, ok := p.parseIdentifier("event name")
	if !ok {
		p.recoverTopLevel()
		return parsedEvent{}, false
	}
	if !p.require("{") {
		p.recoverTopLevel()
		return parsedEvent{}, false
	}
	event := parsedEvent{Event: Event{Name: name, Fields: []Field{}}, nameSpan: nameSpan}
	fieldNames := make(map[string]sourceSpan)
	for !p.at("}") && !p.atEOF() {
		before := p.position
		fieldType, parsed := p.parseType()
		if !parsed {
			p.recoverMember()
		} else {
			fieldName, fieldSpan, named := p.parseIdentifier("event field name")
			if named {
				p.require(";")
				p.validateType(fieldType, false)
				field := parsedField{Field: Field{Name: fieldName, Type: fieldType}, nameSpan: fieldSpan}
				if previous, duplicate := fieldNames[fieldName]; duplicate {
					p.addDuplicate("event field", fieldName, fieldSpan, previous)
				} else {
					fieldNames[fieldName] = fieldSpan
					event.fields = append(event.fields, field)
					event.Fields = append(event.Fields, field.Field)
				}
			} else {
				p.recoverMember()
			}
		}
		if p.position == before && !p.atEOF() {
			p.advance()
		}
	}
	p.require("}")
	event.span = sourceSpan{start: start, end: p.previousEnd()}
	return event, true
}

func (p *interfaceParser) parseAttributes() []parsedAttribute {
	attributes := make([]parsedAttribute, 0, 2)
	for p.match("@") {
		start := p.previous().span.start
		name, nameSpan, ok := p.parseIdentifier("attribute name")
		if !ok {
			return attributes
		}
		attribute := parsedAttribute{name: name, span: sourceSpan{start: start, end: nameSpan.end}}
		if p.match("(") {
			p.addDiagnostic("CIDL1006", "contract IDL v0 attributes do not accept arguments", attribute.span)
			depth := 1
			for !p.atEOF() && depth > 0 {
				switch {
				case p.match("("):
					depth++
				case p.match(")"):
					depth--
				default:
					p.advance()
				}
			}
			attribute.span.end = p.previousEnd()
		}
		attributes = append(attributes, attribute)
	}
	return attributes
}
