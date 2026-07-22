package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestPersistentCacheProcessReportPreservesRawPairedEvidence(t *testing.T) {
	t.Parallel()
	report := validPersistentCacheProcessReport()
	first, err := buildmeasure.MarshalPersistentCacheProcess(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalPersistentCacheProcess(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) ||
		!strings.HasPrefix(string(first), `{"schema":"fairyc.persistent-cache-process-measurement/v0"`) ||
		strings.Contains(string(first), `C:\`) || strings.Contains(string(first), "build/cache") ||
		strings.Contains(string(first), `"speedup"`) {
		t.Fatalf("persistent-cache process measurement JSON = %s", first)
	}
}

func TestPersistentCacheProcessReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.PersistentCacheProcessReport)
	}{
		{name: "speedup claim", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Decision = "faster" }},
		{name: "invalid compiler build", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.CompilerBuild = "fairyc-dev" }},
		{name: "retry policy", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.RetryPolicy = "retry-noise" }},
		{name: "policy ceiling", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Policy.MaxEntries = 65_537 }},
		{name: "missing pair", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples = value.Samples[:17] }},
		{name: "pair ordering", mutate: func(value *buildmeasure.PersistentCacheProcessReport) {
			value.Samples[1].Condition = buildmeasure.PersistentCacheProcessCold
		}},
		{name: "warm lowering", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples[1].Lowered = true }},
		{name: "partial warm hit", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples[1].Hits = 1 }},
		{name: "disk write failure", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples[0].StoreFailed = 1 }},
		{name: "record rejection", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples[1].RecordRejected = 1 }},
		{name: "clock classification", mutate: func(value *buildmeasure.PersistentCacheProcessReport) { value.Samples[0].BelowClockResolution = true }},
		{name: "oversize retention", mutate: func(value *buildmeasure.PersistentCacheProcessReport) {
			value.Samples[0].RetainedBytes = value.Policy.MaxBytes + 1
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validPersistentCacheProcessReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalPersistentCacheProcess(report); err == nil {
				t.Fatal("expected validation failure")
			}
		})
	}
}

func validPersistentCacheProcessReport() buildmeasure.PersistentCacheProcessReport {
	report := buildmeasure.PersistentCacheProcessReport{
		Schema: buildmeasure.PersistentCacheProcessSchema, Workload: "workspace_quickstart",
		Scope: "caller-process-run-through-exit;workspace-and-cache-io-included", Clock: "caller-monotonic-wall-nanoseconds",
		ClockResolutionNS: 10, CacheImplementation: "bounded-immutable-disk-package-mir-v1",
		Decision: "raw-paired-distributions-only-no-speedup-claim",
		Setup:    "fresh-cache-per-pair;separate-fairyc-process-per-sample", RetryPolicy: "none;preserve-first-observation",
		IterationsPerCondition: buildmeasure.PersistentCacheProcessIterations,
		CompilerBuild:          "fairyc.compiler-executable-sha256/v1:" + strings.Repeat("a", 64),
		Environment:            buildmeasure.CacheEnvironment{OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc"},
		Policy: buildmeasure.PersistentCachePolicy{
			MaxEntries: 8, MaxBytes: 67108864, MaxRecordBytes: 8388608,
			MaxArtifactBytes: 4194304, TempMaxAgeSecs: 60,
		},
	}
	for pair := 1; pair <= buildmeasure.PersistentCacheProcessIterations; pair++ {
		report.Samples = append(report.Samples,
			buildmeasure.PersistentCacheProcessSample{
				Sequence: pair*2 - 1, Pair: pair, Condition: buildmeasure.PersistentCacheProcessCold,
				DurationNS: 100, Outcome: "fallback", Packages: 2, Misses: 2, Stored: 2,
				Lowered: true, MIRFunctions: 2, RetainedEntries: 2, RetainedBytes: 6000, RecordMisses: 2,
			},
			buildmeasure.PersistentCacheProcessSample{
				Sequence: pair * 2, Pair: pair, Condition: buildmeasure.PersistentCacheProcessWarm,
				DurationNS: 80, Outcome: "all-hit", Packages: 2, Hits: 2,
				MIRFunctions: 2, RetainedEntries: 2, RetainedBytes: 6000, RecordHits: 2, VerifiedHits: 2,
			},
		)
	}
	return report
}
