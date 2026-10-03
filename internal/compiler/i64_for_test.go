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

func TestI64ForArtifactExecution(t *testing.T) {
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "i64_for", "src", "main.fy"))
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
		if err != nil || !strings.Contains(unit.Source, "const int64_t *data;") {
			t.Fatalf("emit: %v", err)
		}
	}
}

func TestI64ForCheckedSum(t *testing.T) {
	for _, values := range []string{"9223372036854775807,1", "-9223372036854775808,-1"} {
		source := "module main; fn i32 main() { let i64[2] data = [" + values + "]; let ref i64[] view = ref data; var i64 total = 0; for (let i64 value in view) { total += value; } return 0; }"
		checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
		if err != nil || !checked.Verified {
			t.Fatalf("check: %v %+v", err, checked.Semantic)
		}
		outcome, err := interpreter.Execute(checked.MIR, "main::main")
		if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != "integer-overflow" {
			t.Fatalf("overflow lost: %v %+v", err, outcome)
		}
	}
}

func TestI64ForCleanup(t *testing.T) {
	// Reuse the observer's independent exact cleanup-order oracle, replacing only
	// the iterable type and item-to-observer mapping. No narrowing cast is used.
	for _, item := range []struct {
		name, source, expected string
		drops                  int
	}{
		{"exits", byteForCleanupSource, "aAbBaAbB", 4},
		{"propagation", byteForResultCleanupSource, "aAbB", 2},
	} {
		t.Run(item.name, func(t *testing.T) {
			source := strings.NewReplacer("ref u8[]", "ref i64[]", "for (let u8 byte in view)", "for (let i64 number in view)", "step(u8 byte)", "step(i32 byte)", "byte as i32", "byte").Replace(item.source)
			source = strings.Replace(source, `let ref i64[] view = b"AB"[..];`, "let i64[2] data = [65,66]; let ref i64[] view = ref data;", 1)
			source = strings.ReplaceAll(source, "for (let i64 number in view) {", "for (let i64 number in view) { assert(number == 65 || number == 66); let i32 byte = if (number == 65) { 65 } else { 66 };")
			runByteForCleanupProof(t, source, item.expected, item.drops)
		})
	}
}
