package cli

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
)

func testFairyScalarAdapter(t *testing.T, native func(string) string) []string {
	cases := []struct {
		input string
		clean bool
	}{
		{"module main;fn i32 answer(){return 42;}", true},
		{"module main;\r\nfn i32 pick(i32 z,i32 a,){return a;}\r\n", true},
		{"module main;fn i32 zero(){return 0;}", true},
		{"module main;fn i32 max(){return 2147483647;}", true},
		{"module main;fn i32 bad(){return missing;}", false},
		{"module main;fn i32 duplicate(i32 x,i32 x){return x;}", false},
		{"module main;fn i32 overflow(){return 2147483648;}", false},
		{"module main;fn i32 f(i32 a,i32 b){return (a+2)*-b;}", true},
		{"module main;fn i32 f(i32 a,i32 b,i32 c){return a-b-c;}", true},
		{"module main;fn i32 f(i32 a,i32 b){return +a*b-2;}", true},
		{"module main;fn i32 f(){return -1;}", true},
		{"module main;fn i32 f(){return -2147483648;}", true},
		{"module main;fn i32 f(){return (-2147483648);}", true},
		{"module main;fn i32 f(){return -2147483649;}", false},
		{"module main;fn i32 f(i32 a){return a+missing;}", false},
		{"module main;fn i32 f(){return +2147483648;}", false},
		{"module main;\r\nfn i32 f(i32 a,i32 b){return +(a /* 연산 */ - -b);}", true},
		{"module main;fn i32 f(){return " + strings.Repeat("+ ", 60) + "1;}", true},
		{"module main;fn i32 f(){let i32 x=1;return x;}", true},
		{"module main;fn i32 f(i32 a){let i32 x=a+2;let i32 y=(x-3)*-a;return y+x;}", true},
		{"module main;\r\nfn i32 f(i32 a){\r\nlet i32 x = +(a /* 초기값 */ - 2);\r\nlet i32 y=x*3;\r\nreturn y;\r\n}", true},
		{"module main;fn i32 f(){let i32 x=-2147483648;return x;}", true},
		{"module main;fn i32 f(){let i32 x=2147483648;return x;}", false},
		{"module main;fn i32 f(){let i32 x=missing;return x;}", false},
		{"module main;fn i32 f(){let i32 x=y;let i32 y=1;return x;}", false},
		{"module main;fn i32 f(){let i32 x=x;return x;}", false},
		{"module main;fn i32 f(){let i32 x=1;let i32 x=2;return x;}", false},
		{"module main;fn i32 f(i32 x){let i32 x=x+2;return x;}", true},
		{"module main;fn i32 f(){let i32 x=" + strings.Repeat("+ ", 60) + "1;return x;}", true},
	}
	var firstWire string
	var params []string
	for i := 0; i < 16; i++ {
		params = append(params, fmt.Sprintf("i32 p%d", i))
	}
	wide := "module main;fn i32 wide(" + strings.Join(params, ",") + "){return p15;}"
	var balanced func(int) string
	balanced = func(n int) string {
		if n == 1 {
			return "1"
		}
		return "(" + balanced(n/2) + "+" + balanced(n-n/2) + ")"
	}
	wideExpr := balanced(74)
	budgetHeader := "module main;fn i32 wide(" + strings.Join(params, ",") + "){return "
	base := "module main;fn i32 f(){return 1;}"
	atLimit := "module main;" + strings.Repeat(" ", 4096-len(base)) + "fn i32 f(){return 1;}"
	cases = append(cases, struct {
		input string
		clean bool
	}{wide, true}, struct {
		input string
		clean bool
	}{atLimit, true})
	cases = append(cases, struct {
		input string
		clean bool
	}{budgetHeader + wideExpr + ";}", true}) // 256 records
	var locals strings.Builder
	for i := 0; i < 83; i++ {
		fmt.Fprintf(&locals, "let i32 x%d=1;", i)
	}
	localHeader := "module main;fn i32 f(){" + locals.String()
	cases = append(cases, struct {
		input string
		clean bool
	}{localHeader + "return + + x82;}", true}) // 3 + 83*3 + 4 = 256 records
	for i, c := range cases {
		wire := native(c.input)
		if i == 0 {
			firstWire = wire
		}
		module, err := adaptFairyScalar(c.input, wire)
		if err != nil {
			t.Fatalf("adapter %q: %v", c.input, err)
		}
		// Oracle-only Go parsing. Its AST is never passed into the adapter.
		ref, err := frontend.AnalyzeSyntax(source.NewFile("main.fy", []byte(c.input)), source.DefaultLimits())
		if err != nil || ref.AST == nil {
			t.Fatalf("oracle: %v", err)
		}
		if !reflect.DeepEqual(module.Functions, ref.AST.Functions) || !reflect.DeepEqual(module.Bindings, ref.AST.Bindings) || !reflect.DeepEqual(module.Literals, ref.AST.Literals) {
			t.Fatalf("AST fields or summary mismatch %q", c.input)
		}
		got := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: module}})
		want := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: ref.AST}})
		if (len(want.Diagnostics) == 0 && len(want.Unsupported) == 0) != c.clean {
			t.Fatalf("oracle class %q: %+v", c.input, want)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("semantic mismatch %q:\ngot=%+v\nwant=%+v", c.input, got, want)
		}
	}
	unsupported := []string{
		"module other;fn i32 f(){return 1;}",
		"module main;pub fn i32 f(){return 1;}",
		"module main;unsafe fn i32 f(){return 1;}",
		"module main;fn i32 Owner::f(){return 1;}",
		"module main;@tag fn i32 f(){return 1;}",
		"module main;fn i32 f<T>(){return 1;}",
		"module main;fn bool f(){return true;}",
		"module main;fn i32 f(u8 x){return x;}",
		"module main;fn i32 f(){return 6/2;}",
		"module main;fn i32 f(){return 6%2;}",
		"module main;fn i32 f(){return 1<2;}",
		"module main;fn i32 f(){return true&&false;}",
		"module main;fn i32 f(){return ~1;}",
		"module main;fn i32 f(i32 x){return *x;}",
		"module main;fn i32 f(){return 0x2A;}",
		"module main;fn i32 f(){return 1_000;}",
		"module main;fn i32 f(){return call();}",
		"module main;fn i32 f(){let x=1;return x;}",
		"module main;fn i32 f(){var i32 x=1;return x;}",
		"module main;fn i32 f(){let u8 x=1;return x;}",
		"module main;fn i32 f(){let i32 x=call();return x;}",
		"module main;fn i32 f(){let i32 x=1;x=2;return x;}",
		"module main;fn i32 f(){let i32 x={1};return x;}",
		"module main;fn i32 f(){let i32 x=1;}",
		"module main;fn i32 f(){return 1;let i32 x=2;}",
		"module main;fn i32 f(){return 1;return 2;}",
		"module main;fn i32 f(){if(true){return 1;}return 0;}",
		"module main;fn i32 wide(" + strings.Join(params, ",") + ",i32 extra){return p15;}",
		atLimit + " ",
		"module main;fn i32 f(){return 01;}",
		"module main;fn i32 f(){return pkg::value;}",
		"module main;fn i32 f(){return true;}",
		"module main;fn i32 f(){return;}",
	}
	for _, input := range unsupported {
		if module, err := adaptFairyScalar(input, native(input)); err == nil || module != nil {
			t.Fatalf("adapter accepted %q", input)
		}
	}
	input := cases[0].input
	fields := strings.Fields(firstWire)
	mutate := func(index int, value string) string {
		copy := append([]string(nil), fields...)
		copy[index] = value
		return strings.Join(copy, " ")
	}
	root := 2 + len(input)
	for _, wire := range []string{
		"", strings.Join(fields[:len(fields)-1], " "), firstWire + "0 ",
		mutate(0, "2"), mutate(2, "0"), mutate(root, "999"), mutate(root+2, "99999"),
		mutate(root+3, "257"), mutate(root+6, "1"), mutate(root+1, "-1"), strings.Repeat("0 ", 131073),
	} {
		if module, err := adaptFairyScalar(input, wire); err == nil || module != nil {
			t.Fatal("accepted malformed private transport")
		}
	}
	arithmetic := "module main;fn i32 f(){return 1+2;}"
	parts := strings.Fields(native(arithmetic))
	foundBinary := false
	for p := 2 + len(arithmetic); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "204" {
			foundBinary = true
			for _, change := range []struct {
				offset int
				value  string
			}{{7, "47"}, {6, "0"}, {3, "1"}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(arithmetic, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed arithmetic record")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundBinary {
		t.Fatal("arithmetic witness has no binary record")
	}
	// Insert one unary record into an otherwise valid boundary-sized native
	// stream. This tests the reader's limits separately from producer rejection.
	for _, boundary := range []string{"module main;fn i32 f(){return " + strings.Repeat("+ ", 60) + "1;}", budgetHeader + wideExpr + ";}"} {
		parts := strings.Fields(native(boundary))
		inserted := false
		for p := 2 + len(boundary); p < len(parts); {
			op, _ := strconv.Atoi(parts[p+6])
			if parts[p] == "202" || parts[p] == "203" {
				record := []string{"203", parts[p+1], parts[p+2], "1", "0", "0", "1", "43"}
				changed := append([]string(nil), parts[:p]...)
				changed = append(changed, record...)
				changed = append(changed, parts[p:]...)
				if module, err := adaptFairyScalar(boundary, strings.Join(changed, " ")); err == nil || module != nil {
					t.Fatal("reader accepted excess arithmetic depth/nodes")
				}
				inserted = true
				break
			}
			p += 7 + op
		}
		if !inserted {
			t.Fatal("missing arithmetic boundary node")
		}
	}
	bindings := "module main;fn i32 f(){let i32 x=1;let i32 y=2;return x+y;}"
	parts = strings.Fields(native(bindings))
	var bindingRecords []int
	for p := 2 + len(bindings); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "109" {
			bindingRecords = append(bindingRecords, p)
		}
		p += 7 + op
	}
	if len(bindingRecords) != 2 {
		t.Fatal("binding witness must contain two declarations")
	}
	p, q := bindingRecords[0], bindingRecords[1]
	for _, change := range []struct {
		index int
		value string
	}{
		{p, "110"},           // mutable binding
		{p + 3, "1"},         // missing child role
		{p + 4, "0"},         // invalid name span
		{p + 7, "200"},       // non-type first child
		{p + 5, parts[p+16]}, // name overlaps initializer
		{q + 1, parts[p+1]},  // overlapping statement envelopes
	} {
		copy := append([]string(nil), parts...)
		copy[change.index] = change.value
		if module, err := adaptFairyScalar(bindings, strings.Join(copy, " ")); err == nil || module != nil {
			t.Fatal("accepted malformed binding record")
		}
	}
	t.Logf("scalar adapter: %d semantic comparisons, %d unsupported shapes, 22 malformed transports", len(cases), len(unsupported))
	return []string{"module main;fn i32 f(){return " + strings.Repeat("+ ", 61) + "1;}", budgetHeader + "+" + wideExpr + ";}", localHeader + "let i32 x83=1;return x83;}"}
}
