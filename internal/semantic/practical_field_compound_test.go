package semantic_test

import "testing"

func TestPracticalFieldCompoundRejectsUnsafePlaces(t *testing.T) {
	for name, body := range map[string]string{
		"shared":      "fn void edit(ref State s) { s.count += 1; }",
		"boolean":     "fn void edit(ref mut State s) { s.ready += true; }",
		"nested":      "fn void edit(ref mut Outer s) { s.inner.count += 1; }",
		"active loan": "fn i32 main() { var State s=State{.count=1,.ready=false}; let ref State view=ref s; s.count += 1; assert(view.count==1); return 0; }",
	} {
		t.Run(name, func(t *testing.T) {
			result := analyzeText(t, map[string]string{"main.fy": "module main; struct State { i32 count; bool ready; } struct Outer { State inner; } " + body})
			if len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
				t.Fatal("unsafe/unsupported compound field target admitted")
			}
		})
	}
}
