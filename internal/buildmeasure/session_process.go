package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	SessionProcessSchema = "fairyc.workspace-session-process-measurement/v0"

	SessionProcessCold              = "cold"
	SessionProcessNoOp              = "no-op"
	SessionProcessPrivateRoot       = "private-root-edit"
	SessionProcessNoOpAfterRoot     = "no-op-after-root-edit"
	SessionProcessPrivateDependency = "private-dependency-edit"
	SessionProcessNoOpAfterPrivate  = "no-op-after-private-dependency-edit"
	SessionProcessPublicDependency  = "public-dependency-edit"
	SessionProcessNoOpAfterPublic   = "no-op-after-public-dependency-edit"
)

var sessionProcessConditions = []string{
	SessionProcessCold,
	SessionProcessNoOp,
	SessionProcessPrivateRoot,
	SessionProcessNoOpAfterRoot,
	SessionProcessPrivateDependency,
	SessionProcessNoOpAfterPrivate,
	SessionProcessPublicDependency,
	SessionProcessNoOpAfterPublic,
}

// SessionProcessMemory records process-reported Windows peaks. Working set is resident memory;
// pagefile usage is committed pagefile-backed memory. Neither is a Go live-heap measurement.
type SessionProcessMemory struct {
	PeakWorkingSetBytes uint64 `json:"peakWorkingSetBytes"`
	PeakPagefileBytes   uint64 `json:"peakPagefileBytes"`
}

// SessionProcessSample is one caller-observed request/response round trip. File mutation happens
// before the clock starts and is excluded from DurationNS.
type SessionProcessSample struct {
	Sequence        int    `json:"sequence"`
	Condition       string `json:"condition"`
	DurationNS      int64  `json:"durationNs"`
	Outcome         string `json:"outcome"`
	Hits            int    `json:"hits"`
	Misses          int    `json:"misses"`
	Lowered         bool   `json:"lowered"`
	MIRFunctions    int    `json:"mirFunctions"`
	RetainedEntries int    `json:"retainedEntries"`
	RetainedBytes   uint64 `json:"retainedBytes"`
	VerifiedHits    uint64 `json:"verifiedHits"`
}

// SessionProcessReport preserves one local editor-shaped process observation without selecting a
// reference machine, budget, percentile, retry, default cache policy, or speedup conclusion.
type SessionProcessReport struct {
	Schema              string                 `json:"schema"`
	Workload            string                 `json:"workload"`
	Scope               string                 `json:"scope"`
	Clock               string                 `json:"clock"`
	Setup               string                 `json:"setup"`
	RetryPolicy         string                 `json:"retryPolicy"`
	MemoryMetric        string                 `json:"memoryMetric"`
	CacheImplementation string                 `json:"cacheImplementation"`
	Decision            string                 `json:"decision"`
	Environment         CacheEnvironment       `json:"environment"`
	Policy              CachePolicy            `json:"policy"`
	Memory              SessionProcessMemory   `json:"memory"`
	Samples             []SessionProcessSample `json:"samples"`
}

func MarshalSessionProcess(report SessionProcessReport) ([]byte, error) {
	if err := ValidateSessionProcess(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode workspace-session process measurement: %w", err)
	}
	return encoded, nil
}

func ValidateSessionProcess(report SessionProcessReport) error {
	if report.Schema != SessionProcessSchema || report.Workload != "workspace_quickstart-editor-sequence" {
		return fmt.Errorf("workspace-session process measurement has an unsupported schema or workload")
	}
	if report.Scope != "request-write-through-response-read;mutation-excluded" || report.Clock != "caller-monotonic-wall-nanoseconds" ||
		report.Setup != "one-standalone-fairyc-process;sequential-requests" || report.RetryPolicy != "none;preserve-first-observation" {
		return fmt.Errorf("workspace-session process measurement boundaries are incompatible")
	}
	if report.MemoryMetric != "windows-process-memory-counters-peak-working-set-and-pagefile-bytes" ||
		report.CacheImplementation != "bounded-process-package-mir-v0" ||
		report.Decision != "local-observation-only-no-reference-budget-or-speedup-claim" {
		return fmt.Errorf("workspace-session process measurement has an unsupported metric or decision")
	}
	if report.Environment.OS != "windows" || report.Environment.Arch != "amd64" || report.Environment.Target != "x86_64-windows-msvc" {
		return fmt.Errorf("workspace-session process measurement requires the Windows amd64 smoke environment")
	}
	if report.Policy.MaxEntries < 8 || report.Policy.MaxBytes == 0 || report.Policy.MaxArtifactBytes == 0 || report.Policy.MaxArtifactBytes > report.Policy.MaxBytes {
		return fmt.Errorf("workspace-session process measurement cache policy is incomplete")
	}
	if report.Memory.PeakWorkingSetBytes == 0 || report.Memory.PeakPagefileBytes == 0 {
		return fmt.Errorf("workspace-session process measurement requires positive process-memory peaks")
	}
	if len(report.Samples) != len(sessionProcessConditions) {
		return fmt.Errorf("workspace-session process measurement requires %d samples", len(sessionProcessConditions))
	}
	for index, sample := range report.Samples {
		if sample.Sequence != index+1 || sample.Condition != sessionProcessConditions[index] || sample.DurationNS < 0 ||
			sample.MIRFunctions <= 0 || sample.RetainedEntries <= 0 || sample.RetainedEntries > report.Policy.MaxEntries ||
			sample.RetainedBytes == 0 || sample.RetainedBytes > report.Policy.MaxBytes {
			return fmt.Errorf("workspace-session process sample %d has incomplete or out-of-bound data", index+1)
		}
		if err := validateSessionProcessOutcome(sample); err != nil {
			return fmt.Errorf("workspace-session process sample %d: %w", index+1, err)
		}
	}
	return nil
}

func validateSessionProcessOutcome(sample SessionProcessSample) error {
	switch sample.Condition {
	case SessionProcessCold:
		if sample.Outcome != "fallback" || sample.Hits != 0 || sample.Misses != 2 || !sample.Lowered || sample.RetainedEntries != 2 {
			return fmt.Errorf("cold request does not prove complete fallback")
		}
	case SessionProcessPrivateRoot:
		if sample.Outcome != "fallback" || sample.Hits != 1 || sample.Misses != 1 || !sample.Lowered || sample.RetainedEntries != 3 {
			return fmt.Errorf("private root edit does not prove one-package invalidation")
		}
	case SessionProcessPrivateDependency:
		if sample.Outcome != "fallback" || sample.Hits != 1 || sample.Misses != 1 || !sample.Lowered || sample.RetainedEntries != 4 {
			return fmt.Errorf("private dependency edit does not prove provider-only invalidation")
		}
	case SessionProcessPublicDependency:
		if sample.Outcome != "fallback" || sample.Hits != 0 || sample.Misses != 2 || !sample.Lowered {
			return fmt.Errorf("public dependency edit does not prove dependency-and-dependent invalidation")
		}
	default:
		if sample.Outcome != "all-hit" || sample.Hits != 2 || sample.Misses != 0 || sample.Lowered {
			return fmt.Errorf("no-op request does not prove complete verified reuse")
		}
	}
	if sample.Condition == SessionProcessPublicDependency || sample.Condition == SessionProcessNoOpAfterPublic {
		if sample.MIRFunctions != 3 {
			return fmt.Errorf("public dependency edit must retain three MIR functions")
		}
	} else if sample.MIRFunctions != 2 {
		return fmt.Errorf("sample must retain two MIR functions before the public edit")
	}
	return nil
}
