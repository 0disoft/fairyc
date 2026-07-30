package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestAtlasCallProcessReportPreservesRawBoundedProcessEvidence(t *testing.T) {
	t.Parallel()
	report := validAtlasCallProcessReport()
	first, err := buildmeasure.MarshalAtlasCallProcess(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalAtlasCallProcess(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(first)
	if text != string(second) || !strings.HasPrefix(text, `{"schema":"fairyc.atlas-call-process-measurement/v0"`) ||
		strings.Contains(text, `C:\`) || strings.Contains(text, "src/main.fy") ||
		strings.Contains(text, `"speedup"`) || strings.Contains(text, "percentile") {
		t.Fatalf("Atlas call process JSON = %s", first)
	}
}

func TestAtlasCallProcessReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.AtlasCallProcessReport)
	}{
		{name: "reference claim", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Decision = "reference-budget" }},
		{name: "timed discovery", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Scope = "build-and-discovery-included" }},
		{name: "retry selection", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.RetryPolicy = "retry-until-fast" }},
		{name: "wrong catalog", mutate: func(value *buildmeasure.AtlasCallProcessReport) {
			value.CatalogVersion = "fairyc.workspace-call-catalog/v1"
		}},
		{name: "missing source denominator", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Shape.SourceBytes = 0 }},
		{name: "calls over policy", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Shape.Calls = 9 }},
		{name: "missing sample", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Samples = value.Samples[:8] }},
		{name: "clock mismatch", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Samples[0].BelowClockResolution = true }},
		{name: "missing memory", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Samples[2].PeakWorkingSetBytes = 0 }},
		{name: "output drift", mutate: func(value *buildmeasure.AtlasCallProcessReport) { value.Samples[4].OutputBytes++ }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validAtlasCallProcessReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalAtlasCallProcess(report); err == nil {
				t.Fatal("invalid Atlas call process report accepted")
			}
		})
	}
}

func validAtlasCallProcessReport() buildmeasure.AtlasCallProcessReport {
	report := buildmeasure.AtlasCallProcessReport{
		Schema: buildmeasure.AtlasCallProcessSchema, Workload: "workspace_quickstart-root-main-outgoing",
		Scope: "caller-command-start-through-exit;compiler-build-symbol-discovery-and-warmup-excluded",
		Clock: "caller-monotonic-wall-nanoseconds", ClockResolutionNS: 10,
		Setup:          "one-untimed-warmup;fresh-standalone-fairyc-process-per-sample;locked-workspace;os-caches-uncontrolled",
		RetryPolicy:    "none;preserve-first-observation",
		MemoryMetric:   "windows-process-memory-counters-peak-working-set-and-pagefile-bytes",
		Decision:       "local-control-only-no-reference-budget-speedup-traversal-or-impact-claim",
		Iterations:     buildmeasure.AtlasCallProcessIterations,
		CompilerBuild:  "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		CatalogVersion: "fairyc.workspace-call-catalog/v0", CatalogEncoding: "fairyc.workspace-call-catalog-json/v0",
		Environment: buildmeasure.AtlasCallProcessEnvironment{OS: "windows", Arch: "amd64"},
		Policy:      buildmeasure.AtlasCallProcessPolicy{MaxPackages: 8, MaxCalls: 8, MaxOutputBytes: 2 << 20},
		Shape:       buildmeasure.AtlasCallProcessShape{Packages: 2, SourceFiles: 2, SourceBytes: 200, Calls: 1, CrossPackageCalls: 1, SelectedCalls: 1},
	}
	for index := 0; index < buildmeasure.AtlasCallProcessIterations; index++ {
		report.Samples = append(report.Samples, buildmeasure.AtlasCallProcessSample{
			Sequence: index + 1, DurationNS: int64(20 + index),
			PeakWorkingSetBytes: uint64(10_000 + index), PeakPagefileBytes: uint64(20_000 + index), OutputBytes: 4_096,
		})
	}
	return report
}
