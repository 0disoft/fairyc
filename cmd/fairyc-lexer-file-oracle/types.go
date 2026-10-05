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

type typeFixture struct {
	Name        string                 `json:"name"`
	InputHex    string                 `json:"inputHex"`
	Outcome     string                 `json:"outcome"`
	Message     string                 `json:"message"`
	Diagnostics []semantic.Diagnostic  `json:"diagnostics"`
	Unsupported []semantic.Unsupported `json:"unsupported"`
}

func writeTypes(destination string) error {
	var fixtures []typeFixture
	add := func(name, input, outcome, message string) error {
		file := source.NewFile("main.fy", []byte(input))
		tree := syntax.Parse(file, source.DefaultLimits())
		if !tree.ErrorFree() {
			return fmt.Errorf("type syntax fixture %s: %+v", name, tree.Diagnostics)
		}
		module, err := ast.Lower(tree)
		if err != nil {
			return err
		}
		result := semantic.Analyze([]semantic.File{{Path: "main.fy", Module: module}})
		if outcome == "pass" && (len(result.Diagnostics) != 0 || len(result.Unsupported) != 0) {
			return fmt.Errorf("type-positive %s: %+v %+v", name, result.Diagnostics, result.Unsupported)
		}
		if outcome == "error" && len(result.Diagnostics) == 0 && len(result.Unsupported) == 0 {
			return fmt.Errorf("type-negative %s unexpectedly passes", name)
		}
		fixtures = append(fixtures, typeFixture{name, hex.EncodeToString([]byte(input)), outcome, message, result.Diagnostics, result.Unsupported})
		return nil
	}
	for _, s := range []struct{ name, input string }{
		{"module-only", "module main;"},
		{"struct", "module main; struct S { i32 x; }"},
		{"i64-forward-call", "module main; fn i64 f() { return g(4000000000); } fn i64 g(i64 x) { return x; }"},
		{"recursive", "module main; fn i64 f(i64 x) { return f(x); }"},
		{"inferred-shadow", "module main; fn i64 f(i64 x) { let y=x; { let y=y+1; y; } return y; }"},
		{"compound-assignment", "module main; fn i64 f() { var i64 x=4000000000; x+=2; x-=1; x*=2; x/=2; x%=7; x&=3; x|=2; x^=1; x<<=1; x>>=1; return x; }"},
		{"bool-operators", "module main; fn bool f(bool x, bool y) { return !(x==y) || x && y; }"},
		{"void-call", "module main; fn void f() { g(); return; } fn void g() {}"},
		{"block-tail", "module main; fn i64 f(i64 x) { let i64 y={ let z=x+1; z }; return y; }"},
		{"signed-minima", "module main; fn i32 f() { return -2147483648; } fn i64 g() { return -9223372036854775808; }"},
		{"signed-minima-radix", "module main; fn i32 f() { return -0x80000000; } fn i64 g() { return -0x8000000000000000; }"},
		{"integer-radices", "module main; fn i64 f() { return 0b1111+0xFF+4_000_000_000; }"},
		{"UTF8-CRLF-no-final-newline", "module main; // 한글\r\nfn i64 f() {\r\n return 4000000000;\r\n}"},
		{"source-limit", "module main;" + strings.Repeat(" ", 32768-len("module main;"))},
	} {
		if err := add(s.name, s.input, "pass", ""); err != nil {
			return err
		}
	}
	for _, t := range []struct{ name, maximum string }{{"i32", "2147483647"}, {"i64", "9223372036854775807"}, {"u8", "255"}, {"u32", "4294967295"}, {"usize", "18446744073709551615"}} {
		for _, expr := range []string{t.maximum, "0", "~0", "1+2*3", "1+x", "x+1", "(1+2)*x", "x*(1+2)", "(1<<2)|3"} {
			input := fmt.Sprintf("module main; fn %s f(%s x) { return %s; }", t.name, t.name, expr)
			if err := add(t.name+"-"+expr, input, "pass", ""); err != nil {
				return err
			}
		}
		if err := add(t.name+"-comparison", fmt.Sprintf("module main; fn bool f(%s x) { return 0 < x && x >= 1; }", t.name), "pass", ""); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input, message string }{
		{"binding-mismatch", "module main; fn i32 f() { let i32 x=true; return 0; }", "type mismatch"},
		{"return-mismatch", "module main; fn i64 f() { return true; }", "type mismatch"},
		{"return-empty", "module main; fn i32 f() { return; }", "type mismatch"},
		{"void-return-value", "module main; fn void f() { return 1; }", "type mismatch"},
		{"assignment-mismatch", "module main; fn i32 f() { var i32 x=1; x=true; return x; }", "type mismatch"},
		{"immutable-assignment", "module main; fn i32 f() { let i32 x=1; x=2; return x; }", "assignment target is immutable"},
		{"immutable-parameter", "module main; fn i32 f(i32 x) { x=2; return x; }", "assignment target is immutable"},
		{"mixed-width", "module main; fn i64 f(i32 x,i64 y) { return x+y; }", "type mismatch"},
		{"peer-over-destination", "module main; fn u32 f(u8 x) { return 1+x; }", "type mismatch"},
		{"call-count", "module main; fn i64 f() { return g(); } fn i64 g(i64 x) { return x; }", "call argument count mismatch"},
		{"extra-call-argument", "module main; fn i64 f() { return g(1,2); } fn i64 g(i64 x) { return x; }", "call argument count mismatch"},
		{"call-argument-type", "module main; fn i64 f() { return g(true); } fn i64 g(i64 x) { return x; }", "type mismatch"},
		{"not-callable", "module main; fn i64 f(i64 x) { return x(); }", "expression is not callable"},
		{"missing-return", "module main; fn i64 f() { let i64 x=1; }", "missing return value"},
		{"nested-return-only", "module main; fn i64 f() { { return 1; } }", "missing return value"},
		{"not-bool-prefix", "module main; fn bool f() { return !1; }", "expected bool operand"},
		{"not-bool-left", "module main; fn bool f() { return 1 && true; }", "expected bool operand"},
		{"not-bool-right", "module main; fn bool f() { return true || 1; }", "expected bool operand"},
		{"unsigned-minus", "module main; fn u8 f(u8 x) { return -x; }", "type mismatch"},
		{"bool-order", "module main; fn bool f() { return true < false; }", "invalid scalar operator"},
		{"bool-add", "module main; fn bool f() { return true+false; }", "invalid scalar operator"},
		{"bool-complement", "module main; fn bool f() { return ~true; }", "invalid scalar operator"},
		{"i32-too-large", "module main; fn i32 f() { return 2147483648; }", "integer literal out of range"},
		{"i64-too-large", "module main; fn i64 f() { return 9223372036854775808; }", "integer literal out of range"},
		{"u8-too-large", "module main; fn u8 f() { return 256; }", "integer literal out of range"},
		{"u32-too-large", "module main; fn u32 f() { return 4294967296; }", "integer literal out of range"},
		{"usize-too-large", "module main; fn usize f() { return 18446744073709551616; }", "integer literal out of range"},
		{"i64-negative-too-large", "module main; fn i64 f() { return -9223372036854775809; }", "integer literal out of range"},
		{"minimum-parenthesized", "module main; fn i64 f() { return -(9223372036854775808); }", "integer literal out of range"},
		{"minimum-unary-plus", "module main; fn i64 f() { return +9223372036854775808; }", "integer literal out of range"},
		{"default-i32", "module main; fn i64 f() { let x=4000000000; return x; }", "integer literal out of range"},
		{"literal-no-cancellation", "module main; fn u8 f() { return 256-1; }", "integer literal out of range"},
		{"unresolved", "module main; fn i32 f() { return missing; }", "unresolved name"},
	} {
		if err := add(s.name, s.input, "error", s.message); err != nil {
			return err
		}
	}
	for _, s := range []struct{ name, input string }{
		{"switch", "module main; fn i32 f() { return switch(1) { case .none: { 0 } }; }"},
		{"import", "module main; import std::io as io;"},
	} {
		if err := add(s.name, s.input, "unsupported", "unsupported name-check syntax"); err != nil {
			return err
		}
	}
	for _, count := range []int{1365, 1366} {
		outcome, message := "pass", ""
		if count == 1366 {
			outcome, message = "limit", "type-check limit"
		}
		if err := add(fmt.Sprintf("expression-cap-%d", count), "module main; fn i32 f() {"+strings.Repeat("1+1;", count)+"return 0; }", outcome, message); err != nil {
			return err
		}
	}
	if err := addControlTypes(add); err != nil {
		return err
	}
	data, err := json.MarshalIndent(struct {
		Schema   string        `json:"schema"`
		Fixtures []typeFixture `json:"fixtures"`
	}{"fairyc.types-file-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
		return err
	}
	return os.WriteFile(destination, append(data, '\n'), 0644)
}
