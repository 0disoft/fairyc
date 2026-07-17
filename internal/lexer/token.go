// Package lexer converts immutable Fairy C source bytes into a trivia-preserving token stream.
package lexer

import (
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
)

// Kind classifies a raw source token without discarding its original byte span.
type Kind uint8

const (
	Invalid Kind = iota
	EOF
	Trivia
	Identifier
	Keyword
	Integer
	Float
	String
	ByteString
	CString
	Symbol
)

// Token refers into Result.Source. Cooked is populated only for literal tokens.
type Token struct {
	Kind   Kind
	Span   diagnostic.Span
	Cooked string
}

// Text returns the token's exact original source spelling.
func (t Token) Text(source []byte) string {
	if !t.Span.Valid(len(source)) {
		return ""
	}
	return string(source[t.Span.StartByte:t.Span.EndByte])
}

// Result owns one immutable byte snapshot and every token that accounts for it.
type Result struct {
	Source      []byte
	Tokens      []Token
	Diagnostics []diagnostic.Diagnostic
}

// Reconstruct concatenates every real token span and returns the original bytes on success.
func (r Result) Reconstruct() ([]byte, bool) {
	reconstructed := make([]byte, 0, len(r.Source))
	position := 0
	for _, token := range r.Tokens {
		if token.Kind == EOF {
			continue
		}
		if token.Span.StartByte != position || !token.Span.Valid(len(r.Source)) {
			return nil, false
		}
		reconstructed = append(reconstructed, r.Source[token.Span.StartByte:token.Span.EndByte]...)
		position = token.Span.EndByte
	}
	return reconstructed, position == len(r.Source) && slices.Equal(reconstructed, r.Source)
}

// LiteralKind is the manifest-facing literal class.
func LiteralKind(kind Kind) string {
	switch kind {
	case Integer:
		return "integer"
	case Float:
		return "float"
	case String:
		return "string"
	case ByteString:
		return "byte-string"
	case CString:
		return "c-string"
	case Keyword:
		return "keyword"
	default:
		return ""
	}
}
