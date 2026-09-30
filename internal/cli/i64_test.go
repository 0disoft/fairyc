package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestI64NativeRuntimeAndFormatting(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	// Reuse the exact interpreter input; two independent execution paths must agree.
	data, err := os.ReadFile(filepath.Join("..", "compiler", "testdata", "i64.fy"))
	if err != nil {
		t.Fatal(err)
	}
	normal := string(data)
	evidenceRoot := filepath.Join("..", "..", ".cache", "i64-native")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	format, err := os.ReadFile(filepath.Join("..", "..", "examples", "i64_quickstart", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, source, stdout, trap string }{
		{"containers", normal, "", ""},
		{"formatting", string(format), "-9223372036854775808,9223372036854775807,0,-5000000000\n", ""},
	}
	for _, tc := range []struct{ expression, trap string }{
		{"hi + 1", "integer-overflow"}, {"lo - 1", "integer-overflow"},
		{"lo * -1", "integer-overflow"}, {"hi * 2", "integer-overflow"},
		{"-lo", "integer-overflow"}, {"lo / -1", "signed-division-overflow"},
		{"lo % -1", "signed-division-overflow"}, {"hi / 0", "division-by-zero"},
		{"hi % 0", "division-by-zero"}, {"hi << 1", "integer-overflow"},
		{"hi << -1", "invalid-shift"}, {"lo >> 64", "invalid-shift"},
	} {
		cases = append(cases, struct{ name, source, stdout, trap string }{tc.expression,
			"module main; fn i32 main() { let i64 lo = -9223372036854775808; let i64 hi = 9223372036854775807; let i64 v = " + tc.expression + "; assert(v == 0); return 0; }", "", tc.trap})
	}
	for index, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.i64\"\nname = \"i64\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", tc.source)
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
				artifact, code := compileWindowsProgram(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, &diagnostics)
				if code != ExitSuccess {
					_, emitErr := cbackend.Emit(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
					t.Fatalf("%s; emission: %v", diagnostics.String(), emitErr)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				if err != nil {
					t.Fatal(err)
				}
				category, trapped := windowsHostTrapCategory(outcome.ExitCode)
				sum := sha256.Sum256(artifact.Bytes)
				sourceSum := sha256.Sum256([]byte(tc.source))
				image := filepath.Join(evidenceRoot, fmt.Sprintf("%02d-%s.exe", index, profile.Name))
				if err := os.WriteFile(image, artifact.Bytes, 0700); err != nil {
					t.Fatal(err)
				}
				evidence = append(evidence, map[string]any{"case": tc.name, "profile": profile.Name, "sourceSHA256": hex.EncodeToString(sourceSum[:]), "exeSHA256": hex.EncodeToString(sum[:]), "exeBytes": len(artifact.Bytes), "stdout": outcome.Stdout, "stderr": outcome.Stderr, "exitCode": outcome.ExitCode, "trap": category, "expectedTrap": tc.trap, "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
				if tc.trap != "" {
					if !trapped || category != tc.trap || outcome.Stdout != "" {
						t.Fatalf("%s: want %s, got %+v", profile.Name, tc.trap, outcome)
					}
				} else if outcome.ExitCode != 0 || outcome.Stdout != tc.stdout || outcome.Stderr != "" {
					t.Fatalf("%s: %+v", profile.Name, outcome)
				}
			}
		})
	}
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.i64-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
