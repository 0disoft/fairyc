package semantic

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/hir"
)

func referenceBorrowValuesOverlap(left, right borrowValueState) bool {
	for _, a := range left.origins {
		for _, b := range right.origins {
			if sameFlowOrigin(a, b) {
				return true
			}
		}
	}
	return false
}

func TestBorrowValuesOverlapPreservesIdentityAndInputs(t *testing.T) {
	random := rand.New(rand.NewPCG(42, 17))
	pool := make([]flowOrigin, 64)
	for index := range pool {
		pool[index] = flowOrigin{
			name:        fmt.Sprint(index % 3),
			declaration: hir.Span{Package: hir.PackageID(index % 2), Source: fmt.Sprint(index % 5), StartByte: index % 7, EndByte: index % 11},
		}
	}
	for trial := range 300 {
		left, right := borrowValueState{}, borrowValueState{}
		for range random.IntN(40) {
			left.origins = append(left.origins, pool[random.IntN(len(pool))])
		}
		for range random.IntN(40) {
			right.origins = append(right.origins, pool[random.IntN(len(pool))])
		}
		if trial%2 == 0 {
			left.origins = compactOrigins(left.origins)
			right.origins = compactOrigins(right.origins)
		}
		wantLeft, wantRight := slices.Clone(left.origins), slices.Clone(right.origins)
		want := referenceBorrowValuesOverlap(left, right)
		if got := borrowValuesOverlap(left, right); got != want || borrowValuesOverlap(right, left) != want {
			t.Fatalf("trial %d overlap = %v, want %v", trial, got, want)
		}
		if !slices.Equal(left.origins, wantLeft) || !slices.Equal(right.origins, wantRight) {
			t.Fatalf("trial %d mutated an input", trial)
		}
	}
	// Allocation pointers are evidence, not part of sameFlowOrigin identity.
	left := borrowValueState{origins: make([]flowOrigin, 16)}
	right := borrowValueState{origins: make([]flowOrigin, 16)}
	for index := range 16 {
		left.origins[index] = flowOrigin{name: "left", declaration: hir.Span{StartByte: index}}
		right.origins[index] = flowOrigin{name: "right", declaration: hir.Span{StartByte: index}}
	}
	if borrowValuesOverlap(left, right) {
		t.Fatal("equal declaration spans with different origin names overlap")
	}
	right.origins[15] = left.origins[0]
	left.origins[0].allocation = &hir.Span{StartByte: 100}
	right.origins[15].allocation = &hir.Span{StartByte: 200}
	if !borrowValuesOverlap(left, right) {
		t.Fatal("different allocation evidence hid the same lexical origin")
	}
}

func TestBorrowOriginOverlapAtCalls(t *testing.T) {
	for _, conflict := range []bool{false, true} {
		result := Analyze(borrowLiveFiles(t, borrowOverlapSource(32, 1, conflict)))
		if len(result.Unsupported) != 0 {
			t.Fatalf("unsupported = %+v", result.Unsupported)
		}
		if !conflict && (len(result.Diagnostics) != 0 || result.CompletedThrough != PhaseFlow) {
			t.Fatalf("disjoint origins rejected: %+v", result.Diagnostics)
		}
		if conflict && (len(result.Diagnostics) != 1 || result.Diagnostics[0].Code != "E2101" || len(result.Diagnostics[0].Related) != 2) {
			t.Fatalf("overlap must retain one loan/last-use conflict: %+v", result.Diagnostics)
		}
	}
}

func TestSortedBorrowOriginOverlapDoesNotAllocate(t *testing.T) {
	left, right := borrowValueState{}, borrowValueState{}
	for index := range 1024 {
		left.origins = append(left.origins, flowOrigin{name: "Param", declaration: hir.Span{StartByte: index * 2}})
		right.origins = append(right.origins, flowOrigin{name: "Param", declaration: hir.Span{StartByte: index*2 + 1}})
	}
	if allocations := testing.AllocsPerRun(10, func() {
		if borrowValuesOverlap(left, right) {
			t.Fatal("interleaved disjoint origins overlap")
		}
	}); allocations != 0 {
		t.Fatalf("sorted origin check allocated %v objects", allocations)
	}
	// A match near the end must survive advancing both cursors.
	right.origins[len(right.origins)-1] = left.origins[len(left.origins)-1]
	if !borrowValuesOverlap(left, right) {
		t.Fatal("sorted origin check missed the final shared origin")
	}
}

func BenchmarkBorrowValuesOverlap(b *testing.B) {
	for _, count := range []int{4, 1024} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			left, right := borrowValueState{}, borrowValueState{}
			for index := range count {
				left.origins = append(left.origins, flowOrigin{name: "Local", declaration: hir.Span{Source: "main.fy", StartByte: index}})
				right.origins = append(right.origins, flowOrigin{name: "Local", declaration: hir.Span{Source: "main.fy", StartByte: count + index}})
			}
			b.ReportAllocs()
			for b.Loop() {
				if borrowValuesOverlap(left, right) {
					b.Fatal("disjoint origins overlap")
				}
			}
		})
	}
}

func borrowOverlapSource(count, calls int, conflict bool) string {
	var text strings.Builder
	text.WriteString("module main;fn void both(ref i32 left, ref mut i32 right){} fn void f(bool flag")
	for _, prefix := range []string{"l", "r"} {
		for index := range count {
			fmt.Fprintf(&text, ",ref mut i32 %s%d", prefix, index)
		}
	}
	text.WriteString("){var ref mut i32 left=l0;var ref mut i32 right=r0;")
	for index := 1; index < count; index++ {
		fmt.Fprintf(&text, "if(flag){left=l%d;right=r%d;}", index, index)
	}
	if conflict {
		text.WriteString("right=left;")
	}
	text.WriteString(strings.Repeat("both(left,right);", calls))
	text.WriteString("}")
	return text.String()
}

func borrowOverlapFixture(tb testing.TB, count, calls int) (*analyzer, *moduleScope, *ast.Function) {
	tb.Helper()
	a := newAnalyzer(borrowLiveFiles(tb, borrowOverlapSource(count, calls, false)), nil)
	a.resolve()
	a.typeCheck()
	if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
		tb.Fatalf("overlap source: %+v %+v", a.diagnostics, a.unsupported)
	}
	module := a.moduleOrder[0]
	return a, module, &module.file.Module.Functions[1]
}

func BenchmarkBorrowFlowOriginOverlap(b *testing.B) {
	a, module, function := borrowOverlapFixture(b, 128, 1024)
	b.ReportAllocs()
	for b.Loop() {
		a.borrows = nil
		a.borrowFlowFunction(module, function)
		if len(a.diagnostics) != 0 || len(a.unsupported) != 0 {
			b.Fatalf("borrow flow: %+v %+v", a.diagnostics, a.unsupported)
		}
	}
}
