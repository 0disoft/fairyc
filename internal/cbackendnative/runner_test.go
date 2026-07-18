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
	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/cli"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/interpreter"
	"github.com/0disoft/fairyc/internal/layoutfixture"
	"github.com/0disoft/fairyc/internal/target"
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

func TestNativeReportSchemaTracksNativeEvidenceShape(t *testing.T) {
	t.Parallel()
	if reportSchemaVersion != "fairyc.native-c-report/v10" {
		t.Fatalf("report schema = %q", reportSchemaVersion)
	}
	encoded, err := json.Marshal(Report{
		SchemaVersion: reportSchemaVersion, Executions: 68, DropTraceExecutions: 68,
		LayoutCases: []string{"pass.array-i32-three", "pass.pointer"}, LayoutProbeExecutions: 4,
		Sanitizer: cdriver.AddressUndefinedSanitizerMode, SanitizerExecutions: 34,
		ABICases:      []string{"pass.aggregate-rollback", "pass.void-return"},
		ABIDirections: []string{"clang-caller-gcc-callee", "gcc-caller-clang-callee"}, ABIProbeExecutions: 56,
		SystemHeapExecutions: 4,
		BufferExecutions:     4,
		ArenaExecutions:      4,
		TextViewExecutions:   4,
		OwnedTextExecutions:  4,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encoded), `"schemaVersion":"fairyc.native-c-report/v10"`) ||
		!strings.Contains(string(encoded), `"dropTraceExecutions":68`) ||
		!strings.Contains(string(encoded), `"layoutCases":["pass.array-i32-three","pass.pointer"]`) ||
		!strings.Contains(string(encoded), `"layoutProbeExecutions":4`) ||
		!strings.Contains(string(encoded), `"sanitizer":"address-undefined-o1"`) ||
		!strings.Contains(string(encoded), `"sanitizerExecutions":34`) ||
		!strings.Contains(string(encoded), `"abiCases":["pass.aggregate-rollback","pass.void-return"]`) ||
		!strings.Contains(string(encoded), `"abiDirections":["clang-caller-gcc-callee","gcc-caller-clang-callee"]`) ||
		!strings.Contains(string(encoded), `"abiProbeExecutions":56`) ||
		!strings.Contains(string(encoded), `"systemHeapExecutions":4`) ||
		!strings.Contains(string(encoded), `"bufferExecutions":4`) ||
		!strings.Contains(string(encoded), `"arenaExecutions":4`) ||
		!strings.Contains(string(encoded), `"textViewExecutions":4`) ||
		!strings.Contains(string(encoded), `"ownedTextExecutions":4`) {
		t.Fatalf("report JSON = %s", encoded)
	}
}

func TestSystemHeapProbeRunnerRejectsEmptySourceBeforeToolchainUse(t *testing.T) {
	t.Parallel()
	toolchains := []cdriver.Toolchain{{Name: "clang"}, {Name: "gcc"}}
	if _, err := runStandardLibraryProbe(t.TempDir(), toolchains, "SystemHeap", "system-heap", "", ""); err == nil || !strings.Contains(err.Error(), "must be non-empty") {
		t.Fatalf("runStandardLibraryProbe() error = %v", err)
	}
}

func TestSanitizerEnvironmentOverridesAmbientConfiguration(t *testing.T) {
	t.Parallel()
	environment := sanitizerEnvironment([]string{
		"PATH=/bin", "ASAN_OPTIONS=detect_leaks=0", "LSAN_OPTIONS=exitcode=0", "UBSAN_OPTIONS=halt_on_error=0", "HOME=/tmp/home",
	})
	if len(environment) != 5 || environment[0] != "PATH=/bin" || environment[1] != "HOME=/tmp/home" {
		t.Fatalf("sanitizer environment = %v", environment)
	}
	joined := strings.Join(environment, "\n")
	for _, required := range []string{
		"ASAN_OPTIONS=abort_on_error=1:color=never:detect_leaks=1:halt_on_error=1:symbolize=0",
		"LSAN_OPTIONS=exitcode=23:report_objects=0",
		"UBSAN_OPTIONS=color=never:halt_on_error=1:print_stacktrace=0",
	} {
		if strings.Count(joined, required) != 1 {
			t.Errorf("sanitizer environment is missing unique %q: %v", required, environment)
		}
	}
}

func TestGenerateLayoutProbeFromRepositoryFixtures(t *testing.T) {
	t.Parallel()
	manifest := filepath.Join("..", "..", "spec", "fixtures", "layout", "manifest-v0.json")
	targetID, cases, err := layoutfixture.NativeProbeCases(manifest)
	if err != nil {
		t.Fatal(err)
	}
	unit, err := generateLayoutProbe(targetID, cases)
	if err != nil {
		t.Fatal(err)
	}
	repeated, err := generateLayoutProbe(targetID, cases)
	if err != nil || repeated.Source != unit.Source || !slices.Equal(repeated.Cases, unit.Cases) {
		t.Fatalf("repeated layout probe drifted: %v", err)
	}
	if unit.Target != target.X8664LinuxGNU || len(unit.Cases) != 9 || !strings.HasSuffix(unit.Source, "int main(void) { return 0; }\n") {
		t.Fatalf("layout probe unit = %+v", unit)
	}
	for _, required := range []string{
		"typedef int32_t",
		"typedef long",
		"typedef void *",
		"typedef void (*",
		"_Static_assert(sizeof(",
		"_Static_assert(_Alignof(",
		"_Static_assert(offsetof(",
	} {
		if !strings.Contains(unit.Source, required) {
			t.Errorf("layout probe is missing %q", required)
		}
	}
	for _, probeCase := range unit.Cases {
		if strings.Contains(unit.Source, probeCase) {
			t.Errorf("layout probe source leaks fixture case ID %q", probeCase)
		}
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
