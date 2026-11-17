package semantic

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func retiredOwnerFixture(tb testing.TB, count int) (*analyzer, *moduleScope, *ast.Function) {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;fn void f(ref i32 keep){var i32 owner=0;")
	text.WriteString(strings.Repeat("{let ref i32 dead=ref owner;discard *dead;}owner=1;", count))
	text.WriteString("discard *keep;}")
	a := newAnalyzer(borrowLiveFiles(tb, text.String()), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("invalid retired-owner fixture: %v %v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	return a, module, &module.file.Module.Functions[0]
}

func evaluateRetiredOwner(a *analyzer, module *moduleScope, function *ast.Function) *borrowFlowAnalyzer {
	a.borrowWork = nil
	f := &borrowFlowAnalyzer{analyzer: a, module: module, function: function,
		liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap),
		loanByKey: make(map[borrowLoanKey]int)}
	f.liveBlock(function.Body, nil, borrowLiveContext{})
	f.evaluateBlock(function.Body, f.parameterEnvironment(), true)
	return f
}

func TestBorrowOwnerRetiredScopesStayBounded(t *testing.T) {
	const count = 256
	a, module, function := retiredOwnerFixture(t, count)
	f := evaluateRetiredOwner(a, module, function)
	assignment := &function.Body.Statements[2*count]
	owner := a.resolved[assignment.Target]
	live := f.liveAfter[assignment]
	visited := 0
	inspected := f.visitLiveOwnerBindings(owner, live, f.parameterEnvironment(), func(borrowValueState, hir.Span) { visited++ })
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 || len(f.loans) != count || len(live) != 1 || visited != 0 || inspected != 1 {
		t.Fatalf("retired scopes: loans=%d live=%d inspected=%d visited=%d diagnostics=%v unsupported=%v", len(f.loans), len(live), inspected, visited, a.diagnostics, a.unsupported)
	}
}

func TestBorrowOwnerLiveIntersectionKeepsOrderAndForkFilters(t *testing.T) {
	owner, unrelated := &symbol{}, &symbol{}
	bindings := make([]*symbol, 128)
	f := &borrowFlowAnalyzer{}
	environment := make(borrowEnvironment)
	for index := range bindings {
		bindings[index] = &symbol{}
		f.ownerBindings.add(owner, bindings[index])
		f.ownerBindings.add(owner, bindings[index]) // Duplicate adds must keep rank.
		if index%3 != 0 {
			environment[bindings[index]] = borrowValueState{known: true}
		}
	}
	for _, count := range []int{0, 1, 3, 12, 64, 128} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			live := borrowLiveMap{unrelated: {EndByte: 999}}
			for index := count - 1; index >= 0; index-- {
				live[bindings[index]] = hir.Span{EndByte: index + 1}
			}
			var want []hir.Span
			for _, binding := range bindings {
				if use, retained := live[binding]; retained {
					if _, present := environment[binding]; present {
						want = append(want, use)
					}
				}
			}
			var got []hir.Span
			f.visitLiveOwnerBindings(owner, live, environment, func(_ borrowValueState, use hir.Span) { got = append(got, use) })
			if !slices.Equal(got, want) {
				t.Fatalf("changed owner traversal: got=%v want=%v", got, want)
			}
		})
	}
}

func BenchmarkBorrowOwnerRetiredScopes(b *testing.B) {
	for _, count := range []int{256, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			a, module, function := retiredOwnerFixture(b, count)
			b.ReportAllocs()
			for b.Loop() {
				f := evaluateRetiredOwner(a, module, function)
				if len(a.diagnostics) != 0 || len(a.unsupported) != 0 || len(f.loans) != count {
					b.Fatalf("incomplete flow: loans=%d diagnostics=%v unsupported=%v", len(f.loans), a.diagnostics, a.unsupported)
				}
			}
		})
	}
}
