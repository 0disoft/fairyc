package lexer

import (
	"cmp"
	"slices"
	"unicode/utf8"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/source"
)

var keywords = map[string]struct{}{
	"as": {}, "break": {}, "const": {}, "continue": {}, "discard": {},
	"drop": {}, "else": {}, "enum": {}, "extern": {}, "false": {},
	"fn": {}, "if": {}, "impl": {}, "import": {}, "in": {}, "let": {},
	"match": {}, "module": {}, "mut": {}, "null": {}, "pub": {},
	"region": {}, "resource": {}, "return": {}, "self": {}, "struct": {},
	"true": {}, "unsafe": {}, "var": {}, "while": {},
}

var symbols = []string{
	"::", "->", "=>", "==", "!=", "<=", ">=", "<<", ">>", "&&", "||",
	"@", "(", ")", "{", "}", "[", "]", ",", ";", ":", ".", "?", "=",
	"+", "-", "*", "/", "%", "!", "~", "&", "|", "^", "<", ">",
}

// Lex performs one bounded, batch lex over the file's original bytes.
func Lex(file source.File, budget *source.Budget) Result {
	lexer := scanner{
		source: append([]byte(nil), file.Bytes...),
		budget: budget,
	}
	lexer.run()
	return Result{
		Source:      lexer.source,
		Tokens:      lexer.tokens,
		Diagnostics: lexer.diagnostics,
	}
}

type scanner struct {
	source       []byte
	position     int
	budget       *source.Budget
	tokens       []Token
	diagnostics  []diagnostic.Diagnostic
	hasExhausted bool
}

func (s *scanner) run() {
	if s.budget == nil || s.budget.Exhausted() {
		s.emitExhausted()
		if len(s.source) > 0 {
			s.emitWithoutBudget(Invalid, 0, len(s.source), "")
			s.position = len(s.source)
		}
		s.emitEOF()
		return
	}
	if len(s.source) >= 3 && s.source[0] == 0xef && s.source[1] == 0xbb && s.source[2] == 0xbf {
		if !s.budget.ChargeWork(3) {
			s.emitExhausted()
			s.emitWithoutBudget(Invalid, 0, len(s.source), "")
			s.position = len(s.source)
			s.emitEOF()
			return
		}
		s.emit(Trivia, 0, 3, "")
		s.position = 3
	}
	for s.position < len(s.source) && !s.budget.Exhausted() {
		start := s.position
		if !s.budget.ChargeWork(1) {
			break
		}
		s.scanOne()
		if s.position <= start {
			s.position = start + 1
			s.emit(Invalid, start, s.position, "")
			s.emitDiagnostic("E0001", start, s.position, diagnostic.RepairNone, "")
		}
	}
	if s.budget.Exhausted() {
		s.emitExhausted()
		if s.position < len(s.source) {
			s.emitWithoutBudget(Invalid, s.position, len(s.source), "")
			s.position = len(s.source)
		}
	}
	s.emitEOF()
	slices.SortFunc(s.diagnostics, diagnostic.Compare)
}

func (s *scanner) scanOne() {
	current := s.source[s.position]
	if isHorizontalSpace(current) || current == '\n' || current == '\r' {
		s.scanWhitespace()
		return
	}
	if current == '/' && s.matchAt(s.position, "//") {
		s.scanLineComment()
		return
	}
	if current == '/' && s.matchAt(s.position, "/*") {
		s.scanBlockComment()
		return
	}
	if (current == 'b' || current == 'c') && s.position+1 < len(s.source) && s.source[s.position+1] == '"' {
		kind := ByteString
		if current == 'c' {
			kind = CString
		}
		s.scanString(kind, 2)
		return
	}
	if current == '"' {
		s.scanString(String, 1)
		return
	}
	if isIdentifierStart(current) {
		s.scanIdentifier()
		return
	}
	if isDecimalDigit(current) {
		s.scanNumber()
		return
	}
	for _, symbol := range symbols {
		if s.matchAt(s.position, symbol) {
			start := s.position
			if len(symbol) > 1 && !s.advanceCharged(len(symbol)-1) {
				return
			}
			if len(symbol) == 1 {
				s.position++
			} else {
				s.position++
			}
			s.emit(Symbol, start, s.position, "")
			return
		}
	}
	if current == 0 {
		s.position++
		s.emit(Invalid, s.position-1, s.position, "")
		s.emitDiagnostic("E0001", s.position-1, s.position, diagnostic.RepairNone, "")
		return
	}
	if current >= utf8.RuneSelf {
		_, width := utf8.DecodeRune(s.source[s.position:])
		if width == 0 {
			width = 1
		}
		start := s.position
		if width > 1 && !s.budget.ChargeWork(width-1) {
			return
		}
		s.position += width
		s.emit(Invalid, start, s.position, "")
		s.emitDiagnostic("E0001", start, s.position, diagnostic.RepairNone, "")
		return
	}
	start := s.position
	s.position++
	s.emit(Invalid, start, s.position, "")
	s.emitDiagnostic("E0001", start, s.position, diagnostic.RepairNone, "")
}

