package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/target"
)

func TestParseReportTopLevelMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-parse-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	reportBytes, err := json.Marshal(parseReport{
		SchemaVersion: parseReportSchema,
		SourcePath:    "main.fy",
		Status:        "ok",
		Root:          -1,
		Tokens:        []tokenReport{},
		Nodes:         []nodeReport{},
		Diagnostics:   []diagnosticReport{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(reportBytes, &fields); err != nil {
		t.Fatal(err)
	}
	schemaFields := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		schemaFields = append(schemaFields, name)
	}
	reportFields := make([]string, 0, len(fields))
	for name := range fields {
		reportFields = append(reportFields, name)
	}
	slices.Sort(schemaFields)
	slices.Sort(reportFields)
	if !slices.Equal(schemaFields, reportFields) {
		t.Fatalf("report fields = %v, schema fields = %v", reportFields, schemaFields)
	}
	slices.Sort(schema.Required)
	if !slices.Equal(schema.Required, schemaFields) {
		t.Fatalf("required fields = %v, schema fields = %v", schema.Required, schemaFields)
	}
}

func TestTargetSpecTopLevelMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-target-spec-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	spec, found := target.Lookup(target.X8664LinuxGNU)
	if !found {
		t.Fatal("target spec is missing")
	}
	reportBytes, err := json.Marshal(spec)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(reportBytes, &fields); err != nil {
		t.Fatal(err)
	}
	schemaFields := make([]string, 0, len(schema.Properties))
	for name := range schema.Properties {
		schemaFields = append(schemaFields, name)
	}
	reportFields := make([]string, 0, len(fields))
	for name := range fields {
		reportFields = append(reportFields, name)
	}
	slices.Sort(schemaFields)
	slices.Sort(reportFields)
	if !slices.Equal(schemaFields, reportFields) {
		t.Fatalf("report fields = %v, schema fields = %v", reportFields, schemaFields)
	}
	slices.Sort(schema.Required)
	if !slices.Equal(schema.Required, schemaFields) {
		t.Fatalf("required fields = %v, schema fields = %v", schema.Required, schemaFields)
	}
}
