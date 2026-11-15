package contractprofile

import (
	"bytes"
	"encoding/json"
	"slices"
	"testing"
)

func reportWithSharedCollections() Report {
	location := &Location{Source: "main.fy", StartByte: 1, EndByte: 2}
	diagnostic := SourceDiagnostic{Code: "E1001", Location: *location, Message: "first"}
	violation := Violation{Code: "CP1001", Function: "a::run", Location: location, Message: "first"}
	report := NewReport()
	report.Functions = []FunctionReport{
		{Name: "z::run", InternalCalls: []string{"z::call", "a::call", "a::call"}},
		{Name: "a::run"},
	}
	report.SourceDiagnostics = []SourceDiagnostic{
		{Code: "E1002", Location: Location{Source: "main.fy", StartByte: 5, EndByte: 6}, Message: "second"},
		diagnostic, diagnostic,
	}
	report.Violations = []Violation{
		{Code: "CP1002", Function: "z::run", Message: "second"},
		violation, violation,
	}
	return report
}

func TestReportCanonicalizationPreservesSharedCollections(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"JSON", "Finalize"} {
		t.Run(method, func(t *testing.T) {
			input := reportWithSharedCollections()
			before, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			report := input // A value copy still shares every backing array.
			if method == "JSON" {
				data, err := report.JSON()
				if err != nil {
					t.Fatal(err)
				}
				var decoded Report
				if err := json.Unmarshal(data, &decoded); err != nil {
					t.Fatal(err)
				}
				report = decoded
			} else {
				report.Finalize()
			}
			after, err := json.Marshal(input)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(before, after) {
				t.Errorf("%s changed caller collections:\nbefore=%s\nafter=%s", method, before, after)
			}
			if len(report.Functions) != 2 || report.Functions[0].Name != "a::run" ||
				report.Functions[0].InternalCalls == nil ||
				!slices.Equal(report.Functions[1].InternalCalls, []string{"a::call", "z::call"}) {
				t.Fatalf("canonical functions = %+v", report.Functions)
			}
			if len(report.SourceDiagnostics) != 2 || report.SourceDiagnostics[0].Code != "E1001" ||
				len(report.Violations) != 2 || report.Violations[0].Code != "CP1001" ||
				report.Accepted || report.Status != "source-diagnostics" {
				t.Fatalf("canonical decision = %+v", report)
			}
		})
	}
}

func TestReportJSONRepeatedSerialization(t *testing.T) {
	t.Parallel()
	report := reportWithSharedCollections()
	first, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	second, err := report.JSON()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("serializing the same report twice changed output:\nfirst=%s\nsecond=%s", first, second)
	}
}

func TestReportFinalizeDecisionPrecedence(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		name        string
		diagnostics []SourceDiagnostic
		violations  []Violation
		wantStatus  string
	}{
		{name: "empty", wantStatus: "accepted"},
		{name: "violation", violations: []Violation{{Code: "CP1001"}}, wantStatus: "profile-rejected"},
		{name: "diagnostic", diagnostics: []SourceDiagnostic{{Code: "E1001"}}, wantStatus: "source-diagnostics"},
		{name: "both", diagnostics: []SourceDiagnostic{{Code: "E1001"}}, violations: []Violation{{Code: "CP1001"}}, wantStatus: "source-diagnostics"},
	} {
		t.Run(test.name, func(t *testing.T) {
			report := Report{Accepted: true, Status: "stale", SourceDiagnostics: test.diagnostics, Violations: test.violations}
			report.Finalize()
			if report.Status != test.wantStatus || report.Accepted != (test.wantStatus == "accepted") {
				t.Fatalf("decision = %q, %v", report.Status, report.Accepted)
			}
			if report.Functions == nil || report.SourceDiagnostics == nil || report.Violations == nil {
				t.Fatalf("nil canonical collections: %+v", report)
			}
			before, err := json.Marshal(report)
			if err != nil {
				t.Fatal(err)
			}
			report.Finalize()
			after, err := json.Marshal(report)
			if err != nil || !bytes.Equal(before, after) {
				t.Fatalf("Finalize is not idempotent: %s, %v", after, err)
			}
		})
	}
	var report *Report
	report.Finalize()
}
