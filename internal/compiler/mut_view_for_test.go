package compiler_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestMutableViewForArtifactExecution(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "mut_view_for", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(source))})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(program, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
	for _, profile := range []cbackend.Profile{cbackend.Stage0Profile(), cbackend.Stage0WindowsMSVCProfile()} {
		unit, err := cbackend.Emit(program, "main::main", profile)
		if err != nil || !strings.Contains(unit.Source, "fy_mut_i64_slice") || !strings.Contains(unit.Source, "fy_mut_byte_slice") {
			t.Fatalf("emit: %v", err)
		}
	}
}

func TestMutableViewForCheckedArithmetic(t *testing.T) {
	for name, source := range map[string]string{
		"u8-overflow":   "var u8[1] data = [255]; let ref mut u8[] view = ref mut data; for (let u8 value in view) { view[0] = value + 1; }",
		"u8-underflow":  "var u8[1] data = [0]; let ref mut u8[] view = ref mut data; for (let u8 value in view) { view[0] = value - 1; }",
		"i64-overflow":  "var i64[1] data = [9223372036854775807]; let ref mut i64[] view = ref mut data; for (let i64 value in view) { view[0] = value + 1; }",
		"i64-underflow": "var i64[1] data = [-9223372036854775808]; let ref mut i64[] view = ref mut data; for (let i64 value in view) { view[0] = value - 1; }",
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; fn i32 main() { "+source+" return 0; }")})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked.Semantic)
			}
			outcome, err := interpreter.Execute(checked.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "integer-overflow" {
				t.Fatalf("trap: %v %+v", err, outcome)
			}
		})
	}
}

func TestMutableViewForCleanup(t *testing.T) {
	for _, element := range []string{"u8", "i64"} {
		for _, item := range []struct {
			name, source, expected string
			drops                  int
		}{
			{"exits", byteForCleanupSource, "aAbBaAbB", 4},
			{"propagation", byteForResultCleanupSource, "aAbB", 2},
		} {
			t.Run(element+"/"+item.name, func(t *testing.T) {
				source := strings.ReplaceAll(item.source, "ref u8[]", "ref mut "+element+"[]")
				source = strings.Replace(source, `let ref mut `+element+`[] view = b"AB"[..];`, "var "+element+"[2] data = [65,66]; let ref mut "+element+"[] view = ref mut data;", 1)
				if element == "i64" {
					source = strings.NewReplacer("for (let u8 byte in view)", "for (let i64 number in view)", "step(u8 byte)", "step(i32 byte)", "byte as i32", "byte").Replace(source)
					source = strings.ReplaceAll(source, "for (let i64 number in view) {", "for (let i64 number in view) { assert(number == 65 || number == 66); let i32 byte = if (number == 65) { 65 } else { 66 };")
				}
				runByteForCleanupProof(t, source, item.expected, item.drops)
			})
		}
	}
}
