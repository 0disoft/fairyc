package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsResultPropagationOperandExit(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC; native Linux remains batched")
	}
	for _, f := range []struct {
		name, body string
		trap       bool
	}{
		{"trap", "let n={assert(false);read()}?;return .ok(n);", true},
		{"return", "let n={return .err(3);read()}?;return .ok(n);", false},
		{"result-payload", "return .ok(take(read()?,{assert(false);0}));", true},
	} {
		t.Run(f.name, func(t *testing.T) {
			source := "module main;fn Result<i32,i32> read(){return .ok(7);}fn i32 take(i32 a,i32 b){return a+b;}fn Result<i32,i32> exercise(){" + f.body + "}fn i32 main(){return switch(exercise()){case .ok(n):{n}case .err(e):{e}};}"
			root := writeWorkspaceSessionFixture(t, source)
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					var diagnostics strings.Builder
					compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
					if code != ExitSuccess {
						t.Fatalf("terminated propagation operand compile: %d %s", code, &diagnostics)
					}
					out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, strings.NewReader(""))
					if err != nil || out.Stdout != "" || out.Stderr != "" {
						t.Fatalf("terminated propagation operand execution: %+v %v", out, err)
					}
					if !f.trap {
						if out.ExitCode != 3 || len(out.TrapControl) != 0 {
							t.Fatalf("lost original return: %+v", out)
						}
						return
					}
					if len(out.TrapControl) == 0 {
						t.Fatalf("lost original trap: %+v", out)
					}
					category, ok := cdriver.NativeTrapCategory(out.TrapControl[0])
					if !ok || category != "assertion-failed" || out.ExitCode != cdriver.NativeTrapExitBase+int32(out.TrapControl[0]) {
						t.Fatalf("changed original trap: %+v", out)
					}
				})
			}
		})
	}
}
