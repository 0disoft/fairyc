package lexer

import (
	"bytes"

	"github.com/0disoft/fairyc/internal/diagnostic"
)

// AttachedDocumentationSpan returns the exact continuous bytes occupied by consecutive physical
// /// lines immediately above a declaration token. The final line separator and declaration
// indentation are deliberately excluded.
func AttachedDocumentationSpan(tokens []Token, source []byte, declarationToken int) (diagnostic.Span, bool) {
	if declarationToken <= 1 || declarationToken >= len(tokens) || tokens[declarationToken].Kind == Trivia ||
		!tokens[declarationToken].Span.Valid(len(source)) {
		return diagnostic.Span{}, false
	}
	separator := declarationToken - 1
	if !isSingleDocumentationSeparator(tokens[separator], source) ||
		tokens[separator].Span.EndByte != tokens[declarationToken].Span.StartByte {
		return diagnostic.Span{}, false
	}
	documentation := separator - 1
	if !isAttachableDocumentationLine(tokens[documentation], source) ||
		tokens[documentation].Span.EndByte != tokens[separator].Span.StartByte {
		return diagnostic.Span{}, false
	}
	start := tokens[documentation].Span.StartByte
	end := tokens[documentation].Span.EndByte
	for documentation >= 2 {
		separator = documentation - 1
		previous := separator - 1
		if !isSingleDocumentationSeparator(tokens[separator], source) ||
			!isAttachableDocumentationLine(tokens[previous], source) ||
			tokens[previous].Span.EndByte != tokens[separator].Span.StartByte ||
			tokens[separator].Span.EndByte != tokens[documentation].Span.StartByte {
			break
		}
		start = tokens[previous].Span.StartByte
		documentation = previous
	}
	return diagnostic.Span{StartByte: start, EndByte: end}, true
}

func isSingleDocumentationSeparator(token Token, source []byte) bool {
	if token.Kind != Trivia || token.Trivia != TriviaWhitespace || !token.Span.Valid(len(source)) {
		return false
	}
	value := source[token.Span.StartByte:token.Span.EndByte]
	if len(value) == 0 {
		return false
	}
	position := 0
	if value[0] == '\n' {
		position = 1
	} else if len(value) >= 2 && value[0] == '\r' && value[1] == '\n' {
		position = 2
	} else {
		return false
	}
	for ; position < len(value); position++ {
		if value[position] != ' ' && value[position] != '\t' {
			return false
		}
	}
	return true
}

func isAttachableDocumentationLine(token Token, source []byte) bool {
	if token.Kind != Trivia || token.Trivia != TriviaDocumentationLine || !token.Span.Valid(len(source)) ||
		token.Span.EndByte-token.Span.StartByte < 3 ||
		!bytes.Equal(source[token.Span.StartByte:token.Span.StartByte+3], []byte("///")) {
		return false
	}
	lineStart := bytes.LastIndexByte(source[:token.Span.StartByte], '\n') + 1
	if lineStart == 0 && len(source) >= 3 && bytes.Equal(source[:3], []byte{0xef, 0xbb, 0xbf}) {
		lineStart = 3
	}
	for position := lineStart; position < token.Span.StartByte; position++ {
		if source[position] != ' ' && source[position] != '\t' {
			return false
		}
	}
	return true
}
