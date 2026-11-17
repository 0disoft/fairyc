package semantic

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

func wideBorrowLiveSource(count int) string {
	var text strings.Builder
	text.WriteString("module main;fn void wide(")
	for index := range count {
		if index > 0 {
			text.WriteByte(',')
		}
		fmt.Fprintf(&text, "ref i32 p%d", index)
	}
	text.WriteString("){")
	for index := range count {
		fmt.Fprintf(&text, "discard *p%d;", index)
	}
	text.WriteByte('}')
	return text.String()
}

func TestBorrowLiveBudgetRejectsWideSnapshots(t *testing.T) {
	for _, test := range []struct {
		count, limit int
		reject       bool
	}{{128, 4096, true}, {2048, defaultBorrowWorkLimit, true}, {128, defaultBorrowWorkLimit, false}} {
		t.Run(fmt.Sprintf("%d_refs_%d_units", test.count, test.limit), func(t *testing.T) {
			files := borrowLiveFiles(t, wideBorrowLiveSource(test.count))
			result, err := analyzeContextWithOptions(context.Background(), files, nil, nil, nil, analysisOptions{borrowWorkLimit: test.limit})
			if err != nil || len(result.Diagnostics) != 0 || (result.CompletedThrough == PhaseFlow) == test.reject {
				t.Fatalf("phase=%s diagnostics=%v unsupported=%v err=%v", result.CompletedThrough, result.Diagnostics, result.Unsupported, err)
			}
			if test.reject && (len(result.Unsupported) != 1 || result.Unsupported[0].Feature != borrowWorkLimitFeature) || !test.reject && len(result.Unsupported) != 0 {
				t.Fatalf("unsupported=%v", result.Unsupported)
			}
		})
	}
}

func TestBorrowLiveBudgetOperations(t *testing.T) {
	p, q, absent := &symbol{name: "p"}, &symbol{name: "q"}, &symbol{name: "absent"}
	input := borrowLiveMap{p: {EndByte: 10}, q: {EndByte: 20}}
	want := maps.Clone(input)
	at := diagnostic.Span{StartByte: 1, EndByte: 2}
	for _, test := range []struct {
		name        string
		limit, used int
		exhausted   bool
		run         func(*borrowFlowAnalyzer) borrowLiveMap
		want        borrowLiveMap
	}{
		{"copy-exact", 2, 2, false, func(f *borrowFlowAnalyzer) borrowLiveMap { result, _ := f.cloneBorrowLive(at, input); return result }, want},
		{"copy-over", 1, 0, true, func(f *borrowFlowAnalyzer) borrowLiveMap { result, _ := f.cloneBorrowLive(at, input); return result }, nil},
		{"remove-exact", 2, 2, false, func(f *borrowFlowAnalyzer) borrowLiveMap { return f.withoutBorrowLive(at, input, p) }, borrowLiveMap{q: input[q]}},
		{"remove-over", 1, 0, true, func(f *borrowFlowAnalyzer) borrowLiveMap { return f.withoutBorrowLive(at, input, p) }, want},
		{"remove-absent", 1, 0, false, func(f *borrowFlowAnalyzer) borrowLiveMap { return f.withoutBorrowLive(at, input, absent) }, want},
		{"join-exact", 4, 4, false, func(f *borrowFlowAnalyzer) borrowLiveMap { return f.mergeBorrowLive(at, input, input) }, want},
		{"join-over", 3, 2, true, func(f *borrowFlowAnalyzer) borrowLiveMap { return f.mergeBorrowLive(at, input, input) }, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := &borrowFlowAnalyzer{analyzer: &analyzer{borrowWorkLimit: test.limit}, function: &ast.Function{}, module: &moduleScope{file: File{Path: "main.fy"}}}
			result := test.run(f)
			budget := f.analyzer.borrowBudget(f.function)
			if !maps.Equal(result, test.want) || budget.exhausted != test.exhausted || budget.remaining != test.limit-test.used || !maps.Equal(input, want) {
				t.Fatalf("result=%v budget=%+v input=%v", result, budget, input)
			}
			if test.exhausted {
				if len(f.analyzer.unsupported) != 1 || f.analyzer.unsupported[0].Span != span("main.fy", at) {
					t.Fatalf("missing source-located exhaustion: %v", f.analyzer.unsupported)
				}
			} else if test.used > 0 {
				result[p] = hir.Span{EndByte: 999}
				if !maps.Equal(input, want) {
					t.Fatal("budgeted map operation mutated its input snapshot")
				}
			}
		})
	}
}

func BenchmarkBorrowLiveWideSnapshots(b *testing.B) {
	files := borrowLiveFiles(b, wideBorrowLiveSource(2048))
	a := newAnalyzer(files, nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		b.Fatalf("invalid fixture: %v %v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	function := &module.file.Module.Functions[0]
	b.ReportAllocs()
	for b.Loop() {
		a.borrowWork = nil
		a.unsupported = nil
		a.diagnosticBudget = newDiagnosticBudget(0)
		flow := borrowFlowAnalyzer{analyzer: a, module: module, function: function,
			liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap)}
		flow.liveBlock(function.Body, nil, borrowLiveContext{})
		b.ReportMetric(float64(len(flow.liveAfterExpression)), "snapshots/op")
	}
}
