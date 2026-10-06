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
		{"field", `module main;struct E{}struct B{E e;i32 n;}fn i32 accept(E e){return 6;}fn i32 main(){var b=B{.e=E{},.n=1};b.e=E{};return accept(b.e);}`, 6},
		{"field-rhs-effects", `module main;struct E{}struct B{E e;i32 n;}fn i32 main(){var b=B{.e=E{},.n=1};b.e={b.n=9;E{}};return b.n;}`, 9},
		{"field-rhs-return", `module main;struct E{}struct B{E e;}fn i32 main(){var b=B{.e=E{}};b.e={return 7;E{}};return 0;}`, 7},
		{"nested-field", `module main;struct E{}struct B{E e;i32 n;}struct C{B b;}fn i32 main(){var c=C{.b=B{.e=E{},.n=1}};c.b.e={c.b.n=8;E{}};return c.b.n;}`, 8},
		{"field-root-replacement", `module main;struct E{}struct B{E e;i32 n;}fn i32 main(){var b=B{.e=E{},.n=1};let copy=b;b.e={b=B{.e=E{},.n=9};E{}};return b.n+copy.n;}`, 10},
		{"field-reference", `module main;struct E{}struct B{E e;i32 n;}fn void edit(ref mut B b){b.e={b.n=11;E{}};}fn i32 main(){var b=B{.e=E{},.n=1};edit(ref mut b);return b.n;}`, 11},
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
		`module main;struct E{}struct B{E e;}fn i32 main(){let b=B{.e=E{}};b.e=E{};return 0;}`,
		`module main;struct E{}struct B{E e;}fn void bad(ref B b){b.e=E{};}fn i32 main(){return 0;}`,
		`module main;struct E{}struct Other{}struct B{E e;}fn i32 main(){var b=B{.e=E{}};b.e=Other{};return 0;}`,
		`module main;struct E{}struct B{E e;}fn i32 main(){var b=B{.e=E{}};b.e+=E{};return 0;}`,
	} {
		checked, err := compiler.Check([]semantic.File{parseCompilerFile(t, hir.DefaultPackage, "main.fy", source)})
		if err == nil && checked.Verified {
			t.Fatalf("unexpectedly admitted %s", source)
		}
	}
}
