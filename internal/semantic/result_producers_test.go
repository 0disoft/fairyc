package semantic

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestResultProducerIdentityOrderAndScratchReuse(t *testing.T) {
	var set resultProducerSet
	var want []hir.Span
	for index := range 12 {
		value := hir.Span{Package: 1, Source: "main.fy", StartByte: index, EndByte: index + 1}
		set.add(value, value)
		want = append(want, value)
		if index < 8 && set.indexed {
			t.Fatal("small producer set allocated an index")
		}
	}
	// Every component of compareSpan's identity must participate in membership.
	for _, value := range []hir.Span{
		{Package: 2, Source: "main.fy", EndByte: 1},
		{Package: 1, Source: "other.fy", EndByte: 1},
		{Package: 1, Source: "main.fy", EndByte: 2},
	} {
		set.add(value)
		want = append(want, value)
	}
	set.add(want...)
	if !set.indexed || !slices.Equal(set.values, want) {
		t.Fatalf("producer order/identity = %+v, want %+v", set.values, want)
	}
	prior := slices.Clone(set.values)
	next := resultProducerSet{seen: set.seen}
	next.add(want[:3]...)
	if next.indexed || !slices.Equal(next.values, want[:3]) {
		t.Fatal("inactive scratch index suppressed small-set producers")
	}
	next.add(want[3:]...)
	if !slices.Equal(next.values, want) || !slices.Equal(set.values, prior) {
		t.Fatal("reused index changed membership or a retained producer list")
	}
}

func TestMergeResultProducerReachabilityAndBaseline(t *testing.T) {
	value := &symbol{}
	first := hir.Span{Source: "main.fy", StartByte: 1}
	second := hir.Span{Source: "main.fy", StartByte: 2}
	baseline := resultEnvironment{value: pendingResult(first)}
	actual := resultEnvironment{value: {pending: true, producers: []hir.Span{second, first, second}}}
	var a analyzer
	target := make(resultEnvironment)
	if !a.mergeResultBranches(target, baseline, []resultBranch{
		{environment: actual, reachable: true},
		{environment: nil, reachable: true}, // Missing entries use baseline evidence.
		{environment: resultEnvironment{value: pendingResult(hir.Span{StartByte: 3})}},
		{environment: resultEnvironment{value: {}}, reachable: true},
	}) || !target[value].pending || !slices.Equal(target[value].producers, []hir.Span{second, first}) {
		t.Fatalf("merged obligation = %+v", target[value])
	}
	producers := target[value].producers
	producers[0] = hir.Span{}
	if actual[value].producers[0] != second || baseline[value].producers[0] != first {
		t.Fatal("merged producers alias branch evidence")
	}
	if a.mergeResultBranches(target, baseline, nil) || target[value].pending || len(target[value].producers) != 0 {
		t.Fatal("unreachable branches retained an obligation")
	}
}

func TestResultFlowAccumulatedProducerDiagnostics(t *testing.T) {
	for _, handled := range []bool{false, true} {
		text := "module main;fn void f(bool flag){var Result<i32,i32> r=.ok(0);" +
			strings.Repeat("if(flag){discard r;r=.ok(1);}", 32)
		if handled {
			text += "discard r;"
		}
		text += "}"
		result := Analyze(borrowLiveFiles(t, text))
		if len(result.Unsupported) != 0 || handled && len(result.Diagnostics) != 0 || !handled && len(result.Diagnostics) != 33 {
			t.Fatalf("handled=%v: diagnostics %+v, unsupported %+v", handled, result.Diagnostics, result.Unsupported)
		}
		offset := 0
		for _, diagnostic := range result.Diagnostics {
			start := offset + strings.Index(text[offset:], ".ok(")
			if diagnostic.Code != "E4101" || diagnostic.Primary != sourceSpan("main.fy", start, start+6) {
				t.Fatalf("producer diagnostic = %+v, want offset %d", diagnostic, start)
			}
			offset = start + 6
		}
	}
	a, module, function := resultProducerFixture(t, 32)
	a.resultFlowFunction(module, function)
	if a.resultProducerIndex != nil {
		t.Fatal("function retained its temporary producer index")
	}
}

func resultProducerFixture(tb testing.TB, count int) (*analyzer, *moduleScope, *ast.Function) {
	tb.Helper()
	text := "module main;fn void f(bool flag){var Result<i32,i32> r=.ok(0);" +
		strings.Repeat("if(flag){discard r;r=.ok(1);}", count) + "discard r;}"
	a := newAnalyzer(borrowLiveFiles(tb, text), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("result producer fixture: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	return a, module, &module.file.Module.Functions[0]
}

func BenchmarkResultFlowAccumulatedProducers(b *testing.B) {
	for _, count := range []int{128, 512} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			a, module, function := resultProducerFixture(b, count)
			b.ReportAllocs()
			for b.Loop() {
				a.resultHandling = nil
				a.resultFlowFunction(module, function)
				if len(a.diagnostics) != 0 {
					b.Fatalf("result flow: %+v", a.diagnostics)
				}
			}
		})
	}
}
