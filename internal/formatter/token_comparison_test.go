package formatter

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/lexer"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func comparisonTree(text string) *syntax.Tree {
	file := source.NewFile("main.fy", []byte(text))
	scanned := lexer.Lex(file, source.NewBudget(source.DefaultLimits(), len(file.Bytes)))
	return &syntax.Tree{Source: scanned.Source, Tokens: scanned.Tokens}
}

func TestSignificantTokenComparison(t *testing.T) {
	for _, test := range []struct {
		name, left, right string
		want              bool
	}{
		{"empty", "", "", true},
		{"trivia only", " \r\n// comment", "/* other */\n", true},
		{"trivia and offsets", "let value = 12;\r\n", "/* lead */let  value=12; // tail", true},
		{"spelling", "let value = 12;", "let other = 12;", false},
		{"literal spelling", `"a"`, `"\x61"`, false},
		{"extra token", "let value = 12;", "let value = 12;;", false},
		{"order", "first + second", "second + first", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			left, right := comparisonTree(test.left), comparisonTree(test.right)
			if got := equivalentSignificantTokens(left, right); got != test.want {
				t.Fatalf("forward comparison = %v, want %v", got, test.want)
			}
			if got := equivalentSignificantTokens(right, left); got != test.want {
				t.Fatalf("reverse comparison = %v, want %v", got, test.want)
			}
		})
	}
	left := comparisonTree(`"long literal"`)
	for _, field := range []string{"kind", "cooked"} {
		t.Run(field, func(t *testing.T) {
			right := *left
			right.Tokens = slices.Clone(left.Tokens)
			if field == "kind" {
				right.Tokens[0].Kind = lexer.ByteString
			} else {
				right.Tokens[0].Cooked = "changed"
			}
			if equivalentSignificantTokens(left, &right) {
				t.Fatalf("comparison ignored %s", field)
			}
		})
	}
}

func TestSignificantTokenComparisonDetachedStreams(t *testing.T) {
	left := comparisonTree("first + second")
	right := *left
	right.Tokens = slices.Clone(left.Tokens[:len(left.Tokens)-1])
	// EOF tokens are ignored wherever they appear, including before real tokens.
	right.Tokens = append([]lexer.Token{{Kind: lexer.EOF}}, right.Tokens...)
	if !equivalentSignificantTokens(left, &right) {
		t.Fatal("EOF placement changed significant token equality")
	}
	for _, kind := range []lexer.Kind{lexer.Invalid, lexer.Identifier, lexer.String} {
		invalid := &syntax.Tree{Tokens: []lexer.Token{{Kind: kind, Span: diagnostic.Span{StartByte: -1}}}}
		empty := &syntax.Tree{Tokens: []lexer.Token{{Kind: kind}}}
		if !equivalentSignificantTokens(invalid, empty) {
			t.Fatalf("invalid span no longer follows Token.Text empty-string behavior for kind %v", kind)
		}
	}
}

func BenchmarkSignificantTokenComparison(b *testing.B) {
	text := strings.Repeat("fn i32 sample(i32 argument) { return argument + 12345; }\r\n", 1024)
	left := comparisonTree(text)
	right := comparisonTree(strings.ReplaceAll(text, "\r\n", "\n"))
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if !equivalentSignificantTokens(left, right) {
			b.Fatal("equivalent source rejected")
		}
	}
}
