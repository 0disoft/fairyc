package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	AtlasCallRetainedSchema             = "fairyc.atlas-call-retained-heap-measurement/v0"
	AtlasCallRetainedIterationsPerShape = 5
	AtlasCallRetainedIndexesPerSample   = 16
)

type AtlasCallRetainedSample struct {
	Sequence               int    `json:"sequence"`
	RetainedHeapAllocBytes uint64 `json:"retainedHeapAllocBytes"`
	RetainedHeapObjects    uint64 `json:"retainedHeapObjects"`
}

type AtlasCallRetainedShape struct {
	Name             string                    `json:"name"`
	Packages         int                       `json:"packages"`
	Calls            int                       `json:"calls"`
	MapRecords       uint64                    `json:"mapRecords"`
	IndexesPerSample int                       `json:"indexesPerSample"`
	Samples          []AtlasCallRetainedSample `json:"samples"`
}

// AtlasCallRetainedReport records forced-GC process live-heap deltas for batches of retained
// workspace call indexes. It is a local structural control, not exact Atlas ownership or a budget.
type AtlasCallRetainedReport struct {
	Schema             string                    `json:"schema"`
	Workload           string                    `json:"workload"`
	Scope              string                    `json:"scope"`
	MemoryMetric       string                    `json:"memoryMetric"`
	Setup              string                    `json:"setup"`
	Decision           string                    `json:"decision"`
	IterationsPerShape int                       `json:"iterationsPerShape"`
	CompilerBuild      string                    `json:"compilerBuild"`
	LogicalSchema      string                    `json:"logicalSchema"`
	Environment        AtlasCallScaleEnvironment `json:"environment"`
	Shapes             []AtlasCallRetainedShape  `json:"shapes"`
}

func MarshalAtlasCallRetained(report AtlasCallRetainedReport) ([]byte, error) {
	if err := ValidateAtlasCallRetained(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode Atlas call retained-heap measurement: %w", err)
	}
	return encoded, nil
}

func ValidateAtlasCallRetained(report AtlasCallRetainedReport) error {
	if report.Schema != AtlasCallRetainedSchema || report.Workload != "generated-star-workspace-call-retained-index-batches-v0" {
		return fmt.Errorf("Atlas call retained-heap measurement has an unsupported schema or workload")
	}
	if report.Scope != "post-forced-gc-process-live-heap-delta-for-retained-workspace-call-index-batches;prepared-inputs-kept-live;workspace-generation-loading-compilation-and-map-planning-excluded" ||
		report.MemoryMetric != "runtime-memstats-heapalloc-and-heapobjects-post-forced-gc-delta;process-wide;not-atlas-only" ||
		report.Setup != "deterministic-generated-star-workspaces;prepare-and-warm-before-sampling;forced-gc-before-baseline-and-after-batch;first-observations-only" ||
		report.Decision != "retained-heap-control-only-no-per-index-budget-default-traversal-impact-or-regression-claim" ||
		report.IterationsPerShape != AtlasCallRetainedIterationsPerShape {
		return fmt.Errorf("Atlas call retained-heap measurement boundary is incompatible")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.LogicalSchema != "fairyc.fymap-logical/v1" || report.Environment.OS == "" || report.Environment.Arch == "" {
		return fmt.Errorf("Atlas call retained-heap measurement compatibility or environment is incomplete")
	}
	definitions := AtlasCallScaleDefinitions()
	if len(report.Shapes) != len(definitions) {
		return fmt.Errorf("Atlas call retained-heap measurement requires %d shapes", len(definitions))
	}
	for shapeIndex, shape := range report.Shapes {
		expected := definitions[shapeIndex]
		if shape.Name != expected.Name || shape.Packages != expected.Packages || shape.Calls != expected.Calls ||
			shape.MapRecords == 0 || shape.IndexesPerSample != AtlasCallRetainedIndexesPerSample {
			return fmt.Errorf("Atlas call retained-heap shape %d is incompatible", shapeIndex+1)
		}
		if len(shape.Samples) != report.IterationsPerShape {
			return fmt.Errorf("Atlas call retained-heap shape %q requires %d samples", shape.Name, report.IterationsPerShape)
		}
		for sampleIndex, sample := range shape.Samples {
			if sample.Sequence != sampleIndex+1 || sample.RetainedHeapAllocBytes == 0 || sample.RetainedHeapObjects == 0 {
				return fmt.Errorf("Atlas call retained-heap shape %q sample %d is incomplete", shape.Name, sampleIndex+1)
			}
		}
	}
	return nil
}
