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

func TestSharedReborrowNative(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("Windows AMD64 MSVC")
	}
	root := t.TempDir()
	data, err := os.ReadFile(filepath.Join("..", "..", "examples", "shared_reborrow", "src", "main.fy"))
	if err != nil {
		t.Fatal(err)
	}
	writeTestSource(t, root, "fairy.toml", "schema = \"fairyc.manifest/v1\"\n[package]\nuid = \"test.shared-reborrow\"\nname = \"shared-reborrow\"\nversion = \"1.0.0\"\nlanguage = \""+sdkstdlib.LanguageRevision+"\"\nsource = \"src\"\n")
	writeTestSource(t, root, "src/main.fy", string(data))
	workspace, err := workspaceloader.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	checked, err := workspacecompiler.Check(workspace)
	if err != nil || !checked.Verified {
		t.Fatalf("check: %v %+v %+v", err, checked.Semantic.Diagnostics, checked.Semantic.Unsupported)
	}
	evidenceRoot := filepath.Join("..", "..", ".cache", "shared-reborrow-native")
	if err := os.MkdirAll(evidenceRoot, 0700); err != nil {
		t.Fatal(err)
	}
	var evidence []map[string]any
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var diagnostics bytes.Buffer
		artifact, code := compileWindowsProgram(checked.MIR, tokenScannerEntry(t, checked), target.X8664WindowsMSVC, profile, &diagnostics)
		if code != ExitSuccess {
			t.Fatalf("%s: %s", profile.Name, diagnostics.String())
		}
		outcome, err := cdriver.ExecuteWindowsMSVCTarget(artifact)
		if err != nil {
			t.Fatal(err)
		}
		if outcome.ExitCode != 0 || outcome.Stdout != "" || outcome.Stderr != "" {
			t.Fatalf("%s: %+v", profile.Name, outcome)
		}
		sum, sourceSum := sha256.Sum256(artifact.Bytes), sha256.Sum256(data)
		if err := os.WriteFile(filepath.Join(evidenceRoot, profile.Name+".exe"), artifact.Bytes, 0700); err != nil {
			t.Fatal(err)
		}
		evidence = append(evidence, map[string]any{"profile": profile.Name, "sourceSHA256": hex.EncodeToString(sourceSum[:]), "exeSHA256": hex.EncodeToString(sum[:]), "exeBytes": len(artifact.Bytes), "stdout": outcome.Stdout, "stderr": outcome.Stderr, "exitCode": outcome.ExitCode, "toolchain": artifact.Toolchain.Name + " " + artifact.Toolchain.Version})
	}
	encoded, err := json.MarshalIndent(map[string]any{"schema": "fairyc.shared-reborrow-native/v1", "sdk": sdkstdlib.Version, "cases": evidence}, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(evidenceRoot, "receipt.json"), append(encoded, '\n'), 0600); err != nil {
		t.Fatal(err)
	}
}
