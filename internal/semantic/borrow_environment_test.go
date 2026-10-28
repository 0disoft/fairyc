package semantic

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func BenchmarkBorrowFlowBranchAliases(b *testing.B) {
	var text strings.Builder
	text.WriteString("module main;fn i32 f(bool flag")
	for index := range 64 {
		fmt.Fprintf(&text, ",ref i32 p%d", index)
	}
	text.WriteString("){var ref i32 selected=p0;")
	for index := 1; index < 64; index++ {
		fmt.Fprintf(&text, "if(flag){selected=p%d;}", index)
	}
	for index := range 32 {
		fmt.Fprintf(&text, "let ref i32 alias%d=selected;", index)
	}
	text.WriteString("var i32 sum=0;")
	text.WriteString(strings.Repeat("if(flag){sum=sum+1;}", 128))
	text.WriteString("return sum")
	for index := range 32 {
		fmt.Fprintf(&text, "+*alias%d", index)
	}
	text.WriteString(";}")
	a := newAnalyzer(borrowLiveFiles(b, text.String()), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		b.Fatalf("branch aliases: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	function := &module.file.Module.Functions[0]
	b.ReportAllocs()
	for b.Loop() {
		a.borrows = nil
		a.borrowFlowFunction(module, function)
		if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
			b.Fatalf("branch aliases flow: %+v %+v", a.diagnostics, a.unsupported)
		}
	}
}

func TestBorrowEnvironmentForksIsolateUpdates(t *testing.T) {
	binding := &symbol{}
	origins := make([]flowOrigin, 2, 4)
	origins[0] = flowOrigin{name: "Local(a)", declaration: hir.Span{Source: "main.fy", StartByte: 20}}
	origins[1] = flowOrigin{name: "Local(b)", declaration: hir.Span{Source: "main.fy", StartByte: 10}}
	loans := make([]int, 2, 4)
	loans[0], loans[1] = 4, 2
	pending := make([]pendingBorrowLoan, 1, 4)
	pending[0] = pendingBorrowLoan{owner: binding, kind: "mutable", origin: origins[0], borrow: hir.Span{StartByte: 42}}
	// Spare capacity detects appends through a shared backing array too.
	origins[:cap(origins)][2] = flowOrigin{name: "spare"}
	loans[:cap(loans)][2] = 999
	pending[:cap(pending)][1] = pendingBorrowLoan{kind: "spare"}
	wantOrigins := slices.Clone(origins[:cap(origins)])
	wantLoans := slices.Clone(loans[:cap(loans)])
	wantPending := slices.Clone(pending[:cap(pending)])
	input := borrowEnvironment{binding: {known: true, origins: origins, loans: loans, pending: pending}}
	left, right := cloneBorrowEnvironment(input), cloneBorrowEnvironment(input)
	flow := borrowFlowAnalyzer{loanByKey: make(map[borrowLoanKey]int)}
	left[binding] = flow.materialize(left[binding], binding)
	if !left[binding].known || len(left[binding].pending) != 0 || !slices.Equal(left[binding].loans, []int{0, 2, 4}) {
		t.Fatalf("materialized branch = %+v", left[binding])
	}
	if !sameBorrowValue(input[binding], right[binding]) || !reflect.DeepEqual(origins[:cap(origins)], wantOrigins) ||
		!slices.Equal(loans[:cap(loans)], wantLoans) || !reflect.DeepEqual(pending[:cap(pending)], wantPending) {
		t.Fatal("materialization changed another branch or its spare capacity")
	}
	// A join must produce independently writable arrays, even with shared inputs.
	joined := mergeBorrowEnvironments(input, []borrowEnvironment{left, right})
	joined[binding].origins[0] = flowOrigin{name: "changed"}
	joined[binding].loans[0] = 100
	joined[binding].pending[0].kind = "changed"
	if !reflect.DeepEqual(origins[:cap(origins)], wantOrigins) || !slices.Equal(loans[:cap(loans)], wantLoans) ||
		!reflect.DeepEqual(pending[:cap(pending)], wantPending) {
		t.Fatal("join mutated an incoming environment")
	}
	// Environment replacement preserves source values; later writes detach first.
	old := &symbol{}
	replaced := borrowEnvironment{old: {known: true}}
	replaceBorrowEnvironment(replaced, right)
	if len(replaced) != 1 || !sameBorrowValue(replaced[binding], input[binding]) {
		t.Fatal("replacement did not retain exactly the source bindings")
	}
	updated := cloneBorrowValue(replaced[binding])
	updated.origins[0].name = "new"
	updated.loans[0] = 101
	updated.pending[0].kind = "new"
	replaced[binding] = updated
	delete(left, binding)
	if len(input) != 1 || len(right) != 1 || !reflect.DeepEqual(origins[:cap(origins)], wantOrigins) ||
		!slices.Equal(loans[:cap(loans)], wantLoans) || !reflect.DeepEqual(pending[:cap(pending)], wantPending) {
		t.Fatal("reassignment or deletion changed another environment")
	}
}

func TestBorrowEnvironmentEmptyForksAreIndependent(t *testing.T) {
	for _, input := range []borrowEnvironment{nil, {}} {
		cloned := cloneBorrowEnvironment(input)
		cloned[&symbol{}] = borrowValueState{known: true}
		if len(input) != 0 || len(cloned) != 1 {
			t.Fatal("empty fork must be independently writable")
		}
	}
}

func BenchmarkCloneBorrowEnvironmentEvidence(b *testing.B) {
	for _, count := range []int{1, 256} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			input := make(borrowEnvironment)
			for local := range 64 {
				state := borrowValueState{known: true}
				for index := range count {
					state.origins = append(state.origins, flowOrigin{name: "Local", declaration: hir.Span{Source: "main.fy", StartByte: local*count + index}})
					state.loans = append(state.loans, local*count+index)
				}
				input[&symbol{}] = state
			}
			b.ReportAllocs()
			for b.Loop() {
				if cloned := cloneBorrowEnvironment(input); len(cloned) != len(input) {
					b.Fatal("incomplete environment clone")
				}
			}
		})
	}
}
