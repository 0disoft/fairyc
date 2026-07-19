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

func TestTargetSpecScalarProviderSchemaMatchesSemanticValidator(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-target-spec-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Defs struct {
			Scalar struct {
				DependentRequired map[string][]string `json:"dependentRequired"`
				Properties        map[string]struct {
					Pattern string   `json:"pattern"`
					Enum    []string `json:"enum"`
				} `json:"properties"`
			} `json:"scalar"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	if !slices.Equal(schema.Defs.Scalar.DependentRequired["header"], []string{"availability"}) ||
		!slices.Equal(schema.Defs.Scalar.DependentRequired["availability"], []string{"header"}) {
		t.Fatalf("scalar provider dependencies = %+v", schema.Defs.Scalar.DependentRequired)
	}
	if got := schema.Defs.Scalar.Properties["header"].Pattern; got != `^[A-Za-z0-9_-][A-Za-z0-9_.-]*(?:/[A-Za-z0-9_-][A-Za-z0-9_.-]*)*\.h$` {
		t.Fatalf("scalar provider header pattern = %q", got)
	}
	if got := schema.Defs.Scalar.Properties["availability"].Enum; !slices.Equal(got, []string{"iso-c11", "posix.1-2008"}) {
		t.Fatalf("scalar provider availability enum = %v", got)
	}
}

func TestTargetSpecSchemaProfilesMatchKnownTargets(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-target-spec-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		OneOf []struct {
			Properties map[string]struct {
				Const string `json:"const"`
			} `json:"properties"`
		} `json:"oneOf"`
		Defs struct {
			Stack struct {
				Required   []string `json:"required"`
				Properties map[string]struct {
					Minimum    *uint32 `json:"minimum"`
					MultipleOf uint32  `json:"multipleOf"`
				} `json:"properties"`
			} `json:"stack"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	var profileIDs []string
	for _, profile := range schema.OneOf {
		profileIDs = append(profileIDs, profile.Properties["id"].Const)
	}
	wantIDs := []string{string(target.X8664LinuxGNU), string(target.X8664WindowsMSVC)}
	if !slices.Equal(profileIDs, wantIDs) {
		t.Fatalf("target schema profiles = %v, want %v", profileIDs, wantIDs)
	}
	wantStackFields := []string{"callAlignmentBytes", "redZoneBytes", "shadowSpaceBytes"}
	if !slices.Equal(schema.Defs.Stack.Required, wantStackFields) {
		t.Fatalf("target stack required fields = %v, want %v", schema.Defs.Stack.Required, wantStackFields)
	}
	shadow := schema.Defs.Stack.Properties["shadowSpaceBytes"]
	if shadow.Minimum == nil || *shadow.Minimum != 0 || shadow.MultipleOf != 8 {
		t.Fatalf("target shadow-space schema = %+v", shadow)
	}
}
