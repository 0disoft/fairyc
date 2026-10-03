package compiler_test

import (
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
	"testing"
)

func TestIntegerViewForInterpreter(t *testing.T) {
	for _, element := range []string{"i32", "u32", "usize"} {
		t.Run(element, func(t *testing.T) {
			source := "module main; fn i32 main() { var " + element + "[3] data = [1,2,3]; let ref " + element + "[] shared = ref data; var " + element + " sum = 0; for(let " + element + " item in shared) { sum += item; } assert(sum == 6); var usize visits = 0; for(let " + element + " item in shared) { visits += 1; if(item == 1) {continue;} break; } assert(visits == 2); let ref mut " + element + "[] view = ref mut data; var usize index = 0; for(let " + element + " item in view) { if(index == 0) { view[1] = 9; } if(index == 1) {assert(item == 9);} index += 1; } assert(data[1] == 9 && index == 3); return 0; }"
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
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
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
				t.Fatalf("execute: %v %+v", err, outcome)
			}
		})
	}
}
