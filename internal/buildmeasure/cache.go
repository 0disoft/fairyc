package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	CacheSchema        = "fairyc.process-cache-measurement/v0"
	CacheConditionCold = "cold-fallback"
	CacheConditionWarm = "warm-all-hit"
)

// CachePolicy records harness-local bounds. These values are measurement inputs, not product defaults.
type CachePolicy struct {
	MaxEntries       int    `json:"maxEntries"`
	MaxBytes         uint64 `json:"maxBytes"`
	MaxArtifactBytes uint64 `json:"maxArtifactBytes"`
}

// CacheEnvironment names reproducibility facts without claiming a native toolchain participates.
type CacheEnvironment struct {
	OS     string `json:"os"`
	Arch   string `json:"arch"`
	Target string `json:"target"`
}

// CacheSample is one raw semantic-through-verified-MIR observation from the same Session.
type CacheSample struct {
	Sequence           int    `json:"sequence"`
	Condition          string `json:"condition"`
	DurationNS         int64  `json:"durationNs"`
	Outcome            string `json:"outcome"`
	Packages           int    `json:"packages"`
	Hits               int    `json:"hits"`
	Misses             int    `json:"misses"`
	Rejected           int    `json:"rejected"`
	Stored             int    `json:"stored"`
	StoreSkipped       int    `json:"storeSkipped"`
	Lowered            bool   `json:"lowered"`
	ReassemblyFallback bool   `json:"reassemblyFallback"`
	MIRFunctions       int    `json:"mirFunctions"`
	InterpreterResult  int    `json:"interpreterResult"`
	RetainedEntries    int    `json:"retainedEntries"`
	RetainedBytes      uint64 `json:"retainedBytes"`
	VerifiedHits       uint64 `json:"verifiedHits"`
}

// CacheReport preserves two raw observations and explicitly forbids a speedup conclusion.
type CacheReport struct {
	Schema              string           `json:"schema"`
	Workload            string           `json:"workload"`
	Scope               string           `json:"scope"`
	Clock               string           `json:"clock"`
	CacheImplementation string           `json:"cacheImplementation"`
	Decision            string           `json:"decision"`
	Environment         CacheEnvironment `json:"environment"`
	Policy              CachePolicy      `json:"policy"`
	Samples             []CacheSample    `json:"samples"`
}

// MarshalCache validates and emits compact deterministic JSON without host paths or source text.
func MarshalCache(report CacheReport) ([]byte, error) {
	if err := ValidateCache(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode process-cache measurement: %w", err)
	}
	return encoded, nil
}

// ValidateCache proves cache control flow and retained-state bounds, not a timing distribution.
func ValidateCache(report CacheReport) error {
	if report.Schema != CacheSchema {
		return fmt.Errorf("process-cache measurement schema must be %q", CacheSchema)
	}
	if report.Workload == "" || report.Scope != "semantic-through-verified-mir" || report.Clock != "process-monotonic-wall-nanoseconds" {
		return fmt.Errorf("process-cache measurement requires workload, phase scope, and monotonic wall clock semantics")
	}
	if report.CacheImplementation != "bounded-process-package-mir-v0" || report.Decision != "observed-only-no-speedup-claim" {
		return fmt.Errorf("process-cache measurement has an unsupported implementation or decision")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.Target == "" {
		return fmt.Errorf("process-cache measurement environment is incomplete")
	}
	if report.Policy.MaxEntries <= 0 || report.Policy.MaxBytes == 0 || report.Policy.MaxArtifactBytes == 0 {
		return fmt.Errorf("process-cache measurement policy requires positive bounds")
	}
	if len(report.Samples) != 2 {
		return fmt.Errorf("process-cache measurement requires exactly cold and warm samples")
	}
	cold, warm := report.Samples[0], report.Samples[1]
	if cold.Sequence != 1 || cold.Condition != CacheConditionCold || warm.Sequence != 2 || warm.Condition != CacheConditionWarm {
		return fmt.Errorf("process-cache measurement samples are not canonical cold then warm observations")
	}
	for _, sample := range report.Samples {
		if sample.DurationNS <= 0 || sample.Packages <= 0 || sample.MIRFunctions <= 0 || sample.InterpreterResult != 42 {
			return fmt.Errorf("process-cache measurement sample %d has incomplete outcome data", sample.Sequence)
		}
		if sample.Rejected != 0 || sample.StoreSkipped != 0 || sample.ReassemblyFallback {
			return fmt.Errorf("process-cache measurement sample %d did not follow the clean comparison path", sample.Sequence)
		}
		if sample.RetainedEntries < 0 || sample.RetainedEntries > report.Policy.MaxEntries || sample.RetainedBytes > report.Policy.MaxBytes {
			return fmt.Errorf("process-cache measurement sample %d exceeds retained-state bounds", sample.Sequence)
		}
	}
	if cold.Outcome != "fallback" || cold.Hits != 0 || cold.Misses != cold.Packages || cold.Stored != cold.Packages || !cold.Lowered || cold.VerifiedHits != 0 {
		return fmt.Errorf("process-cache cold sample does not prove complete fallback and population")
	}
	if warm.Packages != cold.Packages || warm.Outcome != "all-hit" || warm.Hits != warm.Packages || warm.Misses != 0 ||
		warm.Stored != 0 || warm.Lowered || warm.VerifiedHits != uint64(warm.Packages) {
		return fmt.Errorf("process-cache warm sample does not prove complete verified reuse")
	}
	if warm.RetainedEntries < warm.Packages || warm.RetainedBytes == 0 {
		return fmt.Errorf("process-cache warm sample has no retained artifact state")
	}
	return nil
}
