package buildmeasure

import (
	"encoding/json"
	"fmt"
)

const (
	PersistentCacheGrowthSchema = "fairyc.persistent-cache-growth-measurement/v1"

	PersistentGrowthBaselineCold        = "baseline-cold"
	PersistentGrowthPrivateRootOne      = "private-root-v1"
	PersistentGrowthPrivateRootTwo      = "private-root-v2"
	PersistentGrowthPrivateDependency   = "private-dependency-v1"
	PersistentGrowthPublicDependency    = "public-dependency-v1"
	PersistentGrowthPublicNoOp          = "public-no-op"
	PersistentGrowthPrivateRootPublic   = "private-root-on-public-dependency"
	PersistentGrowthPrivateRootNoOp     = "private-root-public-no-op"
	PersistentGrowthRepeatedNoOp        = "repeated-private-root-public-no-op"
	PersistentCacheGrowthSampleCount    = 9
	persistentCacheGrowthPackageCount   = 2
	persistentCacheGrowthMaximumEntries = 6
)

// PersistentCacheGrowthSample is one standalone compiler process in a fixed edit sequence. Entry
// and byte deltas compare retained state with the immediately preceding completed process.
type PersistentCacheGrowthSample struct {
	Sequence           int    `json:"sequence"`
	Condition          string `json:"condition"`
	Outcome            string `json:"outcome"`
	Packages           int    `json:"packages"`
	Hits               int    `json:"hits"`
	Misses             int    `json:"misses"`
	Rejected           int    `json:"rejected"`
	Stored             int    `json:"stored"`
	StoreSkipped       int    `json:"storeSkipped"`
	StoreFailed        int    `json:"storeFailed"`
	Lowered            bool   `json:"lowered"`
	ReassemblyFallback bool   `json:"reassemblyFallback"`
	MIRFunctions       int    `json:"mirFunctions"`
	RetainedEntries    int    `json:"retainedEntries"`
	RetainedBytes      uint64 `json:"retainedBytes"`
	RetainedEntryDelta int    `json:"retainedEntryDelta"`
	RetainedByteDelta  int64  `json:"retainedByteDelta"`
	RecordHits         uint64 `json:"recordHits"`
	RecordMisses       uint64 `json:"recordMisses"`
	RecordRejected     uint64 `json:"recordRejected"`
	Evictions          uint64 `json:"evictions"`
	VerifiedHits       uint64 `json:"verifiedHits"`
}

// PersistentCacheGrowthReport preserves a deterministic edit-and-retention staircase. It does not
// select a default cache policy or claim the control fixture represents real repository growth.
type PersistentCacheGrowthReport struct {
	Schema              string                        `json:"schema"`
	Workload            string                        `json:"workload"`
	Scope               string                        `json:"scope"`
	CacheImplementation string                        `json:"cacheImplementation"`
	Decision            string                        `json:"decision"`
	Setup               string                        `json:"setup"`
	RetryPolicy         string                        `json:"retryPolicy"`
	CompilerBuild       string                        `json:"compilerBuild"`
	Environment         CacheEnvironment              `json:"environment"`
	Policy              PersistentCachePolicy         `json:"policy"`
	Samples             []PersistentCacheGrowthSample `json:"samples"`
}

// MarshalPersistentCacheGrowth validates and emits deterministic, path-free raw evidence.
func MarshalPersistentCacheGrowth(report PersistentCacheGrowthReport) ([]byte, error) {
	if err := ValidatePersistentCacheGrowth(report); err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return nil, fmt.Errorf("encode persistent-cache growth measurement: %w", err)
	}
	return encoded, nil
}

