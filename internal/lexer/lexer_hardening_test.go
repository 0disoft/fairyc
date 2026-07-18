package lexer

import (
	"bytes"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/source"
)

const fuzzInputLimit = 64 << 10

func TestScannerWorkBudgetCoversLongTokens(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input []byte
	}{
		{name: "identifier", input: bytes.Repeat([]byte("a"), 64)},
		{name: "number", input: bytes.Repeat([]byte("9"), 64)},
		{name: "string", input: append(append([]byte{'"'}, bytes.Repeat([]byte("a"), 64)...), '"')},
		{name: "line-comment", input: append([]byte("//"), bytes.Repeat([]byte("a"), 64)...)},
		{name: "whitespace", input: bytes.Repeat([]byte(" "), 64)},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			limits := source.DefaultLimits()
			limits.MaxWorkUnits = 8
			result := Lex(source.NewFile("main.fy", testCase.input), source.NewBudget(limits, len(testCase.input)))
			assertSingleExhaustion(t, result.Diagnostics)
			assertLosslessLexResult(t, testCase.input, result)
		})
	}
}

func TestTokenBudgetExhaustionRemainsLossless(t *testing.T) {
	t.Parallel()
	input := []byte("module main;")
	limits := source.DefaultLimits()
	limits.MaxTokens = 1
	result := Lex(source.NewFile("main.fy", input), source.NewBudget(limits, len(input)))
	assertSingleExhaustion(t, result.Diagnostics)
	assertLosslessLexResult(t, input, result)
}

func FuzzLexLosslessDeterministic(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		[]byte("module main;\n"),
		[]byte("\xef\xbb\xbfmodule main;\r\n/* outer /* inner */ done */"),
		{0xff, 0x00, 0xc3, 0x28},
		[]byte("c\"bad\\0value\" 1__0 @ ((("),
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > fuzzInputLimit {
			t.Skip()
		}
		limits := source.DefaultLimits()
		first := Lex(source.NewFile("fuzz.fy", input), source.NewBudget(limits, len(input)))
		second := Lex(source.NewFile("fuzz.fy", input), source.NewBudget(limits, len(input)))
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("Lex() is nondeterministic for %x", input)
		}
		assertLosslessLexResult(t, input, first)
		for index, value := range first.Diagnostics {
			if !value.Span.Valid(len(input)) {
				t.Fatalf("diagnostic %d has invalid span %+v for %d bytes", index, value.Span, len(input))
			}
			if index > 0 && diagnostic.Compare(first.Diagnostics[index-1], value) > 0 {
				t.Fatalf("diagnostics are not sorted: %+v", first.Diagnostics)
			}
		}
	})
}

func assertLosslessLexResult(t testing.TB, input []byte, result Result) {
	t.Helper()
	reconstructed, ok := result.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %x, %v; want %x, true; tokens=%+v", reconstructed, ok, input, result.Tokens)
	}
	for index, token := range result.Tokens {
		if !token.Span.Valid(len(input)) {
			t.Fatalf("token %d has invalid span %+v for %d bytes", index, token.Span, len(input))
		}
	}
}

func assertSingleExhaustion(t testing.TB, values []diagnostic.Diagnostic) {
	t.Helper()
	count := 0
	for _, value := range values {
		if value.Code == "E0010" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("E0010 count = %d, want 1; diagnostics=%+v", count, values)
	}
}
