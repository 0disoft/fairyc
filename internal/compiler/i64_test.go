package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"os"
	"path/filepath"
	"testing"
)

func TestI64InterpreterArtifactAndValueContainers(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "i64.fy"))
	if err != nil {
		t.Fatal(err)
	}
	program, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", string(data))})
	if err != nil || !program.Verified {
		t.Fatalf("compile: %v %+v %+v", err, program.Semantic.Diagnostics, program.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(program.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := interpreter.Execute(decoded, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
		t.Fatalf("execute: %v %+v", err, outcome)
	}
}

func TestI64InterpreterTrapBoundaries(t *testing.T) {
	for _, tc := range []struct{ expression, trap string }{
		{"hi + 1", "integer-overflow"}, {"lo - 1", "integer-overflow"},
		{"lo * -1", "integer-overflow"}, {"hi * 2", "integer-overflow"},
		{"-lo", "integer-overflow"}, {"lo / -1", "signed-division-overflow"},
		{"lo % -1", "signed-division-overflow"}, {"hi / 0", "division-by-zero"},
		{"hi % 0", "division-by-zero"}, {"hi << 1", "integer-overflow"},
		{"hi << -1", "invalid-shift"}, {"lo >> 64", "invalid-shift"},
	} {
		t.Run(tc.expression, func(t *testing.T) {
			text := "module main; fn i32 main() { let i64 lo = -9223372036854775808; let i64 hi = 9223372036854775807; let i64 v = " + tc.expression + "; assert(v == 0); return 0; }"
			result, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !result.Verified {
				t.Fatalf("compile: %v %+v %+v", err, result.Semantic.Diagnostics, result.Semantic.Unsupported)
			}
			outcome, err := interpreter.Execute(result.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != tc.trap {
				t.Fatalf("want %s: %+v %v", tc.trap, outcome, err)
			}
		})
	}
}
