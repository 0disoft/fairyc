package semantic_test

import (
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
	if d.Code != "E1006" || d.ExpectedType != "bool" || d.ActualType != "i32" || len(d.Related) != 1 || d.Related[0].Role != "expected-type" {
		t.Fatalf("diagnostic = %+v", d)
	}
	argument := strings.Index(text, "42")
	parameter := strings.Index(text, "bool")
	assertSpan(t, d.Primary, "main.fy", argument, argument+2)
	assertSpan(t, d.Related[0].Span, "main.fy", parameter, parameter+4)
}

func TestCallTypeDiagnosticsQualifiedNames(t *testing.T) {
	result := analyzeText(t, map[string]string{
		"left.fy":  "module left; pub struct Index { usize raw; } pub fn void take(Index value) {}",
		"right.fy": "module right; pub struct Index { usize raw; }",
		"main.fy":  "module main; import left as l; import right as r; fn void caller() { let r::Index value = r::Index { .raw = 0 }; l::take(value); }",
	})
	if len(result.Diagnostics) != 1 || result.Diagnostics[0].ExpectedType != "left::Index" || result.Diagnostics[0].ActualType != "right::Index" {
		t.Fatalf("result = %+v", result)
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
