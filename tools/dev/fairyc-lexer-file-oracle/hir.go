package main

// This oracle is deliberately independent of the Fairy candidate's flat tables.
// All accepted trees come from semantic.Analyze, never candidate syntax/output.
// Main owns dispatch: writeHIR(destination) supplies --hir's implementation.
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

type hirOracleSource struct {
	Path     string `json:"path"`
	InputHex string `json:"inputHex"`
}

type hirOracleFixture struct {
	Name             string                     `json:"name"`
	InputHex         string                     `json:"inputHex,omitempty"` // Convenience for single-file fixtures.
	Files            []hirOracleSource          `json:"files"`
	Outcome          string                     `json:"outcome"` // Go result: pass, error, or unsupported.
	CompletedThrough semantic.Phase             `json:"completedThrough"`
	FailedPhase      semantic.Phase             `json:"failedPhase"`
	HIR              *hirOraclePackage          `json:"hir"`     // Null for every rejected input.
	Symbols          map[string]hirOracleSymbol `json:"symbols"` // Build-local ID -> stable identity.
	Resolutions      []semantic.Resolution      `json:"resolutions"`
	Types            []semantic.TypeFact        `json:"types"`
	Diagnostics      []semantic.Diagnostic      `json:"diagnostics"`
	Unsupported      []semantic.Unsupported     `json:"unsupported"`
}

