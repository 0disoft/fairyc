package semantic

import (
	"fmt"
	"maps"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/syntax"
)

func TestBorrowLiveSnapshotsSurviveEarlierWrites(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 f(ref i32 p,ref i32 q){var ref i32 local=p;local=q;return *local;}")
	statements := f.function.Body.Statements[:2]
	local := f.analyzer.bindingSymbols[statements[0].Binding]
	p := f.analyzer.resolved[statements[0].Binding.Expression]
	q := f.analyzer.resolved[statements[1].Expression]
	later := hir.Span{Source: "main.fy", StartByte: 1000, EndByte: 1001}
	out := borrowLiveMap{local: later}
	live := f.liveBlock(&ast.Block{Statements: statements}, out, borrowLiveContext{})
	if !maps.Equal(out, borrowLiveMap{local: later}) {
		t.Fatalf("binding/assignment mutated input: %+v", out)
	}
	if snapshot := f.liveAfter[&statements[1]]; !maps.Equal(snapshot, out) {
		t.Fatalf("assignment killed its future snapshot: %+v", snapshot)
	}
	qUse := span(f.module.file.Path, statements[1].Expression.Span)
	if snapshot := f.liveAfter[&statements[0]]; !maps.Equal(snapshot, borrowLiveMap{q: qUse}) {
		t.Fatalf("earlier binding changed assignment result: %+v", snapshot)
	}
	if snapshot := f.liveAfterExpression[statements[0].Binding.Expression]; !maps.Equal(snapshot, borrowLiveMap{q: qUse}) {
		t.Fatalf("path insertion changed expression snapshot: %+v", snapshot)
	}
	if len(live) != 2 || live[p] != span(f.module.file.Path, statements[0].Binding.Expression.Span) || live[q] != qUse {
		t.Fatalf("live inputs = %+v, want p and q", live)
	}
}

func TestBorrowLiveSharedBranchAndLoopEdges(t *testing.T) {
	for _, edge := range []syntax.Kind{syntax.KindBreak, syntax.KindContinue} {
		t.Run(string(edge), func(t *testing.T) {
			f, _ := scalarLoopFlow(t, "module main;fn i32 f(ref i32 p){let ref i32 local=p;return *local;}")
			binding := f.function.Body.Statements[0]
			local := f.analyzer.bindingSymbols[binding.Binding]
			later := hir.Span{Source: "main.fy", StartByte: 1000, EndByte: 1001}
			out := borrowLiveMap{local: later}
			context := borrowLiveContext{breakLive: out, continueLive: out}
			block := &ast.Block{Statements: []ast.Statement{binding, {Kind: edge}}}
			f.liveBlock(block, nil, context)
			if !maps.Equal(out, borrowLiveMap{local: later}) {
				t.Fatalf("loop edge mutated by earlier binding: %+v", out)
			}
			// The first branch kills local. The sibling must still see it.
			sibling := &ast.Expression{Kind: syntax.KindPath}
			branch := &ast.Expression{Kind: syntax.KindIf, Children: []*ast.Expression{
				nil, {Kind: syntax.KindBlock, Block: &ast.Block{Statements: []ast.Statement{binding}}}, sibling,
			}}
			f.liveExpression(branch, out, context)
			if snapshot := f.liveAfterExpression[sibling]; !maps.Equal(snapshot, out) || len(out) != 1 {
				t.Fatalf("sibling branch lost live borrow: %+v, input %+v", snapshot, out)
			}
		})
	}
}

func TestBorrowLiveSharedPatternAndForInitializer(t *testing.T) {
	f, _ := scalarLoopFlow(t, "module main;fn i32 f(ref i32 p){let ref i32 local=p;return *local;}")
	binding := f.function.Body.Statements[0]
	local := f.analyzer.bindingSymbols[binding.Binding]
	later := hir.Span{Source: "main.fy", StartByte: 1000, EndByte: 1001}
	out := borrowLiveMap{local: later}
	patternBinding := &ast.Expression{Kind: syntax.KindPath}
	f.analyzer.resolved[patternBinding] = local
	armBody := &ast.Expression{Kind: syntax.KindPath}
	match := &ast.Expression{Kind: syntax.KindSwitch, Children: []*ast.Expression{
		nil, {Children: []*ast.Expression{{Children: []*ast.Expression{nil, patternBinding}}, armBody}},
	}}
	if live := f.liveExpression(match, out, borrowLiveContext{}); len(live) != 0 {
		t.Fatalf("pattern binding escaped its arm: %+v", live)
	}
	if !maps.Equal(out, borrowLiveMap{local: later}) || !maps.Equal(f.liveAfterExpression[armBody], out) {
		t.Fatal("pattern removal changed the arm's input or snapshot")
	}
	f.liveFor(&ast.ForLoop{Initializer: &binding, Condition: &ast.Expression{Kind: syntax.KindLiteral}, Body: &ast.Block{}}, out, borrowLiveContext{})
	if !maps.Equal(out, borrowLiveMap{local: later}) {
		t.Fatalf("for initializer changed loop exit liveness: %+v", out)
	}
	read := binding.Binding.Expression
	p := f.analyzer.resolved[read]
	earlier := hir.Span{Source: "main.fy", EndByte: 1}
	input := borrowLiveMap{p: earlier}
	live := f.liveExpression(read, input, borrowLiveContext{})
	if live[p] != span(f.module.file.Path, read.Span) || input[p] != earlier || f.liveAfterExpression[read][p] != earlier {
		t.Fatal("advancing a last use mutated its input or snapshot")
	}
}

func borrowLiveSharedFixture(tb testing.TB, count int) (*analyzer, *moduleScope, *ast.Function) {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;fn i32 keep(")
	for index := range 64 {
		if index != 0 {
			text.WriteByte(',')
		}
		fmt.Fprintf(&text, "ref i32 p%d", index)
	}
	text.WriteString("){var i32 x=0;")
	text.WriteString(strings.Repeat("x=x+1;", count))
	text.WriteString("return x")
	for index := range 64 {
		fmt.Fprintf(&text, "+*p%d", index)
	}
	text.WriteString(";}")
	a := newAnalyzer(borrowLiveFiles(tb, text.String()), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("shared liveness fixture: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	return a, module, &module.file.Module.Functions[0]
}

func BenchmarkBorrowLiveRetainedReferences(b *testing.B) {
	for _, count := range []int{128, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			a, module, function := borrowLiveSharedFixture(b, count)
			b.ReportAllocs()
			for b.Loop() {
				flow := borrowFlowAnalyzer{analyzer: a, module: module, function: function,
					liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap)}
				if live := flow.liveBlock(function.Body, nil, borrowLiveContext{}); len(live) != 64 {
					b.Fatalf("live parameters = %d, want 64", len(live))
				}
			}
		})
	}
}
