package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	AtlasCallScaleSchema             = "fairyc.atlas-call-scale-measurement/v0"
	AtlasCallScaleIterationsPerShape = 9
	AtlasCallScaleMaxOutputBytes     = uint64(64 << 20)
)

type AtlasCallScaleDefinition struct {
	Name     string
	Packages int
	Calls    int
}

var atlasCallScaleShapes = []AtlasCallScaleDefinition{
	{Name: "small-p2-c8", Packages: 2, Calls: 8},
	{Name: "medium-p8-c64", Packages: 8, Calls: 64},
	{Name: "large-p32-c512", Packages: 32, Calls: 512},
}

// AtlasCallScaleDefinitions returns the detached closed workload shapes owned by this schema.
func AtlasCallScaleDefinitions() []AtlasCallScaleDefinition {
	return append([]AtlasCallScaleDefinition(nil), atlasCallScaleShapes...)
}

type AtlasCallScaleEnvironment struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// AtlasCallScaleSample records one complete index construction plus exact outgoing catalog encode.
// Allocation counters are process-wide runtime deltas, not retained heap or an Atlas-only profile.
type AtlasCallScaleSample struct {
	Sequence             int    `json:"sequence"`
	DurationNS           int64  `json:"durationNs"`
	BelowClockResolution bool   `json:"belowClockResolution"`
	TotalAllocBytes      uint64 `json:"totalAllocBytes"`
	Mallocs              uint64 `json:"mallocs"`
	OutputBytes          uint64 `json:"outputBytes"`
}

type AtlasCallScaleShape struct {
	Name              string                 `json:"name"`
	Packages          int                    `json:"packages"`
	SourceFiles       int                    `json:"sourceFiles"`
	SourceBytes       uint64                 `json:"sourceBytes"`
	MapRecords        uint64                 `json:"mapRecords"`
	Calls             int                    `json:"calls"`
	CrossPackageCalls int                    `json:"crossPackageCalls"`
	SelectedCalls     int                    `json:"selectedCalls"`
	Policy            AtlasCallProcessPolicy `json:"policy"`
	Samples           []AtlasCallScaleSample `json:"samples"`
}

// AtlasCallScaleReport preserves three fixed generated workload distributions without selecting a
// product default, complexity law, traversal, impact behavior, reference machine, or regression gate.
type AtlasCallScaleReport struct {
	Schema             string                    `json:"schema"`
	Workload           string                    `json:"workload"`
	Scope              string                    `json:"scope"`
	Clock              string                    `json:"clock"`
	ClockResolutionNS  int64                     `json:"clockResolutionNs"`
	AllocationMetric   string                    `json:"allocationMetric"`
	Setup              string                    `json:"setup"`
	Decision           string                    `json:"decision"`
	IterationsPerShape int                       `json:"iterationsPerShape"`
	CompilerBuild      string                    `json:"compilerBuild"`
	LogicalSchema      string                    `json:"logicalSchema"`
	CatalogVersion     string                    `json:"catalogVersion"`
	CatalogEncoding    string                    `json:"catalogEncoding"`
	Environment        AtlasCallScaleEnvironment `json:"environment"`
	Shapes             []AtlasCallScaleShape     `json:"shapes"`
}

func MarshalAtlasCallScale(report AtlasCallScaleReport) ([]byte, error) {
	if err := ValidateAtlasCallScale(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode Atlas call scale measurement: %w", err)
	}
	return encoded, nil
}

func ValidateAtlasCallScale(report AtlasCallScaleReport) error {
	if report.Schema != AtlasCallScaleSchema || report.Workload != "generated-star-workspace-call-shapes-v0" {
		return fmt.Errorf("Atlas call scale measurement has an unsupported schema or workload")
	}
	if report.Scope != "workspace-call-index-construction-plus-outgoing-catalog-encode;workspace-generation-loading-compilation-map-planning-symbol-discovery-excluded" ||
		report.Clock != "process-monotonic-wall-nanoseconds" || report.ClockResolutionNS <= 0 ||
		report.AllocationMetric != "runtime-memstats-totalalloc-and-mallocs-delta;process-wide;gc-uncontrolled" ||
		report.Setup != "deterministic-generated-star-workspaces;prepare-and-warm-each-shape-before-sampling;fixed-partitions;first-observations-only" ||
		report.Decision != "scaling-control-only-no-complexity-default-traversal-impact-or-regression-claim" ||
		report.IterationsPerShape != AtlasCallScaleIterationsPerShape {
		return fmt.Errorf("Atlas call scale measurement boundary is incompatible")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.LogicalSchema != "fairyc.fymap-logical/v1" ||
		report.CatalogVersion != "fairyc.workspace-call-catalog/v0" ||
		report.CatalogEncoding != "fairyc.workspace-call-catalog-json/v0" {
		return fmt.Errorf("Atlas call scale measurement compatibility identity is incomplete")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" {
		return fmt.Errorf("Atlas call scale measurement environment is incomplete")
	}
	if len(report.Shapes) != len(atlasCallScaleShapes) {
		return fmt.Errorf("Atlas call scale measurement requires %d shapes", len(atlasCallScaleShapes))
	}
	var previousSourceBytes, previousMapRecords, previousOutputBytes uint64
	for shapeIndex, shape := range report.Shapes {
		expected := atlasCallScaleShapes[shapeIndex]
		if shape.Name != expected.Name || shape.Packages != expected.Packages || shape.SourceFiles != shape.Packages ||
			shape.SourceBytes == 0 || shape.MapRecords == 0 || shape.Calls != expected.Calls ||
			shape.CrossPackageCalls != shape.Calls || shape.SelectedCalls != shape.Calls ||
			shape.Policy.MaxPackages != shape.Packages || shape.Policy.MaxCalls != shape.Calls ||
			shape.Policy.MaxOutputBytes != AtlasCallScaleMaxOutputBytes {
			return fmt.Errorf("Atlas call scale shape %d is incompatible", shapeIndex+1)
		}
		if shapeIndex > 0 && (shape.SourceBytes <= previousSourceBytes || shape.MapRecords <= previousMapRecords) {
			return fmt.Errorf("Atlas call scale shape %d did not increase its fixed denominator", shapeIndex+1)
		}
		if len(shape.Samples) != report.IterationsPerShape {
			return fmt.Errorf("Atlas call scale shape %q requires %d samples", shape.Name, report.IterationsPerShape)
		}
		var outputBytes uint64
		for sampleIndex, sample := range shape.Samples {
			if sample.Sequence != sampleIndex+1 || sample.DurationNS < 0 ||
				sample.BelowClockResolution != (sample.DurationNS < report.ClockResolutionNS) ||
				sample.TotalAllocBytes == 0 || sample.Mallocs == 0 || sample.OutputBytes == 0 ||
				sample.OutputBytes > shape.Policy.MaxOutputBytes {
				return fmt.Errorf("Atlas call scale shape %q sample %d is incomplete", shape.Name, sampleIndex+1)
			}
			if outputBytes == 0 {
				outputBytes = sample.OutputBytes
			} else if sample.OutputBytes != outputBytes {
				return fmt.Errorf("Atlas call scale shape %q sample %d changed canonical output size", shape.Name, sample.Sequence)
			}
		}
		if shapeIndex > 0 && outputBytes <= previousOutputBytes {
			return fmt.Errorf("Atlas call scale shape %q did not increase canonical output size", shape.Name)
		}
		previousSourceBytes, previousMapRecords, previousOutputBytes = shape.SourceBytes, shape.MapRecords, outputBytes
	}
	return nil
}
