package lexer

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestLosslessTokenStreamPreservesBOMCRLFAndNestedComments(t *testing.T) {
	t.Parallel()
	input := append([]byte{0xef, 0xbb, 0xbf}, []byte("module main;\r\n/* outer /* nested */ done */\r\n")...)
	result := Lex(source.NewFile("main.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	reconstructed, ok := result.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, want original %q", reconstructed, input)
	}
}

func TestInvalidUTF8PreservesRawByteAndReportsE0001(t *testing.T) {
	t.Parallel()
	input := []byte{'m', 'o', 'd', 'u', 'l', 'e', ' ', 0xff, ';'}
	result := Lex(source.NewFile("main.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E0001" || result.Diagnostics[0].Span.StartByte != 7 || result.Diagnostics[0].Span.EndByte != 8 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	if _, ok := result.Reconstruct(); !ok {
		t.Fatal("invalid UTF-8 byte was not retained in the token stream")
	}
}

func TestNumericAndStringCooking(t *testing.T) {
	t.Parallel()
	input := []byte("1_024.5e-2 0xDEAD_BEEF \"line\\n\\u{03BB}\" b\"\\x00A\" c\"fairy\"")
	result := Lex(source.NewFile("literals.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	want := []string{"10.245", "3735928559", "6c696e650acebb", "0041", "666169727900"}
	var got []string
	for _, token := range result.Tokens {
		if token.Cooked != "" {
			got = append(got, token.Cooked)
		}
	}
	if !slices.Equal(got, want) {
		t.Fatalf("cooked literals = %v, want %v", got, want)
	}
}
