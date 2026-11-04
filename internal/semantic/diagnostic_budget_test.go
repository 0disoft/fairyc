package semantic

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

func semanticReportFiles(tb testing.TB, calls int) []File {
	tb.Helper()
	var source strings.Builder
	source.WriteString("module main;fn i32 f(){")
	for index := range calls {
		fmt.Fprintf(&source, "missing%d();", index)
	}
	source.WriteString("return 0;}")
	return borrowLiveFiles(tb, source.String())
}

func analyzeSemanticReport(t *testing.T, files []File, limit int) Result {
	t.Helper()
	result, err := analyzeContextWithOptions(context.Background(), files, nil, nil, nil, analysisOptions{reportLimit: limit})
	if err != nil {
		t.Fatalf("analyze semantic report: %v", err)
	}
	return result
}

func TestSemanticDiagnosticBudgetBelowAtAndOver(t *testing.T) {
	for _, test := range []struct {
		calls       int
		diagnostics int
		marker      bool
	}{
		{calls: 3, diagnostics: 3},
		{calls: 4, diagnostics: 4},
		{calls: 5, diagnostics: 5, marker: true},
	} {
		t.Run(fmt.Sprintf("%d-of-4", test.calls), func(t *testing.T) {
			result := analyzeSemanticReport(t, semanticReportFiles(t, test.calls), 4)
			if len(result.Unsupported) != 0 || len(result.Diagnostics) != test.diagnostics {
				t.Fatalf("diagnostics=%+v unsupported=%+v", result.Diagnostics, result.Unsupported)
			}
			markers := 0
			for _, value := range result.Diagnostics {
				if value.Code == SemanticReportExhaustedCode {
					markers++
				}
			}
			if markers != map[bool]int{false: 0, true: 1}[test.marker] {
				t.Fatalf("markers=%d diagnostics=%+v", markers, result.Diagnostics)
			}
		})
	}
}

func TestSemanticDiagnosticBudgetBoundsUnresolvedCalls(t *testing.T) {
	const calls = 4096
	result := analyzeSemanticReport(t, semanticReportFiles(t, calls), 0)
	if result.FailedPhase != PhaseResolve || result.CompletedThrough == PhaseFlow {
		t.Fatalf("bounded report became success: %+v", result)
	}
	if len(result.Diagnostics) != DefaultSemanticReportLimit+1 || len(result.Unsupported) != 0 {
		t.Fatalf("diagnostics=%d unsupported=%d", len(result.Diagnostics), len(result.Unsupported))
	}
	for index, value := range result.Diagnostics[:DefaultSemanticReportLimit] {
		if value.Code != "E0101" {
			t.Fatalf("diagnostic %d = %+v", index, value)
		}
	}
	if marker := result.Diagnostics[len(result.Diagnostics)-1]; marker.Code != SemanticReportExhaustedCode {
		t.Fatalf("terminal marker = %+v", marker)
	}
	for index := 1; index < DefaultSemanticReportLimit; index++ {
		if result.Diagnostics[index-1].Primary.StartByte >= result.Diagnostics[index].Primary.StartByte {
			t.Fatalf("admitted error order changed at %d: %+v", index, result.Diagnostics)
		}
	}
	again := analyzeSemanticReport(t, semanticReportFiles(t, calls), 0)
	if !reflect.DeepEqual(result.Diagnostics, again.Diagnostics) {
		t.Fatal("repeated bounded reports differ")
	}
}

func TestSemanticDiagnosticBudgetBoundsDirectTypeAppend(t *testing.T) {
	const calls = 12
	var source strings.Builder
	source.WriteString("module main;fn void take(bool value){}fn void caller(){")
	for range calls {
		source.WriteString("take(42);")
	}
	source.WriteString("}")
	result := analyzeSemanticReport(t, borrowLiveFiles(t, source.String()), 4)
	if result.FailedPhase != PhaseType || len(result.Unsupported) != 0 || len(result.Diagnostics) != 5 {
		t.Fatalf("direct type report = %+v", result)
	}
	for _, value := range result.Diagnostics[:4] {
		if value.Code != "E1006" || value.ExpectedType != hir.ParseType("bool") || value.ActualType != hir.ParseType("i32") {
			t.Fatalf("admitted type diagnostic = %+v", value)
		}
	}
	if result.Diagnostics[4].Code != SemanticReportExhaustedCode {
		t.Fatalf("terminal type marker = %+v", result.Diagnostics[4])
	}
}

