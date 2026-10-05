package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

type nameFixture struct {
	Name           string         `json:"name"`
	InputHex       string         `json:"inputHex"`
	Outcome        string         `json:"outcome"`
	Expected       *parserGolden  `json:"expected"`
	ReferencePhase semantic.Phase `json:"referenceFailedPhase"`
	Symbols        []nameSymbol   `json:"symbols"`
	Resolutions    []nameUse      `json:"resolutions"`
}
type nameSymbol struct {
	Kind  string `json:"kind"`
	Start int    `json:"start"`
	End   int    `json:"end"`
}
type nameUse struct {
	Start       int    `json:"start"`
	End         int    `json:"end"`
	TargetStart int    `json:"targetStart"`
	TargetEnd   int    `json:"targetEnd"`
	TargetKind  string `json:"targetKind"`
}

func writeNames(destination string) error {
	var fixtures []nameFixture
	add := func(name, input, outcome string) error {
		file := source.NewFile("main.fy", []byte(input))
		tree := syntax.Parse(file, source.DefaultLimits())
		if !tree.ErrorFree() {
			return fmt.Errorf("names syntax fixture %s: %+v", name, tree.Diagnostics)
		}
		module, err := ast.Lower(tree)
		if err != nil {
			return err
		}
		result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: module}})
		f := nameFixture{Name: name, InputHex: hex.EncodeToString([]byte(input)), Outcome: outcome, ReferencePhase: result.FailedPhase}
		if outcome == "pass" {
			for _, function := range module.Functions {
				f.Symbols = append(f.Symbols, nameSymbol{"function", function.NameSpan.StartByte, function.NameSpan.EndByte})
				for _, parameter := range function.Parameters {
					f.Symbols = append(f.Symbols, nameSymbol{"parameter", parameter.NameSpan.StartByte, parameter.NameSpan.EndByte})
				}
			}
			for _, binding := range module.Bindings {
				kind := "local-let"
				if binding.Mutable {
					kind = "local-var"
				}
				f.Symbols = append(f.Symbols, nameSymbol{kind, binding.NameSpan.StartByte, binding.NameSpan.EndByte})
			}
			for _, use := range result.Resolutions {
				f.Resolutions = append(f.Resolutions, nameUse{use.Use.StartByte, use.Use.EndByte, use.Target.Declaration.StartByte, use.Target.Declaration.EndByte, use.Target.Kind})
			}
		}
		var naming []semantic.Diagnostic
		for _, d := range result.Diagnostics {
			if d.Code == "E0101" || d.Code == "E0102" {
				naming = append(naming, d)
			}
		}
		if outcome == "pass" && (len(naming) != 0 || result.FailedPhase == semantic.PhaseResolve) {
			return fmt.Errorf("name-positive %s: %+v", name, result.Diagnostics)
		}
		if outcome == "unresolved" || outcome == "duplicate" {
			code, message := "E0101", "unresolved name"
			if outcome == "duplicate" {
				code, message = "E0102", "duplicate name in scope"
			}
			if len(naming) != 1 || naming[0].Code != code {
				return fmt.Errorf("name-negative %s: %+v", name, result.Diagnostics)
			}
			span := naming[0].Primary
			f.Expected = &parserGolden{message, span.StartByte, span.EndByte}
		}
		fixtures = append(fixtures, f)
		return nil
	}
	for _, s := range []struct{ name, input string }{
		{"module-only", "module main;"},
		{"params", "module main; fn i64 f(i64 price, i64 fee) { let i64 total=price+fee*2; return total; }"},
		{"forward-call", "module main; fn i64 f() { return later(3); } fn i64 later(i64 x) { return x; }"},
		{"recursive-name", "module main; fn i64 f(i64 x) { return f(x); }"},
		{"inner-shadow-initializer", "module main; fn i64 f(i64 x) { let i64 total=x; { let i64 total=total+1; total; } return total; }"},
		{"parameter-shadow", "module main; fn i64 f(i64 x) { let i64 x=x+1; return x; }"},
		{"siblings", "module main; fn i64 f() { { let i64 x=1; x; } { let i64 x=2; x; } return 0; }"},
		{"separate-functions", "module main; fn i64 f() { let i64 x=1; return x; } fn i64 g() { let i64 x=2; return x; }"},
		{"assignment", "module main; fn i64 f() { var i64 x=1; x+=2; return x; }"},
		{"inferred", "module main; fn i64 f(i64 x) { let y=-(x+1); return y; }"},
		{"bool", "module main; fn bool f(bool value) { return !value || false; }"},
		{"void", "module main; fn void f() { return; }"},
		{"CRLF-no-final-newline", "module main;\r\nfn i64 f(i64 x) {\r\n return x;\r\n}"},
		{"UTF8-comment", "module main; // 한글\nfn i64 f() { return 4000000000; }"},
		{"name-only-type-error", "module main; fn i64 f() { return true; }"},
		{"name-only-immutable-assignment", "module main; fn i64 f() { let i64 x=1; x=2; return x; }"},
		{"if", "module main; fn i64 f(i64 x) { if(x>0) { let i64 y=x; return y; } else { let i64 y=0; return y; } }"},
		{"else-if", "module main; fn i64 f(i64 x) { return if(x>0) { x } else if(x<0) { -x } else { 0 }; }"},
		{"while", "module main; fn i64 f(i64 x) { var i64 y=x; while(y>0) { y-=1; if(y==1) { continue; } if(y==0) { break; } } return y; }"},
		{"for-scope", "module main; fn i64 f(i64 i) { for(var i64 i=i; i<3; i+=1) { let i64 i=i+1; i; } return i; }"},
		{"for-assignment", "module main; fn i64 f() { var i64 i=0; for(i=1; i<3; i+=1) { i; } return i; }"},
		{"for-empty", "module main; fn void f() { for(;;) { break; } }"},
		{"loop-siblings", "module main; fn i64 f() { for(var i64 i=0; i<1; i+=1) { i; } for(var i64 i=0; i<1; i+=1) { i; } return 0; }"},
		{"assert-discard-defer", "module main; fn void f(i64 x) { assert(x>0); discard x+1; defer g(x); } fn void g(i64 x) {}"},
		{"break-name-only", "module main; fn void f() { break; }"},
		{"source-limit", "module main;" + strings.Repeat(" ", 32768-len("module main;"))},
	} {
		if err := add(s.name, s.input, "pass"); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input, outcome string }{
		{"undeclared", "module main; fn i64 f() { return missing; }", "unresolved"},
		{"before-local", "module main; fn i64 f() { x; let i64 x=1; return x; }", "unresolved"},
		{"self-initializer", "module main; fn i64 f() { let i64 x=x; return x; }", "unresolved"},
		{"inner-leak", "module main; fn i64 f() { { let i64 x=1; } return x; }", "unresolved"},
		{"unknown-target", "module main; fn i64 f() { x=1; return 0; }", "unresolved"},
		{"unknown-argument", "module main; fn i64 f() { return g(missing); } fn i64 g(i64 x) { return x; }", "unresolved"},
		{"unknown-callee", "module main; fn i64 f() { return missing(1); }", "unresolved"},
		{"duplicate-param", "module main; fn i64 f(i64 x,i64 x) { return x; }", "duplicate"},
		{"duplicate-local", "module main; fn i64 f() { let i64 x=1; let i64 x=2; return x; }", "duplicate"},
		{"duplicate-function", "module main; fn i64 f() { return 1; } fn i64 f() { return 2; }", "duplicate"},
		{"branch-leak", "module main; fn i64 f() { if(true) { let i64 x=1; } return x; }", "unresolved"},
		{"branch-sibling-leak", "module main; fn i64 f() { if(true) { let i64 x=1; } else { x; } return 0; }", "unresolved"},
		{"while-leak", "module main; fn i64 f() { while(false) { let i64 x=1; } return x; }", "unresolved"},
		{"for-initializer-leak", "module main; fn i64 f() { for(var i64 i=0; i<1; i+=1) {} return i; }", "unresolved"},
		{"for-post-body-local", "module main; fn i64 f() { for(var i64 i=0; i<1; i+=x) { let i64 x=1; } return 0; }", "unresolved"},
		{"for-self-initializer", "module main; fn i64 f() { for(var i64 i=i; i<1; i+=1) {} return 0; }", "unresolved"},
		{"late-error", "module main; fn i64 good() { return 1; }\nfn i64 bad() { let i64 total=unknown; return total; }", "unresolved"},
	} {
		if err := add(s.name, s.input, s.outcome); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input string }{
		{"import", "module main; import std::io as io;"},
		{"constant", "module main; const i64 n=1;"},
		{"struct", "module main; struct S { i64 x; }"},
		{"generic", "module main; fn T f<T>(T x) { return x; }"},
		{"attribute", "module main; @must_use fn i64 f() { return 1; }"},
		{"unsafe", "module main; unsafe fn i64 f() { return 1; }"},
		{"borrow-type", "module main; fn i64 f(ref i64 x) { return *x; }"},
		{"user-type", "module main; fn Missing f() { return 0; }"},
		{"string", "module main; fn i64 f() { \"text\"; return 0; }"},
		{"float", "module main; fn i64 f() { 1.5; return 0; }"},
		{"null", "module main; fn i64 f() { null; return 0; }"},
		{"qualified", "module main; fn i64 f() { return other::name; }"},
		{"labeled-call", "module main; fn i64 f() { return g(.x=1); } fn i64 g(i64 x) { return x; }"},
		{"array", "module main; fn i64 f() { let i64[1] x=[1]; return 0; }"},
		{"intrinsic", "module main; fn i64 f() { source_location(); return 0; }"},
	} {
		if err := add(s.name, s.input, "unsupported"); err != nil {
			return err
		}
	}
	for _, count := range []int{128, 129} {
		var globals, locals strings.Builder
		globals.WriteString("module main;")
		locals.WriteString("module main; fn i64 f() {")
		for i := 0; i < count; i++ {
			fmt.Fprintf(&globals, "fn i64 f%d() { return 0; }", i)
			fmt.Fprintf(&locals, "let i64 x%d=1;", i)
		}
		locals.WriteString("return x0; }")
		outcome := "pass"
		if count == 129 {
			outcome = "limit"
		}
		if err := add(fmt.Sprintf("globals-%d", count), globals.String(), outcome); err != nil {
			return err
		}
		if err := add(fmt.Sprintf("active-locals-%d", count), locals.String(), outcome); err != nil {
			return err
		}
	}
	if err := add("walk-depth", "module main; fn i64 f() { return "+strings.Repeat("1+", 70)+"1; }", "limit"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Schema   string        `json:"schema"`
		Fixtures []nameFixture `json:"fixtures"`
	}{"fairyc.names-file-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0644)
}
