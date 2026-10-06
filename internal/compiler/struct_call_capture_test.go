package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestStructCallArgumentsCaptureBeforeLaterMutation(t *testing.T) {
	cases := []struct {
		name, source string
		want         int32
	}{
		{"struct", `module main;struct A{i32 x;}fn i32 sum(A a,A b){return a.x*10+b.x;}fn i32 main(){var a=A{.x=1};return sum(a,{a=A{.x=3};a});}`, 13},
		{"block-tail", `module main;struct A{i32 x;}fn i32 sum(A a,i32 b){return a.x*10+b;}fn i32 main(){var a=A{.x=1};return sum({a},{a.x=3;3});}`, 13},
		{"nested", `module main;struct A{i32 x;}struct B{A a;i32 z;}fn i32 sum(B b,i32 x){return b.a.x*100+b.z*10+x;}fn i32 main(){var b=B{.a=A{.x=1},.z=2};return sum(b,{b.a.x=3;b.z=4;5});}`, 125},
		{"early-return", `module main;struct A{i32 x;}fn i32 sum(A a,i32 b){return a.x+b;}fn i32 main(){var a=A{.x=1};return sum(a,{return 7;3});}`, 7},
		{"scalar", `module main;fn i32 sum(i32 a,i32 b){return a*10+b;}fn i32 main(){var i32 x=1;return sum(x,{x=3;x});}`, 13},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", c.source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked.Semantic)
			}
			got, err := interpreter.Execute(checked.MIR, "main::main")
			if err != nil || got.Kind != interpreter.OutcomeReturn || got.Value != c.want {
				t.Fatalf("execute: %v %+v, want %d", err, got, c.want)
			}
			if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
				return
			}
			unit, err := cbackend.Emit(checked.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
			if err != nil {
				t.Fatal(err)
			}
			tool, err := cdriver.ResolveWindowsHostMSVC()
			if err != nil {
				t.Fatal(err)
			}
			library, err := cruntime.WindowsRunSource(120)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				artifact, err := cdriver.BuildWindowsHost(tool, profile, unit.Source, library)
				if err != nil {
					t.Fatal(err)
				}
				artifact.TrapControl = true
				outcome, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || outcome.ExitCode != c.want || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
					t.Fatalf("native %s: %v %+v", profile.Name, err, outcome)
				}
			}
		})
	}
}
