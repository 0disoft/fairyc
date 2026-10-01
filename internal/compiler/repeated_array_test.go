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

func TestRepeatedArrayArtifactExecution(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "repeated_array", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []struct{ name, before, after, trap string }{
		{"normal", "", "", ""},
		{"initializer trap", "var i32 divisor = 2;", "var i32 divisor = 0;", "division-by-zero"},
		{"index bounds", "assert(values[3] == 1);", "assert(values[4] == 1);", "bounds"},
	} {
		t.Run(item.name, func(t *testing.T) {
			text := string(data)
			if item.before != "" {
				text = strings.Replace(text, item.before, item.after, 1)
			}
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", text)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			encoded, err := mircodec.Encode(checked.MIR, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := mircodec.Decode(encoded, 1<<20)
			if err != nil {
				t.Fatal(err)
			}
			outcome, err := interpreter.Execute(decoded, "main::main")
			if err != nil || item.trap == "" && (outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0) || item.trap != "" && (outcome.Kind != interpreter.OutcomeTrap || outcome.TrapCategory != item.trap) {
				t.Fatalf("execution: %v %+v", err, outcome)
			}
			if _, err := cbackend.Emit(decoded, "main::main", cbackend.Stage0WindowsMSVCProfile()); err != nil {
				t.Fatal(err)
			}
		})
	}
}
