package semantic

import (
	"context"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
)

func nestedBorrowLoops(depth int, inner string) string {
	return "module main;fn i32 f(ref i32 p){" + strings.Repeat("while(false){", depth) + inner + strings.Repeat("}", depth) + "return *p;}"
}

func TestBorrowWorkNeutralLoopsPreserveParameterState(t *testing.T) {
	f, _ := scalarLoopFlow(t, nestedBorrowLoops(20, "let i32 x=1;"))
	environment := f.parameterEnvironment()
	before := cloneBorrowEnvironment(environment)
	f.evaluateBlock(f.function.Body, environment, true)
	budget := f.analyzer.borrowBudget(f.function)
	used := defaultBorrowWorkLimit - budget.remaining
	if budget.exhausted || used > 512 || len(f.analyzer.diagnostics) != 0 || len(f.analyzer.unsupported) != 0 || !sameBorrowEnvironment(before, environment) {
		t.Fatalf("neutral nested loops: used=%d exhausted=%t diagnostics=%v unsupported=%v", used, budget.exhausted, f.analyzer.diagnostics, f.analyzer.unsupported)
	}
	t.Logf("20 nested loops with a borrowed parameter: %d work units", used)
}

func TestBorrowWorkRejectsNonNeutralNestedLoops(t *testing.T) {
	files := borrowLiveFiles(t, nestedBorrowLoops(20, "discard 0;"))
	result, err := analyzeContextWithOptions(context.Background(), files, nil, nil, nil, analysisOptions{borrowWorkLimit: 1024})
	if err != nil || len(result.Unsupported) != 1 || result.Unsupported[0].Feature != borrowWorkLimitFeature || result.CompletedThrough == PhaseFlow {
		t.Fatalf("incomplete proof accepted: phase=%s diagnostics=%v unsupported=%v err=%v", result.CompletedThrough, result.Diagnostics, result.Unsupported, err)
	}
}

func TestBorrowWorkInclusiveSharedBudgetAndCancellation(t *testing.T) {
	function := &ast.Function{}
	a := &analyzer{borrowWorkLimit: 3}
	f := &borrowFlowAnalyzer{analyzer: a, function: function, module: &moduleScope{file: File{Path: "main.fy"}}}
	if !f.consumeBorrowWork(diagnostic.Span{}, 2) || !f.consumeBorrowWork(diagnostic.Span{}, 1) {
		t.Fatal("exact budget rejected")
	}
	// A second inference/observation visitor shares the same function allowance.
	other := *f
	if other.consumeBorrowWork(diagnostic.Span{}, 1) || f.consumeBorrowWork(diagnostic.Span{}, 1) || len(a.unsupported) != 1 {
		t.Fatal("budget reset or duplicate terminal report")
	}
	other.function = &ast.Function{}
	if !other.consumeBorrowWork(diagnostic.Span{}, 3) {
		t.Fatal("independent function lost its allowance")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	a.ctx = ctx
	if other.consumeBorrowWork(diagnostic.Span{}, 1) || len(a.unsupported) != 1 {
		t.Fatal("cancellation produced a budget report")
	}
}
