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

func TestByteForCompilerPublishesVerifiedMIR(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
fn i32 main() {
    let ref u8[] view = b"abc"[..];
    var i32 total = 0;
    for (let u8 byte in view) { total += byte as i32; }
    return total;
}`)
	result, err := compiler.Check([]semantic.File{file})
	if err != nil || !result.Verified || len(result.Semantic.Diagnostics) != 0 || len(result.Semantic.Unsupported) != 0 {
		t.Fatalf("compile byte loop: result = %+v, error = %v", result.Semantic, err)
	}
	outcome, err := interpreter.Execute(result.MIR, "main::main")
	if err != nil || outcome.Kind != interpreter.OutcomeReturn || outcome.Value != 294 {
		t.Fatalf("execute byte loop: outcome = %+v, error = %v", outcome, err)
	}
}

func TestByteForCompilerRejectsOwnerMutationBeforeMIR(t *testing.T) {
	file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", `module main;
fn i32 main() {
    var u8[2] data = [7, 8];
    let ref u8[] view = ref data;
    for (let u8 byte in view) { data[0] = byte; }
    return 0;
}`)
	result, err := compiler.Check([]semantic.File{file})
	if err != nil || result.Verified || len(result.Semantic.Diagnostics) != 1 || result.Semantic.Diagnostics[0].Code != "E2101" || len(result.MIR.Functions) != 0 {
		t.Fatalf("owner mutation reached MIR: result = %+v, error = %v", result.Semantic, err)
	}
}

func TestByteForCompilerWindowsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("native proof requires Windows amd64")
	}
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	scalarRuntime, err := cruntime.ScalarSource(192)
	if err != nil {
		t.Fatal(err)
	}
	for _, sample := range []struct {
		name string
		body string
		exit int32
	}{
		{name: "empty", body: `let ref u8[] view = b"*"[0..0]; for (let u8 byte in view) { return byte as i32; } return 0;`, exit: 0},
		{name: "continue-break", body: `let ref u8[] view = b"AB*"[..]; var i32 value = 0; for (let u8 byte in view) { if (byte != 42) { continue; } value += byte as i32; break; } return value;`, exit: 42},
	} {
		file := parseCompilerFile(t, hir.DefaultPackage, "main.fy", "module main; fn i32 main() { "+sample.body+" }")
		result, err := compiler.Check([]semantic.File{file})
		if err != nil || !result.Verified {
			t.Fatalf("compile %s: %v %+v", sample.name, err, result.Semantic)
		}
		unit, err := cbackend.Emit(result.MIR, "main::main", cbackend.Stage0WindowsMSVCProfile())
		if err != nil {
			t.Fatal(err)
		}
		for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
			t.Run(sample.name+"/"+profile.Name, func(t *testing.T) {
				artifact, err := cdriver.BuildWindowsHost(toolchain, profile, unit.Source, scalarRuntime)
				if err != nil {
					t.Fatal(err)
				}
				outcome, err := cdriver.ExecuteWindowsHost(artifact)
				if err != nil || outcome.ExitCode != sample.exit || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("native execution: %+v %v", outcome, err)
				}
			})
		}
	}
}
