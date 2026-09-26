package cli

import (
	"bytes"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestFixedTrapGroupNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC native trap evidence")
	}
	for _, tc := range []struct{ name, body, trap string }{
		{"first-bounds", `let ref u8[] data = b""; let u8 first = data[0]; let u8 second = data[1]; return if (first == second) { 0 } else { 1 };`, "bounds"},
		{"second-bounds", `let ref u8[] data = b"a"; let u8 first = data[0]; let u8 second = data[1]; return if (first == second) { 0 } else { 1 };`, "bounds"},
		{"first-assert", `let u8 first = 1; assert(first == 0); let u8 second = 2; assert(second == 0); return 0;`, "assertion-failed"},
		{"second-assert", `let u8 first = 1; assert(first == 1); let u8 second = 2; assert(second == 0); return 0;`, "assertion-failed"},
		{"separate-categories", `let ref u8[] data = b"a"; let u8 first = data[0]; assert(first == 0); return 0;`, "assertion-failed"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.trap-group\"\nname = \"trap-group\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", "module main;\nfn i32 main() { "+tc.body+" }\n")
			w, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := workspacecompiler.Check(w)
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				var diagnostics bytes.Buffer
				artifact, code := compileWindowsProgram(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, &diagnostics)
				if code != ExitSuccess {
					t.Fatal(diagnostics.String())
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil {
					t.Fatal(err)
				}
				category, trapped := windowsHostTrapCategory(outcome.ExitCode)
				if !trapped || category != tc.trap || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("%s: want %s, got %+v", profile.Name, tc.trap, outcome)
				}
			}
		})
	}
}
