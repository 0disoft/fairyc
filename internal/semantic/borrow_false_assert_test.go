package semantic_test

import "testing"

func TestAnalyzeBorrowFlowFalseAssertTerminates(t *testing.T) {
	for _, tc := range []struct {
		name string
		body string
		code string
	}{
		{"option payload", `fn ref i32 f(Option<ref i32> x){return switch(x){case .some(v):{v}case .none:{assert(false);}};}`, ""},
		{"local payload", `fn ref i32 f(Option<i32> x){return switch(x){case .some(v):{ref v}case .none:{assert(false);}};}`, "E2102"},
		{"nested payload", `fn ref i32 f(Option<Option<ref i32>> x){return switch(x){case .some(v):{switch(v){case .some(a):{a}case .none:{assert(false);}}}case .none:{assert(false);}};}`, ""},
		{"unreachable escape", `fn ref i32 f(){assert(false);let i32 n=1;return ref n;}`, ""},
		{"true assertion continues", `fn ref i32 f(Option<ref i32> x,ref i32 fallback){return switch(x){case .some(v):{v}case .none:{assert(true);fallback}};}`, "E2103"},
		{"dynamic assertion continues", `fn ref i32 f(Option<ref i32> x,ref i32 fallback,bool flag){return switch(x){case .some(v):{v}case .none:{assert(flag);fallback}};}`, "E2103"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := analyzeText(t, map[string]string{"main.fy": "module main;" + tc.body})
			if len(r.Unsupported) != 0 {
				t.Fatalf("unsupported = %+v", r.Unsupported)
			}
			if tc.code == "" {
				if len(r.Diagnostics) != 0 {
					t.Fatalf("diagnostics = %+v", r.Diagnostics)
				}
			} else if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != tc.code {
				t.Fatalf("diagnostics = %+v, want %s", r.Diagnostics, tc.code)
			}
		})
	}
}
