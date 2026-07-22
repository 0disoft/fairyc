package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestCacheEditReportMarshalPreservesRawOrderedDistributions(t *testing.T) {
	t.Parallel()
	report := validCacheEditReport()
	first, err := buildmeasure.MarshalCacheEdits(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.MarshalCacheEdits(report)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) || !strings.HasPrefix(string(first), `{"schema":"fairyc.process-cache-edit-measurement/v0"`) ||
		strings.Contains(string(first), "C:\\") || strings.Contains(string(first), "percentile") || strings.Contains(string(first), "speedup\"") {
		t.Fatalf("cache edit measurement JSON = %s", first)
	}
}

func TestCacheEditReportRejectsIncomparableOrMisclassifiedSamples(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.CacheEditReport)
	}{
		{name: "wrong repetitions", mutate: func(value *buildmeasure.CacheEditReport) { value.IterationsPerCondition-- }},
		{name: "timed setup ambiguity", mutate: func(value *buildmeasure.CacheEditReport) { value.Setup = "unknown" }},
		{name: "missing sample", mutate: func(value *buildmeasure.CacheEditReport) { value.Samples = value.Samples[:len(value.Samples)-1] }},
		{name: "reordered condition", mutate: func(value *buildmeasure.CacheEditReport) {
			value.Samples[0].Condition = buildmeasure.CacheEditPrivateRoot
		}},
		{name: "private root over-invalidates", mutate: func(value *buildmeasure.CacheEditReport) { value.Samples[9].Hits = 0 }},
		{name: "private dependency over-invalidates", mutate: func(value *buildmeasure.CacheEditReport) { value.Samples[18].Hits = 0 }},
		{name: "public dependency under-invalidates", mutate: func(value *buildmeasure.CacheEditReport) { value.Samples[27].Misses = 1 }},
		{name: "sub-resolution sample not marked", mutate: func(value *buildmeasure.CacheEditReport) {
			value.Samples[1].DurationNS = value.ClockResolutionNS - 1
		}},
		{name: "resolved sample marked sub-resolution", mutate: func(value *buildmeasure.CacheEditReport) {
			value.Samples[1].BelowClockResolution = true
		}},
		{name: "speedup claim", mutate: func(value *buildmeasure.CacheEditReport) { value.Decision = "faster" }},
		{name: "host path", mutate: func(value *buildmeasure.CacheEditReport) { value.Workload = `C:\private\repo` }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validCacheEditReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalCacheEdits(report); err == nil {
				t.Fatal("invalid cache edit measurement report accepted")
			}
		})
	}
}

func validCacheEditReport() buildmeasure.CacheEditReport {
	report := buildmeasure.CacheEditReport{
		Schema: buildmeasure.CacheEditSchema, Workload: "workspace_quickstart",
		Scope: "semantic-through-verified-mir", Clock: "process-monotonic-wall-nanoseconds", ClockResolutionNS: 10,
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "raw-distributions-only-no-speedup-claim",
		Setup: "fresh-session-per-sample;edit-and-load-outside-timer", WarmupRequests: 1,
		IterationsPerCondition: buildmeasure.CacheEditIterationsPerCondition,
		Environment:            buildmeasure.CacheEnvironment{OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc"},
		Policy:                 buildmeasure.CachePolicy{MaxEntries: 8, MaxBytes: 32 << 20, MaxArtifactBytes: 8 << 20},
	}
	conditions := []string{
		buildmeasure.CacheEditNoOpWarm,
		buildmeasure.CacheEditPrivateRoot,
		buildmeasure.CacheEditPrivateDependency,
		buildmeasure.CacheEditPublicDependency,
	}
	for _, condition := range conditions {
		for iteration := 1; iteration <= buildmeasure.CacheEditIterationsPerCondition; iteration++ {
			sample := buildmeasure.CacheEditSample{
				Sequence: len(report.Samples) + 1, Condition: condition, Iteration: iteration,
				DurationNS: 100, Packages: 2, MIRFunctions: 2, InterpreterResult: 42, RetainedBytes: 100,
			}
			switch condition {
			case buildmeasure.CacheEditNoOpWarm:
				sample.Outcome, sample.Hits, sample.VerifiedHits, sample.RetainedEntries = "all-hit", 2, 2, 2
			case buildmeasure.CacheEditPrivateRoot:
				sample.Outcome, sample.Hits, sample.Misses, sample.Stored, sample.Lowered = "fallback", 1, 1, 2, true
				sample.VerifiedHits, sample.RetainedEntries = 1, 3
			case buildmeasure.CacheEditPrivateDependency:
				sample.Outcome, sample.Hits, sample.Misses, sample.Stored, sample.Lowered = "fallback", 1, 1, 2, true
				sample.VerifiedHits, sample.RetainedEntries = 1, 3
			default:
				sample.Outcome, sample.Misses, sample.Stored, sample.Lowered, sample.RetainedEntries = "fallback", 2, 2, true, 4
			}
			if condition == buildmeasure.CacheEditPublicDependency {
				sample.MIRFunctions = 3
			}
			report.Samples = append(report.Samples, sample)
		}
	}
	report.Samples[0].DurationNS = 0
	report.Samples[0].BelowClockResolution = true
	return report
}
