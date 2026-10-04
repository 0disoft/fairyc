package semantic_test

import (
	"strings"
	"testing"
)

func TestI64ForSourceRejections(t *testing.T) {
	for name, loop := range map[string]string{
		"wrong-item":       "for (let u8 value in view) {}",
		"unsupported-item": "for (let i32 value in view) {}",
		"reference-width":  "for (let ref u8 value in view) {}",
		"mutable-binding":  "var ref i64[] alias = view; for (let i64 value in alias) {}",
		"slice-expression": "for (let i64 value in view[..]) {}",
		"call-expression":  "for (let i64 value in relay(view)) {}",
		"bool-array":       "let bool[2] flags=[true,false];for (let i64 value in flags) {}",
	} {
		t.Run(name, func(t *testing.T) {
			text := "module main; fn ref i64[] relay(ref i64[] view) { return view; } fn void run() { let i64[2] data = [1,2]; let ref i64[] view = ref data; " + loop + " }"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if len(result.Unsupported) == 0 {
				t.Fatalf("invalid iteration accepted: %+v", result.Diagnostics)
			}
		})
	}
}

func TestI64ForOwnerBorrow(t *testing.T) {
	for _, item := range []struct{ name, body, code string }{
		{"mutation", "for (let i64 value in view) { data[0] = value; }", "E2101"},
		{"mutation-before-break", "for (let i64 value in view) { data[0] = value; break; }", "E2101"},
		{"mutation-before-return", "for (let i64 value in view) { data[0] = value; return; }", "E2101"},
		{"mutation-after-loop", "for (let i64 value in view) { assert(value > 0); } data[0] = 9;", ""},
		{"mutation-after-break", "for (let i64 value in view) { assert(value > 0); break; } data[0] = 9;", ""},
		{"immutable-item", "for (let i64 value in view) { value = 9; }", "E1008"},
		{"item-scope", "for (let i64 value in view) {} let i64 outside = value;", "E0101"},
	} {
		t.Run(item.name, func(t *testing.T) {
			text := "module main; fn void run() { var i64[2] data = [1,2]; let ref i64[] view = ref data; " + item.body + " }"
			result := analyzeText(t, map[string]string{"main.fy": text})
			if item.code == "" {
				if len(result.Diagnostics) != 0 || len(result.Unsupported) != 0 {
					t.Fatalf("valid loop rejected: %+v %+v", result.Diagnostics, result.Unsupported)
				}
				return
			}
			found := false
			for _, diagnostic := range result.Diagnostics {
				found = found || diagnostic.Code == item.code
			}
			if !found {
				t.Fatalf("expected %s: %+v %+v", item.code, result.Diagnostics, result.Unsupported)
			}
		})
	}
}

func TestI64ForRejectsByteView(t *testing.T) {
	result := analyzeText(t, map[string]string{"main.fy": `module main; fn void run(ref u8[] bytes) { for (let i64 value in bytes) {} }`})
	if len(result.Unsupported) != 1 || !strings.Contains(result.Unsupported[0].Feature, "ref i64[]") {
		t.Fatalf("byte view accepted: %+v", result.Unsupported)
	}
}
