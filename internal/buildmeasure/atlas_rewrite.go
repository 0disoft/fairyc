package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	AtlasRewriteSchema     = "fairyc.atlas-rewrite-measurement/v0"
	AtlasRewriteIterations = 9

	atlasRewriteMaximumMapBytes = uint64(1 << 30)
)

// AtlasRewriteEnvironment records only host facts relevant to the in-process target-independent
// encoder. Toolchain, target, paths, environment variables, and source text are intentionally absent.
type AtlasRewriteEnvironment struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// AtlasRewriteShape fixes the input denominator prepared before timing begins.
type AtlasRewriteShape struct {
	Packages    int    `json:"packages"`
	SourceFiles int    `json:"sourceFiles"`
	SourceBytes uint64 `json:"sourceBytes"`
	MapRecords  uint64 `json:"mapRecords"`
}

// AtlasRewriteSample is one complete in-memory canonical rewrite of every prepared package map.
type AtlasRewriteSample struct {
	Sequence             int    `json:"sequence"`
	DurationNS           int64  `json:"durationNs"`
	BelowClockResolution bool   `json:"belowClockResolution"`
	Packages             int    `json:"packages"`
	MapRecords           uint64 `json:"mapRecords"`
	EncodedBytes         uint64 `json:"encodedBytes"`
}

// AtlasRewriteReport preserves raw local observations without selecting a format, cache, storage,
// incremental-update strategy, default limit, or performance threshold.
type AtlasRewriteReport struct {
	Schema                string                  `json:"schema"`
	Workload              string                  `json:"workload"`
	Scope                 string                  `json:"scope"`
	Clock                 string                  `json:"clock"`
	ClockResolutionNS     int64                   `json:"clockResolutionNs"`
	Decision              string                  `json:"decision"`
	Setup                 string                  `json:"setup"`
	Iterations            int                     `json:"iterations"`
	CompilerBuild         string                  `json:"compilerBuild"`
	LogicalSchema         string                  `json:"logicalSchema"`
	TransportSchema       string                  `json:"transportSchema"`
	TransportEncoding     string                  `json:"transportEncoding"`
	MaxMapBytesPerPackage uint64                  `json:"maxMapBytesPerPackage"`
	Environment           AtlasRewriteEnvironment `json:"environment"`
	Shape                 AtlasRewriteShape       `json:"shape"`
	Samples               []AtlasRewriteSample    `json:"samples"`
}

// MarshalAtlasRewrite validates and emits compact deterministic path-free measurement JSON.
func MarshalAtlasRewrite(report AtlasRewriteReport) ([]byte, error) {
	if err := ValidateAtlasRewrite(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode Atlas rewrite measurement: %w", err)
	}
	return encoded, nil
}

// ValidateAtlasRewrite keeps the measured phase, denominator, and no-policy conclusion exact.
func ValidateAtlasRewrite(report AtlasRewriteReport) error {
	if report.Schema != AtlasRewriteSchema || report.Workload != "workspace_quickstart" ||
		report.Scope != "canonical-fymap-encode-all-packages;workspace-analysis-map-planning-and-publication-excluded" {
		return fmt.Errorf("Atlas rewrite measurement requires its closed schema, workload, and scope")
	}
	if report.Clock != "process-monotonic-wall-nanoseconds" || report.ClockResolutionNS <= 0 ||
		report.Decision != "raw-distribution-only-no-format-or-incremental-policy" ||
		report.Setup != "one-untimed-warmup;fixed-preplanned-partitions;gc-uncontrolled" ||
		report.Iterations != AtlasRewriteIterations {
		return fmt.Errorf("Atlas rewrite measurement timing policy is incompatible")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.LogicalSchema == "" || report.TransportSchema == "" || report.TransportEncoding == "" ||
		report.MaxMapBytesPerPackage == 0 || report.MaxMapBytesPerPackage > atlasRewriteMaximumMapBytes {
		return fmt.Errorf("Atlas rewrite measurement compatibility or byte bounds are incomplete")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" {
		return fmt.Errorf("Atlas rewrite measurement environment is incomplete")
	}
	if report.Shape.Packages <= 0 || report.Shape.SourceFiles <= 0 || report.Shape.SourceBytes == 0 || report.Shape.MapRecords == 0 {
		return fmt.Errorf("Atlas rewrite measurement workload shape is incomplete")
	}
	if len(report.Samples) != report.Iterations {
		return fmt.Errorf("Atlas rewrite measurement requires %d samples", report.Iterations)
	}
	var encodedBytes uint64
	for index, sample := range report.Samples {
		if sample.Sequence != index+1 || sample.DurationNS < 0 ||
			sample.BelowClockResolution != (sample.DurationNS < report.ClockResolutionNS) {
			return fmt.Errorf("Atlas rewrite sample %d has invalid sequence or clock evidence", index+1)
		}
		if sample.Packages != report.Shape.Packages || sample.MapRecords != report.Shape.MapRecords || sample.EncodedBytes == 0 {
			return fmt.Errorf("Atlas rewrite sample %d changed the fixed workload", sample.Sequence)
		}
		if uint64(sample.Packages) > ^uint64(0)/report.MaxMapBytesPerPackage ||
			sample.EncodedBytes > uint64(sample.Packages)*report.MaxMapBytesPerPackage {
			return fmt.Errorf("Atlas rewrite sample %d exceeds the declared package byte bounds", sample.Sequence)
		}
		if encodedBytes == 0 {
			encodedBytes = sample.EncodedBytes
		} else if sample.EncodedBytes != encodedBytes {
			return fmt.Errorf("Atlas rewrite sample %d changed canonical encoded size", sample.Sequence)
		}
	}
	return nil
}
