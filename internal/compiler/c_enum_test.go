package compiler_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/mir"
	"github.com/0disoft/fairyc/internal/mircodec"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestCEnumMetadataRoundTrip(t *testing.T) {
	checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
@repr(c::ushort) enum State { Ready; Busy; Done; }
fn i32 main() { let State s=.Done; return switch(s) { case .Ready: {0} case .Busy: {1} case .Done: {42} }; }
`)})
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v diagnostics=%+v unsupported=%+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	encoded, err := mircodec.Encode(checked.MIR, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	program, err := mircodec.Decode(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if program.EnumTypes[0].CBase != "c.ushort" {
		t.Fatal("C enum base lost in MIR codec")
	}
	result, err := interpreter.Execute(program, "main::main")
	if err != nil || result.Kind != interpreter.OutcomeReturn || result.Value != 42 {
		t.Fatalf("execute: %+v %v", result, err)
	}
	program.EnumTypes[0].CBase = "c.float"
	if len(mir.Verify(program)) == 0 {
		t.Fatal("MIR accepted floating enum base")
	}
	program.EnumTypes[0].CBase = "c.uint"
	program.EnumTypes[0].Variants[0].Fields = []mir.EnumField{{Name: "code", Type: hir.TypeI32}}
	if len(mir.Verify(program)) == 0 {
		t.Fatal("MIR accepted C enum payload")
	}
}
