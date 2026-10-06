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

func TestArrayInitializersFreezeValuesAndStopOnReturn(t *testing.T) {
	for name, source := range map[string]string{
		"scalar": `module main;fn i32 main(){var i32 n=1;let a=[n,{n=3;n},n];assert(a[0]==1 && a[1]==3 && a[2]==3);return 0;}`,
		"wide":   `module main;fn i32 main(){var i64 n=4000000000;let i64[2] a=[n,{n+=1;n}];assert(a[0]==4000000000 && a[1]==4000000001);return 0;}`,
		"return": `module main;fn i32 main(){let i32[2] a=[1,{return 0;2}];return 9;}`,
	} {
		t.Run(name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v", err, checked.Semantic)
			}
			outcome, err := interpreter.Execute(checked.MIR, "main::main")
			if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 0 {
				t.Fatalf("execute: %v %+v", err, outcome)
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
				native, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || native.ExitCode != 0 || native.TrapControl != "" || native.Stdout != "" || native.Stderr != "" {
					t.Fatalf("native %s: %v %+v", profile.Name, err, native)
				}
			}
		})
	}
}
