package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestAtlasCallScaleReportPreservesClosedRawShapeEvidence(t *testing.T) {
	t.Parallel()
	report := validAtlasCallScaleReport()
	first, err := buildmeasure.MarshalAtlasCallScale(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalAtlasCallScale(report)
	if err != nil {
		t.Fatal(err)
	}
	text := string(first)
	if text != string(second) || !strings.HasPrefix(text, `{"schema":"fairyc.atlas-call-scale-measurement/v0"`) ||
		strings.Contains(text, `C:\`) || strings.Contains(text, "src/main.fy") ||
		strings.Contains(text, `"speedup"`) || strings.Contains(text, "percentile") {
		t.Fatalf("Atlas call scale JSON = %s", first)
	}
}

func TestAtlasCallScaleDefinitionsAreClosedAndDetached(t *testing.T) {
	t.Parallel()
	definitions := buildmeasure.AtlasCallScaleDefinitions()
	if len(definitions) != 3 ||
		definitions[0] != (buildmeasure.AtlasCallScaleDefinition{Name: "small-p2-c8", Packages: 2, Calls: 8}) ||
		definitions[1] != (buildmeasure.AtlasCallScaleDefinition{Name: "medium-p8-c64", Packages: 8, Calls: 64}) ||
		definitions[2] != (buildmeasure.AtlasCallScaleDefinition{Name: "large-p32-c512", Packages: 32, Calls: 512}) {
		t.Fatalf("Atlas call scale definitions = %+v", definitions)
	}
	definitions[0].Calls = 1
	if current := buildmeasure.AtlasCallScaleDefinitions(); current[0].Calls != 8 {
		t.Fatalf("Atlas call scale definitions retained caller mutation: %+v", current)
	}
}

func TestAtlasCallScaleReportRejectsDriftAndPolicyClaims(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.AtlasCallScaleReport)
	}{
		{name: "complexity claim", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Decision = "linear-complexity-proven" }},
		{name: "timed compilation", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Scope = "compile-and-join" }},
		{name: "retained-memory claim", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.AllocationMetric = "retained-atlas-heap" }},
		{name: "missing shape", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes = value.Shapes[:2] }},
		{name: "shape name drift", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[1].Name = "medium" }},
		{name: "call count drift", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[2].Calls-- }},
		{name: "non-growing source", mutate: func(value *buildmeasure.AtlasCallScaleReport) {
			value.Shapes[1].SourceBytes = value.Shapes[0].SourceBytes
		}},
		{name: "missing sample", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[0].Samples = value.Shapes[0].Samples[:8] }},
		{name: "clock mismatch", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[0].Samples[0].BelowClockResolution = true }},
		{name: "missing allocation", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[1].Samples[2].TotalAllocBytes = 0 }},
		{name: "output drift", mutate: func(value *buildmeasure.AtlasCallScaleReport) { value.Shapes[2].Samples[4].OutputBytes++ }},
		{name: "non-growing output", mutate: func(value *buildmeasure.AtlasCallScaleReport) {
			for index := range value.Shapes[1].Samples {
				value.Shapes[1].Samples[index].OutputBytes = value.Shapes[0].Samples[0].OutputBytes
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validAtlasCallScaleReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalAtlasCallScale(report); err == nil {
				t.Fatal("invalid Atlas call scale report accepted")
			}
		})
	}
}

func validAtlasCallScaleReport() buildmeasure.AtlasCallScaleReport {
	report := buildmeasure.AtlasCallScaleReport{
		Schema: buildmeasure.AtlasCallScaleSchema, Workload: "generated-star-workspace-call-shapes-v0",
		Scope: "workspace-call-index-construction-plus-outgoing-catalog-encode;workspace-generation-loading-compilation-map-planning-symbol-discovery-excluded",
		Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: 10,
		AllocationMetric:   "runtime-memstats-totalalloc-and-mallocs-delta;process-wide;gc-uncontrolled",
		Setup:              "deterministic-generated-star-workspaces;prepare-and-warm-each-shape-before-sampling;fixed-partitions;first-observations-only",
		Decision:           "scaling-control-only-no-complexity-default-traversal-impact-or-regression-claim",
		IterationsPerShape: buildmeasure.AtlasCallScaleIterationsPerShape,
		CompilerBuild:      "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		LogicalSchema:      "fairyc.fymap-logical/v1", CatalogVersion: "fairyc.workspace-call-catalog/v0",
		CatalogEncoding: "fairyc.workspace-call-catalog-json/v0",
		Environment:     buildmeasure.AtlasCallScaleEnvironment{OS: "windows", Arch: "amd64"},
	}
	definitions := []struct {
		name                    string
		packages, calls         int
		sourceBytes, mapRecords uint64
		outputBytes             uint64
	}{
		{name: "small-p2-c8", packages: 2, calls: 8, sourceBytes: 200, mapRecords: 40, outputBytes: 4_096},
		{name: "medium-p8-c64", packages: 8, calls: 64, sourceBytes: 1_600, mapRecords: 180, outputBytes: 32_768},
		{name: "large-p32-c512", packages: 32, calls: 512, sourceBytes: 12_800, mapRecords: 1_200, outputBytes: 262_144},
	}
	for _, definition := range definitions {
		shape := buildmeasure.AtlasCallScaleShape{
			Name: definition.name, Packages: definition.packages, SourceFiles: definition.packages,
			SourceBytes: definition.sourceBytes, MapRecords: definition.mapRecords,
			Calls: definition.calls, CrossPackageCalls: definition.calls, SelectedCalls: definition.calls,
			Policy: buildmeasure.AtlasCallProcessPolicy{
				MaxPackages: definition.packages, MaxCalls: definition.calls,
				MaxOutputBytes: buildmeasure.AtlasCallScaleMaxOutputBytes,
			},
		}
		for index := 0; index < buildmeasure.AtlasCallScaleIterationsPerShape; index++ {
			shape.Samples = append(shape.Samples, buildmeasure.AtlasCallScaleSample{
				Sequence: index + 1, DurationNS: int64(20 + index), TotalAllocBytes: uint64(1_000 + index),
				Mallocs: uint64(100 + index), OutputBytes: definition.outputBytes,
			})
		}
		report.Shapes = append(report.Shapes, shape)
	}
	return report
}
