// Package buildmeasure defines the bounded, path-free schema emitted by configured build probes.
package buildmeasure

import (
	"encoding/json"
	"fmt"
	"slices"
)

const Schema = "fairyc.build-measurement/v0"

const (
	ConditionOrdinary = "ordinary-no-compiler-cache"
	ConditionLocked   = "locked-repeat-no-compiler-cache"
)

// Environment names only reproducibility facts that are safe to print and retain.
type Environment struct {
	OS        string `json:"os"`
	Arch      string `json:"arch"`
	Target    string `json:"target"`
	Toolchain string `json:"toolchain"`
}

// Sample is one complete package-aware build observation. DurationNS is monotonic elapsed wall
// time within the measuring process; it is not CPU time and must not be compared across machines.
type Sample struct {
	Sequence      int    `json:"sequence"`
	Profile       string `json:"profile"`
	Condition     string `json:"condition"`
	LockState     string `json:"lockState"`
	DurationNS    int64  `json:"durationNs"`
	ArtifactBytes int64  `json:"artifactBytes"`
	NativeResult  int    `json:"nativeResult"`
}

// Report intentionally preserves raw samples and does not calculate a speedup from a two-sample
// local calibration. CacheImplementation remains none until a compiler cache actually serves data.
type Report struct {
	Schema              string      `json:"schema"`
	Workload            string      `json:"workload"`
	Clock               string      `json:"clock"`
	CacheImplementation string      `json:"cacheImplementation"`
	Decision            string      `json:"decision"`
	Environment         Environment `json:"environment"`
	Samples             []Sample    `json:"samples"`
}

// Marshal validates and emits compact deterministic JSON without paths, source, arguments, or
// environment variables.
func Marshal(report Report) ([]byte, error) {
	if err := Validate(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode build measurement: %w", err)
	}
	return encoded, nil
}

// Validate keeps event names, units, denominators, and the no-cache conclusion exact.
func Validate(report Report) error {
	if report.Schema != Schema {
		return fmt.Errorf("build measurement schema must be %q", Schema)
	}
	if report.Workload == "" || report.Clock != "process-monotonic-wall-nanoseconds" {
		return fmt.Errorf("build measurement requires workload and monotonic wall clock semantics")
	}
	if report.CacheImplementation != "none" || report.Decision != "baseline-only" {
		return fmt.Errorf("prototype build measurement must not claim compiler-cache reuse or a comparison decision")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.Target == "" || report.Environment.Toolchain == "" {
		return fmt.Errorf("build measurement environment is incomplete")
	}
	if len(report.Samples) == 0 {
		return fmt.Errorf("build measurement has no samples")
	}

	type pair struct {
		profile   string
		condition string
	}
	seen := make(map[pair]struct{}, len(report.Samples))
	for index, sample := range report.Samples {
		if sample.Sequence != index+1 {
			return fmt.Errorf("build measurement sample sequence must be contiguous from one")
		}
		if !slices.Contains([]string{"low", "optimized"}, sample.Profile) {
			return fmt.Errorf("build measurement sample %d has unsupported profile %q", sample.Sequence, sample.Profile)
		}
		if !slices.Contains([]string{ConditionOrdinary, ConditionLocked}, sample.Condition) {
			return fmt.Errorf("build measurement sample %d has unsupported condition %q", sample.Sequence, sample.Condition)
		}
		if !slices.Contains([]string{"updated", "unchanged", "verified"}, sample.LockState) {
			return fmt.Errorf("build measurement sample %d has unsupported lock state %q", sample.Sequence, sample.LockState)
		}
		if sample.DurationNS <= 0 || sample.ArtifactBytes <= 0 || sample.NativeResult != 42 {
			return fmt.Errorf("build measurement sample %d has incomplete outcome data", sample.Sequence)
		}
		key := pair{profile: sample.Profile, condition: sample.Condition}
		if _, duplicate := seen[key]; duplicate {
			return fmt.Errorf("build measurement repeats profile %q condition %q", sample.Profile, sample.Condition)
		}
		seen[key] = struct{}{}
	}
	for _, profile := range []string{"low", "optimized"} {
		for _, condition := range []string{ConditionOrdinary, ConditionLocked} {
			if _, found := seen[pair{profile: profile, condition: condition}]; !found {
				return fmt.Errorf("build measurement is missing profile %q condition %q", profile, condition)
			}
		}
	}
	return nil
}
