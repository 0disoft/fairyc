package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsRegionAllocationOperandTrap(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC; native Linux remains batched")
	}
	for _, f := range []struct{ name, expression string }{
		{"place", "r.place({assert(false);7})"},
		{"zeroed-bytes", "r.alloc_zeroed_bytes({assert(false);0})"},
		{"copy-bytes", "r.copy_bytes({assert(false);b\"ab\"})"},
	} {
		t.Run(f.name, func(t *testing.T) {
			root := writeWorkspaceSessionFixture(t, "module main;import std::alloc as alloc;fn i32 main(){var alloc::Arena arena=switch(alloc::create()){case .ok(a):{a}case .err(_):{return 1;}};region r in ref mut arena{discard "+f.expression+";}return 0;}")
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					var diagnostics strings.Builder
					compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
					if code != ExitSuccess {
						t.Fatalf("trapping allocation compile: %d %s", code, &diagnostics)
					}
					out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, strings.NewReader(""))
					if err != nil || out.Stdout != "" || out.Stderr != "" || len(out.TrapControl) == 0 {
						t.Fatalf("trapping allocation execution: %+v %v", out, err)
					}
					category, ok := cdriver.NativeTrapCategory(out.TrapControl[0])
					if !ok || category != "assertion-failed" || out.ExitCode != cdriver.NativeTrapExitBase+int32(out.TrapControl[0]) {
						t.Fatalf("trapping allocation receipt/status: %+v", out)
					}
				})
			}
		})
	}
}
