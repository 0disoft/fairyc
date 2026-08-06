package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
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

func TestStableTestListReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-list-v1.schema.json"))
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
	reportBytes, err := json.Marshal(stableTestListReport{
		SchemaVersion: stableTestListSchema,
		Tests:         []string{"tests::main::passes"},
		Count:         1,
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
}

func TestStableWorkspaceTestListReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-list-v2.schema.json"))
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
	reportBytes, err := json.Marshal(stableTestListReport{
		SchemaVersion: stableWorkspaceTestListSchema,
		Tests:         []string{"example.app::tests::main::passes"},
		Count:         1,
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
}

func TestStableLocatedTestListReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-list-v3.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Location struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"location"`
			Source struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"source"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	source := schemaTestSource()
	location := stableTestLocation{Identity: "tests::main::passes", Source: source}
	reportBytes, err := json.Marshal(stableTestListReport{
		SchemaVersion: stableLocatedTestListSchema,
		Tests:         []string{location.Identity},
		Locations:     &[]stableTestLocation{location},
		Count:         1,
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	locationBytes, _ := json.Marshal(location)
	assertJSONFieldsMatchSchema(t, locationBytes, schema.Defs.Location.Properties, schema.Defs.Location.Required)
	sourceBytes, _ := json.Marshal(source)
	assertJSONFieldsMatchSchema(t, sourceBytes, schema.Defs.Source.Properties, schema.Defs.Source.Required)
}

func TestStableTestExecutionReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-report-v1.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Outcome struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"outcome"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	target := "x86_64-windows-msvc"
	profile := "release"
	category := "assertion-failed"
	outcome := stableTestExecutionOutcome{
		Identity: "tests::main::fails", Outcome: "failed", TrapCategory: &category,
	}
	reportBytes, err := json.Marshal(stableTestExecutionReport{
		SchemaVersion: stableTestReportSchema,
		Engine:        "native",
		Target:        &target,
		Profile:       &profile,
		Tests:         []stableTestExecutionOutcome{outcome},
		Count:         1,
		Failed:        1,
		Status:        "failed",
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	outcomeBytes, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, outcomeBytes, schema.Defs.Outcome.Properties, schema.Defs.Outcome.Required)
}

func TestStableWorkspaceTestExecutionReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-report-v2.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Outcome struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"outcome"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	category := "assertion-failed"
	outcome := stableTestExecutionOutcome{
		Identity: "example.app::tests::main::fails", Outcome: "failed", TrapCategory: &category,
	}
	reportBytes, err := json.Marshal(stableTestExecutionReport{
		SchemaVersion: stableWorkspaceTestReportSchema,
		Engine:        "interpreter",
		Tests:         []stableTestExecutionOutcome{outcome},
		Count:         1,
		Failed:        1,
		Status:        "failed",
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	outcomeBytes, err := json.Marshal(outcome)
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, outcomeBytes, schema.Defs.Outcome.Properties, schema.Defs.Outcome.Required)
}

func TestStableLocatedTestExecutionReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-report-v3.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Outcome struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"outcome"`
			Source struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"source"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	source := schemaTestSource()
	outcome := stableTestExecutionOutcome{
		Identity: "tests::main::passes", Outcome: "passed", Source: &source,
	}
	reportBytes, err := json.Marshal(stableTestExecutionReport{
		SchemaVersion: stableLocatedTestReportSchema,
		Engine:        "interpreter",
		Tests:         []stableTestExecutionOutcome{outcome},
		Count:         1,
		Passed:        1,
		Status:        "ok",
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	outcomeBytes, _ := json.Marshal(outcome)
	assertJSONFieldsMatchSchema(t, outcomeBytes, schema.Defs.Outcome.Properties, schema.Defs.Outcome.Required)
	sourceBytes, _ := json.Marshal(source)
	assertJSONFieldsMatchSchema(t, sourceBytes, schema.Defs.Source.Properties, schema.Defs.Source.Required)
}

func TestStableCapturedTestExecutionReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "stable-test-report-v4.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Outcome struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"outcome"`
			Source struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"source"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	source := schemaTestSource()
	stdout := "AP8="
	stderr := "gA=="
	outcome := stableTestExecutionOutcome{
		Identity: "tests::main::passes", Outcome: "passed", Source: &source,
		StdoutBase64: &stdout, StderrBase64: &stderr,
	}
	target := "x86_64-windows-msvc"
	profile := "release"
	reportBytes, err := json.Marshal(stableTestExecutionReport{
		SchemaVersion: stableCapturedTestReportSchema,
		Engine:        "native",
		Target:        &target,
		Profile:       &profile,
		Tests:         []stableTestExecutionOutcome{outcome},
		Count:         1,
		Passed:        1,
		Status:        "ok",
		Lock:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	outcomeBytes, _ := json.Marshal(outcome)
	assertJSONFieldsAllowedBySchema(t, outcomeBytes, schema.Defs.Outcome.Properties, schema.Defs.Outcome.Required)
	sourceBytes, _ := json.Marshal(source)
	assertJSONFieldsMatchSchema(t, sourceBytes, schema.Defs.Source.Properties, schema.Defs.Source.Required)
}

func schemaTestSource() stableTestSource {
	return stableTestSource{
		PackageUID:      "example.app",
		Path:            "tests/main.fy",
		Line:            3,
		ByteColumn:      9,
		StartByte:       32,
		EndByte:         38,
		FileIdentity:    "fyfile1-sha256-" + strings.Repeat("0", 64),
		LineMapIdentity: "fylm1-sha256-" + strings.Repeat("1", 64),
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

func TestWorkspaceSessionResponseMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-workspace-check-session-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Defs struct {
			Request struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"request"`
			Response struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"response"`
			Cache struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"cache"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	requestBytes, err := json.Marshal(struct {
		SchemaVersion string `json:"schemaVersion"`
		Kind          string `json:"kind"`
		Sequence      int    `json:"sequence"`
	}{SchemaVersion: workspaceSessionSchema, Kind: workspaceSessionRequestKind, Sequence: 1})
	if err != nil {
		t.Fatal(err)
	}
	responseBytes, err := json.Marshal(workspaceSessionResponseForSchema())
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, requestBytes, schema.Defs.Request.Properties, schema.Defs.Request.Required)
	assertJSONFieldsMatchSchema(t, responseBytes, schema.Defs.Response.Properties, schema.Defs.Response.Required)
	cacheBytes, err := json.Marshal(workspaceSessionCacheReport{})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, cacheBytes, schema.Defs.Cache.Properties, schema.Defs.Cache.Required)
}

func TestWorkspaceCacheReportMatchesSchema(t *testing.T) {
	t.Parallel()
	schemaBytes, err := os.ReadFile(filepath.Join("..", "..", "spec", "cli", "experimental-workspace-cache-check-v0.schema.json"))
	if err != nil {
		t.Fatalf("read schema: %v", err)
	}
	var schema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
		Defs       struct {
			Cache struct {
				Properties map[string]json.RawMessage `json:"properties"`
				Required   []string                   `json:"required"`
			} `json:"cache"`
		} `json:"$defs"`
	}
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		t.Fatalf("decode schema contract: %v", err)
	}
	reportBytes, err := json.Marshal(workspaceCacheReportForSchema())
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, reportBytes, schema.Properties, schema.Required)
	cacheBytes, err := json.Marshal(workspaceCacheReuseReport{})
	if err != nil {
		t.Fatal(err)
	}
	assertJSONFieldsMatchSchema(t, cacheBytes, schema.Defs.Cache.Properties, schema.Defs.Cache.Required)
}

func assertJSONFieldsMatchSchema(t *testing.T, encoded []byte, properties map[string]json.RawMessage, required []string) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	schemaFields := make([]string, 0, len(properties))
	for name := range properties {
		schemaFields = append(schemaFields, name)
	}
	encodedFields := make([]string, 0, len(fields))
	for name := range fields {
		encodedFields = append(encodedFields, name)
	}
	slices.Sort(schemaFields)
	slices.Sort(encodedFields)
	if !slices.Equal(schemaFields, encodedFields) {
		t.Fatalf("encoded fields = %v, schema fields = %v", encodedFields, schemaFields)
	}
	slices.Sort(required)
	if !slices.Equal(required, schemaFields) {
		t.Fatalf("required fields = %v, schema fields = %v", required, schemaFields)
	}
}

func assertJSONFieldsAllowedBySchema(t *testing.T, encoded []byte, properties map[string]json.RawMessage, required []string) {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	for field := range fields {
		if _, found := properties[field]; !found {
			t.Fatalf("encoded field %q is absent from schema properties", field)
		}
	}
	for _, field := range required {
		if _, found := fields[field]; !found {
			t.Fatalf("required schema field %q is absent from encoded object", field)
		}
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
