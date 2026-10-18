package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
	"os"
	"path/filepath"
)

// Independent Go semantic observations, never inputs to candidate checking.
func writeNominal(path string) error {
	cases := []struct {
		name, input string
		valid       bool
	}{
		{"fields", "module main;struct A{i64 x;bool y;}fn i64 f(){let a=A{.y=true,.x=4000000000};return a.x;}", true},
		{"nested", "module main;struct A{i32 x;}struct B{A a;}fn i32 f(){var b=B{.a=A{.x=1}};b.a.x+=2;let copy=b;return copy.a.x;}", true},
		{"forward", "module main;struct B{A a;}struct A{i32 x;}fn B f(){return B{.a=A{.x=3}};}", true},
		{"parameter-return", "module main;struct A{i32 x;}fn A id(A a){return a;}fn i32 f(){return id(A{.x=4}).x;}", true},
		{"empty", "module main;struct A{}fn A f(){return A{};}", true},
		{"shorthand", "module main;struct A{i32 x;bool y;}fn A f(i32 x,bool y){return A{.x,.y};}", true},
		{"branch", "module main;struct A{i32 x;}fn A f(bool b){return if(b){A{.x=1}}else{A{.x=2}};}", true},
		{"whole-assignment", "module main;struct A{i32 x;}fn A f(){var a=A{.x=1};a=A{.x=2};return a;}", true},
		{"context-minimum", "module main;struct A{i64 x;}fn A f(){return A{.x=-9223372036854775808};}", true},
		{"CRLF-UTF8", "module main; // 한글\r\nstruct A{i32 x;}fn A f(){return A{.x=3};}", true},
		{"nominal-mismatch", "module main;struct A{i32 x;}struct B{i32 x;}fn A f(B b){return b;}", false},
		{"field-type", "module main;struct A{i32 x;}fn A f(){return A{.x=true};}", false},
		{"missing-field", "module main;struct A{i32 x;}fn A f(){return A{};}", false},
		{"duplicate-initializer", "module main;struct A{i32 x;}fn A f(){return A{.x=1,.x=2};}", false},
		{"unknown-initializer", "module main;struct A{i32 x;}fn A f(){return A{.z=1};}", false},
		{"unknown-member", "module main;struct A{i32 x;}fn i32 f(A a){return a.z;}", false},
		{"immutable-parameter", "module main;struct A{i32 x;}fn void f(A a){a.x=2;}", false},
		{"immutable-let", "module main;struct A{i32 x;}fn void f(){let a=A{.x=1};a.x=2;}", false},
		{"temporary-place", "module main;struct A{i32 x;}fn A make(){return A{.x=1};}fn void f(){make().x=2;}", false},
		{"duplicate-field", "module main;struct A{i32 x;i32 x;}", false},
		{"direct-cycle", "module main;struct A{A x;}", true},
		{"indirect-cycle", "module main;struct A{B x;}struct B{A x;}", true},
		{"unknown-type", "module main;struct A{Missing x;}", false},
		{"bare-type-value", "module main;struct A{i32 x;}fn A f(){return A;}", true},
		{"aggregate-operator", "module main;struct A{i32 x;}fn bool f(A a,A b){return a==b;}", false},
		{"literal-range", "module main;struct A{u8 x;}fn A f(){return A{.x=256};}", false},
	}
	fixtures := []typeFixture{}
	for _, c := range cases {
		tree := syntax.Parse(source.NewFile("main.fy", []byte(c.input)), source.DefaultLimits())
		if !tree.ErrorFree() {
			return fmt.Errorf("nominal syntax %s: %+v", c.name, tree.Diagnostics)
		}
		module, err := ast.Lower(tree)
		if err != nil {
			return err
		}
		checked, checkErr := compiler.Check([]semantic.File{{Path: "main.fy", Module: module}})
		result := checked.Semantic
		valid := len(result.Diagnostics) == 0 && len(result.Unsupported) == 0
		if valid != c.valid {
			return fmt.Errorf("nominal reference %s expected valid=%t: %+v %+v error=%v", c.name, c.valid, result.Diagnostics, result.Unsupported, checkErr)
		}
		outcome := "error"
		if valid {
			outcome = "pass"
		}
		if c.name == "direct-cycle" || c.name == "indirect-cycle" {
			outcome = "candidate-layout-rejection"
		}
		if c.name == "bare-type-value" {
			outcome = "candidate-value-rejection"
		}
		message := ""
		if checkErr != nil {
			message = checkErr.Error()
		}
		fixtures = append(fixtures, typeFixture{Message: message, Name: c.name, InputHex: hex.EncodeToString([]byte(c.input)), Outcome: outcome, Diagnostics: result.Diagnostics, Unsupported: result.Unsupported})
	}
	data, err := json.MarshalIndent(struct {
		Schema   string        `json:"schema"`
		Fixtures []typeFixture `json:"fixtures"`
	}{"fairyc.nominal-types-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
