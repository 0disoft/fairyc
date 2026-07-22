package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	CacheEditSchema = "fairyc.process-cache-edit-measurement/v0"

	CacheEditNoOpWarm          = "no-op-warm"
	CacheEditPrivateRoot       = "private-root-edit"
	CacheEditPrivateDependency = "private-dependency-edit"
	CacheEditPublicDependency  = "public-dependency-edit"

	CacheEditIterationsPerCondition = 9
)

var cacheEditConditions = []string{
	CacheEditNoOpWarm,
	CacheEditPrivateRoot,
	CacheEditPrivateDependency,
	CacheEditPublicDependency,
}

// CacheEditSample is one independent fresh-Session observation after one untimed warmup request.
type CacheEditSample struct {
	Sequence             int    `json:"sequence"`
	Condition            string `json:"condition"`
	Iteration            int    `json:"iteration"`
	DurationNS           int64  `json:"durationNs"`
	BelowClockResolution bool   `json:"belowClockResolution"`
	Outcome              string `json:"outcome"`
	Packages             int    `json:"packages"`
	Hits                 int    `json:"hits"`
	Misses               int    `json:"misses"`
	Rejected             int    `json:"rejected"`
	Stored               int    `json:"stored"`
	StoreSkipped         int    `json:"storeSkipped"`
	Lowered              bool   `json:"lowered"`
	ReassemblyFallback   bool   `json:"reassemblyFallback"`
	MIRFunctions         int    `json:"mirFunctions"`
	InterpreterResult    int    `json:"interpreterResult"`
	RetainedEntries      int    `json:"retainedEntries"`
	RetainedBytes        uint64 `json:"retainedBytes"`
	VerifiedHits         uint64 `json:"verifiedHits"`
}

// CacheEditReport preserves raw edit-class observations without calculating a preferred summary.
type CacheEditReport struct {
	Schema                 string            `json:"schema"`
	Workload               string            `json:"workload"`
	Scope                  string            `json:"scope"`
	Clock                  string            `json:"clock"`
	ClockResolutionNS      int64             `json:"clockResolutionNs"`
	CacheImplementation    string            `json:"cacheImplementation"`
	Decision               string            `json:"decision"`
	Setup                  string            `json:"setup"`
	WarmupRequests         int               `json:"warmupRequests"`
	IterationsPerCondition int               `json:"iterationsPerCondition"`
	Environment            CacheEnvironment  `json:"environment"`
	Policy                 CachePolicy       `json:"policy"`
	Samples                []CacheEditSample `json:"samples"`
}

// MarshalCacheEdits validates and emits deterministic path-free raw distribution JSON.
func MarshalCacheEdits(report CacheEditReport) ([]byte, error) {
	if err := ValidateCacheEdits(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode process-cache edit measurement: %w", err)
	}
	return encoded, nil
}

