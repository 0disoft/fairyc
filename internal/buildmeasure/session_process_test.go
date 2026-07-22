package buildmeasure_test

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestSessionProcessReportPreservesRawProcessEvidence(t *testing.T) {
	t.Parallel()
	report := validSessionProcessReport()
	encoded, err := buildmeasure.MarshalSessionProcess(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(encoded), `{"schema":"fairyc.workspace-session-process-measurement/v0"`) ||
		strings.Contains(string(encoded), `C:\`) || strings.Contains(string(encoded), "speedup\"") || strings.Contains(string(encoded), "percentile") {
		t.Fatalf("session process JSON = %s", encoded)
	}
}

func TestSessionProcessReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.SessionProcessReport)
	}{
		{name: "reference claim", mutate: func(value *buildmeasure.SessionProcessReport) { value.Decision = "reference-budget" }},
		{name: "retry selection", mutate: func(value *buildmeasure.SessionProcessReport) { value.RetryPolicy = "retry-until-positive" }},
		{name: "missing peak", mutate: func(value *buildmeasure.SessionProcessReport) { value.Memory.PeakWorkingSetBytes = 0 }},
		{name: "missing sample", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples = value.Samples[:7] }},
		{name: "reordered sample", mutate: func(value *buildmeasure.SessionProcessReport) {
			value.Samples[1].Condition = buildmeasure.SessionProcessPrivateRoot
		}},
		{name: "no-op miss", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples[1].Misses = 1 }},
		{name: "root over-invalidation", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples[2].Hits = 0 }},
		{name: "private dependency over-invalidation", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples[4].Hits = 0 }},
		{name: "wrong public functions", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples[6].MIRFunctions = 2 }},
		{name: "retention over bound", mutate: func(value *buildmeasure.SessionProcessReport) { value.Samples[7].RetainedEntries = 9 }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			report := validSessionProcessReport()
			test.mutate(&report)
			if _, err := buildmeasure.MarshalSessionProcess(report); err == nil {
				t.Fatal("invalid session process report accepted")
			}
		})
	}
}

func validSessionProcessReport() buildmeasure.SessionProcessReport {
	conditions := []string{
		buildmeasure.SessionProcessCold,
		buildmeasure.SessionProcessNoOp,
		buildmeasure.SessionProcessPrivateRoot,
		buildmeasure.SessionProcessNoOpAfterRoot,
		buildmeasure.SessionProcessPrivateDependency,
		buildmeasure.SessionProcessNoOpAfterPrivate,
		buildmeasure.SessionProcessPublicDependency,
		buildmeasure.SessionProcessNoOpAfterPublic,
	}
	report := buildmeasure.SessionProcessReport{
		Schema: buildmeasure.SessionProcessSchema, Workload: "workspace_quickstart-editor-sequence",
		Scope: "request-write-through-response-read;mutation-excluded", Clock: "caller-monotonic-wall-nanoseconds",
		Setup: "one-standalone-fairyc-process;sequential-requests", RetryPolicy: "none;preserve-first-observation",
		MemoryMetric:        "windows-process-memory-counters-peak-working-set-and-pagefile-bytes",
		CacheImplementation: "bounded-process-package-mir-v0", Decision: "local-observation-only-no-reference-budget-or-speedup-claim",
		Environment: buildmeasure.CacheEnvironment{OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc"},
		Policy:      buildmeasure.CachePolicy{MaxEntries: 8, MaxBytes: 32 << 20, MaxArtifactBytes: 8 << 20},
		Memory:      buildmeasure.SessionProcessMemory{PeakWorkingSetBytes: 10, PeakPagefileBytes: 20},
	}
	for index, condition := range conditions {
		sample := buildmeasure.SessionProcessSample{Sequence: index + 1, Condition: condition, DurationNS: int64(index), MIRFunctions: 2, RetainedEntries: 2, RetainedBytes: 100}
		switch condition {
		case buildmeasure.SessionProcessCold:
			sample.Outcome, sample.Misses, sample.Lowered = "fallback", 2, true
		case buildmeasure.SessionProcessPrivateRoot:
			sample.Outcome, sample.Hits, sample.Misses, sample.Lowered, sample.RetainedEntries = "fallback", 1, 1, true, 3
		case buildmeasure.SessionProcessPrivateDependency:
			sample.Outcome, sample.Hits, sample.Misses, sample.Lowered, sample.RetainedEntries = "fallback", 1, 1, true, 4
		case buildmeasure.SessionProcessPublicDependency:
			sample.Outcome, sample.Misses, sample.Lowered, sample.RetainedEntries, sample.MIRFunctions = "fallback", 2, true, 6, 3
		default:
			sample.Outcome, sample.Hits = "all-hit", 2
			if condition == buildmeasure.SessionProcessNoOpAfterRoot {
				sample.RetainedEntries = 3
			} else if condition == buildmeasure.SessionProcessNoOpAfterPrivate {
				sample.RetainedEntries = 4
			} else if condition == buildmeasure.SessionProcessNoOpAfterPublic {
				sample.RetainedEntries, sample.MIRFunctions = 6, 3
			}
		}
		report.Samples = append(report.Samples, sample)
	}
	return report
}
