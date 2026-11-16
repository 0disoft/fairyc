package semantic

import (
	"fmt"
	"reflect"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/hir"
)

func unrelatedBorrowCreation(count int) (*borrowFlowAnalyzer, *ast.Expression, *symbol, borrowEnvironment) {
	expression := &ast.Expression{Span: diagnostic.Span{StartByte: 1, EndByte: 2}}
	owner := &symbol{name: "target"}
	live := make(borrowLiveMap, count)
	environment := make(borrowEnvironment, count)
	flow := &borrowFlowAnalyzer{
		analyzer:            &analyzer{diagnosticBudget: newDiagnosticBudget(DefaultSemanticReportLimit)},
		module:              &moduleScope{file: File{Path: "main.fy"}},
		liveAfterExpression: map[*ast.Expression]borrowLiveMap{expression: live},
	}
	for index := range count {
		binding, other := &symbol{name: fmt.Sprint(index)}, &symbol{name: "other"}
		borrow := hir.Span{Source: "main.fy", StartByte: index + 10, EndByte: index + 11}
		flow.loans = append(flow.loans, borrowLoanRecord{owner: other, fact: BorrowFact{Kind: "shared", Borrow: borrow}})
		value := borrowValueState{known: true, loans: []int{index}}
		live[binding], environment[binding] = borrow, value
		flow.ownerBindings.addValue(value, binding, flow.loans)
	}
	return flow, expression, owner, environment
}

func TestBorrowCreationKeepsLiveEvidenceBeforeHeldDuplicates(t *testing.T) {
	flow, expression, owner, environment := unrelatedBorrowCreation(64)
	binding := &symbol{name: "view"}
	borrow := hir.Span{Source: "main.fy", StartByte: 100, EndByte: 110}
	lastUse := hir.Span{Source: "main.fy", StartByte: 200, EndByte: 210}
	value := borrowValueState{known: true, loans: []int{-1, len(flow.loans), len(flow.loans) + 1}}
	flow.loans = append(flow.loans, borrowLoanRecord{owner: owner, fact: BorrowFact{Kind: "mutable", Borrow: borrow}})
	value.pending = []pendingBorrowLoan{{owner: owner, kind: "mutable", borrow: borrow}}
	environment[binding] = value
	flow.liveAfterExpression[expression][binding] = lastUse
	flow.ownerBindings.addValue(value, binding, flow.loans)
	flow.callBorrows = []borrowValueState{value}

	// Cumulative candidates can disappear from liveness or change owners in a fork.
	stale := &symbol{name: "stale"}
	environment[stale] = value
	flow.ownerBindings.add(owner, stale)
	for binding := range environment {
		flow.ownerBindings.add(owner, binding)
	}
	flow.checkNamedBorrowCreation(expression, owner, "shared", environment)
	want := []Diagnostic{{Code: "E2101", Primary: span("main.fy", expression.Span), Related: []RelatedSpan{
		{Role: "loan", Span: borrow}, {Role: "last-use", Span: lastUse},
	}}}
	if !reflect.DeepEqual(flow.analyzer.diagnostics, want) {
		t.Fatalf("diagnostics = %+v, want %+v", flow.analyzer.diagnostics, want)
	}
}

func TestBorrowCreationConflictsPreservesKindsBoundsAndInputs(t *testing.T) {
	for _, test := range []struct {
		name, kind, priorKind string
		conflict              bool
	}{
		{"shared/shared", "shared", "shared", false},
		{"shared/mutable", "shared", "mutable", true},
		{"mutable/shared", "mutable", "shared", true},
		{"mutable/mutable", "mutable", "mutable", true},
	} {
		t.Run(test.name, func(t *testing.T) {
			owner, other := &symbol{name: "owner"}, &symbol{name: "other"}
			retained := hir.Span{Source: "main.fy", StartByte: 10, EndByte: 20}
			pending := hir.Span{Source: "main.fy", StartByte: 30, EndByte: 40}
			loans := []borrowLoanRecord{
				{owner: other, fact: BorrowFact{Kind: "mutable", Borrow: retained}},
				{owner: owner, fact: BorrowFact{Kind: test.priorKind, Borrow: retained}},
			}
			value := borrowValueState{loans: []int{-1, 0, len(loans) - 1, len(loans)}, pending: []pendingBorrowLoan{
				{owner: other, kind: "mutable", borrow: pending},
				{owner: owner, kind: test.priorKind, borrow: pending},
			}}
			wantValue, wantLoans := cloneBorrowValue(value), slices.Clone(loans)
			got := borrowCreationConflicts(value, loans, owner, test.kind)
			var want []hir.Span
			if test.conflict {
				want = []hir.Span{retained, pending}
			}
			if !slices.Equal(got, want) {
				t.Fatalf("conflicts = %+v, want %+v", got, want)
			}
			if len(got) != 0 {
				got[0].Source = "changed by caller"
			}
			if !reflect.DeepEqual(value, wantValue) || !reflect.DeepEqual(loans, wantLoans) {
				t.Fatal("conflict selection or result mutation changed an input")
			}
			if !test.conflict {
				if allocations := testing.AllocsPerRun(100, func() {
					_ = borrowCreationConflicts(value, loans, owner, test.kind)
				}); allocations != 0 {
					t.Fatalf("compatible accesses allocated %v objects", allocations)
				}
			}
		})
	}
}

func BenchmarkBorrowCreationUnrelatedBindings(b *testing.B) {
	for _, count := range []int{0, 64, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			flow, expression, owner, environment := unrelatedBorrowCreation(count)
			b.ReportAllocs()
			for b.Loop() {
				flow.checkNamedBorrowCreation(expression, owner, "mutable", environment)
			}
			if len(flow.analyzer.diagnostics) != 0 {
				b.Fatal("unrelated bindings produced conflicts")
			}
		})
	}
}
