package cli

import (
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsByteViewRegionReuseAfterLoopExits(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	root := t.TempDir()
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.byte-view-region\"\nname = \"byte-view-region\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", `module main;
import std::alloc as alloc;
fn i32 run(alloc::Arena owned) {
    var alloc::Arena arena = owned;
    let ref u8[] view = b"AB"[..];
    for (let u8 byte in view) {
        region scratch in ref mut arena {
            assert(switch (scratch.place(byte as i32)) {
                case .ok(stored): { *stored == byte as i32 }
                case .err(_): { false }
            });
            if (byte == 65) { continue; }
            break;
        }
    }
    region final in ref mut arena {
        assert(switch (final.place(42)) {
            case .ok(stored): { *stored == 42 }
            case .err(_): { false }
        });
    }
    return 42;
}
fn i32 main() {
    return switch (alloc::create()) {
        case .ok(arena): { run(arena) }
        case .err(_): { 1 }
    };
}`)
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			var diagnostics strings.Builder
			compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &diagnostics)
			if code != ExitSuccess {
				t.Fatalf("compile: %d %s", code, diagnostics.String())
			}
			outcome, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, strings.NewReader(""))
			if err != nil || outcome.ExitCode != 42 || outcome.Stdout != "" || outcome.Stderr != "" || outcome.TrapControl != "" {
				t.Fatalf("native Region reuse: %+v %v", outcome, err)
			}
		})
	}
}
