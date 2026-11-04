package semantic

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestBorrowOwnerBindingIndexFiltersForkedLiveUses(t *testing.T) {
	owner := &symbol{}
	first := &symbol{}
	second := &symbol{}
	otherOwner := &symbol{}
	stale := &symbol{}

	var index borrowOwnerBindingIndex
	index.add(owner, first)
	index.add(owner, second)
	index.add(owner, first)
	index.add(otherOwner, stale)
	flow := borrowFlowAnalyzer{ownerBindings: index}

	environment := borrowEnvironment{
		first:  {known: true, loans: []int{0}},
		second: {known: true, pending: []pendingBorrowLoan{{owner: owner}}},
		stale:  {known: true, loans: []int{1}},
	}
	firstUse := hir.Span{Source: "main.fy", StartByte: 10, EndByte: 11}
	secondUse := hir.Span{Source: "main.fy", StartByte: 20, EndByte: 21}
	for binding := range 512 {
		environment[&symbol{}] = borrowValueState{known: true, loans: []int{binding}}
	}

	var uses []hir.Span
	if candidates := flow.visitLiveOwnerBindings(owner,
		borrowLiveMap{first: firstUse, second: secondUse}, environment, func(_ borrowValueState, use hir.Span) {
			uses = append(uses, use)
		}); candidates != 2 {
		t.Fatalf("mutable fork inspected %d candidates, want 2", candidates)
	}
	if len(uses) != 2 || uses[0] != firstUse || uses[1] != secondUse {
		t.Fatalf("mutable fork uses = %+v", uses)
	}

	// The same index must follow a later fork's environment and liveness instead
	// of retaining the first branch's last-use evidence.
	uses = nil
	if candidates := flow.visitLiveOwnerBindings(owner, borrowLiveMap{second: secondUse},
		borrowEnvironment{first: environment[first], second: environment[second]},
		func(_ borrowValueState, use hir.Span) { uses = append(uses, use) }); candidates != 2 {
		t.Fatalf("second fork inspected %d candidates, want 2", candidates)
	}
	if len(uses) != 1 || uses[0] != secondUse {
		t.Fatalf("second fork uses = %+v", uses)
	}
}

func unrelatedOwnerBorrowFlow(tb testing.TB, borrows, assignments int) (*borrowFlowAnalyzer, *symbol) {
	tb.Helper()
	var text strings.Builder
	text.WriteString("module main;fn i32 f(){")
	for index := range borrows {
		fmt.Fprintf(&text, "var i32 a%d=%d;", index, index)
	}
	for index := range borrows {
		fmt.Fprintf(&text, "let ref i32 p%d=ref a%d;", index, index)
	}
	text.WriteString("var i32 sink=0;")
	for index := range assignments {
		fmt.Fprintf(&text, "sink=%d;", index)
	}
	for index := range borrows {
		fmt.Fprintf(&text, "discard *p%d;", index)
	}
	text.WriteString("return 0;}")

	a := newAnalyzer(borrowLiveFiles(tb, text.String()), nil)
	a.resolve()
	_ = a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("unrelated owner fixture: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	function := &module.file.Module.Functions[0]
	statement := &function.Body.Statements[2*borrows+1]
	sink := a.resolved[statement.Target]
	if sink == nil || len(statement.Target.Children) != 0 {
		tb.Fatalf("sink assignment target = %+v, symbol = %+v", statement.Target, sink)
	}

	flow := &borrowFlowAnalyzer{
		analyzer: a, module: module, function: function,
		liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap),
		loanByKey: make(map[borrowLoanKey]int),
	}
	flow.liveBlock(function.Body, make(borrowLiveMap), borrowLiveContext{})
	flow.evaluateBlock(function.Body, flow.parameterEnvironment(), true)
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("unrelated owner flow: %+v %+v", a.diagnostics, a.unsupported)
	}
	return flow, sink
}

