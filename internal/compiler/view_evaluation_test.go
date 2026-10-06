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

func TestViewEvaluationFreezesDescriptorsAndSliceStart(t *testing.T) {
	for name, source := range map[string]string{
		"index-owner":  `module main;fn i32 main(){var b=b"A";let u8 x=b[{b=b"";0}];assert(x==65 && b.is_empty);return 0;}`,
		"index-data":   `module main;fn i32 main(){var b=b"A";let u8 x=b[{b=b"B";0}];assert(x==65 && b[0]==66);return 0;}`,
		"index-return": `module main;fn i32 main(){let b=b"A";discard b[{return 0;0}];return 9;}`,
		"slice-owner":  `module main;fn i32 main(){var b=b"ABC";let s=b[{b=b"";0}..];assert(s.len==3 && s[2]==67 && b.is_empty);return 0;}`,
		"slice-start":  `module main;fn i32 main(){var usize n=0;let s=b"ABC"[n..{n=2;3}];assert(s.len==3 && s[0]==65 && n==2);return 0;}`,
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
				observed, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || observed.ExitCode != 0 || observed.TrapControl != "" || observed.Stdout != "" || observed.Stderr != "" {
					t.Fatalf("native %s: %v %+v", profile.Name, err, observed)
				}
			}
		})
	}
}
