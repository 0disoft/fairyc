package semantic

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestSubviewLoanMembershipPreservesDiagnostics(t *testing.T) {
	for _, tc := range []struct {
		name       string
		access     []int
		retained   []int
		borrowByte int
	}{
		{"sorted", []int{0, 1, 2, 3}, []int{0, 1, 2, 3}, 0},
		{"unsorted-access", []int{3, 1, 0, 2}, []int{0, 1, 2, 3}, 0},
		{"unsorted-retained", []int{0, 1, 2, 3}, []int{3, 1, 0, 2}, 0},
		{"duplicates", []int{0, 0, 1, 2, 3}, []int{0, 1, 1, 2, 3}, 0},
		{"missing-sorted", []int{0, 2, 3}, []int{0, 1, 2, 3}, 2},
		{"missing-unsorted", []int{3, 0, 2}, []int{3, 1, 0, 2}, 2},
		{"first-conflict-order", []int{0, 2}, []int{3, 1, 0, 2}, 4},
		{"invalid-ids", []int{0, 2}, []int{-1, 0, 2, 4}, 0},
		{"empty-access", nil, []int{0, 1, 2, 3}, 1},
		{"empty-retained", []int{0, 1, 2, 3}, nil, 0},
		{"single-access", []int{3}, []int{3}, 0},
		{"single-retained", []int{0, 1, 2, 3}, []int{3}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			flow, expression, owner, environment := subviewMembershipFixture(4)
			access := environment[owner]
			access.loans = tc.access
			environment[owner] = access
			lastUse := hir.Span{Source: "main.fy", StartByte: 9000}
			for retained := range flow.liveAfterExpression[expression] {
				value := environment[retained]
				value.loans = tc.retained
				environment[retained] = value
				flow.liveAfterExpression[expression][retained] = lastUse
			}
			wantAccess, wantRetained := slices.Clone(tc.access), slices.Clone(tc.retained)
			flow.checkSubviewPathAccess(expression, owner, environment)
			var want []Diagnostic
			if tc.borrowByte != 0 {
				want = []Diagnostic{{Code: "E2101", Primary: hir.Span{Source: "main.fy"}, Related: []RelatedSpan{
					{Role: "loan", Span: hir.Span{StartByte: tc.borrowByte}}, {Role: "last-use", Span: lastUse},
				}}}
			}
			if !reflect.DeepEqual(flow.analyzer.diagnostics, want) {
				t.Fatalf("diagnostics = %+v, want %+v", flow.analyzer.diagnostics, want)
			}
			if !slices.Equal(tc.access, wantAccess) || !slices.Equal(tc.retained, wantRetained) {
				t.Fatal("membership check mutated its input")
			}
		})
	}
}

func TestNestedSubviewLoanMembership(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		source := nestedSubviewMembershipSource(128, 2)
		if conflict {
			source = strings.Replace(source, "return sum;", "let u8 invalid=child0[0];return sum+child127.len;", 1)
		}
		result := Analyze(borrowLiveFiles(t, source))
		if len(result.Unsupported) != 0 {
			t.Fatalf("unsupported: %+v", result.Unsupported)
		}
		if !conflict && (len(result.Diagnostics) != 0 || result.CompletedThrough != PhaseFlow) {
			t.Fatalf("nested descendant rejected: %+v", result.Diagnostics)
		}
		if conflict && (len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" || len(result.Diagnostics[0].Related) != 2) {
			t.Fatalf("ancestor access lost its loan/last-use evidence: %+v", result.Diagnostics)
		}
	}
}

func subviewMembershipFixture(count int) (*borrowFlowAnalyzer, *ast.Expression, *symbol, borrowEnvironment) {
	owner, retained := &symbol{}, &symbol{}
	expression := &ast.Expression{}
	value := borrowValueState{known: true, origins: []flowOrigin{{name: "Param(view)"}}}
	flow := &borrowFlowAnalyzer{
		analyzer: &analyzer{}, module: &moduleScope{file: File{Path: "main.fy"}},
		ownerReadPlace:      expression,
		liveAfterExpression: map[*ast.Expression]borrowLiveMap{expression: {retained: {StartByte: 9000}}},
	}
	for index := range count {
		value.loans = append(value.loans, index)
		flow.loans = append(flow.loans, borrowLoanRecord{subview: true, owner: owner, fact: BorrowFact{Borrow: hir.Span{StartByte: index + 1}}})
	}
	return flow, expression, owner, borrowEnvironment{owner: value, retained: value}
}

func BenchmarkSubviewPathLoanMembership(b *testing.B) {
	for _, count := range []int{256, 1024, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			flow, expression, owner, environment := subviewMembershipFixture(count)
			b.ReportAllocs()
			for b.Loop() {
				flow.checkSubviewPathAccess(expression, owner, environment)
				if len(flow.analyzer.diagnostics) != 0 {
					b.Fatal("ancestor loans rejected a descendant access")
				}
			}
		})
	}
}

func nestedSubviewMembershipSource(count, reads int) string {
	var text strings.Builder
	text.WriteString("module main;fn usize f(ref mut u8[] view){let ref mut u8[] child0=view[..];")
	for index := 1; index < count; index++ {
		fmt.Fprintf(&text, "let ref mut u8[] child%d=child%d[..];", index, index-1)
	}
	text.WriteString("var usize sum=0;")
	for range reads {
		fmt.Fprintf(&text, "sum=sum+child%d.len;", count-1)
	}
	text.WriteString("return sum;}")
	return text.String()
}

func BenchmarkBorrowFlowNestedSubviewMembership(b *testing.B) {
	a := newAnalyzer(borrowLiveFiles(b, nestedSubviewMembershipSource(128, 128)), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		b.Fatalf("nested subview types: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	b.ReportAllocs()
	for b.Loop() {
		a.borrows = nil
		a.borrowFlowFunction(module, &module.file.Module.Functions[0])
		if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
			b.Fatalf("nested subview flow: %+v %+v", a.diagnostics, a.unsupported)
		}
	}
}
