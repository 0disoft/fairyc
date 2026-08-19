package contractprofile

import "testing"

func TestAnalyzeSourcesConsumesMaxIterOverlayBeforeCoreSemantics(t *testing.T) {
	t.Parallel()
	source := `module main;

@max_iter(8)
fn i32 main() {
    var i32 value = 0;
    while (value < 2) {
        value += 1;
    }
    return value;
}
`
	report, err := AnalyzeSources([]SourceInput{{Path: "main.fy", Bytes: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if !report.Accepted || report.Status != "accepted" {
		t.Fatalf("report = %+v", report)
	}
	if len(report.Functions) != 1 {
		t.Fatalf("functions = %+v", report.Functions)
	}
	function := report.Functions[0]
	if function.Name != "main::main" || !function.HasControlFlowCycle || function.MeteredCycleEdges == 0 || function.DeclaredLoopBudget != 8 || !function.Bounded {
		t.Fatalf("function = %+v", function)
	}
}

func TestAnalyzeSourcesRejectsLoopWithoutMaxIter(t *testing.T) {
	t.Parallel()
	source := `module main;

fn i32 main() {
    var i32 value = 0;
    while (value < 2) {
        value += 1;
    }
    return value;
}
`
	report, err := AnalyzeSources([]SourceInput{{Path: "main.fy", Bytes: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || report.Status != "profile-rejected" || !hasViolationCode(report, "CP1007") {
		t.Fatalf("report = %+v", report)
	}
}

func TestAnalyzeSourcesRejectsMalformedMaxIter(t *testing.T) {
	t.Parallel()
	source := `module main;

@max_iter(0)
fn i32 main() {
    return 0;
}
`
	report, err := AnalyzeSources([]SourceInput{{Path: "main.fy", Bytes: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || !hasViolationCode(report, "CP0102") {
		t.Fatalf("report = %+v", report)
	}
	if len(report.SourceDiagnostics) != 0 {
		t.Fatalf("unexpected source diagnostics = %+v", report.SourceDiagnostics)
	}
}

func TestAnalyzeSourcesRejectsMisplacedMaxIter(t *testing.T) {
	t.Parallel()
	source := `module main;

@max_iter(4)
struct Counter {
    i32 value;
}

fn i32 main() {
    return 0;
}
`
	report, err := AnalyzeSources([]SourceInput{{Path: "main.fy", Bytes: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || !hasViolationCode(report, "CP0103") {
		t.Fatalf("report = %+v", report)
	}
}

func TestAnalyzeSourcesReturnsCoreDiagnosticsWithoutInventingMIR(t *testing.T) {
	t.Parallel()
	source := `module main;

fn i32 main() {
    return missing;
}
`
	report, err := AnalyzeSources([]SourceInput{{Path: "main.fy", Bytes: []byte(source)}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Accepted || report.Status != "source-diagnostics" || len(report.SourceDiagnostics) != 1 {
		t.Fatalf("report = %+v", report)
	}
	if report.SourceDiagnostics[0].Code != "E0101" || len(report.Functions) != 0 {
		t.Fatalf("diagnostics/functions = %+v / %+v", report.SourceDiagnostics, report.Functions)
	}
}
