package buildmeasure_test

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/buildmeasure"
)

func TestReportMarshalPreservesRawNoCacheSamples(t *testing.T) {
	t.Parallel()
	report := validReport()
	first, err := buildmeasure.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	second, err := buildmeasure.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("the same report produced different JSON")
	}
	wantPrefix := `{"schema":"fairyc.build-measurement/v0","workload":"workspace_quickstart"`
	if !bytes.HasPrefix(first, []byte(wantPrefix)) {
		t.Fatalf("report = %s", first)
	}
}

func TestReportRejectsMisleadingOrIncompleteEvidence(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		mutate func(*buildmeasure.Report)
	}{
		{name: "cache claim", mutate: func(value *buildmeasure.Report) { value.CacheImplementation = "persistent" }},
		{name: "decision claim", mutate: func(value *buildmeasure.Report) { value.Decision = "faster" }},
		{name: "missing sample", mutate: func(value *buildmeasure.Report) { value.Samples = value.Samples[:3] }},
		{name: "duplicate sample", mutate: func(value *buildmeasure.Report) { value.Samples[1].Condition = value.Samples[0].Condition }},
		{name: "zero duration", mutate: func(value *buildmeasure.Report) { value.Samples[0].DurationNS = 0 }},
		{name: "wrong result", mutate: func(value *buildmeasure.Report) { value.Samples[0].NativeResult = 0 }},
		{name: "missing environment", mutate: func(value *buildmeasure.Report) { value.Environment.Toolchain = "" }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			report := validReport()
			test.mutate(&report)
			if _, err := buildmeasure.Marshal(report); err == nil {
				t.Fatal("invalid measurement report accepted")
			}
		})
	}
}

func validReport() buildmeasure.Report {
	return buildmeasure.Report{
		Schema:              buildmeasure.Schema,
		Workload:            "workspace_quickstart",
		Clock:               "process-monotonic-wall-nanoseconds",
		CacheImplementation: "none",
		Decision:            "baseline-only",
		Environment: buildmeasure.Environment{
			OS: "windows", Arch: "amd64", Target: "x86_64-windows-msvc", Toolchain: "msvc 14.51",
		},
		Samples: []buildmeasure.Sample{
			{Sequence: 1, Profile: "low", Condition: buildmeasure.ConditionOrdinary, LockState: "updated", DurationNS: 10, ArtifactBytes: 100, NativeResult: 42},
			{Sequence: 2, Profile: "low", Condition: buildmeasure.ConditionLocked, LockState: "verified", DurationNS: 9, ArtifactBytes: 100, NativeResult: 42},
			{Sequence: 3, Profile: "optimized", Condition: buildmeasure.ConditionOrdinary, LockState: "unchanged", DurationNS: 8, ArtifactBytes: 80, NativeResult: 42},
			{Sequence: 4, Profile: "optimized", Condition: buildmeasure.ConditionLocked, LockState: "verified", DurationNS: 7, ArtifactBytes: 80, NativeResult: 42},
		},
	}
}
