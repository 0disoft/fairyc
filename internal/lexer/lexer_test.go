package lexer

import (
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/source"
)

func TestRetiredLegacyKeywordsAreIdentifiersInBothCurrentRevisions(t *testing.T) {
	t.Parallel()
	input := []byte("impl match drop self switch")
	file := source.NewFile("main.fy", input)
	current := Lex(file, source.NewBudget(source.DefaultLimits(), len(input)))
	previous := LexPreviousRevision(file, source.NewBudget(source.DefaultLimits(), len(input)))
	currentKinds := significantKinds(current.Tokens)
	previousKinds := significantKinds(previous.Tokens)
	if !slices.Equal(currentKinds, []Kind{Identifier, Identifier, Identifier, Identifier, Keyword, EOF}) {
		t.Fatalf("Lex() kinds = %v", currentKinds)
	}
	if !slices.Equal(previousKinds, []Kind{Identifier, Identifier, Identifier, Identifier, Keyword, EOF}) {
		t.Fatalf("LexPreviousRevision() kinds = %v", previousKinds)
	}
}

func TestForIsReservedByCurrentAndImmediatelyPreviousRevision(t *testing.T) {
	t.Parallel()
	input := []byte("for format")
	file := source.NewFile("main.fy", input)
	current := Lex(file, source.NewBudget(source.DefaultLimits(), len(input)))
	previous := LexPreviousRevision(file, source.NewBudget(source.DefaultLimits(), len(input)))
	if got := significantKinds(current.Tokens); !slices.Equal(got, []Kind{Keyword, Identifier, EOF}) {
		t.Fatalf("Lex() kinds = %v", got)
	}
	if got := significantKinds(previous.Tokens); !slices.Equal(got, []Kind{Keyword, Identifier, EOF}) {
		t.Fatalf("LexPreviousRevision() kinds = %v", got)
	}
}

func TestDeferIsReservedOnlyByCurrentRevision(t *testing.T) {
	t.Parallel()
	input := []byte("defer record")
	file := source.NewFile("main.fy", input)
	current := Lex(file, source.NewBudget(source.DefaultLimits(), len(input)))
	previous := LexPreviousRevision(file, source.NewBudget(source.DefaultLimits(), len(input)))
	if got := significantKinds(current.Tokens); !slices.Equal(got, []Kind{Keyword, Identifier, EOF}) {
		t.Fatalf("Lex() kinds = %v", got)
	}
	if got := significantKinds(previous.Tokens); !slices.Equal(got, []Kind{Identifier, Identifier, EOF}) {
		t.Fatalf("LexPreviousRevision() kinds = %v", got)
	}
}

func significantKinds(tokens []Token) []Kind {
	result := make([]Kind, 0, len(tokens))
	for _, token := range tokens {
		if token.Kind != Trivia {
			result = append(result, token.Kind)
		}
	}
	return result
}

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

func TestTriviaClassificationPreservesDocumentationWithoutChangingTokenKind(t *testing.T) {
	t.Parallel()
	input := append([]byte{0xef, 0xbb, 0xbf}, []byte(" \n// ordinary\n  /// declaration docs\n/* block */")...)
	result := Lex(source.NewFile("main.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	var kinds []TriviaKind
	var documentation string
	for _, token := range result.Tokens {
		if token.Kind != Trivia {
			continue
		}
		kinds = append(kinds, token.Trivia)
		if token.Trivia == TriviaDocumentationLine {
			documentation = token.Text(result.Source)
		}
	}
	want := []TriviaKind{
		TriviaPreamble, TriviaWhitespace, TriviaLineComment, TriviaWhitespace,
		TriviaDocumentationLine, TriviaWhitespace, TriviaBlockComment,
	}
	if !slices.Equal(kinds, want) || documentation != "/// declaration docs" {
		t.Fatalf("trivia = %v, documentation = %q, want %v", kinds, documentation, want)
	}
	reconstructed, ok := result.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
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

func TestRangePunctuationIsOneLosslessToken(t *testing.T) {
	t.Parallel()
	input := []byte(`b"fairy"[1..4]`)
	result := Lex(source.NewFile("slice.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	var ranges int
	for _, token := range result.Tokens {
		if token.Text(result.Source) == ".." {
			ranges++
		}
	}
	if ranges != 1 {
		t.Fatalf("range token count = %d, tokens = %+v", ranges, result.Tokens)
	}
	reconstructed, ok := result.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want %q, true", reconstructed, ok, input)
	}
}

func TestCompoundAssignmentUsesLongestLosslessTokens(t *testing.T) {
	t.Parallel()
	input := []byte("a += 1; b -= 1; c *= 2; d /= 2; e %= 3; f <<= 1; g >>= 1; h &= 7; i ^= 1; j |= 2;")
	result := Lex(source.NewFile("assignment.fy", input), source.NewBudget(source.DefaultLimits(), len(input)))
	if len(result.Diagnostics) != 0 {
		t.Fatalf("Lex() diagnostics = %+v", result.Diagnostics)
	}
	var operators []string
	for _, token := range result.Tokens {
		if token.Kind == Symbol && slices.Contains([]string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|="}, token.Text(result.Source)) {
			operators = append(operators, token.Text(result.Source))
		}
	}
	want := []string{"+=", "-=", "*=", "/=", "%=", "<<=", ">>=", "&=", "^=", "|="}
	if !slices.Equal(operators, want) {
		t.Fatalf("operators = %v, want %v", operators, want)
	}
	if reconstructed, ok := result.Reconstruct(); !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %q, %v; want exact input", reconstructed, ok)
	}
}

func TestStringHexEscapesMustFormValidUTF8(t *testing.T) {
	t.Parallel()
	validInput := []byte(`"\xCE\xBB"`)
	valid := Lex(source.NewFile("valid.fy", validInput), source.NewBudget(source.DefaultLimits(), len(validInput)))
	if len(valid.Diagnostics) != 0 || len(valid.Tokens) == 0 || valid.Tokens[0].Kind != String || valid.Tokens[0].Cooked != "cebb" {
		t.Fatalf("valid hex-escaped UTF-8 = %+v", valid)
	}

	invalidInput := []byte(`"\xFF"`)
	invalid := Lex(source.NewFile("invalid.fy", invalidInput), source.NewBudget(source.DefaultLimits(), len(invalidInput)))
	if len(invalid.Diagnostics) != 1 || invalid.Diagnostics[0].Code != "E0005" || invalid.Diagnostics[0].Span.StartByte != 0 || invalid.Diagnostics[0].Span.EndByte != len(invalidInput) || invalid.Diagnostics[0].Repair != "none" {
		t.Fatalf("invalid hex-escaped UTF-8 diagnostics = %+v", invalid.Diagnostics)
	}
}
