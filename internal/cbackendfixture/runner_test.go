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

func TestCEmissionManifestV0(t *testing.T) {
	t.Parallel()
	manifestPath := filepath.Join("..", "..", "spec", "fixtures", "c-emission", "manifest-v0.json")
	report, err := Run(manifestPath)
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if report.CasesRun != 18 {
		t.Fatalf("Run() cases = %d, want 18", report.CasesRun)
	}
	for _, failure := range report.Failures {
		t.Errorf("%s: %s", failure.CaseID, failure.Detail)
	}
	if len(report.Emitted) != 18 {
		t.Errorf("Run() emitted = %d, want 18", len(report.Emitted))
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
