package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestCacheReportMarshalPreservesRawControlFlowEvidence(t *testing.T) {
	t.Parallel()
	report := validCacheReport()
	first, err := buildmeasure.MarshalCache(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalCache(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || !strings.HasPrefix(string(first), `{"schema":"fairyc.process-cache-measurement/v0"`) ||
		strings.Contains(string(first), "C:\\") || strings.Contains(string(first), "speedup\"") {
		t.Fatalf("cache measurement JSON = %s", first)
	}
}

func TestCacheReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.CacheReport)
	}{
		{name: "speedup claim", mutate: func(value *buildmeasure.CacheReport) { value.Decision = "faster" }},
		{name: "warm lowering", mutate: func(value *buildmeasure.CacheReport) { value.Samples[1].Lowered = true }},
		{name: "partial warm hit", mutate: func(value *buildmeasure.CacheReport) { value.Samples[1].Hits = 1 }},
		{name: "cold hit", mutate: func(value *buildmeasure.CacheReport) { value.Samples[0].Hits = 1 }},
		{name: "wrong result", mutate: func(value *buildmeasure.CacheReport) { value.Samples[1].InterpreterResult = 0 }},
		{name: "oversize retention", mutate: func(value *buildmeasure.CacheReport) { value.Samples[1].RetainedBytes = value.Policy.MaxBytes + 1 }},
		{name: "reassembly fallback", mutate: func(value *buildmeasure.CacheReport) { value.Samples[1].ReassemblyFallback = true }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validCacheReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalCache(report); err == nil {
				t.Fatal("invalid cache measurement report accepted")
			}
		})
	}
}

func validCacheReport() buildmeasure.CacheReport {
	return buildmeasure.CacheReport{
		Schema: buildmeasure.CacheSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds",
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "observed-only-no-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc"},
		Policy:      buildmeasure.CachePolicy{MaxEntries: 8, MaxBytes: 1024, MaxArtifactBytes: 512},
		Samples: []buildmeasure.CacheSample{
			{Sequence: 1, Condition: buildmeasure.CacheConditionCold, DurationNS: 10, Outcome: "fallback", Packages: 2, Misses: 2, Stored: 2, Lowered: true, MIRFunctions: 2, InterpreterResult: 42, RetainedEntries: 2, RetainedBytes: 100},
			{Sequence: 2, Condition: buildmeasure.CacheConditionWarm, DurationNS: 8, Outcome: "all-hit", Packages: 2, Hits: 2, MIRFunctions: 2, InterpreterResult: 42, RetainedEntries: 2, RetainedBytes: 100, VerifiedHits: 2},
		},
	}
}
