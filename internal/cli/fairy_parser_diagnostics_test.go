package cli

import (
	"fmt"
	"strconv"
	"strings"
	"testing"
)

func TestWindowsFairyParserDiagnostics(t *testing.T) {
	cases := []struct {
		input, actual    string
		offset, expected int
	}{
		{"fn void kept() {}", "fn", 0, 12},
		{"module ;", ";", 7, 1},
		{"module main", "", 11, 8},
		{"module main; import x y;", "y", 22, 10},
		{"module main; fn Thing bad() {}", "Thing", 16, 2},
		{"module main; fn void f { }", "{", 23, 4},
		{"module main; fn void f() ", "", 25, 6},
		{"module main; fn void f() {", "", 25, 7},
		{"module main; fn void f() { return +; }", ";", 34, 3},
		{"module main; fn void f() { x = 1 }", "}", 32, 8},
		{"module main; fn void f() { x = a[1; }", ";", 33, 11},
		{"module main; fn void f() { x = a.; }", ";", 32, 1},
		{"module main; fn void f() { while (true { } }", "{", 38, 5},
	}
	var p strings.Builder
	p.WriteString("fn i32 main() {\n")
	for i, c := range cases {
		// Derive the exact fault offset independently from the preserved spelling.
		offset := len(c.input)
		if c.actual != "" {
			offset = strings.LastIndex(c.input, c.actual)
		}
		if c.expected == 12 {
			offset = 0
		}
		if c.expected == 5 {
			offset = strings.Index(c.input, "true ") + 5
		}
		fmt.Fprintf(&p, "{ let i32 result = switch(make(b%s)) { case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 1); let syntax::Problem d = syntax::problem_at(ref parsed,0); assert(d.expected == %d && d.start == %d && d.end == %d); let ref u8[] actual = syntax::problem_actual(ref parsed,0); assert(actual.len == %d);", strconv.Quote(c.input), c.expected, offset, offset+len(c.actual), len(c.actual))
		for j, b := range []byte(c.actual) {
			fmt.Fprintf(&p, "assert(actual[%d] == %d);", j, b)
		}
		fmt.Fprintf(&p, "assert(syntax::expected_text(d.expected).len > 0); 0 } case .err(_): { 1 } }; assert(result == 0); } // case %d\n", i)
	}
	// A later unsupported item must not inherit the earlier semicolon expectation.
	p.WriteString(`{ let i32 result = switch(make(b"module main; fn void bad() { x = 1 } unsupported;")) {
 case .ok(parsed): { assert(syntax::problem_count(ref parsed) == 2); assert(syntax::problem_at(ref parsed,0).expected == 8); assert(syntax::problem_at(ref parsed,1).expected == 0); 0 }
 case .err(_): { 1 } }; assert(result == 0); }
return 0; }`)
	runParserProbe(t, p.String())
}
