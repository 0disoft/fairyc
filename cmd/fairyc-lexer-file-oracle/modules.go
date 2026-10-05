package main

// This file prepares independent multi-file semantic observations. The Fairy
// executable reads and checks the authored sources itself; it never reads these
// reference diagnostics at runtime.
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

type moduleSource struct {
	Path     string `json:"path"`
	InputHex string `json:"inputHex"`
}
type moduleFixture struct {
	Name        string                 `json:"name"`
	Files       []moduleSource         `json:"files"`
	Outcome     string                 `json:"outcome"`
	Diagnostics []semantic.Diagnostic  `json:"diagnostics"`
	Unsupported []semantic.Unsupported `json:"unsupported"`
}

func writeModules(destination string) error {
	var fixtures []moduleFixture
	add := func(name, outcome string, inputs ...string) error {
		var files []semantic.File
		var sources []moduleSource
		for _, input := range inputs {
			modulePath := strings.TrimSpace(strings.SplitN(strings.TrimPrefix(input, "module "), ";", 2)[0])
			path := strings.ReplaceAll(modulePath, "::", "/") + ".fy"
			tree := syntax.Parse(source.NewFile(path, []byte(input)), source.DefaultLimits())
			if !tree.ErrorFree() {
				return fmt.Errorf("module syntax fixture %s: %+v", name, tree.Diagnostics)
			}
			module, err := ast.Lower(tree)
			if err != nil {
				return err
			}
			files = append(files, semantic.File{Path: path, Module: module})
			sources = append(sources, moduleSource{path, hex.EncodeToString([]byte(input))})
		}
		result := semantic.Analyze(files)
		if outcome == "pass" && (len(result.Diagnostics) != 0 || len(result.Unsupported) != 0) {
			return fmt.Errorf("module-positive %s: %+v %+v", name, result.Diagnostics, result.Unsupported)
		}
		if outcome == "error" && len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
			return fmt.Errorf("module-negative %s unexpectedly passes", name)
		}
		fixtures = append(fixtures, moduleFixture{name, sources, outcome, result.Diagnostics, result.Unsupported})
		return nil
	}
	for _, s := range []struct {
		name   string
		inputs []string
	}{
		{"single-empty-module", []string{"module main;"}},
		{"forward-import", []string{"module main; import library as lib; fn i64 f(){ return lib::identity(4000000000); }", "module library; pub fn i64 identity(i64 x){ return x; }"}},
		{"reversed-files", []string{"module library; pub fn i64 identity(i64 x){ return x; }", "module main; import library as lib; fn i64 f(){ return lib::identity(4000000000); }"}},
		{"nested-module-path", []string{"module main; import util::math as m; fn i32 f(){ return m::answer(); }", "module util::math; pub fn i32 answer(){ return 42; }"}},
		{"explicit-alias", []string{"module main; import library as renamed; fn bool f(){ return renamed::check(true); }", "module library; pub fn bool check(bool x){ return x; }"}},
		{"two-aliases-one-module", []string{"module main; import library as a; import library as b; fn i32 f(){ return a::f()+b::f(); }", "module library; pub fn i32 f(){ return 1; }"}},
		{"private-local-call", []string{"module main; import library as lib; fn i32 f(){ return lib::exposed(); }", "module library; fn i32 hidden(){ return 1; } pub fn i32 exposed(){ return hidden(); }"}},
		{"qualified-control-flow", []string{"module main; import library as lib; fn i64 f(bool x){ if(x){ return lib::f(); } else { return 2; } }", "module library; pub fn i64 f(){ return 4000000000; }"}},
		{"mutual-module-calls", []string{"module a; import b as b; pub fn i32 f(i32 x){ return b::g(x); }", "module b; import a as a; pub fn i32 g(i32 x){ return a::f(x); }"}},
		{"three-module-chain", []string{"module main; import a as a; fn i64 f(){ return a::f(); }", "module a; import b as b; pub fn i64 f(){ return b::f(); }", "module b; pub fn i64 f(){ return 4000000000; }"}},
		{"void-cross-call", []string{"module main; import library as lib; fn void f(){ lib::f(); }", "module library; pub fn void f(){}"}},
		{"unused-import", []string{"module main; import library as lib;", "module library;"}},
		{"UTF8-CRLF-final-line", []string{"module main; // 한글\r\nimport library as lib;\r\nfn i32 f(){ return lib::f(); }", "module library; pub fn i32 f(){ return 1; }"}},
		{"local-name-matches-alias", []string{"module main; import library as lib; fn i32 f(){ let lib=1; return lib+lib::f(); }", "module library; pub fn i32 f(){ return 2; }"}},
		{"qualified-path-trivia", []string{"module main; import util /* p */ :: math as m; fn i64 f(){return m /* call */ :: f(4000000000);}", "module util::math; pub fn i64 f(i64 x){return x;}"}},
	} {
		if err := add(s.name, "pass", s.inputs...); err != nil {
			return err
		}
	}
	for _, t := range []struct{ kind, value string }{{"i32", "2147483647"}, {"i64", "9223372036854775807"}, {"u8", "255"}, {"u32", "4294967295"}, {"usize", "18446744073709551615"}, {"bool", "true"}} {
		if err := add("scalar-"+t.kind, "pass", fmt.Sprintf("module main; import library as lib; fn %s f(){ return lib::f(%s); }", t.kind, t.value), fmt.Sprintf("module library; pub fn %s f(%s x){ return x; }", t.kind, t.kind)); err != nil {
			return err
		}
	}
	for _, s := range []struct {
		name   string
		inputs []string
	}{
		{"missing-import", []string{"module main; import absent as lib;"}},
		{"missing-alias", []string{"module main; fn i32 f(){ return lib::f(); }", "module library; pub fn i32 f(){ return 1; }"}},
		{"missing-item", []string{"module main; import library as lib; fn i32 f(){ return lib::missing(); }", "module library; pub fn i32 f(){ return 1; }"}},
		{"private-item", []string{"module main; import library as lib; fn i32 f(){ return lib::f(); }", "module library; fn i32 f(){ return 1; }"}},
		{"duplicate-alias", []string{"module main; import library as lib; import library as lib;", "module library;"}},
		{"alias-item-conflict", []string{"module main; import library as f; fn i32 f(){ return 1; }", "module library;"}},
		{"duplicate-function", []string{"module main; fn i32 f(){return 0;} fn i32 f(){return 1;}"}},
		{"not-reexported", []string{"module main; import a as a; fn i32 f(){ return a::f(); }", "module a; import b as b;", "module b; pub fn i32 f(){ return 1; }"}},
		{"unimported-module-path", []string{"module main; fn i32 f(){ return library::f(); }", "module library; pub fn i32 f(){return 1;}"}},
		{"wrong-argument-count", []string{"module main; import library as lib; fn i32 f(){ return lib::f(); }", "module library; pub fn i32 f(i32 x){return x;}"}},
		{"extra-argument", []string{"module main; import library as lib; fn i32 f(){ return lib::f(1,2); }", "module library; pub fn i32 f(i32 x){return x;}"}},
		{"wrong-argument-type", []string{"module main; import library as lib; fn i32 f(){ return lib::f(true); }", "module library; pub fn i32 f(i32 x){return x;}"}},
		{"wrong-result-type", []string{"module main; import library as lib; fn i32 f(){ return lib::f(); }", "module library; pub fn bool f(){return true;}"}},
		{"argument-literal-overflow", []string{"module main; import library as lib; fn u8 f(){ return lib::f(256); }", "module library; pub fn u8 f(u8 x){return x;}"}},
		{"imported-body-error", []string{"module main; import library as lib; fn i32 f(){ return lib::f(); }", "module library; pub fn i32 f(){return true;}"}},
		{"imported-body-name-error", []string{"module main; import library as lib; fn i32 f(){ return lib::f(); }", "module library; pub fn i32 f(){return missing;}"}},
		{"private-unqualified", []string{"module main; import library as lib; fn i32 f(){ return hidden(); }", "module library; pub fn i32 hidden(){return 1;}"}},
	} {
		if err := add(s.name, "error", s.inputs...); err != nil {
			return err
		}
	}
	// Limits are intentional candidate limits; the Go oracle may accept them.
	var modules []string
	for i := 0; i < 33; i++ {
		modules = append(modules, fmt.Sprintf("module m%d;", i))
	}
	if err := add("module-count-boundary", "pass", modules[:32]...); err != nil {
		return err
	}
	if err := add("module-count-over", "limit", modules...); err != nil {
		return err
	}
	if err := add("source-size-boundary", "pass", "module main;"+strings.Repeat(" ", 32768-len("module main;"))); err != nil {
		return err
	}
	if err := add("source-size-over", "limit", "module main;"+strings.Repeat(" ", 32769-len("module main;"))); err != nil {
		return err
	}
	for _, s := range []struct{ name, input string }{{"unsupported-struct", "module main; struct S {i32 x;}"}, {"unsupported-constant", "module main; const i32 x=1;"}, {"unsupported-generics", "module main; fn T f<T>(T x){return x;}"}} {
		if err := add(s.name, "unsupported", s.input); err != nil {
			return err
		}
	}
	if err := add("reserved-intrinsic-alias", "unsupported", "module main; import library as source_location;", "module library;"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Schema   string          `json:"schema"`
		Fixtures []moduleFixture `json:"fixtures"`
	}{"fairyc.modules-file-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0644)
}
