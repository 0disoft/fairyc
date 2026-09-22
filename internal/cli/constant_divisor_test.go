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

func TestConstantDivisorNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows MSVC native evidence")
	}
	for _, tc := range []struct{ name, source, trap string }{
		{"unsigned-boundaries", `fn i32 main() {
    let u8 a = 255; let u32 b = 4294967295; let usize c = 18446744073709551615;
    assert(a / 1 == 255 && a % 255 == 0 && a / 128 == 1);
    assert(b / 1 == 4294967295 && b % 4294967295 == 0 && b % 128 == 127);
    assert(c / 1 == 18446744073709551615 && c % 18446744073709551615 == 0 && c % 128 == 127);
    return 0;
  }`, ""},
		{"unsigned-zero", `fn i32 main() { let u32 n = 42; let u32 a = n / 0; return if (a == 0) { 0 } else { 1 }; }`, "division-by-zero"},
		{"unsigned-zero-remainder", `fn i32 main() { let u8 n = 42; let u8 a = n % 0; return if (a == 0) { 0 } else { 1 }; }`, "division-by-zero"},
		{"dynamic-zero", `fn u32 f(u32 d) { let u32 n = 42; return n % d; } fn i32 main() { let u32 a = f(0); return if (a == 0) { 0 } else { 1 }; }`, "division-by-zero"},
		{"signed-overflow", `fn i32 main() { return -2147483648 / -1; }`, "signed-division-overflow"},
		{"signed-remainder-overflow", `fn i32 main() { return -2147483648 % -1; }`, "signed-division-overflow"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.divisor\"\nname = \"divisor\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", "module main;\n"+tc.source)
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
				if tc.trap != "" {
					if !trapped || category != tc.trap {
						t.Fatalf("%s: want %s, got %+v", profile.Name, tc.trap, outcome)
					}
				} else if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("%s: %+v", profile.Name, outcome)
				}
			}
		})
	}
}
