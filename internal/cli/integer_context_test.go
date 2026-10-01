package cli

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/sdkstdlib"
	"github.com/0disoft/fairyc/internal/target"
	"github.com/0disoft/fairyc/internal/workspacecompiler"
	"github.com/0disoft/fairyc/internal/workspaceloader"
)

func TestContextualIntegerNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "contextual_integer", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "contextual-integer-native")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	for _, item := range []struct{ name, before, after, trap string }{
		{"normal", "", "", ""},
		{"division-trap", "var usize rows = 2;", "var usize rows = 0;", "division-by-zero"},
		{"overflow", "var u8 limit = 254;", "var u8 limit = 255;", "integer-overflow"},
		{"shift-width", "var u32 amount = 1;", "var u32 amount = 32;", "invalid-shift"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			source := string(data)
			if item.before != "" {
				source = strings.Replace(source, item.before, item.after, 1)
			}
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.contextual-integer\"\nname = \"contextual-integer\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
			writeTestSource(t, root, "src/main.fy", source)
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
					_, emitErr := cbackend.Emit(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
					t.Fatalf("%s: %s emission: %v", profile.Name, diagnostics.String(), emitErr)
				}
				outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
				_, trap, classificationErr := classifyStableRunOutcome(outcome)
				if err != nil || classificationErr != nil || trap != item.trap || outcome.Stdout != "" || outcome.Stderr != "" || item.trap == "" && outcome.ExitCode != 0 {
					unit, _ := cbackend.Emit(checked.MIR, tokenScannerEntry(t, checked), cbackend.Stage0WindowsMSVCProfile())
					_ = os.WriteFile(filepath.Join(evidenceRoot, item.name+"-"+profile.Name+"-failed.c"), []byte(unit.Source), 0600)
					t.Fatalf("%s: %+v %v %v", profile.Name, outcome, err, classificationErr)
				}
				sum, sourceSum := sha256.Sum256(artifact.Bytes), sha256.Sum256([]byte(source))
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
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.contextual-integer-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
