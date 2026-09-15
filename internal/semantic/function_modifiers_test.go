package semantic_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/semantic"
)

func TestAnalyzeFunctionResultTypeDoesNotGrantUnsafeScope(t *testing.T) {
	for _, test := range []struct {
		name, prefix, body string
		rejected           bool
	}{
		{"ordinary", "", "raw();return callback;", true},
		{"unsafe function", "unsafe ", "raw();return callback;", false},
		{"unsafe block", "", "unsafe{raw();}return callback;", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			text := "module main;unsafe extern C fn void raw();" + test.prefix + "fn unsafe extern C fn void() choose(unsafe extern C fn void() callback){" + test.body + "}"
			result := analyzeText(t, map[string]string{"main.fy": text})
			// C function values are syntax-supported but still semantically
			// unsupported. Preserve that boundary while testing unsafe checking.
			if len(result.Unsupported) != 2 {
				t.Fatalf("expected result and parameter type exclusions: %+v", result.Unsupported)
			}
			for _, item := range result.Unsupported {
				if item.Feature != "type unsafe extern C fn void()" {
					t.Fatalf("unexpected exclusion: %+v", item)
				}
			}
			if !test.rejected {
				if len(result.Diagnostics) != 0 || result.FailedPhase != "" {
					t.Fatalf("explicit unsafe boundary: %+v", result)
				}
				return
			}
			if result.FailedPhase != semantic.PhaseType || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E5101" {
				t.Fatalf("ordinary function must reject raw call: %+v", result)
			}
			start := strings.LastIndex(text, "raw()")
			assertSpan(t, result.Diagnostics[0].Primary, "main.fy", start, start+len("raw()"))
		})
	}
}
