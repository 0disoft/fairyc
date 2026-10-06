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

func TestEmptyStructConstructionAndCopies(t *testing.T) {
	cases := []struct {
		name, source string
		want         int32
	}{
		{"construct", `module main;struct E{}fn i32 main(){let e=E{};return 1;}`, 1},
		{"copy", `module main;struct E{}fn i32 accept(E e){return 2;}fn i32 main(){let e=E{};let copy=e;return accept(copy);}`, 2},
		{"return", `module main;struct E{}fn E make(){return E{};}fn E id(E e){return e;}fn i32 accept(E e){return 3;}fn i32 main(){return accept(id(make()));}`, 3},
		{"nested", `module main;struct E{}struct Box{E e;i32 n;}fn Box id(Box b){return b;}fn i32 main(){let b=id(Box{.e=E{},.n=4});return b.n;}`, 4},
		{"assign", `module main;struct E{}fn i32 accept(E e){return 5;}fn i32 main(){var e=E{};e=E{};return accept(e);}`, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", c.source)})
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v diagnostics=%+v", err, checked.Semantic.Diagnostics)
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
					t.Fatalf("native %s: %v %+v, want %d", profile.Name, err, outcome, c.want)
				}
			}
		})
	}
}

func TestEmptyStructDoesNotAdmitFieldsOrCABI(t *testing.T) {
	for _, source := range []string{
		`module main;struct E{}fn i32 main(){let e=E{.x=1};return 0;}`,
		`module main;@repr(C) struct E{}fn i32 main(){return 0;}`,
	} {
		checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
		if err == nil && checked.Verified {
			t.Fatalf("unexpectedly admitted %s", source)
		}
	}
}
