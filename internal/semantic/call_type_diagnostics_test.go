package semantic_test

import (
	"github.com/0disoft/fairyc/internal/hir"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCallTypeDiagnostics(t *testing.T) {
	text := "module main;\nfn void take(bool flag) {}\nfn void caller() { take(42); }\n"
	result := analyzeText(t, map[string]string{"main.fy": text})
	if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 {
		t.Fatalf("result = %+v", result)
	}
	d := result.Diagnostics[0]
	if d.Code != "E1006" || d.ExpectedType != hir.ParseType("bool") || d.ActualType != hir.ParseType("i32") || len(d.Related) != 1 || d.Related[0].Role != "expected-type" {
		t.Fatalf("diagnostic = %+v", d)
	}
	argument := strings.Index(text, "42")
	parameter := strings.Index(text, "bool")
	assertSpan(t, d.Primary, "main.fy", argument, argument+2)
	assertSpan(t, d.Related[0].Span, "main.fy", parameter, parameter+4)
}

func TestCallTypeDiagnosticsQualifiedNames(t *testing.T) {
	sources := map[string]string{
		"left.fy":  "module left; pub struct Index { usize raw; } pub fn void take(Index value) {}",
		"right.fy": "module right; pub struct Index { usize raw; }",
		"main.fy":  "module main; import left as l; import right as r; fn void caller() { let r::Index value = r::Index { .raw = 0 }; l::take(value); }",
	}
	result := analyzeText(t, sources)
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].ExpectedType != hir.ParseType("left::Index") || result.Diagnostics[0].ActualType != hir.ParseType("right::Index") {
		t.Fatalf("result = %+v", result)
	}
	d := result.Diagnostics[0]
	if d.Code != "E1006" || len(d.Related) != 3 || d.Related[0].Role != "actual-type-declaration" || d.Related[1].Role != "expected-type" || d.Related[2].Role != "expected-type-declaration" {
		t.Fatalf("diagnostic = %+v", d)
	}
	argument := strings.LastIndex(sources["main.fy"], "value")
	assertSpan(t, d.Primary, "main.fy", argument, argument+5)
	parameter := strings.Index(sources["left.fy"], "Index value")
	assertSpan(t, d.Related[1].Span, "left.fy", parameter, parameter+5)
	for index, path := range []string{"left.fy", "right.fy"} {
		start := strings.Index(sources[path], "Index")
		assertSpan(t, d.Related[2-index*2].Span, path, start, start+5)
	}
}

func TestCallTypeDiagnosticsOneStructDeclaration(t *testing.T) {
	for _, test := range []struct{ parameter, argument, role string }{
		{"Index", "1", "expected-type-declaration"},
		{"i32", "Index { .raw = 1 }", "actual-type-declaration"},
	} {
		text := "module main; struct Index { usize raw; } fn void take(" + test.parameter + " value) {} fn void caller() { take(" + test.argument + "); }"
		result := analyzeText(t, map[string]string{"main.fy": text})
		if len(result.Diagnostics) != 1 {
			t.Fatalf("result = %+v", result)
		}
		d := result.Diagnostics[0]
		if d.Code != "E1006" || len(d.Related) != 2 || d.Related[0].Role != min("expected-type", test.role) || d.Related[1].Role != max("expected-type", test.role) {
			t.Fatalf("diagnostic = %+v", d)
		}
		start := strings.Index(text, "Index")
		for _, related := range d.Related {
			if related.Role == test.role {
				assertSpan(t, related.Span, "main.fy", start, start+5)
			}
		}
	}
}

func TestCallTypeDiagnosticsCleanAndUnresolved(t *testing.T) {
	for _, argument := range []string{"true", "missing"} {
		result := analyzeText(t, map[string]string{"main.fy": "module main; fn void take(bool value) {} fn void caller() { take(" + argument + "); }"})
		if argument == "true" {
			assertCleanTypeResult(t, result)
		} else {
			if len(result.Diagnostics) == 0 {
				t.Fatal("missing unresolved diagnostic")
			}
			for _, d := range result.Diagnostics {
				if d.Code == "E1006" {
					t.Fatalf("cascaded diagnostic: %+v", d)
				}
			}
		}
	}
}