// ValidateCacheEdits freezes the comparable setup and expected invalidation path for each edit.
func ValidateCacheEdits(report CacheEditReport) error {
	if report.Schema != CacheEditSchema {
		return fmt.Errorf("process-cache edit measurement schema must be %q", CacheEditSchema)
	}
	if report.Workload != "workspace_quickstart" || report.Scope != "semantic-through-verified-mir" || report.Clock != "process-monotonic-wall-nanoseconds" {
		return fmt.Errorf("process-cache edit measurement requires its closed workload, phase scope, and clock")
	}
	if report.ClockResolutionNS <= 0 {
		return fmt.Errorf("process-cache edit measurement requires a positive observed clock resolution")
	}
	if report.CacheImplementation != "bounded-process-package-mir-v0" || report.Decision != "raw-distributions-only-no-speedup-claim" {
		return fmt.Errorf("process-cache edit measurement has an unsupported implementation or decision")
	}
	if report.Setup != "fresh-session-per-sample;edit-and-load-outside-timer" || report.WarmupRequests != 1 || report.IterationsPerCondition != CacheEditIterationsPerCondition {
		return fmt.Errorf("process-cache edit measurement setup is incompatible")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.Target == "" {
		return fmt.Errorf("process-cache edit measurement environment is incomplete")
	}
	if report.Policy.MaxEntries <= 0 || report.Policy.MaxBytes == 0 || report.Policy.MaxArtifactBytes == 0 {
		return fmt.Errorf("process-cache edit measurement policy requires positive bounds")
	}
	expectedSamples := len(cacheEditConditions) * CacheEditIterationsPerCondition
	if len(report.Samples) != expectedSamples {
		return fmt.Errorf("process-cache edit measurement requires %d samples", expectedSamples)
	}
	for index, sample := range report.Samples {
		conditionIndex := index / CacheEditIterationsPerCondition
		iteration := index%CacheEditIterationsPerCondition + 1
		if sample.Sequence != index+1 || sample.Condition != cacheEditConditions[conditionIndex] || sample.Iteration != iteration {
			return fmt.Errorf("process-cache edit measurement sample %d is not in canonical condition order", index+1)
		}
		expectedFunctions := 2
		if sample.Condition == CacheEditPublicDependency {
			expectedFunctions = 3
		}
		if sample.DurationNS < 0 || sample.BelowClockResolution != (sample.DurationNS < report.ClockResolutionNS) || sample.Packages != 2 || sample.MIRFunctions != expectedFunctions || sample.InterpreterResult != 42 {
			return fmt.Errorf(
				"process-cache edit measurement sample %d has incomplete outcome data: durationNs=%d belowClockResolution=%t packages=%d mirFunctions=%d interpreterResult=%d",
				sample.Sequence, sample.DurationNS, sample.BelowClockResolution, sample.Packages, sample.MIRFunctions, sample.InterpreterResult,
			)
		}
		if sample.Rejected != 0 || sample.StoreSkipped != 0 || sample.ReassemblyFallback || sample.RetainedBytes == 0 {
			return fmt.Errorf("process-cache edit measurement sample %d did not follow the clean comparison path", sample.Sequence)
		}
		if sample.RetainedEntries <= 0 || sample.RetainedEntries > report.Policy.MaxEntries || sample.RetainedBytes > report.Policy.MaxBytes {
			return fmt.Errorf("process-cache edit measurement sample %d exceeds retained-state bounds", sample.Sequence)
		}
		if err := validateCacheEditOutcome(sample); err != nil {
			return fmt.Errorf("process-cache edit measurement sample %d: %w", sample.Sequence, err)
		}
	}
	return nil
}

func validateCacheEditOutcome(sample CacheEditSample) error {
	switch sample.Condition {
	case CacheEditNoOpWarm:
		if sample.Outcome != "all-hit" || sample.Hits != 2 || sample.Misses != 0 || sample.Stored != 0 || sample.Lowered || sample.VerifiedHits != 2 || sample.RetainedEntries != 2 {
			return fmt.Errorf("no-op sample does not prove complete verified reuse")
		}
	case CacheEditPrivateRoot:
		if sample.Outcome != "fallback" || sample.Hits != 1 || sample.Misses != 1 || sample.Stored != 2 || !sample.Lowered || sample.VerifiedHits != 1 || sample.RetainedEntries != 3 {
			return fmt.Errorf("private root edit does not prove one-package invalidation")
		}
	case CacheEditPrivateDependency, CacheEditPublicDependency:
		if sample.Outcome != "fallback" || sample.Hits != 0 || sample.Misses != 2 || sample.Stored != 2 || !sample.Lowered || sample.VerifiedHits != 0 || sample.RetainedEntries != 4 {
			return fmt.Errorf("dependency edit does not prove dependency-and-dependent invalidation")
		}
	default:
		return fmt.Errorf("unknown edit condition %q", sample.Condition)
	}
	return nil
}
