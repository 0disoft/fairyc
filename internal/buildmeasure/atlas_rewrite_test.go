package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestAtlasRewriteReportPreservesBoundedRawEvidence(t *testing.T) {
	t.Parallel()
	report := validAtlasRewriteReport()
	first, err := buildmeasure.MarshalAtlasRewrite(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalAtlasRewrite(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(first)
	if text != string(second) || !strings.HasPrefix(text, `{"schema":"fairyc.atlas-rewrite-measurement/v0"`) ||
		strings.Contains(text, `C:\`) || strings.Contains(text, "src/main.fy") ||
		strings.Contains(text, `"speedup"`) || strings.Contains(text, `"default"`) {
		t.Fatalf("Atlas rewrite JSON = %s", first)
	}
}

func TestAtlasRewriteReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.AtlasRewriteReport)
	}{
		{name: "policy claim", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Decision = "incremental-required" }},
		{name: "timed planning", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Scope = "analysis-and-encoding" }},
		{name: "missing sample", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Samples = value.Samples[:8] }},
		{name: "clock mismatch", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Samples[0].BelowClockResolution = false }},
		{name: "workload drift", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Samples[3].MapRecords++ }},
		{name: "encoded size drift", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Samples[5].EncodedBytes++ }},
		{name: "unbounded map", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.MaxMapBytesPerPackage = 1 << 31 }},
		{name: "missing source denominator", mutate: func(value *buildmeasure.AtlasRewriteReport) { value.Shape.SourceBytes = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validAtlasRewriteReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalAtlasRewrite(report); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func validAtlasRewriteReport() buildmeasure.AtlasRewriteReport {
	report := buildmeasure.AtlasRewriteReport{
		Schema: buildmeasure.AtlasRewriteSchema, Workload: "workspace_quickstart",
		Scope: "canonical-fymap-encode-all-packages;workspace-analysis-map-planning-and-publication-excluded",
		Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: 10,
		Decision: "raw-distribution-only-no-format-or-incremental-policy",
		Setup:    "one-untimed-warmup;fixed-preplanned-partitions;gc-uncontrolled", Iterations: buildmeasure.AtlasRewriteIterations,
		CompilerBuild: "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		LogicalSchema: "fairyc.fymap-logical/v1", TransportSchema: "fairyc.fymap/v1", TransportEncoding: "fairyc.fymap-json/v1",
		MaxMapBytesPerPackage: 64 << 20,
		Environment:           buildmeasure.AtlasRewriteEnvironment{OS: "windows", Arch: "amd64"},
		Shape:                 buildmeasure.AtlasRewriteShape{Packages: 2, SourceFiles: 2, SourceBytes: 200, MapRecords: 12},
	}
	for index := 0; index < buildmeasure.AtlasRewriteIterations; index++ {
		duration := int64(20 + index)
		if index == 0 {
			duration = 5
		}
		report.Samples = append(report.Samples, buildmeasure.AtlasRewriteSample{
			Sequence: index + 1, DurationNS: duration, BelowClockResolution: duration < report.ClockResolutionNS,
			Packages: 2, MapRecords: 12, EncodedBytes: 4096,
		})
	}
	return report
}
