package contractabi

import (
	"fmt"
	"strconv"
	"strings"
)

func (p *interfaceParser) parseType() (Type, bool) {
	if p.depth >= maxParseDepth {
		p.addDiagnostic("CIDL1001", fmt.Sprintf("type nesting exceeds %d levels", maxParseDepth), p.current().span)
		return Type{}, false
	}
	p.depth++
	defer func() { p.depth-- }()
	start := p.current().span.start
	if p.at("ref") || p.at("ptr") {
		kind := TypeReference
		if p.at("ptr") {
			kind = TypePointer
		}
		p.advance()
		p.match("mut")
		element, ok := p.parseType()
		if !ok {
			return Type{}, false
		}
		return Type{Kind: kind, Element: &element, span: sourceSpan{start: start, end: element.span.end}}, true
	}
	name, _, ok := p.parsePath()
	if !ok {
		p.addDiagnostic("CIDL1002", "expected ABI type", p.current().span)
		return Type{}, false
	}
	value := Type{Kind: TypeNamed, Name: name, span: sourceSpan{start: start, end: p.previousEnd()}}
	if p.match("<") {
		for !p.at(">") && !p.atEOF() {
			argument, parsed := p.parseType()
			if !parsed {
				return Type{}, false
			}
			value.Arguments = append(value.Arguments, argument)
			if !p.match(",") {
				break
			}
		}
		p.require(">")
		value.span.end = p.previousEnd()
	}
	for p.match("[") {
		postfixStart := value.span.start
		if p.match("]") {
			copyValue := value
			value = Type{Kind: TypeSlice, Element: &copyValue, span: sourceSpan{start: postfixStart, end: p.previousEnd()}}
			continue
		}
		lengthToken := p.current()
		if lengthToken.kind != tokenInteger {
			p.addDiagnostic("CIDL1002", "fixed-array length must be a decimal integer", lengthToken.span)
			return Type{}, false
		}
		p.advance()
		if !validDecimalInteger(lengthToken.text) {
			p.addDiagnostic("CIDL1002", "fixed-array length has invalid underscore placement", lengthToken.span)
		}
		lengthText := strings.ReplaceAll(lengthToken.text, "_", "")
		length, err := strconv.ParseUint(lengthText, 10, 64)
		if err != nil {
			p.addDiagnostic("CIDL1007", "fixed-array length is outside the supported unsigned range", lengthToken.span)
			length = 0
		}
		p.require("]")
		copyValue := value
		value = Type{Kind: TypeArray, Element: &copyValue, Length: &length, span: sourceSpan{start: postfixStart, end: p.previousEnd()}}
	}
	return value, true
}

func (p *interfaceParser) validateType(value Type, allowVoid bool) {
	switch value.Kind {
	case TypeReference:
		p.addDiagnostic("CIDL1007", "safe references cannot cross a contract boundary", value.span)
		return
	case TypePointer:
		p.addDiagnostic("CIDL1007", "raw pointers cannot cross a contract boundary", value.span)
		return
	case TypeSlice:
		p.addDiagnostic("CIDL1007", "unbounded slices cannot cross a contract boundary; use Bytes or a fixed array", value.span)
		return
	case TypeArray:
		if value.Length == nil || *value.Length == 0 || *value.Length > maxArrayLength {
			p.addDiagnostic("CIDL1007", fmt.Sprintf("fixed-array length must be in 1..%d", maxArrayLength), value.span)
		}
		if value.Element != nil {
			p.validateType(*value.Element, false)
		}
		return
	case TypeNamed:
		if value.Name == "void" {
			if !allowVoid {
				p.addDiagnostic("CIDL1007", "void is valid only as a function result or Result success payload", value.span)
			}
			if len(value.Arguments) != 0 {
				p.addDiagnostic("CIDL1007", "void does not accept type arguments", value.span)
			}
			return
		}
		if value.Name == "usize" || value.Name == "isize" || value.Name == "str" || strings.HasPrefix(value.Name, "c::") {
			p.addDiagnostic("CIDL1007", fmt.Sprintf("target-dependent or borrowed type %s is not chain-neutral", value.Name), value.span)
		}
		switch value.Name {
		case "Option":
			if len(value.Arguments) != 1 {
				p.addDiagnostic("CIDL1007", "Option requires exactly one ABI type argument", value.span)
				return
			}
			p.validateType(value.Arguments[0], false)
		case "Result":
			if len(value.Arguments) != 2 {
				p.addDiagnostic("CIDL1007", "Result requires exactly two ABI type arguments", value.span)
				return
			}
			p.validateType(value.Arguments[0], true)
			p.validateType(value.Arguments[1], false)
		default:
			if len(value.Arguments) != 0 {
				p.addDiagnostic("CIDL1007", fmt.Sprintf("generic ABI type %s is not supported by contract IDL v0", value.Name), value.span)
			}
		}
	default:
		p.addDiagnostic("CIDL1007", "invalid ABI type", value.span)
	}
}

func (p *interfaceParser) parsePath() (string, sourceSpan, bool) {
	first, span, ok := p.parseIdentifier("path segment")
	if !ok {
		return "", sourceSpan{}, false
	}
	parts := []string{first}
	resultSpan := span
	for p.match("::") {
		part, partSpan, parsed := p.parseIdentifier("path segment")
		if !parsed {
			return strings.Join(parts, "::"), resultSpan, false
		}
		parts = append(parts, part)
		resultSpan.end = partSpan.end
	}
	return strings.Join(parts, "::"), resultSpan, true
}

func (p *interfaceParser) parseIdentifier(role string) (string, sourceSpan, bool) {
	current := p.current()
	if current.kind != tokenIdentifier {
		p.addDiagnostic("CIDL1002", "expected "+role, current.span)
		return "", current.span, false
	}
	if current.text == "_" {
		p.addDiagnostic("CIDL1002", fmt.Sprintf("blank identifier cannot be used as %s", role), current.span)
		p.advance()
		return "", current.span, false
	}
	if _, reserved := reservedWords[current.text]; reserved {
		p.addDiagnostic("CIDL1002", fmt.Sprintf("reserved word %q cannot be used as %s", current.text, role), current.span)
		p.advance()
		return "", current.span, false
	}
	p.advance()
	return current.text, current.span, true
}
