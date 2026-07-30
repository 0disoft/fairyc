package pecofffixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRepositoryCOFFObjectManifestPasses(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-object", "manifest-v0.json")
	report, err := Run(path)
	if err != nil {
		t.Fatal(err)
	}
	if report.Target != fixtureTarget || report.CasesRun != 2 || report.Passed != 2 || len(report.Failures) != 0 {
		t.Fatalf("Run() report = %+v", report)
	}
}

func TestCOFFObjectManifestRejectsUnknownFieldsAndTrailingData(t *testing.T) {
	t.Parallel()
	for _, testCase := range []struct {
		name string
		body string
	}{
		{name: "unknown", body: `{"$schema":"./schema-v0.json","schemaVersion":"fairyc.coff-object-fixtures/v0","target":"x86_64-windows-msvc","contracts":["contract"],"deferred":["deferred"],"cases":[],"unknown":true}`},
		{name: "trailing", body: `{"$schema":"./schema-v0.json","schemaVersion":"fairyc.coff-object-fixtures/v0","target":"x86_64-windows-msvc","contracts":["contract"],"deferred":["deferred"],"cases":[]} {}`},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(path, []byte(testCase.body), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Run(path); err == nil || !strings.Contains(err.Error(), "decode COFF object fixture manifest") {
				t.Fatalf("Run() error = %v", err)
			}
		})
	}
}

func TestCOFFObjectManifestReportsHexAndExpectationFailures(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-object", "manifest-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var value manifest
	if err := json.Unmarshal(contents, &value); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range []struct {
		name   string
		mutate func(*manifest)
		want   string
	}{
		{name: "invalid hex", mutate: func(value *manifest) { value.Cases[0].ObjectHex = "zz" }, want: "object hex must be"},
		{name: "uppercase hex", mutate: func(value *manifest) { value.Cases[0].ObjectHex = strings.ToUpper(value.Cases[0].ObjectHex) }, want: "object hex must be"},
		{name: "invalid case id", mutate: func(value *manifest) { value.Cases[0].ID = "pass.Bad" }, want: "case id must match"},
		{name: "section mismatch", mutate: func(value *manifest) { value.Cases[0].Expect.Sections[0].COMDATSelection = 6 }, want: "section receipt mismatch"},
		{name: "symbol mismatch", mutate: func(value *manifest) { value.Cases[0].Expect.Symbols[0].Name = "wrong" }, want: "symbol receipt mismatch"},
		{name: "relocation mismatch", mutate: func(value *manifest) { value.Cases[0].Expect.Relocations[0].Type = 1 }, want: "relocation receipt mismatch"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			copyValue := value
			copyValue.Cases = append([]fixtureCase(nil), value.Cases...)
			copyValue.Cases[0].Expect.Sections = append([]sectionExpectation(nil), value.Cases[0].Expect.Sections...)
			copyValue.Cases[0].Expect.Symbols = append([]symbolExpectation(nil), value.Cases[0].Expect.Symbols...)
			copyValue.Cases[0].Expect.Relocations = append([]relocationExpectation(nil), value.Cases[0].Expect.Relocations...)
			testCase.mutate(&copyValue)
			encoded, err := json.Marshal(copyValue)
			if err != nil {
				t.Fatal(err)
			}
			manifestPath := filepath.Join(t.TempDir(), "manifest.json")
			if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
				t.Fatal(err)
			}
			report, err := Run(manifestPath)
			if err != nil {
				t.Fatal(err)
			}
			if len(report.Failures) != 1 || !strings.Contains(report.Failures[0].Detail, testCase.want) {
				t.Fatalf("Run() report = %+v", report)
			}
		})
	}
}

func TestCOFFObjectManifestRequiresEverySchemaField(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "coff-object", "manifest-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, group := range []struct {
		name   string
		keys   []string
		locate func(map[string]any) map[string]any
	}{
		{name: "root", keys: []string{"$schema", "schemaVersion", "target", "contracts", "deferred", "cases"}, locate: func(root map[string]any) map[string]any { return root }},
		{name: "case", keys: []string{"id", "objectHex", "expect"}, locate: func(root map[string]any) map[string]any {
			return root["cases"].([]any)[0].(map[string]any)
		}},
		{name: "expect", keys: []string{"sections", "symbols", "relocations"}, locate: func(root map[string]any) map[string]any {
			return root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)
		}},
		{name: "section", keys: []string{"name", "relocations", "characteristics", "comdatSelection", "associativeSection"}, locate: func(root map[string]any) map[string]any {
			expect := root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)
			return expect["sections"].([]any)[0].(map[string]any)
		}},
		{name: "symbol", keys: []string{"tableIndex", "name", "value", "sectionNumber", "type", "storageClass", "auxiliaryRecords"}, locate: func(root map[string]any) map[string]any {
			expect := root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)
			return expect["symbols"].([]any)[0].(map[string]any)
		}},
		{name: "relocation", keys: []string{"section", "virtualAddress", "symbolIndex", "symbol", "type"}, locate: func(root map[string]any) map[string]any {
			expect := root["cases"].([]any)[0].(map[string]any)["expect"].(map[string]any)
			return expect["relocations"].([]any)[0].(map[string]any)
		}},
	} {
		for _, key := range group.keys {
			group := group
			key := key
			t.Run(group.name+"/"+key, func(t *testing.T) {
				var root map[string]any
				if err := json.Unmarshal(contents, &root); err != nil {
					t.Fatal(err)
				}
				delete(group.locate(root), key)
				encoded, err := json.Marshal(root)
				if err != nil {
					t.Fatal(err)
				}
				manifestPath := filepath.Join(t.TempDir(), "manifest.json")
				if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
					t.Fatal(err)
				}
				if _, err := Run(manifestPath); err == nil || !strings.Contains(err.Error(), "missing required field") {
					t.Fatalf("Run() error = %v", err)
				}
			})
		}
	}
}