func (s *scanner) scanWhitespace() {
	start := s.position
	if s.source[s.position] == '\r' && (s.position+1 >= len(s.source) || s.source[s.position+1] != '\n') {
		s.position++
		s.emit(Invalid, start, s.position, "")
		s.emitDiagnostic("E0001", start, s.position, diagnostic.RepairNone, "")
		return
	}
	if s.source[s.position] == '\r' {
		if !s.advanceCharged(1) {
			return
		}
	}
	s.position++
	for s.position < len(s.source) {
		current := s.source[s.position]
		switch {
		case isHorizontalSpace(current) || current == '\n':
			if !s.advanceCharged(1) {
				s.emit(Invalid, start, s.position, "")
				return
			}
		case current == '\r' && s.position+1 < len(s.source) && s.source[s.position+1] == '\n':
			if !s.advanceCharged(2) {
				s.emit(Invalid, start, s.position, "")
				return
			}
		case current == '\r':
			if start < s.position {
				s.emit(Trivia, start, s.position, "")
			}
			invalidStart := s.position
			if !s.advanceCharged(1) {
				return
			}
			s.emit(Invalid, invalidStart, s.position, "")
			s.emitDiagnostic("E0001", invalidStart, s.position, diagnostic.RepairNone, "")
			return
		default:
			s.emit(Trivia, start, s.position, "")
			return
		}
	}
	if start < s.position {
		s.emit(Trivia, start, s.position, "")
	}
}

func (s *scanner) scanLineComment() {
	start := s.position
	if !s.advanceCharged(1) {
		return
	}
	s.position++
	for s.position < len(s.source) && s.source[s.position] != '\n' && s.source[s.position] != '\r' && !s.budget.Exhausted() {
		if !s.advanceCharged(1) {
			break
		}
	}
	s.emit(Trivia, start, s.position, "")
}

func (s *scanner) scanBlockComment() {
	start := s.position
	if !s.advanceCharged(1) {
		return
	}
	s.position++
	depth := 1
	for s.position < len(s.source) && depth > 0 && !s.budget.Exhausted() {
		switch {
		case s.matchAt(s.position, "/*"):
			if !s.advanceCharged(2) {
				break
			}
			depth++
		case s.matchAt(s.position, "*/"):
			if !s.advanceCharged(2) {
				break
			}
			depth--
		default:
			if !s.advanceCharged(1) {
				break
			}
		}
	}
	s.emit(Trivia, start, s.position, "")
	if depth != 0 {
		s.emitDiagnostic("E0001", start, s.position, diagnostic.RepairExit, "*/")
	}
}

func (s *scanner) scanIdentifier() {
	start := s.position
	s.position++
	for s.position < len(s.source) && isIdentifierContinue(s.source[s.position]) && !s.budget.Exhausted() {
		if !s.advanceCharged(1) {
			break
		}
	}
	if s.budget.Exhausted() {
		s.emit(Invalid, start, s.position, "")
		return
	}
	text := string(s.source[start:s.position])
	kind := Identifier
	if _, found := keywords[text]; found {
		kind = Keyword
	}
	s.emit(kind, start, s.position, literalKeywordCooked(text))
}

func (s *scanner) emit(kind Kind, start, end int, cooked string) {
	if !s.budget.ChargeToken() {
		if start < end {
			s.emitWithoutBudget(Invalid, start, end, "")
		}
		return
	}
	s.emitWithoutBudget(kind, start, end, cooked)
}

func (s *scanner) emitWithoutBudget(kind Kind, start, end int, cooked string) {
	s.tokens = append(s.tokens, Token{
		Kind:   kind,
		Span:   diagnostic.Span{StartByte: start, EndByte: end},
		Cooked: cooked,
	})
}

func (s *scanner) emitDiagnostic(code string, start, end int, repair diagnostic.Repair, token string) {
	if !s.budget.ChargeDiagnostic() {
		return
	}
	s.diagnostics = append(s.diagnostics, diagnostic.Diagnostic{
		Code:   code,
		Span:   diagnostic.Span{StartByte: start, EndByte: end},
		Repair: repair,
		Token:  token,
	})
}

func (s *scanner) emitExhausted() {
	if s.hasExhausted {
		return
	}
	s.hasExhausted = true
	s.diagnostics = append(s.diagnostics, source.ExhaustionDiagnostic(s.position))
}

func (s *scanner) emitEOF() {
	position := len(s.source)
	s.emitWithoutBudget(EOF, position, position, "")
}

func (s *scanner) matchAt(position int, value string) bool {
	return position >= 0 && position+len(value) <= len(s.source) && string(s.source[position:position+len(value)]) == value
}

func (s *scanner) advanceCharged(count int) bool {
	if count <= 0 || s.position+count > len(s.source) || !s.budget.ChargeWork(count) {
		return false
	}
	s.position += count
	return true
}

func isHorizontalSpace(value byte) bool {
	return value == ' ' || value == '\t'
}

func isIdentifierStart(value byte) bool {
	return value == '_' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isIdentifierContinue(value byte) bool {
	return isIdentifierStart(value) || isDecimalDigit(value)
}

func isDecimalDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func literalKeywordCooked(value string) string {
	switch value {
	case "true", "false", "null":
		return value
	default:
		return ""
	}
}

func compareToken(a, b Token) int {
	if n := cmp.Compare(a.Span.StartByte, b.Span.StartByte); n != 0 {
		return n
	}
	return cmp.Compare(a.Span.EndByte, b.Span.EndByte)
}
