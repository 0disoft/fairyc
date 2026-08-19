package contractabi

import (
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"
)

const (
	// MaxSourceBytes bounds one contract-interface source before token allocation.
	MaxSourceBytes = 8 << 20
	maxTokens      = 1 << 20
	maxParseDepth  = 256
	maxArrayLength = 65535
)

type tokenKind uint8

const (
	tokenInvalid tokenKind = iota
	tokenEOF
	tokenIdentifier
	tokenInteger
	tokenSymbol
)

type token struct {
	kind tokenKind
	text string
	span sourceSpan
}

var reservedWords = map[string]struct{}{
	"module": {}, "extern": {}, "contract": {}, "fn": {}, "struct": {},
	"ref": {}, "ptr": {}, "mut": {},
}

// Parse reads one experimental contract-interface module. It does not mutate or extend the Stage 0
// product grammar; the dedicated profile is consumed by the contract IDL tool only.
func Parse(input []byte) (Document, []Diagnostic) {
	parser := newInterfaceParser(input)
	document := parser.parseDocument()
	sort.SliceStable(parser.diagnostics, func(i, j int) bool {
		left, right := parser.diagnostics[i], parser.diagnostics[j]
		if left.StartByte != right.StartByte {
			return left.StartByte < right.StartByte
		}
		if left.EndByte != right.EndByte {
			return left.EndByte < right.EndByte
		}
		return left.Code < right.Code
	})
	return document, parser.diagnostics
}

type interfaceParser struct {
	source      []byte
	tokens      []token
	position    int
	depth       int
	diagnostics []Diagnostic
}

func newInterfaceParser(input []byte) *interfaceParser {
	parser := &interfaceParser{source: append([]byte(nil), input...)}
	if len(input) > MaxSourceBytes {
		parser.addDiagnostic("CIDL1001", fmt.Sprintf("contract interface exceeds %d bytes", MaxSourceBytes), sourceSpan{})
		parser.tokens = []token{{kind: tokenEOF, span: sourceSpan{start: len(input), end: len(input)}}}
		return parser
	}
	parser.tokens = parser.scan()
	return parser
}

func (p *interfaceParser) scan() []token {
	if !utf8.Valid(p.source) {
		p.addDiagnostic("CIDL1001", "contract interface is not valid UTF-8", sourceSpan{})
	}
	result := make([]token, 0, len(p.source)/4+1)
	position := 0
	if len(p.source) >= 3 && p.source[0] == 0xef && p.source[1] == 0xbb && p.source[2] == 0xbf {
		position = 3
	}
	appendToken := func(kind tokenKind, start, end int) bool {
		if len(result) >= maxTokens {
			p.addDiagnostic("CIDL1001", fmt.Sprintf("contract interface exceeds %d tokens", maxTokens), sourceSpan{start: start, end: end})
			return false
		}
		result = append(result, token{kind: kind, text: string(p.source[start:end]), span: sourceSpan{start: start, end: end}})
		return true
	}
	for position < len(p.source) {
		current := p.source[position]
		switch {
		case current == ' ' || current == '\t' || current == '\n' || current == '\r':
			position++
		case current == '/' && position+1 < len(p.source) && p.source[position+1] == '/':
			position += 2
			for position < len(p.source) && p.source[position] != '\n' && p.source[position] != '\r' {
				position++
			}
		case current == '/' && position+1 < len(p.source) && p.source[position+1] == '*':
			start := position
			position += 2
			depth := 1
			for position < len(p.source) && depth > 0 {
				switch {
				case position+1 < len(p.source) && p.source[position] == '/' && p.source[position+1] == '*':
					depth++
					position += 2
				case position+1 < len(p.source) && p.source[position] == '*' && p.source[position+1] == '/':
					depth--
					position += 2
				default:
					_, width := utf8.DecodeRune(p.source[position:])
					if width < 1 {
						width = 1
					}
					position += width
				}
			}
			if depth != 0 {
				p.addDiagnostic("CIDL1001", "unterminated block comment", sourceSpan{start: start, end: len(p.source)})
			}
		case isIdentifierStart(current):
			start := position
			position++
			for position < len(p.source) && isIdentifierContinue(p.source[position]) {
				position++
			}
			if !appendToken(tokenIdentifier, start, position) {
				position = len(p.source)
			}
		case current >= '0' && current <= '9':
			start := position
			position++
			for position < len(p.source) && (p.source[position] >= '0' && p.source[position] <= '9' || p.source[position] == '_') {
				position++
			}
			if !appendToken(tokenInteger, start, position) {
				position = len(p.source)
			}
		case current == ':' && position+1 < len(p.source) && p.source[position+1] == ':':
			if !appendToken(tokenSymbol, position, position+2) {
				position = len(p.source)
			} else {
				position += 2
			}
		case strings.ContainsRune("@(){}[]<>,;", rune(current)):
			if !appendToken(tokenSymbol, position, position+1) {
				position = len(p.source)
			} else {
				position++
			}
		default:
			start := position
			_, width := utf8.DecodeRune(p.source[position:])
			if width < 1 {
				width = 1
			}
			position += width
			p.addDiagnostic("CIDL1001", fmt.Sprintf("unexpected source token %q", string(p.source[start:position])), sourceSpan{start: start, end: position})
			if !appendToken(tokenInvalid, start, position) {
				position = len(p.source)
			}
		}
	}
	result = append(result, token{kind: tokenEOF, span: sourceSpan{start: len(p.source), end: len(p.source)}})
	return result
}
