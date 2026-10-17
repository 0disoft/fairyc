package semantic_test

import (
	"encoding/hex"
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func TestAnalyzeExpandsSourceLocationWithExactOriginalByteGeometry(t *testing.T) {
	t.Parallel()
	text := "module nested::main;\r\nfn ref str where() { let ref str prefix = \"λ\"; return source_location(); }\r\n"
	start := strings.Index(text, "source_location")
	lineStart := strings.LastIndex(text[:start], "\n") + 1
	want := fmt.Sprintf("nested/main.fy:2:%d@%d", start-lineStart+1, start)
	result := analyzeText(t, map[string]string{"nested/main.fy": text})
	assertCleanTypeResult(t, result)
	if len(result.HIR.Modules) != 1 || len(result.HIR.Modules[0].Functions) != 1 {
		t.Fatalf("HIR = %+v", result.HIR)
	}
	returned := soleReturnExpression(t, result.HIR.Modules[0].Functions[0])
	if returned.Kind != hir.ExpressionLiteral || returned.Type != hir.SharedBorrow(hir.TypeStr) || returned.Value != fmt.Sprintf("%q", want) ||
		returned.Cooked != hex.EncodeToString([]byte(want)) || returned.Symbol != hir.NoSymbol || returned.Span.StartByte != start {
		t.Fatalf("source location HIR = %+v, want %q", returned, want)
	}
	foundStatic := false
	for _, borrow := range result.Borrows {
		if borrow.Borrow.StartByte == start && borrow.Origin == "Static" {
			foundStatic = true
		}
	}
	if !foundStatic {
		t.Fatalf("static source-location borrow missing from %+v", result.Borrows)
	}
}

func TestAnalyzeRejectsReservedSourceLocationNamesAndMalformedUses(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		code string
	}{
		{name: "function", body: "fn ref str source_location() { return \"bad\"; }", code: "E4401"},
		{name: "parameter", body: "fn i32 bad(i32 source_location) { return source_location; }", code: "E4401"},
		{name: "local", body: "fn i32 bad() { let i32 source_location = 1; return 0; }", code: "E4401"},
		{name: "non-call", body: "fn ref str bad() { return source_location; }", code: "E4401"},
		{name: "argument", body: "fn ref str bad() { return source_location(1); }", code: "E1010"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := analyzeText(t, map[string]string{"main.fy": "module main;\n" + test.body + "\n"})
			if !hasDiagnosticCode(result, test.code) {
				t.Fatalf("diagnostics = %+v, unsupported = %+v, want %s", result.Diagnostics, result.Unsupported, test.code)
			}
		})
	}
}

func TestAnalyzeRejectsSourceLocationWithoutRetainedGeometry(t *testing.T) {
	t.Parallel()
	text := "module main;\nfn ref str where() { return source_location(); }\n"
	parsed, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(text)), source.DefaultLimits())
	if err != nil || parsed.AST == nil {
		t.Fatalf("AnalyzeSyntax() = %+v, %v", parsed, err)
	}
	parsed.AST.SourceLineStarts = nil
	result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: parsed.AST}})
	if !hasDiagnosticCode(result, "E4402") {
		t.Fatalf("diagnostics = %+v, want E4402", result.Diagnostics)
	}
}