// writeHIR writes a bounded corpus. Expected outcomes guard the fixture authors;
// the emitted outcome is classified from the actual Go semantic result.
func writeHIR(destination string) error {
	fixtures := []hirOracleFixture{}
	multiFixtures := []hirOracleFixture{}
	add := func(name, expected string, inputs ...string) error {
		files := []semantic.File{}
		sources := []hirOracleSource{}
		seen := map[string]bool{}
		for _, input := range inputs {
			if len(input) > 4096 || !strings.HasPrefix(input, "module ") {
				return fmt.Errorf("HIR fixture %s: invalid or unbounded source", name)
			}
			modulePath, _, found := strings.Cut(strings.TrimPrefix(input, "module "), ";")
			if !found {
				return fmt.Errorf("HIR fixture %s: missing module declaration", name)
			}
			path := strings.ReplaceAll(strings.TrimSpace(modulePath), "::", "/") + ".fy"
			if seen[path] {
				return fmt.Errorf("HIR fixture %s: duplicate source %s", name, path)
			}
			seen[path] = true
			file := source.NewFile(path, []byte(input))
			if _, err := file.ExpectedModule(); err != nil {
				return err
			}
			tree := syntax.Parse(file, source.DefaultLimits())
			if !tree.ErrorFree() {
				return fmt.Errorf("HIR syntax fixture %s: %+v", name, tree.Diagnostics)
			}
			module, err := ast.Lower(tree)
			if err != nil {
				return fmt.Errorf("HIR AST fixture %s: %w", name, err)
			}
			files = append(files, semantic.File{Path: path, Module: module})
			sources = append(sources, hirOracleSource{path, hex.EncodeToString([]byte(input))})
		}
		result := semantic.Analyze(files)
		outcome := "pass"
		if len(result.Diagnostics) > 0 {
			outcome = "error"
		} else if len(result.Unsupported) > 0 {
			outcome = "unsupported"
		}
		if outcome != expected {
			return fmt.Errorf("HIR fixture %s: expected %s, Go returned %s: %+v %+v", name, expected, outcome, result.Diagnostics, result.Unsupported)
		}
		f := hirOracleFixture{
			Name: name, Files: sources, Outcome: outcome, CompletedThrough: result.CompletedThrough, FailedPhase: result.FailedPhase,
			Symbols: map[string]hirOracleSymbol{}, Resolutions: []semantic.Resolution{}, Types: []semantic.TypeFact{},
			Diagnostics: append([]semantic.Diagnostic{}, result.Diagnostics...), Unsupported: append([]semantic.Unsupported{}, result.Unsupported...),
		}
		if len(sources) == 1 {
			f.InputHex = sources[0].InputHex
		}
		if outcome == "pass" {
			observed, symbols, err := observeOracleHIR(result.HIR, files)
			if err != nil {
				return fmt.Errorf("HIR fixture %s: %w", name, err)
			}
			f.HIR, f.Symbols = observed, symbols
			f.Resolutions = append(f.Resolutions, result.Resolutions...)
			f.Types = append(f.Types, result.Types...)
		}
		if len(sources) == 1 {
			fixtures = append(fixtures, f)
		} else {
			multiFixtures = append(multiFixtures, f)
		}
		return nil
	}
	for _, seed := range []struct{ name, input string }{
		{"module-only", "module main;"},
		{"signed-minima", "module main; fn i32 f(){ return -2147483648; } fn i64 g(){ return -9223372036854775808; }"},
		{"signed-minima-radix", "module main; fn i32 f(){ return -0x80000000; } fn i64 g(){ return -0x8000000000000000; }"},
		{"canonical-radices", "module main; fn i64 f(){ return 0b1111+0xFF+4_000_000_000; }"},
		{"usize-maximum-context", "module main; fn usize f(usize x){ let usize m=18446744073709551615; return (m & x) | (0xFFFFFFFFFFFFFFFF ^ x); }"},
		{"i64-context-grouped", "module main; fn i64 f(i64 x){ return ((1+2)*x)+(x*(1+2)); }"},
		{"u8-context", "module main; fn u8 f(u8 x){ return (1+x) & 255; }"},
		{"u32-context", "module main; fn u32 f(u32 x){ return (x+1) ^ 4294967295; }"},
		{"short-circuit-order", "module main; fn bool f(bool x,bool y){ return !(x==y) || x && g(y); } fn bool g(bool b){ return b; }"},
		{"grouped-unary", "module main; fn i64 f(i64 x){ return ((-x))+(~x)+(+x); }"},
		{"shadow-restored", "module main; fn i64 f(i64 x){ let y=x; { let y=y+1; { let y=y+2; y; } y; } return y; }"},
		{"mutable-inferred", "module main; fn i32 f(){ var x=1; x+=2; return x; }"},
		{"compound-all", "module main; fn i64 f(){ var i64 x=4000000000; x+=2; x-=1; x*=2; x/=2; x%=7; x&=3; x|=2; x^=1; x<<=1; x>>=1; return x; }"},
		{"block-tail-return", "module main; pub fn i64 f(i64 x){ let i64 y={ let z=x+1; z }; return y; }"},
		{"ordered-forward-call", "module main; fn i64 f(i64 x){ return g(x+1,x+2,true); } fn i64 g(i64 a,i64 b,bool c){ return if(c){ a-b }else{ b-a }; }"},
		{"mutual-recursion", "module main; fn i64 f(i64 x){ return g(x); } fn i64 g(i64 x){ return if(x==0){ 0 }else{ f(x-1) }; }"},
		{"UTF8-CRLF", "module main; // 한글\r\nfn i64 f(i64 x){\r\n let y=(x+1);\r\n return y;\r\n}"},
	} {
		if err := add(seed.name, "pass", seed.input); err != nil {
			return err
		}
	}
	// Reuse the existing authored control inputs, not their expected candidate
	// categories. Skip three redundant positives to keep the corpus small.
	controlErrors := map[string]bool{
		"control-if-condition": true, "control-while-condition": true, "control-for-condition": true,
		"control-assert-condition": true, "control-if-branch-types": true, "control-break-outside": true,
		"control-continue-outside": true, "control-for-immutable-post": true, "control-defer-argument-type": true,
	}
	controlUnsupported := map[string]bool{
		"control-if-one-return": true, "control-for-initial-call": true, "control-defer-not-call": true,
	}
	if err := addControlTypes(func(name, input, candidateOutcome, _ string) error {
		if candidateOutcome == "pass" && name != "control-if-inferred" && name != "control-else-if-all-return" && name != "control-if-void" {
			return add(name, "pass", input)
		}
		if controlErrors[name] {
			return add(name, "error", input)
		}
		if controlUnsupported[name] {
			return add(name, "unsupported", input)
		}
		return nil
	}); err != nil {
		return err
	}
	if err := add("import-public", "pass",
		"module main; import util::math as m; fn i64 f(i64 x){ return m::choose(x,4000000000); }",
		"module util::math; fn i64 hidden(i64 x){ return x+1; } pub fn i64 choose(i64 a,i64 b){ return hidden(a)+b; }"); err != nil {
		return err
	}
	if err := add("import-public-reversed", "pass",
		"module library; pub fn void observe(i32 x){}",
		"module main; import library as lib; fn void f(){ defer lib::observe(2); lib::observe(1); }"); err != nil {
		return err
	}
	for _, seed := range []struct{ name, input, outcome string }{
		{"bad-binding", "module main; fn i32 f(){ let i32 x=true; return 0; }", "error"},
		{"bad-return", "module main; fn i64 f(){ return true; }", "error"},
		{"bad-assignment", "module main; fn i32 f(){ var i32 x=1; x=true; return x; }", "error"},
		{"immutable-local", "module main; fn i32 f(){ let i32 x=1; x=2; return x; }", "error"},
		{"immutable-parameter", "module main; fn i32 f(i32 x){ x+=1; return x; }", "error"},
		{"mixed-width", "module main; fn i64 f(i32 x,i64 y){ return x+y; }", "error"},
		{"argument-order-type", "module main; fn i64 f(){ return g(true,1); } fn i64 g(i64 x,bool b){ return x; }", "error"},
		{"arity", "module main; fn i64 f(){ return g(); } fn i64 g(i64 x){ return x; }", "error"},
		{"unresolved-after-valid-function", "module main; fn i32 ok(){ return 1; } fn i32 f(){ return missing; }", "error"},
		{"minimum-parenthesized", "module main; fn i64 f(){ return -(9223372036854775808); }", "error"},
		{"usize-overflow", "module main; fn usize f(){ return 18446744073709551616; }", "error"},
		{"inferred-default-overflow", "module main; fn i64 f(){ let x=4000000000; return x; }", "error"},
		{"u64-unadmitted", "module main; fn u64 f(){ return 18446744073709551615; }", "error"},
		{"not-callable", "module main; fn i32 f(i32 x){ return x(); }", "unsupported"},
	} {
		if err := add(seed.name, seed.outcome, seed.input); err != nil {
			return err
		}
	}
	if err := add("import-private-rejected", "error", "module main; import library as lib; fn i32 f(){ return lib::hidden(); }", "module library; fn i32 hidden(){ return 1; }"); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Schema        string             `json:"schema"`
		SpanPolicy    string             `json:"spanPolicy"`
		Fixtures      []hirOracleFixture `json:"fixtures"`
		MultiFixtures []hirOracleFixture `json:"multiFixtures"`
	}{"fairyc.hir-file-oracle/v1", "UTF-8 byte offsets, end exclusive; symbol declaration spans exact; group nodes retained, comparator may unwrap only single-child groups and ignore their wrapper spans; IDs are observation-local", fixtures, multiFixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0644)
}
