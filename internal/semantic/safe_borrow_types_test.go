package semantic

import (
	"fmt"
	"strconv"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func sharedBorrowResultType(depth int, leaf hir.Type) hir.Type {
	for range depth {
		leaf = hir.Result(leaf, leaf)
	}
	return leaf
}

func TestSafeBorrowLeafSharedGraph(t *testing.T) {
	shared := hir.SharedBorrow(hir.TypeI32)
	exclusive := hir.MutableBorrow(hir.TypeU8)
	for _, test := range []struct {
		name    string
		typ     hir.Type
		element hir.Type
		mutable bool
		count   int
	}{
		{"scalar", hir.TypeI32, hir.TypeInvalid, false, 0},
		{"shared", shared, hir.TypeI32, false, 1},
		{"exclusive", exclusive, hir.TypeU8, true, 1},
		{"failure-borrow", hir.Result(hir.TypeI32, hir.Option(exclusive)), hir.TypeU8, true, 1},
		{"first-origin", hir.Result(shared, exclusive), hir.TypeI32, false, 2},
		{"three-paths", hir.Result(hir.Result(shared, shared), exclusive), hir.TypeI32, false, 3},
		{"shared-borrow-graph", sharedBorrowResultType(20, shared), hir.TypeI32, false, 1 << 20},
		{"shared-scalar-graph", sharedBorrowResultType(80, hir.TypeI32), hir.TypeInvalid, false, 0},
		{"overflow", sharedBorrowResultType(strconv.IntSize+1, exclusive), hir.TypeU8, true, int(^uint(0) >> 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			element, mutable, count := safeBorrowLeaf(test.typ)
			if element != test.element || mutable != test.mutable || count != test.count {
				t.Fatalf("got (%v, %t, %d), want (%v, %t, %d)", element, mutable, count, test.element, test.mutable, test.count)
			}
			if containsSafeBorrow(test.typ) != (test.count > 0) {
				t.Fatal("borrow presence disagrees with leaf count")
			}
		})
	}
}

func BenchmarkSharedSafeBorrowLeaves(b *testing.B) {
	for _, borrowed := range []bool{false, true} {
		for _, depth := range []int{0, 16, 20} {
			b.Run(fmt.Sprintf("borrowed=%t/depth=%d", borrowed, depth), func(b *testing.B) {
				leaf := hir.TypeI32
				want := 0
				if borrowed {
					leaf = hir.SharedBorrow(leaf)
					want = 1 << depth
				}
				typ := sharedBorrowResultType(depth, leaf)
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					_, _, count := safeBorrowLeaf(typ)
					if count != want {
						b.Fatalf("got %d borrow leaves, want %d", count, want)
					}
				}
			})
		}
	}
}