func TestBorrowOwnerIndexSkipsUnrelatedLiveBindings(t *testing.T) {
	const borrows, assignments = 192, 192
	flow, sink := unrelatedOwnerBorrowFlow(t, borrows, assignments)

	if len(flow.ownerBindings) != borrows {
		t.Fatalf("owner index has %d owners, want %d", len(flow.ownerBindings), borrows)
	}
	if candidates := flow.ownerBindings.candidates(sink); len(candidates) != 0 {
		t.Fatalf("sink acquired %d unrelated loan bindings", len(candidates))
	}
	if len(flow.loans) != borrows {
		t.Fatalf("loan facts = %d, want %d", len(flow.loans), borrows)
	}
	for index := 1; index < len(flow.loans); index++ {
		if flow.loans[index-1].fact.Borrow.StartByte >= flow.loans[index].fact.Borrow.StartByte {
			t.Fatalf("loan fact order changed at %d: %+v", index, flow.loans)
		}
	}

	for offset := range assignments {
		statement := &flow.function.Body.Statements[2*borrows+1+offset]
		live := flow.liveAfter[statement]
		if len(live) != borrows {
			t.Fatalf("sink assignment %d has %d live loans, want %d", offset, len(live), borrows)
		}
		environment := make(borrowEnvironment, len(live))
		for binding := range live {
			environment[binding] = borrowValueState{known: true}
		}
		visited := 0
		if candidates := flow.visitLiveOwnerBindings(sink, live, environment,
			func(borrowValueState, hir.Span) { visited++ }); candidates != 0 || visited != 0 {
			t.Fatalf("sink assignment %d inspected %d candidates (%d visits) for %d unrelated live bindings",
				offset, candidates, visited, len(live))
		}
	}
}

func TestBorrowOwnerIndexKeepsLocalOwnerBesideParameterOrigins(t *testing.T) {
	const prelude = `module main;
resource R { cleanup {} }
fn void observe(ref R value) {}
fn void take(R value) {}
`
	for name, test := range map[string]struct{ body, code string }{
		"assignment": {`var R owner = R {}; let ref R view = ref owner; discard *alias; owner = R {}; observe(view);`, "E2101"},
		"move":       {`var R owner = R {}; let ref R view = ref owner; discard *alias; take(owner); observe(view);`, "E2105"},
	} {
		t.Run(name, func(t *testing.T) {
			source := prelude + `fn i32 f(ref i32 parameter) { let ref i32 alias = parameter; ` + test.body + ` return 0; }`
			result := Analyze(borrowLiveFiles(t, source))
			if len(result.Unsupported) != 0 || len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != test.code {
				t.Fatalf("parameter-origin owner %s: diagnostics=%+v unsupported=%+v", name, result.Diagnostics, result.Unsupported)
			}
			roles := make(map[string]bool)
			for _, related := range result.Diagnostics[0].Related {
				roles[related.Role] = true
			}
			if !roles["loan"] || !roles["last-use"] || test.code == "E2105" && !roles["owner"] {
				t.Fatalf("parameter-origin owner %s evidence: %+v", name, result.Diagnostics[0].Related)
			}
		})
	}
}

func BenchmarkBorrowOwnerAssignmentUnrelatedLoans(b *testing.B) {
	for _, count := range []int{64, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			flow, sink := unrelatedOwnerBorrowFlow(b, count, count)
			a, module, function := flow.analyzer, flow.module, flow.function
			b.ReportAllocs()
			for b.Loop() {
				run := &borrowFlowAnalyzer{
					analyzer: a, module: module, function: function,
					liveAfter: make(map[*ast.Statement]borrowLiveMap), liveAfterExpression: make(map[*ast.Expression]borrowLiveMap),
					loanByKey: make(map[borrowLoanKey]int),
				}
				run.liveBlock(function.Body, make(borrowLiveMap), borrowLiveContext{})
				run.evaluateBlock(function.Body, run.parameterEnvironment(), true)
				if candidates := run.ownerBindings.candidates(sink); len(candidates) != 0 {
					b.Fatalf("sink acquired %d unrelated loan bindings", len(candidates))
				}
			}
		})
	}
}
