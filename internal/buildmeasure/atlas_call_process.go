package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	AtlasCallProcessSchema     = "fairyc.atlas-call-process-measurement/v0"
	AtlasCallProcessIterations = 9

	atlasCallProcessMaximumCount = 65_536
	atlasCallProcessMaximumBytes = uint64(1 << 30)
)

// AtlasCallProcessPolicy records the explicit construction and final-output bounds used by every
// fresh CLI process. These are measurement inputs, not recommended defaults.
type AtlasCallProcessPolicy struct {
	MaxPackages    int    `json:"maxPackages"`
	MaxCalls       int    `json:"maxCalls"`
	MaxOutputBytes uint64 `json:"maxOutputBytes"`
}

// AtlasCallProcessEnvironment fixes the host identity for Windows process-memory observations.
type AtlasCallProcessEnvironment struct {
	OS   string `json:"os"`
	Arch string `json:"arch"`
}

// AtlasCallProcessShape fixes the source and joined-call denominator observed during warmup.
type AtlasCallProcessShape struct {
	Packages          int    `json:"packages"`
	SourceFiles       int    `json:"sourceFiles"`
	SourceBytes       uint64 `json:"sourceBytes"`
	Calls             int    `json:"calls"`
	CrossPackageCalls int    `json:"crossPackageCalls"`
	SelectedCalls     int    `json:"selectedCalls"`
}

// AtlasCallProcessSample is one fresh locked standalone compiler process. Windows peak working-set
// and pagefile values are whole-process counters, not a Go live-heap or index-only measurement.
type AtlasCallProcessSample struct {
	Sequence             int    `json:"sequence"`
	DurationNS           int64  `json:"durationNs"`
	BelowClockResolution bool   `json:"belowClockResolution"`
	PeakWorkingSetBytes  uint64 `json:"peakWorkingSetBytes"`
	PeakPagefileBytes    uint64 `json:"peakPagefileBytes"`
	OutputBytes          uint64 `json:"outputBytes"`
}

// AtlasCallProcessReport preserves raw local end-to-end CLI observations without selecting a
// reference machine, default budget, percentile, speedup, traversal, or impact policy.
type AtlasCallProcessReport struct {
	Schema            string                      `json:"schema"`
	Workload          string                      `json:"workload"`
	Scope             string                      `json:"scope"`
	Clock             string                      `json:"clock"`
	ClockResolutionNS int64                       `json:"clockResolutionNs"`
	Setup             string                      `json:"setup"`
	RetryPolicy       string                      `json:"retryPolicy"`
	MemoryMetric      string                      `json:"memoryMetric"`
	Decision          string                      `json:"decision"`
	Iterations        int                         `json:"iterations"`
	CompilerBuild     string                      `json:"compilerBuild"`
	CatalogVersion    string                      `json:"catalogVersion"`
	CatalogEncoding   string                      `json:"catalogEncoding"`
	Environment       AtlasCallProcessEnvironment `json:"environment"`
	Policy            AtlasCallProcessPolicy      `json:"policy"`
	Shape             AtlasCallProcessShape       `json:"shape"`
	Samples           []AtlasCallProcessSample    `json:"samples"`
}

func MarshalAtlasCallProcess(report AtlasCallProcessReport) ([]byte, error) {
	if err := ValidateAtlasCallProcess(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode Atlas call process measurement: %w", err)
	}
	return encoded, nil
}

func ValidateAtlasCallProcess(report AtlasCallProcessReport) error {
	if report.Schema != AtlasCallProcessSchema || report.Workload != "workspace_quickstart-root-main-outgoing" {
		return fmt.Errorf("Atlas call process measurement has an unsupported schema or workload")
	}
	if report.Scope != "caller-command-start-through-exit;compiler-build-symbol-discovery-and-warmup-excluded" ||
		report.Clock != "caller-monotonic-wall-nanoseconds" || report.ClockResolutionNS <= 0 ||
		report.Setup != "one-untimed-warmup;fresh-standalone-fairyc-process-per-sample;locked-workspace;os-caches-uncontrolled" ||
		report.RetryPolicy != "none;preserve-first-observation" || report.Iterations != AtlasCallProcessIterations {
		return fmt.Errorf("Atlas call process measurement timing boundary is incompatible")
	}
	if report.MemoryMetric != "windows-process-memory-counters-peak-working-set-and-pagefile-bytes" ||
		report.Decision != "local-control-only-no-reference-budget-speedup-traversal-or-impact-claim" {
		return fmt.Errorf("Atlas call process measurement metric or decision is incompatible")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.CatalogVersion != "fairyc.workspace-call-catalog/v0" ||
		report.CatalogEncoding != "fairyc.workspace-call-catalog-json/v0" {
		return fmt.Errorf("Atlas call process measurement catalog identity is incompatible")
	}
	if report.Environment.OS != "windows" || report.Environment.Arch != "amd64" {
		return fmt.Errorf("Atlas call process measurement requires the Windows amd64 smoke environment")
	}
	if report.Policy.MaxPackages <= 0 || report.Policy.MaxPackages > atlasCallProcessMaximumCount ||
		report.Policy.MaxCalls <= 0 || report.Policy.MaxCalls > atlasCallProcessMaximumCount ||
		report.Policy.MaxOutputBytes == 0 || report.Policy.MaxOutputBytes > atlasCallProcessMaximumBytes {
		return fmt.Errorf("Atlas call process measurement policy is incomplete or out of bounds")
	}
	shape := report.Shape
	if shape.Packages <= 0 || shape.Packages > report.Policy.MaxPackages || shape.SourceFiles <= 0 || shape.SourceBytes == 0 ||
		shape.Calls <= 0 || shape.Calls > report.Policy.MaxCalls || shape.CrossPackageCalls <= 0 ||
		shape.CrossPackageCalls > shape.Calls || shape.SelectedCalls <= 0 || shape.SelectedCalls > shape.Calls {
		return fmt.Errorf("Atlas call process measurement workload shape is incomplete or out of bounds")
	}
	if len(report.Samples) != report.Iterations {
		return fmt.Errorf("Atlas call process measurement requires %d samples", report.Iterations)
	}
	var outputBytes uint64
	for index, sample := range report.Samples {
		if sample.Sequence != index+1 || sample.DurationNS < 0 ||
			sample.BelowClockResolution != (sample.DurationNS < report.ClockResolutionNS) ||
			sample.PeakWorkingSetBytes == 0 || sample.PeakPagefileBytes == 0 || sample.OutputBytes == 0 ||
			sample.OutputBytes > report.Policy.MaxOutputBytes {
			return fmt.Errorf("Atlas call process sample %d is incomplete or out of bounds", index+1)
		}
		if outputBytes == 0 {
			outputBytes = sample.OutputBytes
		} else if sample.OutputBytes != outputBytes {
			return fmt.Errorf("Atlas call process sample %d changed canonical output size", sample.Sequence)
		}
	}
	return nil
}
