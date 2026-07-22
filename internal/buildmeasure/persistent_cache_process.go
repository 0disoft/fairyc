package buildmeasure

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	PersistentCacheProcessSchema     = "fairyc.persistent-cache-process-measurement/v0"
	PersistentCacheProcessCold       = "cold-fallback"
	PersistentCacheProcessWarm       = "warm-all-hit"
	PersistentCacheProcessIterations = 9
	persistentCacheMaxEntries        = 65_536
	persistentCacheMaxBytes          = 1 << 30
	persistentCacheMaxTempAgeSecs    = 31_536_000
)

// PersistentCachePolicy records caller-selected disk bounds. These are harness inputs, not defaults.
type PersistentCachePolicy struct {
	MaxEntries       int    `json:"maxEntries"`
	MaxBytes         uint64 `json:"maxBytes"`
	MaxRecordBytes   uint64 `json:"maxRecordBytes"`
	MaxArtifactBytes uint64 `json:"maxArtifactBytes"`
	TempMaxAgeSecs   int64  `json:"tempMaxAgeSeconds"`
}

// PersistentCacheProcessSample is one complete standalone compiler process observation. The caller
// starts the timer immediately before process creation and stops it after output capture and exit.
type PersistentCacheProcessSample struct {
	Sequence             int    `json:"sequence"`
	Pair                 int    `json:"pair"`
	Condition            string `json:"condition"`
	DurationNS           int64  `json:"durationNs"`
	BelowClockResolution bool   `json:"belowClockResolution"`
	Outcome              string `json:"outcome"`
	Packages             int    `json:"packages"`
	Hits                 int    `json:"hits"`
	Misses               int    `json:"misses"`
	Rejected             int    `json:"rejected"`
	Stored               int    `json:"stored"`
	StoreSkipped         int    `json:"storeSkipped"`
	StoreFailed          int    `json:"storeFailed"`
	Lowered              bool   `json:"lowered"`
	ReassemblyFallback   bool   `json:"reassemblyFallback"`
	MIRFunctions         int    `json:"mirFunctions"`
	RetainedEntries      int    `json:"retainedEntries"`
	RetainedBytes        uint64 `json:"retainedBytes"`
	RecordHits           uint64 `json:"recordHits"`
	RecordMisses         uint64 `json:"recordMisses"`
	RecordRejected       uint64 `json:"recordRejected"`
	Evictions            uint64 `json:"evictions"`
	VerifiedHits         uint64 `json:"verifiedHits"`
}

// PersistentCacheProcessReport preserves paired raw process observations without selecting a
// preferred sample, summary statistic, default policy, budget, or speedup conclusion.
type PersistentCacheProcessReport struct {
	Schema                 string                         `json:"schema"`
	Workload               string                         `json:"workload"`
	Scope                  string                         `json:"scope"`
	Clock                  string                         `json:"clock"`
	ClockResolutionNS      int64                          `json:"clockResolutionNs"`
	CacheImplementation    string                         `json:"cacheImplementation"`
	Decision               string                         `json:"decision"`
	Setup                  string                         `json:"setup"`
	RetryPolicy            string                         `json:"retryPolicy"`
	IterationsPerCondition int                            `json:"iterationsPerCondition"`
	CompilerBuild          string                         `json:"compilerBuild"`
	Environment            CacheEnvironment               `json:"environment"`
	Policy                 PersistentCachePolicy          `json:"policy"`
	Samples                []PersistentCacheProcessSample `json:"samples"`
}

// MarshalPersistentCacheProcess validates and emits deterministic, path-free raw evidence.
func MarshalPersistentCacheProcess(report PersistentCacheProcessReport) ([]byte, error) {
	if err := ValidatePersistentCacheProcess(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode persistent-cache process measurement: %w", err)
	}
	return encoded, nil
}

