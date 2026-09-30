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

func TestExclusiveSubviewNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "exclusive_subview", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "exclusive-subview-native")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	for _, item := range []struct{ name, bounds, trap string }{
		{"normal", "1..3", ""}, {"reversed", "3..2", "bounds"}, {"past-end", "0..5", "bounds"},
		{"omitted-end", "5..", "bounds"}, {"omitted-start", "..5", "bounds"},
	} {
		t.Run(item.name, func(t *testing.T) {
			root := t.TempDir()
			source := strings.Replace(string(data), "view[1..3]", "view["+item.bounds+"]", 1)
			writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.exclusive-subview\"\nname = \"exclusive-subview\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
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
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.exclusive-subview-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
