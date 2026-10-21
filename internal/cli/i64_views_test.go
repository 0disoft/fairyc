package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func i64ViewNativeCases(t *testing.T) []struct{ name, source, trap string } {
	t.Helper()
	source, err := os.ReadFile(filepath.Join("..", "..", "examples", "i64_view", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	base := "module main; fn i32 main() { let i64[2] data = [4000000000,-3]; let ref i64[] view = ref data; "
	cases := []struct{ name, source, trap string }{
		{"example", string(source), ""},
		{"local-only", base + "assert(view[0] == 4000000000 && view[1] == -3); return 0; }", ""},
		{"ordered-bounds", base + "var usize trace = 0; let ref i64[] child = view[{ trace = trace * 10 + 1; 0 }..{ trace = trace * 10 + 2; 2 }]; assert(trace == 12 && child[0] == 4000000000); return 0; }", ""},
		{"index-end", base + "let i64 unused = view[2]; return 0; }", "bounds"},
		{"index-empty", base + "let i64 unused = view[2..][0]; return 0; }", "bounds"},
		{"slice-reversed", base + "let ref i64[] child = view[2..1]; return 0; }", "bounds"},
		{"slice-past-end", base + "let ref i64[] child = view[..3]; return 0; }", "bounds"},
	}
	return cases
}

func TestI64ViewWindowsNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows amd64 MSVC")
	}
	cases := i64ViewNativeCases(t)
	evidenceRoot := filepath.Join("..", "..", ".cache", "i64-view-native")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.i64-view\"\nname = \"i64-view\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", item.source)
			workspace, err := workspaceloader.Load(root)
			if err != nil {
				t.Fatal(err)
			}
			checked, err := workspacecompiler.Check(workspace)
			if err != nil || !checked.Verified {
				t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				var diagnostics bytes.Buffer
				artifact, code := compileWindowsProgramWithRunReceipt(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, true, &diagnostics)
				if code != ExitSuccess {
					t.Fatalf("%s: %s", profile.Name, diagnostics.String())
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				_, trap, classErr := classifyStableRunOutcome(outcome)
				if err != nil || classErr != nil || trap != item.trap || item.trap == "" && outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
					t.Fatalf("%s: %+v %v %v", profile.Name, outcome, err, classErr)
				}
				sum, sourceSum := sha256.Sum256(artifact.Bytes), sha256.Sum256([]byte(item.source))
				if err := os.WriteFile(filepath.Join(evidenceRoot, item.name+"-"+profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				evidence = append(evidence, map[string]any{"case": item.name, "profile": profile.Name, "sourceSHA256": hex.EncodeToString(sourceSum[:]), "exeSHA256": hex.EncodeToString(sum[:]), "exeBytes": len(artifact.Bytes), "stdout": outcome.Stdout, "stderr": outcome.Stderr, "exitCode": outcome.ExitCode, "trap": trap, "expectedTrap": item.trap, "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
			}
		})
	}
	if t.Failed() {
		return
	}
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.i64-view-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}

// Deferred on Windows; included in the user's later Linux batch so the actual
// stable build/run path, including trap classification, is exercised there.
func TestI64ViewLinuxNative(t *testing.T) {
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("Linux amd64 GCC/Clang; execution deferred to Linux batch")
	}
	for _, item := range i64ViewNativeCases(t) {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.i64-view\"\nname = \"i64-view\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", item.source)
			for _, release := range []bool{false, true} {
				args := []string{"run"}
				if release {
					args = append(args, "--release", "--locked")
				}
				code, out, diag := runForTest(args, root)
				if item.trap != "" {
					if code != ExitProgramTrap || out != "" || !matchesNativeTrapStderr(diag, item.trap, true) {
						t.Fatalf("trap: %d %q %q", code, out, diag)
					}
				} else if code != 0 || out != "" || diag != "" {
					t.Fatalf("run: %d %q %q", code, out, diag)
				}
			}
		})
	}
}
