package lexer

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestSymbolDispatchPreservesVocabularyAndSpans(t *testing.T) {
	input := []byte(strings.Join(symbols, " "))
	for _, lex := range []func(source.File, *source.Budget) Result{Lex, LexPreviousRevision} {
		result := lex(source.NewFile("main.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
		if len(result.Diagnostics) != 0 {
			t.Fatalf("symbol vocabulary rejected: %+v", result.Diagnostics)
		}
		var got []string
		position := 0
		for _, token := range result.Tokens {
			if token.Kind == Trivia || token.Kind == EOF {
				continue
			}
			if token.Kind != Symbol || token.Span.StartByte != position || token.Span.EndByte != position+len(token.Text(result.Source)) {
				t.Fatalf("invalid symbol token or byte span: %+v", token)
			}
			got = append(got, token.Text(result.Source))
			position = token.Span.EndByte + 1
		}
		if !slices.Equal(got, symbols) {
			t.Fatalf("got %q want %q", got, symbols)
		}
		assertLosslessLexResult(t, input, result)
	}
}

func TestSymbolDispatchPreservesLongestMatchAndCommentPriority(t *testing.T) {
	for _, test := range []struct {
		input string
		want  []string
	}{
		{"<<<=", []string{"<<", "<="}},
		{">>>=", []string{">>", ">="}},
		{"::::", []string{"::", "::"}},
		{"....", []string{"..", ".."}},
		{"&&&", []string{"&&", "&"}},
		{"|||=", []string{"||", "|="}},
		{"-->", []string{"-", "->"}},
		{"====>", []string{"==", "==", ">"}},
		{"/=// comment\n/* outer /* nested */ done */ /", []string{"/=", "/"}},
	} {
		t.Run(test.input, func(t *testing.T) {
			input := []byte(test.input)
			result := Lex(source.NewFile("main.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
			var got []string
			for _, token := range result.Tokens {
				if token.Kind == Symbol {
					got = append(got, token.Text(result.Source))
				}
			}
			if len(result.Diagnostics) != 0 || !slices.Equal(got, test.want) {
				t.Fatalf("got %q diagnostics=%+v want %q", got, result.Diagnostics, test.want)
			}
			assertLosslessLexResult(t, input, result)
		})
	}
}

func TestSymbolDispatchKeepsExactWorkBudget(t *testing.T) {
	input := []byte(strings.Join(symbols, " "))
	for _, work := range []int{0, 1, 2, 3, len(input) - 1, len(input)} {
		limits := source.DefaultLimits()
		limits.MaxWorkUnits = work
		result := Lex(source.NewFile("main.fy", input), source.NewBudget(limits, len(input)))
		if work < len(input) {
			assertSingleExhaustion(t, result.Diagnostics)
		} else if len(result.Diagnostics) != 0 {
			t.Fatalf("exact byte budget rejected: %+v", result.Diagnostics)
		}
		assertLosslessLexResult(t, input, result)
	}
}

func BenchmarkLexSymbolDispatch(b *testing.B) {
	for _, input := range []struct {
		name, unit string
	}{
		{"mixed-source", "fn i32 sum(i32 left, i32 right) { var i32 value = left + right; if (value >= 10) { value += 1; } return value; }\n"},
		{"all-symbols", strings.Join(symbols, " ") + " "},
	} {
		b.Run(input.name, func(b *testing.B) {
			file := source.NewFile("main.fy", []byte(strings.Repeat(input.unit, 512)))
			b.SetBytes(int64(len(file.Bytes)))
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result := Lex(file, source.NewBudget(source.DefaultLimits(), len(file.Bytes)))
				if len(result.Diagnostics) != 0 || len(result.Tokens) < 2 {
					b.Fatalf("unexpected lex result: diagnostics=%v tokens=%d", result.Diagnostics, len(result.Tokens))
				}
			}
		})
	}
}
