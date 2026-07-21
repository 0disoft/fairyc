package syntax_test

import (
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

const fuzzInputLimit = 64 << 10

const parseTestDeadline = 5 * time.Second

func TestParseDeterministicForHostileCorpus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name  string
		input []byte
	}{
		{name: "invalid-utf8-and-nul", input: append([]byte("module main;\n"), 0xff, 0x00, 0xc3, 0x28)},
		{name: "nested-comments", input: []byte("module main;\n/* " + strings.Repeat("/* ", 64) + strings.Repeat("*/ ", 64) + "*/")},
		{name: "delimiter-bomb", input: []byte("module main;\nfn i32 main() { " + strings.Repeat("(", 128) + "0;")},
		{name: "repair-heavy", input: []byte("module main fn i32 main( { let = @ return ((( ;")},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			first := parseWithinTestDeadline(t, "main.fy", testCase.input, source.DefaultLimits())
			second := parseWithinTestDeadline(t, "main.fy", testCase.input, source.DefaultLimits())
			if !reflect.DeepEqual(first, second) {
				t.Fatalf("Parse() is nondeterministic for %x", testCase.input)
			}
			assertTreeInvariants(t, testCase.input, first)
		})
	}
}

func TestParseBudgetExhaustionIsSingleAndLossless(t *testing.T) {
	t.Parallel()
	valid := []byte("module main;\nfn i32 main() { return 0; }\n")
	repairHeavy := []byte("module main;\nfn i32 main() { let value = 1 return value; }\n")
	cases := []struct {
		name   string
		input  []byte
		limits func(*source.Limits)
	}{
		{name: "bytes", input: valid, limits: func(value *source.Limits) { value.MaxBytes = len(valid) - 1 }},
		{name: "tokens", input: valid, limits: func(value *source.Limits) { value.MaxTokens = 1 }},
		{name: "nodes", input: valid, limits: func(value *source.Limits) { value.MaxNodes = 1 }},
		{name: "depth", input: valid, limits: func(value *source.Limits) { value.MaxDepth = 1 }},
		{name: "repairs", input: repairHeavy, limits: func(value *source.Limits) { value.MaxRepairs = 0 }},
		{name: "diagnostics", input: []byte{0xff}, limits: func(value *source.Limits) { value.MaxDiagnostics = 0 }},
		{name: "work", input: valid, limits: func(value *source.Limits) { value.MaxWorkUnits = 1 }},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()
			limits := source.DefaultLimits()
			testCase.limits(&limits)
			tree := parseWithinTestDeadline(t, "main.fy", testCase.input, limits)
			assertSingleExhaustion(t, tree.Diagnostics)
			assertTreeInvariants(t, testCase.input, tree)
		})
	}
}

func FuzzParseDeterministicAndBounded(f *testing.F) {
	for _, seed := range [][]byte{
		{},
		[]byte("module main;\nfn i32 main() { return 0; }\n"),
		[]byte("module main;\nfn i32 main() { let x = (((1 + 2; }"),
		[]byte("\xef\xbb\xbfmodule main;\r\n/* outer /* inner */ done */"),
		{0xff, 0x00, 0xc3, 0x28},
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input []byte) {
		if len(input) > fuzzInputLimit {
			t.Skip()
		}
		first := parseWithinTestDeadline(t, "fuzz.fy", input, source.DefaultLimits())
		second := parseWithinTestDeadline(t, "fuzz.fy", input, source.DefaultLimits())
		if !reflect.DeepEqual(first, second) {
			t.Fatalf("Parse() is nondeterministic for %x", input)
		}
		assertTreeInvariants(t, input, first)
		if first.ErrorFree() {
			if _, err := ast.Lower(first); err != nil {
				t.Fatalf("Lower() rejected an error-free tree: %v", err)
			}
		}
	})
}

func parseWithinTestDeadline(t testing.TB, relativePath string, input []byte, limits source.Limits) *syntax.Tree {
	t.Helper()
	result := make(chan *syntax.Tree, 1)
	go func() {
		result <- syntax.Parse(source.NewFile(relativePath, input), limits)
	}()
	select {
	case tree := <-result:
		return tree
	case <-time.After(parseTestDeadline):
		t.Fatalf("Parse() did not terminate within %s for %x", parseTestDeadline, input)
		return nil
	}
}

func assertTreeInvariants(t testing.TB, input []byte, tree *syntax.Tree) {
	t.Helper()
	reconstructed, ok := tree.Reconstruct()
	if !ok || !slices.Equal(reconstructed, input) {
		t.Fatalf("Reconstruct() = %x, %v; want %x, true", reconstructed, ok, input)
	}
	for index, node := range tree.Nodes {
		if !node.Span.Valid(len(input)) {
			t.Fatalf("node %d has invalid span %+v for %d bytes", index, node.Span, len(input))
		}
	}
	for index, value := range tree.Diagnostics {
		if !value.Span.Valid(len(input)) {
			t.Fatalf("diagnostic %d has invalid span %+v for %d bytes", index, value.Span, len(input))
		}
		if index > 0 && diagnostic.Compare(tree.Diagnostics[index-1], value) >= 0 {
			t.Fatalf("diagnostics are not strictly sorted and deduplicated: %+v", tree.Diagnostics)
		}
	}
	count := 0
	for _, value := range tree.Diagnostics {
		if value.Code == "E0010" {
			count++
		}
	}
	if count > 1 {
		t.Fatalf("E0010 count = %d, want at most 1; diagnostics=%+v", count, tree.Diagnostics)
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
