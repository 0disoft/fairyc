package compiler_test

import (
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/compiler"
	"github.com/0disoft/fairyc/internal/cruntime"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/semantic"
)

func TestPrivateNominalBindingPackageMove(t *testing.T) {
	client := parseCompilerFile(t, 1, "main.fy", `module main; import bridge as b;
resource Token { i32 value; cleanup {} }
fn i32 main() {
    let Token owned = Token { .value = 42 };
    let Token returned = b::pass(owned);
    return returned.value;
}`)
	provider := parseCompilerFile(t, 2, "bridge.fy", "module bridge; pub fn i32 pass(i32 value) { return value; }")
	fn := &provider.Module.Functions[0]
	fn.Parameters[0].Type.Kind, fn.Parameters[0].Type.Path = ast.TypeKindBoundNominal, "argument"
	fn.ResultType.Kind, fn.ResultType.Path = ast.TypeKindBoundNominal, "argument"
	input := semantic.PackageInput{Files: []semantic.File{client, provider}, Imports: []semantic.PackageImport{{FromPackage: 1, ImportPath: "bridge", ToPackage: 2}}}
	analysis, err := semantic.AnalyzeBoundNominalPackages(input, []semantic.NominalTypeBinding{{Key: "argument", Package: 1, Module: "main", Name: "Token"}})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.LowerAnalyzed(compiler.Analysis{Semantic: analysis})
	if err != nil || !result.Verified {
		t.Fatalf("bound nominal lowering: %v %+v", err, result.Semantic)
	}
	entry := hir.QualifiedModulePath(1, "main") + "::main"
	outcome, err := interpreter.Execute(result.MIR, entry)
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 42 || len(outcome.DropTrace) != 1 {
		t.Fatalf("bound nominal ownership: %+v %v", outcome, err)
	}
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native nominal binding proof requires Windows amd64")
	}
	unit, err := cbackend.Emit(result.MIR, entry, cbackend.Stage0WindowsMSVCProfile())
	if err != nil {
		t.Fatal(err)
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalar, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalar)
			if err != nil {
				t.Fatal(err)
			}
			native, err := cdriver.ExecuteWindowsHost(artifact)
			if err != nil || native.ExitCode != 42 || native.Stdout != "" || native.Stderr != "" {
				t.Fatalf("bound nominal native: %+v %v", native, err)
			}
		})
	}
}