// ValidatePersistentCacheGrowth proves the fixed invalidation staircase, bounded retention, and
// current-state reuse. It deliberately does not compare time or authorize a product default.
func ValidatePersistentCacheGrowth(report PersistentCacheGrowthReport) error {
	if report.Schema != PersistentCacheGrowthSchema ||
		report.Workload != "workspace_quickstart-retention-sequence" ||
		report.Scope != "retained-disk-state-after-completed-process;source-mutation-excluded" {
		return fmt.Errorf("persistent-cache growth measurement requires its closed schema, workload, and scope")
	}
	if report.CacheImplementation != "bounded-immutable-disk-package-mir-v2" ||
		report.Decision != "bounded-growth-control-only-no-default-policy" ||
		report.Setup != "one-cache;separate-fairyc-process-per-sample;mutation-before-process" ||
		report.RetryPolicy != "none;preserve-first-observation" {
		return fmt.Errorf("persistent-cache growth measurement policy is incompatible")
	}
	if report.Environment.OS == "" || report.Environment.Arch == "" || report.Environment.Target == "" {
		return fmt.Errorf("persistent-cache growth measurement environment is incomplete")
	}
	if err := validateCompilerBuildDigest(report.CompilerBuild); err != nil {
		return err
	}
	if report.Policy.MaxEntries != persistentCacheGrowthMaximumEntries ||
		report.Policy.MaxBytes == 0 || report.Policy.MaxBytes > persistentCacheMaxBytes ||
		report.Policy.MaxRecordBytes == 0 || report.Policy.MaxRecordBytes > report.Policy.MaxBytes ||
		report.Policy.MaxArtifactBytes == 0 || report.Policy.MaxArtifactBytes > report.Policy.MaxRecordBytes ||
		report.Policy.TempMaxAgeSecs <= 0 || report.Policy.TempMaxAgeSecs > persistentCacheMaxTempAgeSecs {
		return fmt.Errorf("persistent-cache growth measurement bounds are incomplete or inconsistent")
	}
	conditions := []string{
		PersistentGrowthBaselineCold,
		PersistentGrowthPrivateRootOne,
		PersistentGrowthPrivateRootTwo,
		PersistentGrowthPrivateDependency,
		PersistentGrowthPublicDependency,
		PersistentGrowthPublicNoOp,
		PersistentGrowthPrivateRootPublic,
		PersistentGrowthPrivateRootNoOp,
		PersistentGrowthRepeatedNoOp,
	}
	wantEntries := []int{2, 3, 4, 5, 6, 6, 6, 6, 6}
	wantDeltas := []int{2, 1, 1, 1, 1, 0, 0, 0, 0}
	if len(report.Samples) != len(conditions) {
		return fmt.Errorf("persistent-cache growth measurement requires %d samples", len(conditions))
	}
	var previousBytes uint64
	for index, sample := range report.Samples {
		if sample.Sequence != index+1 || sample.Condition != conditions[index] {
			return fmt.Errorf("persistent-cache growth sample %d is not in canonical edit order", index+1)
		}
		if sample.Packages != persistentCacheGrowthPackageCount || sample.MIRFunctions < persistentCacheGrowthPackageCount ||
			sample.RetainedEntries != wantEntries[index] || sample.RetainedEntryDelta != wantDeltas[index] ||
			sample.RetainedBytes == 0 || sample.RetainedBytes > report.Policy.MaxBytes {
			return fmt.Errorf("persistent-cache growth sample %d has invalid retained state: %+v", sample.Sequence, sample)
		}
		expectedByteDelta, ok := signedByteDelta(sample.RetainedBytes, previousBytes)
		if !ok || sample.RetainedByteDelta != expectedByteDelta {
			return fmt.Errorf("persistent-cache growth sample %d has an invalid retained-byte delta", sample.Sequence)
		}
		previousBytes = sample.RetainedBytes
		if sample.Rejected != 0 || sample.StoreSkipped != 0 || sample.StoreFailed != 0 || sample.ReassemblyFallback {
			return fmt.Errorf("persistent-cache growth sample %d did not follow the clean path", sample.Sequence)
		}
		if err := validatePersistentCacheGrowthOutcome(sample); err != nil {
			return fmt.Errorf("persistent-cache growth sample %d %+v: %w", sample.Sequence, sample, err)
		}
	}
	return nil
}

func validatePersistentCacheGrowthOutcome(sample PersistentCacheGrowthSample) error {
	noOp := sample.Condition == PersistentGrowthPublicNoOp || sample.Condition == PersistentGrowthPrivateRootNoOp || sample.Condition == PersistentGrowthRepeatedNoOp
	expectedEvictions := uint64(0)
	if sample.Condition == PersistentGrowthPublicDependency {
		expectedEvictions = 1
	}
	if sample.Condition == PersistentGrowthPrivateRootPublic {
		expectedEvictions = 1
	}
	if sample.Evictions != expectedEvictions {
		return fmt.Errorf("evictions = %d, want %d", sample.Evictions, expectedEvictions)
	}
	if sample.RecordRejected != 0 {
		return fmt.Errorf("record rejections = %d, want 0", sample.RecordRejected)
	}
	if noOp {
		if sample.Outcome != "all-hit" || sample.Hits != sample.Packages || sample.Misses != 0 ||
			sample.Stored != 0 || sample.Lowered || sample.RecordHits != uint64(sample.Packages) ||
			sample.RecordMisses != 0 || sample.VerifiedHits != uint64(sample.Packages) {
			return fmt.Errorf("no-op sample does not prove complete verified reuse")
		}
		return nil
	}
	wantHits := 1
	if sample.Condition == PersistentGrowthBaselineCold || sample.Condition == PersistentGrowthPublicDependency {
		wantHits = 0
	}
	wantMisses := sample.Packages - wantHits
	if sample.Outcome != "fallback" || sample.Hits != wantHits || sample.Misses != wantMisses ||
		sample.Stored != sample.Packages || !sample.Lowered || sample.RecordHits != uint64(wantHits) ||
		sample.RecordMisses != uint64(wantMisses) || sample.VerifiedHits != uint64(wantHits) {
		return fmt.Errorf("edit sample does not prove its expected fallback and reuse path")
	}
	return nil
}

func signedByteDelta(current, previous uint64) (int64, bool) {
	const maximumInt64 = uint64(^uint64(0) >> 1)
	if current >= previous {
		delta := current - previous
		return int64(delta), delta <= maximumInt64
	}
	delta := previous - current
	return -int64(delta), delta <= maximumInt64
}
