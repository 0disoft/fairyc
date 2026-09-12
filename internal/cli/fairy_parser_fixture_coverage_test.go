package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/frontend"
	"github.com/0disoft/fairyc/internal/source"
	"github.com/0disoft/fairyc/internal/target"
)

func TestWindowsFairyParserFixtureCoverage(t *testing.T) {
	if runtime.GOOS != "windows" || runtime.GOARCH != "amd64" {
		t.Skip("requires Windows AMD64 MSVC")
	}
	base := filepath.Join("..", "..", "spec", "fixtures", "parser")
	manifest, err := os.ReadFile(filepath.Join(base, "manifest-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var catalog struct {
		SchemaVersion string `json:"schemaVersion"`
		Cases         []struct {
			ID         string `json:"id"`
			SourceRoot string `json:"sourceRoot"`
			Source     string `json:"source"`
			Expect     struct {
				Status string `json:"status"`
			} `json:"expect"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(manifest, &catalog); err != nil {
		t.Fatal(err)
	}
	if catalog.SchemaVersion != "fairyc.parser-fixtures/v0" || len(catalog.Cases) == 0 || len(catalog.Cases) > 32 {
		t.Fatal("unreviewed fixture catalog or batch size")
	}
	root := t.TempDir()
	copyLexerStreamExample(t, root)
	for _, name := range []string{"syntax.fy", "integrity.fy"} {
		data, err := os.ReadFile(filepath.Join("..", "..", "examples", "lexer_candidate", "src", name))
		if err != nil {
			t.Fatal(err)
		}
		writeTestSource(t, root, "src/"+name, string(data))
	}
	writeTestSource(t, root, "src/main.fy", parserCorpusProgram)
	baseline := map[string]string{}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		var stderr strings.Builder
		compiled, code := compileWindowsWorkspace(root, workspaceTargetOptions{targetID: target.X8664WindowsMSVC, profile: profile, locked: profile == cdriver.OptimizedProfile}, workspaceCompilePolicy{stableDiagnostics: true, runTrapReceipt: true}, &stderr)
		if code != ExitSuccess {
			t.Fatalf("compile: %d %s", code, stderr.String())
		}
		totals := map[string]int{}
		seen := map[string]bool{}
		for _, fixture := range catalog.Cases {
			if seen[fixture.ID] {
				t.Fatal("duplicate fixture ID")
			}
			seen[fixture.ID] = true
			if fixture.Expect.Status != "pass" && fixture.Expect.Status != "fail" {
				t.Fatal("unknown fixture status")
			}
			path := filepath.Join(base, fixture.SourceRoot, fixture.Source)
			rel, err := filepath.Rel(base, path)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				t.Fatal("fixture escapes catalog")
			}
			data, err := os.ReadFile(path)
			if err != nil || len(data) > 32768 {
				t.Fatalf("fixture %s: bytes=%d err=%v", fixture.ID, len(data), err)
			}
			oracle, err := frontend.AnalyzeSyntax(source.NewFile(filepath.ToSlash(fixture.Source), data), source.DefaultLimits())
			if err != nil || oracle.Tree == nil {
				t.Fatalf("reference fixture %s: %v", fixture.ID, err)
			}
			referenceOK := len(oracle.Tree.Diagnostics) == 0
			if referenceOK != (fixture.Expect.Status == "pass") {
				t.Fatalf("reference fixture drift: %s", fixture.ID)
			}
			out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(compiled.artifact, bytes.NewReader(data))
			if err != nil || out.ExitCode != 0 || out.Stderr != "" || out.TrapControl != "" {
				t.Fatalf("native fixture %s: %+v %v", fixture.ID, out, err)
			}
			fields := strings.Fields(out.Stdout)
			if len(fields) < 6 {
				t.Fatal("truncated fixture observation")
			}
			row := make([]int, len(fields))
			for i, text := range fields {
				n, err := strconv.Atoi(text)
				if err != nil || n < 0 {
					t.Fatal("invalid observation")
				}
				row[i] = n
			}
			if len(row) != 6+5*row[1] {
				t.Fatal("invalid declaration observation")
			}
			if row[0] != 0 && row[0] != 4 && row[0] != 5 {
				t.Fatalf("unexpected native status for %s: %d", fixture.ID, row[0])
			}
			accepted := row[0] == 0 && row[2] == 0
			if (fixture.ID == "pass.module-path" || fixture.ID == "pass.mutable-parameter" || fixture.ID == "pass.nested-type-close" || fixture.ID == "pass.array-forms" || fixture.ID == "pass.precedence" || fixture.ID == "pass.postfix-chain") && !accepted {
				t.Errorf("supported fixture regressed: %s", fixture.ID)
			}
			class := "rejected"
			if row[0] == 4 {
				class = "budget"
			} else if referenceOK {
				if accepted {
					class = "supported"
					compareParserDeclarations(t, filepath.ToSlash(fixture.Source), data, row[6:])
				} else {
					class = "unsupported"
				}
			} else if accepted {
				class = "difference"
				// The candidate receives bytes, not a file/module identity. Keep this
				// explicit contract gap separate from a grammar false-acceptance finding.
				if fixture.ID == "fail.module-mismatch" {
					class = "context-gap"
				}
			}
			observation := fmt.Sprintf("%s status=%d problems=%d first=%d..%d", class, row[0], row[2], row[4], row[5])
			totals[class]++
			if profile == cdriver.LowProfile {
				baseline[fixture.ID] = fmt.Sprint(row)
				t.Logf("%s %s", fixture.ID, observation)
			} else if baseline[fixture.ID] != fmt.Sprint(row) {
				t.Fatalf("profile difference: %s", fixture.ID)
			}
			if class == "difference" || class == "budget" {
				t.Errorf("unresolved fixture %s: %s", fixture.ID, observation)
			}
		}
		for _, id := range []string{"pass.module-path", "pass.mutable-parameter", "pass.nested-type-close", "pass.array-forms", "pass.precedence", "pass.postfix-chain"} {
			if !seen[id] {
				t.Errorf("missing supported fixture: %s", id)
			}
		}
		t.Logf("profile=%v cases=%d totals=%v", profile, len(catalog.Cases), totals)
	}
}
