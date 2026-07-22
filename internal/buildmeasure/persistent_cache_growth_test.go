package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestPersistentCacheGrowthReportPreservesBoundedRawEvidence(t *testing.T) {
	t.Parallel()
	report := validPersistentCacheGrowthReport()
	first, err := buildmeasure.MarshalPersistentCacheGrowth(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalPersistentCacheGrowth(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) ||
		!strings.HasPrefix(string(first), `{"schema":"fairyc.persistent-cache-growth-measurement/v1"`) ||
		strings.Contains(string(first), `C:\`) || strings.Contains(string(first), "build/cache") ||
		strings.Contains(string(first), `"duration`) || strings.Contains(string(first), `"speedup`) {
		t.Fatalf("persistent-cache growth JSON = %s", first)
	}
}

func TestPersistentCacheGrowthReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.PersistentCacheGrowthReport)
	}{
		{name: "default claim", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Decision = "default-approved" }},
		{name: "retry policy", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.RetryPolicy = "retry-best" }},
		{name: "missing sample", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples = value.Samples[:8] }},
		{name: "wrong edit order", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) {
			value.Samples[2].Condition = buildmeasure.PersistentGrowthPrivateRootOne
		}},
		{name: "unbounded entry policy", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Policy.MaxEntries = 7 }},
		{name: "retained entry drift", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[4].RetainedEntries = 7 }},
		{name: "entry delta drift", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[6].RetainedEntryDelta = 1 }},
		{name: "byte delta drift", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[5].RetainedByteDelta++ }},
		{name: "eviction missing", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[4].Evictions = 0 }},
		{name: "no-op lowering", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[5].Lowered = true }},
		{name: "store failure", mutate: func(value *buildmeasure.PersistentCacheGrowthReport) { value.Samples[0].StoreFailed = 1 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validPersistentCacheGrowthReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalPersistentCacheGrowth(report); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func validPersistentCacheGrowthReport() buildmeasure.PersistentCacheGrowthReport {
	report := buildmeasure.PersistentCacheGrowthReport{
		Schema: buildmeasure.PersistentCacheGrowthSchema, Workload: "workspace_quickstart-retention-sequence",
		Scope:               "retained-disk-state-after-completed-process;source-mutation-excluded",
		CacheImplementation: "bounded-immutable-disk-package-mir-v2", Decision: "bounded-growth-control-only-no-default-policy",
		Setup: "one-cache;separate-fairyc-process-per-sample;mutation-before-process", RetryPolicy: "none;preserve-first-observation",
		CompilerBuild: "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		Environment:   buildmeasure.CacheEnvironment{OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 6, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	conditions := []string{
		buildmeasure.PersistentGrowthBaselineCold,
		buildmeasure.PersistentGrowthPrivateRootOne,
		buildmeasure.PersistentGrowthPrivateRootTwo,
		buildmeasure.PersistentGrowthPrivateDependency,
		buildmeasure.PersistentGrowthPublicDependency,
		buildmeasure.PersistentGrowthPublicNoOp,
		buildmeasure.PersistentGrowthPrivateRootPublic,
		buildmeasure.PersistentGrowthPrivateRootNoOp,
		buildmeasure.PersistentGrowthRepeatedNoOp,
	}
	entries := []int{2, 3, 4, 5, 6, 6, 6, 6, 6}
	deltas := []int{2, 1, 1, 1, 1, 0, 0, 0, 0}
	bytes := []uint64{6000, 9000, 12000, 15000, 21000, 21000, 22000, 22000, 22000}
	previousBytes := uint64(0)
	for index, condition := range conditions {
		sample := buildmeasure.PersistentCacheGrowthSample{
			Sequence: index + 1, Condition: condition, Outcome: "fallback", Packages: 2,
			Hits: 1, Misses: 1, Stored: 2, Lowered: true, MIRFunctions: 2,
			RetainedEntries: entries[index], RetainedBytes: bytes[index], RetainedEntryDelta: deltas[index],
			RetainedByteDelta: int64(bytes[index]) - int64(previousBytes), RecordHits: 1, RecordMisses: 1, VerifiedHits: 1,
		}
		previousBytes = bytes[index]
		if condition == buildmeasure.PersistentGrowthBaselineCold || condition == buildmeasure.PersistentGrowthPublicDependency {
			sample.Hits, sample.Misses, sample.RecordHits, sample.RecordMisses, sample.VerifiedHits = 0, 2, 0, 2, 0
		}
		if condition == buildmeasure.PersistentGrowthPublicDependency {
			sample.Evictions = 1
		}
		if condition == buildmeasure.PersistentGrowthPrivateRootPublic {
			sample.Evictions = 1
		}
		if condition == buildmeasure.PersistentGrowthPublicNoOp || condition == buildmeasure.PersistentGrowthPrivateRootNoOp || condition == buildmeasure.PersistentGrowthRepeatedNoOp {
			sample.Outcome, sample.Hits, sample.Misses, sample.Stored, sample.Lowered = "all-hit", 2, 0, 0, false
			sample.RecordHits, sample.RecordMisses, sample.VerifiedHits = 2, 0, 2
		}
		report.Samples = append(report.Samples, sample)
	}
	return report
}
