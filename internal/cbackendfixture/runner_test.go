package cbackendfixture

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
)

func TestCEmissionSchemaV0Header(t *testing.T) {
	t.Parallel()
	path := filepath.Join("..", "..", "spec", "fixtures", "c-emission", "schema-v0.json")
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Dialect string `json:"$schema"`
		ID      string `json:"$id"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("schema is not JSON: %v", err)
	}
	if schema.Dialect != "https://json-schema.org/draft/2020-12/schema" || schema.ID != "urn:fairyc:c-emission-fixtures:v0" {
		t.Fatalf("schema header = %+v", schema)
	}
}

func TestCEmissionSchemaAllowsEveryManifestFeature(t *testing.T) {
	t.Parallel()
	root := filepath.Join("..", "..", "spec", "fixtures", "c-emission")
	contents, err := os.ReadFile(filepath.Join(root, "schema-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Definitions struct {
			Feature struct {
				Enum []string `json:"enum"`
			} `json:"feature"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(contents, &schema); err != nil {
		t.Fatalf("schema is not JSON: %v", err)
	}
	allowed := make(map[string]struct{}, len(schema.Definitions.Feature.Enum))
	for _, feature := range schema.Definitions.Feature.Enum {
		if _, duplicate := allowed[feature]; duplicate {
			t.Fatalf("schema repeats feature %q", feature)
		}
		allowed[feature] = struct{}{}
	}
	manifestContents, err := os.ReadFile(filepath.Join(root, "manifest-v0.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Cases []struct {
			ID     string `json:"id"`
			Expect struct {
				Features []string `json:"features"`
			} `json:"expect"`
		} `json:"cases"`
	}
	if err := json.Unmarshal(manifestContents, &manifest); err != nil {
		t.Fatalf("manifest is not JSON: %v", err)
	}
	for _, fixture := range manifest.Cases {
		for _, feature := range fixture.Expect.Features {
			if _, found := allowed[feature]; !found {
				t.Errorf("%s uses feature %q absent from schema", fixture.ID, feature)
			}
		}
	}
}

func TestCEmissionManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "c-emission", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 39 {
		t.Fatalf("Run() cases = %d, want 39", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
	if len(report.Emitted) != 39 {
		t.Errorf("Run() emitted = %d, want 39", len(report.Emitted))
	}
	observedDrops := 0
	for _, fixture := range report.Emitted {
		if fixture.Unit.Instrumentation != "" || len(fixture.Unit.DropSites) != 0 || strings.Contains(fixture.Unit.Source, "fy_runtime_trace_drop") {
			t.Errorf("%s production unit contains conformance instrumentation", fixture.ID)
		}
		if fixture.TraceUnit.Instrumentation != cbackend.InstrumentationDropTraceV0 || !strings.Contains(fixture.TraceUnit.Source, "fy_runtime_trace_drop") {
			t.Errorf("%s has no closed drop-trace unit", fixture.ID)
		}
		observedDrops += len(fixture.DropTrace)
	}
	if observedDrops == 0 {
		t.Error("C emission fixtures exercise no interpreter-approved resource drops")
	}
}

func TestResolveWithinRejectsEscape(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if _, err := resolveWithin(root, root, "../escape.fy"); err == nil {
		t.Fatal("resolveWithin() accepted a parent traversal")
	}
}