// ValidatePersistentCacheProcess proves paired cold population and cross-process verified reuse.
// It deliberately does not compare durations or authorize a performance claim.
func ValidatePersistentCacheProcess(report PersistentCacheProcessReport) error {
	if report.Schema != PersistentCacheProcessSchema {
		return fmt.Errorf("persistent-cache process measurement schema must be %q", PersistentCacheProcessSchema)
	}
	if report.Workload != "workspace_quickstart" ||
		report.Scope != "caller-process-run-through-exit;workspace-and-cache-io-included" ||
		report.Clock != "caller-monotonic-wall-nanoseconds" {
		return fmt.Errorf("persistent-cache process measurement requires its closed workload, scope, and clock")
	}
	if report.ClockResolutionNS <= 0 {
		return fmt.Errorf("persistent-cache process measurement requires a positive observed clock resolution")
	}
	if report.CacheImplementation != "bounded-immutable-disk-package-mir-v1" ||
		report.Decision != "raw-paired-distributions-only-no-speedup-claim" {
		return fmt.Errorf("persistent-cache process measurement has an unsupported implementation or decision")
	}
	if report.Setup != "fresh-cache-per-pair;separate-fairyc-process-per-sample" ||
		report.RetryPolicy != "none;preserve-first-observation" ||
		report.IterationsPerCondition != PersistentCacheProcessIterations {
		return fmt.Errorf("persistent-cache process measurement setup is incompatible")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.Target == "" {
		return fmt.Errorf("persistent-cache process measurement environment is incomplete")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.Policy.MaxEntries <= 0 || report.Policy.MaxEntries > persistentCacheMaxEntries ||
		report.Policy.MaxBytes == 0 || report.Policy.MaxBytes > persistentCacheMaxBytes ||
		report.Policy.MaxRecordBytes == 0 || report.Policy.MaxRecordBytes > persistentCacheMaxBytes ||
		report.Policy.MaxArtifactBytes == 0 || report.Policy.MaxArtifactBytes > persistentCacheMaxBytes ||
		report.Policy.TempMaxAgeSecs <= 0 || report.Policy.TempMaxAgeSecs > persistentCacheMaxTempAgeSecs ||
		report.Policy.MaxArtifactBytes > report.Policy.MaxRecordBytes || report.Policy.MaxRecordBytes > report.Policy.MaxBytes {
		return fmt.Errorf("persistent-cache process measurement policy is incomplete or inconsistent")
	}
	expectedSamples := PersistentCacheProcessIterations * 2
	if len(report.Samples) != expectedSamples {
		return fmt.Errorf("persistent-cache process measurement requires %d samples", expectedSamples)
	}
	for index, sample := range report.Samples {
		pair := index/2 + 1
		condition := PersistentCacheProcessCold
		if index%2 == 1 {
			condition = PersistentCacheProcessWarm
		}
		if sample.Sequence != index+1 || sample.Pair != pair || sample.Condition != condition {
			return fmt.Errorf("persistent-cache process measurement sample %d is not in canonical paired order", index+1)
		}
		if sample.DurationNS < 0 || sample.BelowClockResolution != (sample.DurationNS < report.ClockResolutionNS) ||
			sample.Packages != 2 || sample.MIRFunctions != 2 {
			return fmt.Errorf("persistent-cache process measurement sample %d has incomplete outcome data", sample.Sequence)
		}
		if sample.Rejected != 0 || sample.StoreSkipped != 0 || sample.StoreFailed != 0 ||
			sample.ReassemblyFallback || sample.RecordRejected != 0 || sample.Evictions != 0 {
			return fmt.Errorf("persistent-cache process measurement sample %d did not follow the clean comparison path", sample.Sequence)
		}
		if sample.RetainedEntries != sample.Packages || sample.RetainedEntries > report.Policy.MaxEntries ||
			sample.RetainedBytes == 0 || sample.RetainedBytes > report.Policy.MaxBytes {
			return fmt.Errorf("persistent-cache process measurement sample %d has invalid retained state", sample.Sequence)
		}
		if err := validatePersistentCacheProcessOutcome(sample); err != nil {
			return fmt.Errorf("persistent-cache process measurement sample %d: %w", sample.Sequence, err)
		}
	}
	return nil
}

func validateCompilerBuildDigest(value string) error {
	const prefix = "fairyc.compiler-executable-sha256/v1:"
	if !strings.HasPrefix(value, prefix) {
		return fmt.Errorf("persistent-cache process measurement compiler build is not an executable digest")
	}
	digest := strings.TrimPrefix(value, prefix)
	decoded, err := hex.DecodeString(digest)
	if err != nil || len(decoded) != 32 || digest != strings.ToLower(digest) {
		return fmt.Errorf("persistent-cache process measurement compiler build is not a canonical SHA-256 digest")
	}
	return nil
}

func validatePersistentCacheProcessOutcome(sample PersistentCacheProcessSample) error {
	switch sample.Condition {
	case PersistentCacheProcessCold:
		if sample.Outcome != "fallback" || sample.Hits != 0 || sample.Misses != sample.Packages ||
			sample.Stored != sample.Packages || !sample.Lowered || sample.RecordHits != 0 ||
			sample.RecordMisses != uint64(sample.Packages) || sample.VerifiedHits != 0 {
			return fmt.Errorf("cold sample does not prove complete fallback and disk population")
		}
	case PersistentCacheProcessWarm:
		if sample.Outcome != "all-hit" || sample.Hits != sample.Packages || sample.Misses != 0 ||
			sample.Stored != 0 || sample.Lowered || sample.RecordHits != uint64(sample.Packages) ||
			sample.RecordMisses != 0 || sample.VerifiedHits != uint64(sample.Packages) {
			return fmt.Errorf("warm sample does not prove complete verified cross-process reuse")
		}
	default:
		return fmt.Errorf("unknown condition %q", sample.Condition)
	}
	return nil
}
