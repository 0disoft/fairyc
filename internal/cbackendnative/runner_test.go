package cbackendnative

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/cbackend"
	"github.com/0disoft/fairyc/internal/cbackendfixture"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
)

func TestBuildSmokeSourceReachesCEmission(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "main.fy"), []byte(buildSmokeSource), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := cli.Run([]string{"experimental", "emit-c", "main.fy"}, root, &stdout, &stderr)
	if code != cli.ExitSuccess || stderr.Len() != 0 {
		t.Fatalf("build smoke C emission = code %d, stderr %q", code, stderr.String())
	}
	for _, required := range []string{"fy_resource_0", "fy_tagged_0", "fy_tagged_1", "FY_OPTION_SOME", "FY_RESULT_OK"} {
		if !strings.Contains(stdout.String(), required) {
			t.Errorf("build smoke C emission is missing %q", required)
		}
	}
	if strings.Contains(stdout.String(), "fy_runtime_trace_drop") {
		t.Fatal("product emit-c output contains native conformance instrumentation")
	}
}

func TestNativeReportSchemaTracksDropTraceShape(t *testing.T) {
	t.Parallel()
	if reportSchemaVersion != "fairyc.native-c-report/v2" {
		t.Fatalf("report schema = %q", reportSchemaVersion)
	}
	encoded, err := json.Marshal(Report{SchemaVersion: reportSchemaVersion, Executions: 68, DropTraceExecutions: 68})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"schemaVersion":"fairyc.native-c-report/v2"`) ||
		!strings.Contains(string(encoded), `"dropTraceExecutions":68`) {
		t.Fatalf("report JSON = %s", encoded)
	}
}

func TestExpectedExitCode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		fixture cbackendfixture.EmittedCase
		want    int
		fails   bool
	}{
		{name: "return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: 42}, want: 42},
		{name: "overflow", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "integer-overflow"}, want: 194},
		{name: "unknown trap", fixture: cbackendfixture.EmittedCase{ID: "trap", Outcome: "trap", TrapCategory: "unknown"}, fails: true},
		{name: "negative return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: -1}, fails: true},
		{name: "reserved return", fixture: cbackendfixture.EmittedCase{ID: "return", Outcome: "return", Value: trapExitBase}, fails: true},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			got, err := expectedExitCode(test.fixture)
			if test.fails {
				if err == nil {
					t.Fatalf("expectedExitCode() = %d, nil", got)
				}
				return
			}
			if err != nil || got != test.want {
				t.Fatalf("expectedExitCode() = %d, %v; want %d, nil", got, err, test.want)
			}
		})
	}
}

func TestDropTraceRecordsUseSourceLinkedSiteIDs(t *testing.T) {
	t.Parallel()
	span := hir.Span{Source: "main.fy", StartByte: 10, EndByte: 15}
	fixture := cbackendfixture.EmittedCase{
		ID: "drop",
		DropTrace: []interpreter.DropEvent{
			{ResourceType: "main::Token", Binding: span},
			{ResourceType: "main::Token", Binding: span},
		},
		TraceUnit: cbackend.Unit{DropSites: []cbackend.DropSite{{
			ID: 7, ResourceType: "main::Token", Source: "main.fy", StartByte: 10, EndByte: 15,
		}}},
	}
	ids, err := expectedDropSiteIDs(fixture)
	if err != nil || !slices.Equal(ids, []uint32{7, 7}) {
		t.Fatalf("expectedDropSiteIDs() = %v, %v", ids, err)
	}
	decoded, err := decodeDropTrace([]byte{7, 0, 0, 0, 7, 0, 0, 0})
	if err != nil || !slices.Equal(decoded, ids) {
		t.Fatalf("decodeDropTrace() = %v, %v", decoded, err)
	}
	fixture.DropTrace[0].Binding.StartByte++
	if _, err := expectedDropSiteIDs(fixture); err == nil {
		t.Fatal("expectedDropSiteIDs() accepted a missing source-linked site")
	}
	if _, err := decodeDropTrace([]byte{1, 2, 3}); err == nil {
		t.Fatal("decodeDropTrace() accepted a partial record")
	}
}

func TestBoundedBufferRetainsPrefix(t *testing.T) {
	t.Parallel()
	buffer := boundedBuffer{limit: 4}
	if written, err := buffer.Write([]byte("abcdef")); err != nil || written != 6 {
		t.Fatalf("Write() = %d, %v", written, err)
	}
	if buffer.String() != "abcd" || !buffer.truncated {
		t.Fatalf("buffer = %q, truncated=%t", buffer.String(), buffer.truncated)
	}
}

func TestRunRejectsNonLinuxHost(t *testing.T) {
	if runtime.GOOS == "linux" && runtime.GOARCH == "amd64" {
		t.Skip("native host is eligible; actual compiler execution belongs to the configured native command")
	}
	_, err := Run("unused.json")
	if err == nil || !strings.Contains(err.Error(), "requires linux/amd64") {
		t.Fatalf("Run() error = %v", err)
	}
}
