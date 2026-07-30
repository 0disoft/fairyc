package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestAtlasCallRetainedReportPreservesClosedRawHeapEvidence(t *testing.T) {
	t.Parallel()
	report := validAtlasCallRetainedReport()
	first, err := buildmeasure.MarshalAtlasCallRetained(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalAtlasCallRetained(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(first)
	if text != string(second) || !strings.HasPrefix(text, `{"schema":"fairyc.atlas-call-retained-heap-measurement/v0"`) ||
		strings.Contains(text, `C:\`) || strings.Contains(text, "src/main.fy") ||
		strings.Contains(text, "perIndexBytes") || strings.Contains(text, "percentile") {
		t.Fatalf("Atlas call retained-heap JSON = %s", first)
	}
}

func TestAtlasCallRetainedReportRejectsDriftAndPolicyClaims(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.AtlasCallRetainedReport)
	}{
		{name: "Atlas-only claim", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.MemoryMetric = "exact-atlas-retained-heap" }},
		{name: "per-index budget", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Decision = "per-index-budget-accepted" }},
		{name: "timed compilation", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Scope = "compile-and-profile" }},
		{name: "missing shape", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Shapes = value.Shapes[:2] }},
		{name: "shape drift", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Shapes[1].Calls-- }},
		{name: "batch drift", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Shapes[0].IndexesPerSample-- }},
		{name: "missing sample", mutate: func(value *buildmeasure.AtlasCallRetainedReport) {
			value.Shapes[2].Samples = value.Shapes[2].Samples[:4]
		}},
		{name: "zero heap delta", mutate: func(value *buildmeasure.AtlasCallRetainedReport) {
			value.Shapes[0].Samples[0].RetainedHeapAllocBytes = 0
		}},
		{name: "zero object delta", mutate: func(value *buildmeasure.AtlasCallRetainedReport) { value.Shapes[0].Samples[0].RetainedHeapObjects = 0 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validAtlasCallRetainedReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalAtlasCallRetained(report); err == nil {
				t.Fatal("invalid Atlas call retained-heap report accepted")
			}
		})
	}
}

func validAtlasCallRetainedReport() buildmeasure.AtlasCallRetainedReport {
	report := buildmeasure.AtlasCallRetainedReport{
		Schema: buildmeasure.AtlasCallRetainedSchema, Workload: "generated-star-workspace-call-retained-index-batches-v0",
		Scope:              "post-forced-gc-process-live-heap-delta-for-retained-workspace-call-index-batches;prepared-inputs-kept-live;workspace-generation-loading-compilation-and-map-planning-excluded",
		MemoryMetric:       "runtime-memstats-heapalloc-and-heapobjects-post-forced-gc-delta;process-wide;not-atlas-only",
		Setup:              "deterministic-generated-star-workspaces;prepare-and-warm-before-sampling;forced-gc-before-baseline-and-after-batch;first-observations-only",
		Decision:           "retained-heap-control-only-no-per-index-budget-default-traversal-impact-or-regression-claim",
		IterationsPerShape: buildmeasure.AtlasCallRetainedIterationsPerShape,
		CompilerBuild:      "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		LogicalSchema:      "fairyc.fymap-logical/v1",
		Environment:        buildmeasure.AtlasCallScaleEnvironment{OS: "windows", Arch: "amd64"},
	}
	for shapeIndex, definition := range buildmeasure.AtlasCallScaleDefinitions() {
		shape := buildmeasure.AtlasCallRetainedShape{
			Name: definition.Name, Packages: definition.Packages, Calls: definition.Calls,
			MapRecords: uint64(40 * (shapeIndex + 1)), IndexesPerSample: buildmeasure.AtlasCallRetainedIndexesPerSample,
		}
		for sampleIndex := 0; sampleIndex < buildmeasure.AtlasCallRetainedIterationsPerShape; sampleIndex++ {
			shape.Samples = append(shape.Samples, buildmeasure.AtlasCallRetainedSample{
				Sequence: sampleIndex + 1, RetainedHeapAllocBytes: uint64(10_000 + shapeIndex*1_000 + sampleIndex),
				RetainedHeapObjects: uint64(100 + shapeIndex*10 + sampleIndex),
			})
		}
		report.Shapes = append(report.Shapes, shape)
	}
	return report
}
