package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/semantic"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/syntax"
)

func writeNominalHIR(path string) error {
	// Reuse the semantic corpus, without consuming any candidate output.
	seed := path + ".semantic.json"
	if err := writeNominal(seed); err != nil {
		return err
	}
	data, err := os.ReadFile(seed)
	if err != nil {
		return err
	}
	var input struct {
		Fixtures []typeFixture `json:"fixtures"`
	}
	if err = json.Unmarshal(data, &input); err != nil {
		return err
	}
	for _, c := range []struct{ name, source string }{
		{"public-struct", "module main;pub struct A{i32 x;}fn A f(){return A{.x=1};}"},
		{"initializer-order", "module main;struct Pair{i32 x;i32 y;}fn Pair f(){var i32 a=1;return Pair{.y=a,.x={a=3;a}};}"},
		{"initializer-return", "module main;struct Pair{i32 x;i32 y;}fn i32 f(){let p=Pair{.y=1,.x={return 9;2}};return p.x;}"},
		{"nested-assignment", "module main;struct A{i32 x;}struct B{A a;}fn B f(){var b=B{.a=A{.x=1}};b.a=A{.x=8};return b;}"},
		{"distinct-identities", "module main;struct A{i32 x;}struct B{i32 x;}fn i32 f(){let a=A{.x=1};let b=B{.x=2};return a.x+b.x;}"},
	} {
		input.Fixtures = append(input.Fixtures, typeFixture{Name: c.name, InputHex: hex.EncodeToString([]byte(c.source)), Outcome: "pass"})
	}
	fixtures := []hirOracleFixture{}
	for _, f := range input.Fixtures {
		if f.Outcome != "pass" {
			continue
		}
		raw, err := hex.DecodeString(f.InputHex)
		if err != nil {
			return err
		}
		tree := syntax.Parse(source.NewFile("main.fy", raw), source.DefaultLimits())
		module, err := ast.Lower(tree)
		if err != nil {
			return err
		}
		files := []semantic.File{{Path: "main.fy", Module: module}}
		checked := semantic.Analyze(files)
		if len(checked.Diagnostics) != 0 || len(checked.Unsupported) != 0 {
			return fmt.Errorf("nominal HIR reference %s failed", f.Name)
		}
		observed, symbols, err := observeOracleHIR(checked.HIR, files)
		if err != nil {
			return fmt.Errorf("nominal HIR reference %s: %w", f.Name, err)
		}
		fixtures = append(fixtures, hirOracleFixture{Name: f.Name, InputHex: f.InputHex, Outcome: "pass", HIR: observed, Symbols: symbols})
	}
	data, err = json.MarshalIndent(struct {
		Schema   string             `json:"schema"`
		Fixtures []hirOracleFixture `json:"fixtures"`
	}{"fairyc.nominal-hir-oracle/v1", fixtures}, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0644)
}
