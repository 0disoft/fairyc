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
		{"module main;fn i32 f(){var i32 x=1;return x;}", true},
		{"module main;fn i32 f(i32 a){var i32 x=a;x=x+2;x=(x-3)*-a;let i32 y=x;return y;}", true},
		{"module main;\r\nfn i32 f(){var i32 x=1;\r\nx /* 대입 */ = -2147483648;\r\nreturn x;}", true},
		{"module main;fn i32 f(){let i32 x=1;x=2;return x;}", false},
		{"module main;fn i32 f(){missing=2;return 1;}", false},
		{"module main;fn i32 f(){x=2;var i32 x=1;return x;}", false},
		{"module main;fn i32 f(){var i32 x=1;x=missing;return x;}", false},
		{"module main;fn i32 f(){var i32 x=1;x=2147483648;return x;}", false},
		{"module main;fn i32 f(i32 x){x=2;return x;}", false},
		{"module main;fn i32 f(i32 x){var i32 x=x+1;x=x*2;return x;}", true},
		{"module main;fn i32 f(){var i32 x=1;x=" + strings.Repeat("+ ", 60) + "x;return x;}", true},
		{"module main;fn bool f(){return true;}", true},
		{"module main;fn bool f(){return false;}", true},
		{"module main;fn bool f(bool ready){return ready;}", true},
		{"module main;fn bool f(i32 a,bool ready){let bool x=ready;let bool y=a<2;return y;}", true},
		{"module main;fn bool f(i32 a){var bool x=false;x=a>=2;return x;}", true},
		{"module main;fn i32 f(){return true;}", false},
		{"module main;fn i32 f(){return 1<2;}", false},
		{"module main;fn bool f(){return 1;}", false},
		{"module main;fn bool f(){let bool x=1;return x;}", false},
		{"module main;fn i32 f(){let i32 x=false;return x;}", false},
		{"module main;fn bool f(){var bool x=false;x=1;return x;}", false},
		{"module main;fn bool f(){return 1==true;}", false},
		{"module main;fn bool f(){return true+false;}", false},
		{"module main;fn bool f(){return missing<2;}", false},
		{"module main;\r\nfn bool f(i32 a,i32 b){return (a+2)*3 /* 비교 */ <= b-1;}", true},
		{"module main;fn i32 f(i32 a){return if(a<0){-a}else{a};}", true},
		{"module main;fn bool f(bool a){return if(a){false}else{true};}", true},
		{"module main;fn i32 f(i32 a){let i32 x=if(a>0){a+1}else{2};return x;}", true},
		{"module main;fn bool f(i32 a){var bool x=false;x=if(a>0){a<3}else{true};return x;}", true},
		{"module main;\r\nfn i32 f(i32 a){return if(a<0){ /* 분기 */ if(a<-2){1}else{2} }else{3};}", true},
		{"module main;fn i32 f(){return if(1){2}else{3};}", false},
		{"module main;fn i32 f(bool a){return if(a){1}else{false};}", false},
		{"module main;fn bool f(bool a){return if(a){1}else{2};}", false},
		{"module main;fn i32 f(){return if(true){1}else{missing};}", false},
		{"module main;fn i32 f(){return if(missing){1}else{2};}", false},
		{"module main;fn i32 f(){return if(true){let i32 x=1;x}else{2};}", true},
		{"module main;fn i32 f(){var i32 x=1;return if(true){x=2;x}else{2};}", true},
		{"module main;fn i32 f(i32 x){return if(x>0){let i32 x=x+1;x}else{let i32 x=0;x};}", true},
		{"module main;fn bool f(bool a){return if(a){var bool x=false;x=a;x}else{let bool x=true;x};}", true},
		{"module main;fn i32 f(bool a){let i32 x=if(a){let i32 y=if(a){let i32 z=1;z}else{2};y}else{let i32 y=3;y};return x;}", true},
		{"module main;\r\nfn i32 f(i32 a){return if(a>0){\r\nvar i32 x=a; /* 분기 변수 */ x=x+2;\r\nx}else{a};}", true},
		{"module main;fn i32 f(){let i32 y=if(true){let i32 x=1;x}else{2};return x;}", false},
		{"module main;fn i32 f(bool a){return if(a){let i32 x=1;x}else{x};}", false},
		{"module main;fn i32 f(){return if(true){let i32 x=1;let i32 x=2;x}else{3};}", false},
		{"module main;fn i32 f(){return if(true){let i32 x=y;let i32 y=2;x}else{3};}", false},
		{"module main;fn i32 f(){let i32 x=1;return if(true){x=2;x}else{3};}", false},
		{"module main;fn bool f(){return if(true){var bool x=false;x=1;x}else{false};}", false},
		{"module main;fn i32 f(){return if(true){let i32 x=" + strings.Repeat("+ ", 57) + "1;x}else{2};}", true},
		{"module main;fn i32 f(bool a){var i32 x=0;if(a){x=1;}return x;}", true},
		{"module main;fn i32 f(bool a){var i32 x=0;if(a){x=1;}else{x=2;}x=x+3;return x;}", true},
		{"module main;fn i32 f(bool a){if(a){}return 1;}", true},
		{"module main;fn i32 f(bool a){if(a){}else{}return 1;}", true},
		{"module main;fn i32 f(i32 x){if(x>0){let i32 x=x+1;}else{var i32 x=2;x=3;}return x;}", true},
		{"module main;fn i32 f(bool a,bool b){var i32 x=0;if(a){if(b){x=1;}else{x=2;};}return x;}", true},
		{"module main;\r\nfn bool f(bool a){var bool x=false;\r\nif(a){let bool y=if(a){true}else{false}; /* 문장 */ x=y;}\r\nreturn x;}", true},
		{"module main;fn i32 f(bool a){return if(a){var i32 x=0;if(a){x=1;}x}else{2};}", true},
		{"module main;fn i32 f(){if(1){}return 0;}", false},
		{"module main;fn i32 f(bool a){if(a){let i32 x=1;}return x;}", false},
		{"module main;fn i32 f(bool a){var i32 y=0;if(a){let i32 x=1;}else{y=x;}return y;}", false},
		{"module main;fn i32 f(){let i32 x=0;if(true){x=1;}return x;}", false},
		{"module main;fn i32 f(i32 n){var i32 i=0;var i32 sum=0;while(i<n){sum=sum+i;i=i+1;}return sum;}", true},
		{"module main;fn i32 f(){while(false){}return 1;}", true},
		{"module main;fn bool f(bool a){var bool active=a;while(active){active=false;}return active;}", true},
		{"module main;fn i32 f(i32 n){var i32 x=0;while(x<n){let i32 step=1;if(x<2){x=x+step;}else{x=x+2;};}return x;}", true},
		{"module main;fn i32 f(i32 n){var i32 x=0;while(x<n){var i32 y=0;while(y<2){y=y+1;}x=x+y;}return x;}", true},
		{"module main;fn i32 f(bool a){var i32 x=0;if(a){while(x<2){x=x+1;}}return x;}", true},
		{"module main;fn i32 f(){return if(true){var i32 x=0;while(x<2){x=x+1;}x}else{2};}", true},
		{"module main;\r\nfn i32 f(i32 x){while(false){let i32 x=x+1; /* 루프 변수 */ }\r\nreturn x;}", true},
		{"module main;fn i32 f(){while(1){}return 0;}", false},
		{"module main;fn i32 f(){while(missing){}return 0;}", false},
		{"module main;fn i32 f(){while(false){let i32 x=1;}return x;}", false},
		{"module main;fn i32 f(){let i32 x=0;while(false){x=1;}return x;}", false},
	}
	for _, op := range []string{"==", "!=", "<", "<=", ">", ">="} {
		cases = append(cases, struct {
			input string
			clean bool
		}{"module main;fn bool f(i32 a,i32 b){return a" + op + "b;}", true})
	}
	var firstWire string
	// Exercise the same wire depth without multiplying semantic loop fixed points.
	nestedLoops := "while(false){let i32 x=" + strings.Repeat("+ ", 58) + "1;}"
	cases = append(cases, struct {
		input string
		clean bool
	}{"module main;fn i32 f(){" + nestedLoops + "return 0;}", true}) // initializer at depth 63
	nestedStatements := "let i32 x=1;"
	for i := 0; i < 20; i++ {
		nestedStatements = "if(true){" + nestedStatements + "};"
	}
	cases = append(cases, struct {
		input string
		clean bool
	}{"module main;fn i32 f(){" + nestedStatements + "return 0;}", true}) // initializer at depth 63
	nestedIf := "1"
	for i := 0; i < 30; i++ {
		nestedIf = "if(true){" + nestedIf + "}else{2}"
	}
	cases = append(cases, struct {
		input string
		clean bool
	}{"module main;fn i32 f(){return " + nestedIf + ";}", true}) // deepest tail at depth 63
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
		"module main;fn i32 f(u8 x){return x;}",
		"module main;fn i32 f(){return 6/2;}",
		"module main;fn i32 f(){return 6%2;}",
		"module main;fn i32 f(){return true&&false;}",
		"module main;fn bool f(){return true||false;}",
		"module main;fn bool f(){return !true;}",
		"module main;fn bool f(){return 1.0<2.0;}",
		"module main;fn bool f(){return null;}",
		"module main;fn bool f(){if(true){return false;}return true;}",
		"module main;fn i32 f(){if(true){1}return 0;}",
		"module main;fn i32 f(){if(true){}else{1}return 0;}",
		"module main;fn i32 f(){if(true){}else if(false){}return 0;}",
		"module main;fn i32 f(){if(true){call();}return 0;}",
		"module main;fn i32 f(){while(true){break;}return 0;}",
		"module main;fn i32 f(){while(true){continue;}return 0;}",
		"module main;fn i32 f(){while(true){return 1;}return 0;}",
		"module main;fn i32 f(){while(true){call();}return 0;}",
		"module main;fn i32 f(){for(var i32 x=0;x<2;x=x+1){}return 0;}",
		"module main;fn i32 f(){if(true){if(false){}}return 0;}",
		"module main;fn i32 f(){return if(true){1};}",
		"module main;fn i32 f(){return if(true){}else{2};}",
		"module main;fn i32 f(){return if(true){1}else{};}",
		"module main;fn i32 f(){return if(true){let i32 x=1;}else{2};}",
		"module main;fn i32 f(){return if(true){let x=1;x}else{2};}",
		"module main;fn i32 f(){return if(true){var i32 x=1;x+=1;x}else{2};}",
		"module main;fn i32 f(){return if(true){call();1}else{2};}",
		"module main;fn i32 f(){return if(true){return 1;}else{2};}",
		"module main;fn i32 f(){return if(true){1}else if(false){2}else{3};}",
		"module main;fn i32 f(){return ~1;}",
		"module main;fn i32 f(i32 x){return *x;}",
		"module main;fn i32 f(){return 0x2A;}",
		"module main;fn i32 f(){return 1_000;}",
		"module main;fn i32 f(){return call();}",
		"module main;fn i32 f(){let x=1;return x;}",
		"module main;fn i32 f(){var x=1;return x;}",
		"module main;fn i32 f(){var i32 x=1;x+=2;return x;}",
		"module main;fn i32 f(){var i32 x=1;x.field=2;return x;}",
		"module main;fn i32 f(){var i32 x=1;x[0]=2;return x;}",
		"module main;fn i32 f(){var i32 x=1;(x)=2;return x;}",
		"module main;fn i32 f(){var i32 x=1;*x=2;return x;}",
		"module main;fn i32 f(){let u8 x=1;return x;}",
		"module main;fn i32 f(){let i32 x=call();return x;}",
		"module main;fn i32 f(){let i32 x={1};return x;}",
		"module main;fn i32 f(){let i32 x=1;}",
		"module main;fn i32 f(){return 1;let i32 x=2;}",
		"module main;fn i32 f(){return 1;return 2;}",
		"module main;fn i32 f(){if(true){return 1;}return 0;}",
		"module main;fn i32 wide(" + strings.Join(params, ",") + ",i32 extra){return p15;}",
		atLimit + " ",
		"module main;fn i32 f(){return 01;}",
		"module main;fn i32 f(){return pkg::value;}",
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
		{p, "102"},           // inferred mutable binding
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
	assignment := "module main;fn i32 f(){var i32 x=1;x=2;return x;}"
	parts = strings.Fields(native(assignment))
	foundAssignment := false
	for p := 2 + len(assignment); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "104" {
			foundAssignment = true
			for _, change := range []struct {
				offset int
				value  string
			}{{7, "43"}, {6, "0"}, {3, "1"}, {8, "202"}, {10, parts[p+17]}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(assignment, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed assignment record")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundAssignment {
		t.Fatal("missing assignment witness")
	}
	comparison := "module main;fn bool f(i32 a,i32 b){return a<=b;}"
	parts = strings.Fields(native(comparison))
	foundComparison := false
	for p := 2 + len(comparison); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "204" {
			foundComparison = true
			if op != 2 || parts[p+7] != "60" || parts[p+8] != "61" {
				t.Fatal("comparison witness must have a two-byte <= operator")
			}
			for _, change := range []struct {
				offset int
				value  string
			}{{6, "3"}, {7, "256"}, {8, "62"}, {0, "203"}, {0, "104"}, {6, "1"}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(comparison, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed comparison operator")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundComparison {
		t.Fatal("missing comparison witness")
	}
	conditional := "module main;fn i32 f(){return if(true){1}else{2};}"
	parts = strings.Fields(native(conditional))
	foundIf := false
	for p := 2 + len(conditional); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "207" {
			foundIf = true
			if parts[p+14] != "100" || parts[p+28] != "100" {
				t.Fatal("conditional witness must contain two one-value blocks")
			}
			for _, change := range []struct {
				offset int
				value  string
			}{{3, "2"}, {7, "100"}, {14, "206"}, {17, "0"}, {29, parts[p+15]}, {18, parts[p+15]}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(conditional, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed conditional record")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundIf {
		t.Fatal("missing conditional witness")
	}
	branchLocals := "module main;fn i32 f(){return if(true){let i32 x=1;let i32 y=2;y}else{3};}"
	parts = strings.Fields(native(branchLocals))
	var declarations []int
	for p := 2 + len(branchLocals); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "109" {
			declarations = append(declarations, p)
		}
		p += 7 + op
	}
	if len(declarations) != 2 {
		t.Fatal("missing branch declaration witnesses")
	}
	p, q = declarations[0], declarations[1]
	for _, change := range []struct {
		index int
		value string
	}{{q + 1, parts[p+1]}, {q + 22, parts[q+1]}, {p, "103"}} {
		copy := append([]string(nil), parts...)
		copy[change.index] = change.value
		if module, err := adaptFairyScalar(branchLocals, strings.Join(copy, " ")); err == nil || module != nil {
			t.Fatal("accepted malformed branch statements or tail")
		}
	}
	statementIf := "module main;fn i32 f(){if(true){}else{}return 0;}"
	parts = strings.Fields(native(statementIf))
	foundStatementIf := false
	for p := 2 + len(statementIf); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "105" {
			foundStatementIf = true
			if parts[p+7] != "207" || parts[p+21] != "100" {
				t.Fatal("invalid statement conditional witness")
			}
			for _, change := range []struct {
				offset int
				value  string
			}{{3, "0"}, {7, "200"}, {10, "1"}, {21, "206"}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(statementIf, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed statement conditional")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundStatementIf {
		t.Fatal("missing statement conditional witness")
	}
	loop := "module main;fn i32 f(){while(false){}return 0;}"
	parts = strings.Fields(native(loop))
	foundLoop := false
	for p := 2 + len(loop); p < len(parts); {
		op, _ := strconv.Atoi(parts[p+6])
		if parts[p] == "106" {
			foundLoop = true
			if parts[p+14] != "100" {
				t.Fatal("loop witness has no body block")
			}
			for _, change := range []struct {
				offset int
				value  string
			}{{3, "3"}, {7, "100"}, {14, "206"}, {15, parts[p+8]}} {
				copy := append([]string(nil), parts...)
				copy[p+change.offset] = change.value
				if module, err := adaptFairyScalar(loop, strings.Join(copy, " ")); err == nil || module != nil {
					t.Fatal("accepted malformed loop record")
				}
			}
			break
		}
		p += 7 + op
	}
	if !foundLoop {
		t.Fatal("missing loop witness")
	}
	t.Logf("scalar adapter: %d semantic comparisons, %d unsupported shapes, 50 malformed transports", len(cases), len(unsupported))
	return []string{"module main;fn i32 f(){return " + strings.Repeat("+ ", 61) + "1;}", budgetHeader + "+" + wideExpr + ";}", localHeader + "let i32 x83=1;return x83;}", "module main;fn i32 f(){return if(true){" + nestedIf + "}else{2};}", "module main;fn i32 f(){return if(true){let i32 x=" + strings.Repeat("+ ", 58) + "1;x}else{2};}", "module main;fn i32 f(){if(true){" + nestedStatements + "}return 0;}", "module main;fn i32 f(){while(false){" + nestedLoops + "}return 0;}"}
}
