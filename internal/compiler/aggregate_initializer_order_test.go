package compiler_test

import (
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
	"runtime"
	"testing"
)

func TestAggregateInitializerOrderAndCapture(t *testing.T) {
	cases := []struct {
		name, source string
		want         int32
	}{
		{"reordered", `module main;struct Pair{i64 first;bool second;}fn i32 main(){let p=Pair{.second=true,.first=4000000000};return if(p.first==4000000000&&p.second){42}else{1};}`, 42},
		{"written-order", `module main;struct Pair{i32 first;i32 second;}fn i32 main(){var i32 x=3;let p=Pair{.second={x=7;7},.first=x};return p.first*10+p.second;}`, 77},
		{"scalar-capture", `module main;struct Pair{i32 first;i32 second;}fn i32 main(){var i32 x=3;let p=Pair{.first=x,.second={x=7;7}};return p.first*10+p.second;}`, 37},
		{"struct-capture", `module main;struct Inner{i32 x;}struct Outer{Inner first;i32 second;}fn i32 main(){var inner=Inner{.x=3};let p=Outer{.first=inner,.second={inner.x=7;7}};return p.first.x*10+p.second;}`, 37},
		{"resource-order", `module main;resource Pair{i32 first;i32 second;cleanup{}}fn i32 main(){var i32 x=3;let p=Pair{.second={x=7;7},.first=x};return p.first*10+p.second;}`, 77},
		{"resource-scalar-capture", `module main;resource Pair{i32 first;i32 second;cleanup{}}fn i32 main(){var i32 x=3;let p=Pair{.first=x,.second={x=7;7}};return p.first*10+p.second;}`, 37},
		{"initializer-return", `module main;struct Pair{i32 first;i32 second;}fn i32 main(){let p=Pair{.first={return 5;1},.second=2};return p.first;}`, 5},
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