func TestSemanticDiagnosticBudgetCountsUnsupportedFacts(t *testing.T) {
	a := &analyzer{diagnosticBudget: newDiagnosticBudget(2)}
	a.addDiagnostic("E1006", hir.Span{Source: "main.fy", StartByte: 1, EndByte: 2})
	a.addUnsupported("main.fy", diagnostic.Span{StartByte: 10, EndByte: 11}, "first")
	a.addUnsupported("main.fy", diagnostic.Span{StartByte: 20, EndByte: 21}, "second")
	if len(a.unsupported) != 1 || a.unsupported[0].Feature != "first" {
		t.Fatalf("admitted unsupported = %+v", a.unsupported)
	}
	if len(a.diagnostics) != 2 || a.diagnostics[0].Code != "E1006" ||
		a.diagnostics[1].Code != SemanticReportExhaustedCode || a.diagnostics[1].Primary.StartByte != 20 {
		t.Fatalf("bounded diagnostics = %+v", a.diagnostics)
	}
	if !a.diagnosticBudget.exhausted || !a.canceled() {
		t.Fatal("unsupported overflow did not request a fast phase stop")
	}
}

func TestSemanticDiagnosticMarkerHandoffIsNotAdmittedAsAFact(t *testing.T) {
	const originalLimit = 256
	functions := make([]ast.Function, originalLimit+1)
	for index := range originalLimit + 1 {
		functions[index] = ast.Function{
			Public: true,
			Attributes: []ast.Attribute{{Name: testAttribute,
				Span: diagnostic.Span{StartByte: index, EndByte: index + 1}}},
			ResultType: ast.TypeReference{Kind: ast.TypeKindNamed, Path: "void"},
		}
	}
	initial := TestDeclarationDiagnostics([]File{{Path: "tests/main.fy", Test: true,
		Module: &ast.Module{Functions: functions}}})
	if len(initial) != originalLimit+1 || initial[originalLimit].Code != SemanticReportExhaustedCode {
		t.Fatalf("test declaration marker = %+v", initial)
	}

	for _, test := range []struct {
		limit int
		count int
		trust int
	}{
		{limit: 8, count: 9, trust: 8},
		{limit: 300, count: 257, trust: 256},
	} {
		t.Run(fmt.Sprint(test.limit), func(t *testing.T) {
			a := &analyzer{diagnosticBudget: newDiagnosticBudget(test.limit)}
			for _, value := range initial {
				a.addDiagnosticRecord(value)
				if a.diagnosticBudget.exhausted {
					break
				}
			}
			markers := 0
			for _, value := range a.diagnostics {
				if value.Code == SemanticReportExhaustedCode {
					markers++
				}
			}
			if len(a.diagnostics) != test.count || a.diagnosticBudget.admitted != test.trust || markers != 1 {
				t.Fatalf("diagnostics=%d admitted=%d markers=%d", len(a.diagnostics), a.diagnosticBudget.admitted, markers)
			}
			a.addDiagnostic("E1006", hir.Span{Source: "main.fy"})
			if len(a.diagnostics) != test.count {
				t.Fatal("post-marker admission changed bounded output")
			}
		})
	}
}

func TestSemanticDiagnosticBudgetPreservesCancellationCause(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := analyzeContextWithOptions(ctx, semanticReportFiles(t, 32), nil, nil, nil, analysisOptions{reportLimit: 2})
	if !errors.Is(err, context.Canceled) || len(result.Diagnostics) != 0 {
		t.Fatalf("cancellation result=%+v error=%v", result, err)
	}
}
