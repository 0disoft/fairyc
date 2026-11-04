package semantic

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

func resultLoopSource(depth int, before, inner, after string) string {
	return "module main;fn i32 f(Result<i32,i32> p){" + before + strings.Repeat("while(false){", depth) + inner + strings.Repeat("}", depth) + after + "return 0;}"
}

func TestResultWorkNeutralLoopsPreserveObligations(t *testing.T) {
	for _, before := range []string{"", "discard p;"} {
		f, _ := scalarLoopFlow(t, resultLoopSource(20, before, "let i32 x=1;", ""))
		a := f.analyzer
		a.resultFlowFunction(f.module, f.function)
		used := defaultResultWorkLimit - a.resultWork.remaining
		if a.resultWork.exhausted || used > 512 || len(a.unsupported) != 0 {
			t.Fatalf("neutral Result loop: work=%d, unsupported=%v", used, a.unsupported)
		}
		if (len(a.diagnostics) != 0) != (before == "") {
			t.Fatalf("pending obligation changed: before=%q, diagnostics=%v", before, a.diagnostics)
		}
		t.Logf("20 nested loops, handled=%t: %d work units", before != "", used)
	}
}

func TestResultWorkMatchesUnoptimizedDiagnosticsAndFacts(t *testing.T) {
	for _, before := range []string{"", "discard p;"} {
		var results [2]Result
		for mode := range results {
			f, r := scalarLoopFlow(t, resultLoopSource(6, before, "let i32 x=1;", ""))
			a := f.analyzer
			if mode == 1 {
				a.resultNeutralScalars = make(map[*ast.Expression]bool)
				for e := range a.expressionTypes {
					a.resultNeutralScalars[e] = false
				}
			}
			a.resultFlowFunction(f.module, f.function)
			a.finish()
			r.Diagnostics, r.Unsupported, r.ResultHandling = a.diagnostics, a.unsupported, a.resultHandling
			results[mode] = r
		}
		if !reflect.DeepEqual(results[0], results[1]) {
			t.Fatalf("Result obligations or facts changed, handled=%t", before != "")
		}
	}
}

func TestResultWorkBudgetRejectsIncompleteProof(t *testing.T) {
	files := borrowLiveFiles(t, resultLoopSource(12, "discard p;", "discard 0;", ""))
	r, err := analyzeContextWithOptions(context.Background(), files, nil, nil, nil, analysisOptions{resultWorkLimit: 512})
	if err != nil || len(r.Unsupported) != 1 || r.Unsupported[0].Feature != resultWorkLimitFeature || r.CompletedThrough == PhaseFlow {
		t.Fatalf("incomplete Result proof accepted: phase=%s unsupported=%v error=%v", r.CompletedThrough, r.Unsupported, err)
	}
}

func TestResultWorkInclusiveBudgetAndCancellation(t *testing.T) {
	a := &analyzer{resultWorkLimit: 3}
	if !a.consumeResultWork("main.fy", diagnostic.Span{}, 2) || !a.consumeResultWork("main.fy", diagnostic.Span{}, 1) {
		t.Fatal("exact budget rejected")
	}
	if a.consumeResultWork("main.fy", diagnostic.Span{}, 1) || a.consumeResultWork("main.fy", diagnostic.Span{}, 1) || len(a.unsupported) != 1 {
		t.Fatal("overflow accepted or reported twice")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a.ctx, a.resultWork = ctx, newResultWorkBudget(3)
	if a.consumeResultWork("main.fy", diagnostic.Span{}, 1) || a.resultWork.remaining != 3 || len(a.unsupported) != 1 {
		t.Fatal("cancellation consumed work or reported overflow")
	}
}

func TestResultWorkAllowanceResetsForEachFunction(t *testing.T) {
	files := borrowLiveFiles(t, "module main;fn i32 a(){return 0;}fn i32 b(){return 1;}")
	r, err := analyzeContextWithOptions(context.Background(), files, nil, nil, nil, analysisOptions{resultWorkLimit: 2})
	if err != nil || r.CompletedThrough != PhaseFlow || len(r.Unsupported) != 0 || len(r.Diagnostics) != 0 {
		t.Fatalf("independent functions shared allowance: %+v, %v", r, err)
	}
}
