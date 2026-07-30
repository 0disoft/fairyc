package pecofffixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryGeneratedCOFFObjectManifestPasses(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-generated-object", "manifest-v0.json")
	report, err := RunGenerated(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Target != fixtureTarget || report.CasesRun != 4 || report.Passed != 4 || len(report.Failures) != 0 {
		t.Fatalf("RunGenerated() report = %+v", report)
	}
}

func TestGeneratedCOFFObjectManifestRejectsMalformedIdentity(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-generated-object", "manifest-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name string
		edit func(*generatedManifest)
		want string
	}{
		{name: "unknown generator", edit: func(value *generatedManifest) { value.Cases[0].Generator = "unknown" }, want: "unknown generator"},
		{name: "wrong input count", edit: func(value *generatedManifest) { value.Cases[0].Inputs = []string{"one", "two"} }, want: "requires exactly one input"},
		{name: "invalid digest", edit: func(value *generatedManifest) { value.Cases[0].Expect.SHA256 = strings.Repeat("A", 64) }, want: "incomplete generator"},
		{name: "byte length", edit: func(value *generatedManifest) { value.Cases[0].Expect.ByteLength++ }, want: "byte length mismatch"},
		{name: "symbol", edit: func(value *generatedManifest) { value.Cases[0].Expect.Symbols[0].Name = "wrong" }, want: "symbol receipt mismatch"},
		{name: "relocation gap", edit: func(value *generatedManifest) { value.Cases[0].Expect.RelocationRuns[0].Start = 1 }, want: "not a contiguous"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var value generatedManifest
			if err := json.Unmarshal(contents, &value); err != nil {
				t.Fatal(err)
			}
			testCase.edit(&value)
			encoded, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			report, runErr := RunGenerated(manifestPath)
			combined := ""
			if runErr != nil {
				combined = runErr.Error()
			} else if len(report.Failures) > 0 {
				combined = report.Failures[0].Detail
			}
			if !strings.Contains(combined, testCase.want) {
				t.Fatalf("RunGenerated() error/report = %q, want %q", combined, testCase.want)
			}
		})
	}
}

func TestGeneratedCOFFObjectManifestRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	t.Parallel()
	for _, body := range []string{
		`{"$schema":"./schema-v0.json","schemaVersion":"fairyc.coff-generated-object-fixtures/v0","target":"x86_64-windows-msvc","contracts":["c"],"deferred":["d"],"cases":[],"unknown":true}`,
		`{"$schema":"./schema-v0.json","schemaVersion":"fairyc.coff-generated-object-fixtures/v0","target":"x86_64-windows-msvc","contracts":["c"],"deferred":["d"],"cases":[]} {}`,
	} {
		path := filepath.Join(t.TempDir(), "manifest.json")
		if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, err := RunGenerated(path); err == nil || !strings.Contains(err.Error(), "decode generated COFF object fixture manifest") {
			t.Fatalf("RunGenerated() error = %v", err)
		}
	}
}

func TestGeneratedCOFFObjectManifestRequiresSchemaFields(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-generated-object", "manifest-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name   string
		locate func(map[string]any) map[string]any
		key    string
	}{
		{name: "root target", locate: func(root map[string]any) map[string]any { return root }, key: "target"},
		{name: "case generator", locate: func(root map[string]any) map[string]any { return root["cases"].([]any)[0].(map[string]any) }, key: "generator"},
		{name: "expect bigObject", locate: func(root map[string]any) map[string]any {
			return root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)
		}, key: "bigObject"},
		{name: "symbol weakSearch", locate: func(root map[string]any) map[string]any {
			return root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)["symbols"].([]any)[0].(map[string]any)
		}, key: "weakSearch"},
		{name: "relocation count", locate: func(root map[string]any) map[string]any {
			return root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)["relocationRuns"].([]any)[0].(map[string]any)
		}, key: "count"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var root map[string]any
			if err := json.Unmarshal(contents, &root); err != nil {
				t.Fatal(err)
			}
			delete(testCase.locate(root), testCase.key)
			encoded, err := json.Marshal(root)
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := RunGenerated(manifestPath); err == nil || !strings.Contains(err.Error(), "missing required field") {
				t.Fatalf("RunGenerated() error = %v", err)
			}
		})
	}
}
