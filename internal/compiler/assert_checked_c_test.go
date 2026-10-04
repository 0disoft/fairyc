package compiler_test

import (
	"bytes"
	"encoding/json"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAssertCheckedTrapCEmission(t *testing.T) {
	data, err := os.ReadFile("testdata/assert_checked_trap.fy")
	if err != nil {
		t.Fatal(err)
	}
	assertCheckedSource := string(data)
	cases := []struct {
		name, source, trap string
		nonReturning       bool
	}{
		{"shared-call", assertCheckedSource, "integer-overflow", true},
		{"shared-field", `module main;struct Span{i64 start;}struct Token{Span position;}fn void edit(ref mut Token t){let ref i64 before=ref t.position.start;assert(*before==9223372036854775807);t.position.start+=1;}fn i32 main(){var Token t=Token{.position=Span{.start=9223372036854775807}};edit(ref mut t);return 0;}`, "integer-overflow", true},
		{"scalar-parameter", `module main;fn void edit(i64 n){assert(n==9223372036854775807);var i64 value=n;value+=1;}fn i32 main(){edit(9223372036854775807);return 0;}`, "integer-overflow", true},
		{"assertion-first", strings.Replace(assertCheckedSource, ".start=9223372036854775807}}", ".start=41}}", 1), "assertion-failed", true},
		{"field-write", `module main;struct Token{i64 value;}fn void edit(ref mut Token t){assert(t.value==9223372036854775807);t.value=40;t.value+=2;assert(t.value==42);}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, "", false},
		{"scalar-rebind", `module main;fn void edit(i64 input){var i64 n=input;assert(n==9223372036854775807);n=40;let ref mut i64 slot=ref mut n;*slot=*slot+2;assert(n==42);}fn i32 main(){edit(9223372036854775807);return 0;}`, "", false},
		{"unknown-call", `module main;struct Token{i64 value;}fn void reset(ref mut Token t){t.value=40;}fn void edit(ref mut Token t){assert(t.value==9223372036854775807);reset(t);t.value+=2;assert(t.value==42);}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, "", false},
		{"snapshot-write", `module main;struct Token{i64 value;}fn void edit(ref mut Token t){let Token snapshot=*t;assert(snapshot.value==9223372036854775807);t.value=40;t.value+=2;assert(t.value==42);}fn i32 main(){var Token t=Token{.value=9223372036854775807};edit(ref mut t);return 0;}`, "", false},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", item.source)})
			if err != nil || !result.Verified {
				t.Fatalf("check: %v %+v", err, result.Semantic)
			}
			outcome, err := interpreter.Execute(result.MIR, "main::main")
			if err != nil {
				t.Fatal(err)
			}
			if item.trap != "" {
				if outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != item.trap {
					t.Fatalf("execute: %+v", outcome)
				}
			} else if outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
				t.Fatalf("execute: %+v", outcome)
			}
			before, _ := json.Marshal(result.MIR)
			for _, profile := range []cbackend.Profile{cbackend.Stage0WindowsMSVCProfile(), cbackend.Stage0Profile()} {
				unit, err := cbackend.Emit(result.MIR, "main::main", profile)
				if err != nil {
					t.Fatal(err)
				}
				if item.nonReturning {
					if !strings.Contains(unit.Source, "static _Noreturn") || strings.Contains(unit.Source, "return INT32_C(0);") || strings.Contains(unit.Source, "fy_i64_add_overflow(") {
						t.Fatal("impossible continuation/helper retained")
					}
				} else if !strings.Contains(unit.Source, "return INT32_C(0);") {
					t.Fatal("normal continuation pruned")
				}
				if item.name == "shared-call" {
					dir := filepath.Join("..", "..", ".cache", "assert-checked")
					if err := os.MkdirAll(dir, 0700); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(filepath.Join(dir, string(profile.Target)+".c"), []byte(unit.Source), 0600); err != nil {
						t.Fatal(err)
					}
				}
			}
			after, _ := json.Marshal(result.MIR)
			if !bytes.Equal(before, after) {
				t.Fatal("reachability changed verified MIR")
			}
		})
	}
}
