package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func sharedResultViewType(depth int, leaf hir.Type) hir.Type {
	for range depth {
		leaf = hir.Result(leaf, leaf)
	}
	return leaf
}

func TestStructViewSharedResultGraph(t *testing.T) {
	function, properties := repeatedStructViewFixture(0, 1)
	view := function.ResultType
	// This compact graph represents more than four billion leaf paths. Validation
	// must visit shared types once while still rejecting either invalid branch.
	shared := sharedResultViewType(32, view)
	bareSlice := hir.Slice(hir.ParseType("main::Record"))
	for _, test := range []struct {
		name     string
		typ      hir.Type
		failures int
	}{
		{"shared", shared, 0},
		{"invalid-success", hir.Result(bareSlice, shared), 2},
		{"invalid-failure", hir.Result(shared, bareSlice), 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			function.ResultType = test.typ
			function.ParameterTypes = []hir.Type{test.typ}
			failures := verifyStructViewTypes(&function, properties)
			if len(failures) != test.failures {
				t.Fatalf("got %d failures, want %d", len(failures), test.failures)
			}
			for _, failure := range failures {
				if failure.Code != CodeOperationType || failure.Entity != function.Name {
					t.Fatalf("diagnostic identity changed: %+v", failure)
				}
			}
		})
	}
}

func TestStructViewSharedChildResultsAreFresh(t *testing.T) {
	function, properties := repeatedStructViewFixture(0, 1)
	function.ResultType = hir.Option(sharedResultViewType(16, function.ResultType))
	function.ParameterTypes = nil
	definition := properties.structs[hir.ParseType("main::Record")]
	for _, valid := range []bool{true, false, true} {
		definition.Fields[0].Type = hir.TypeI32
		if !valid {
			definition.Fields[0].Type = hir.TypeRegion
		}
		failures := verifyStructViewTypes(&function, properties)
		if valid && len(failures) != 0 || !valid && len(failures) != 1 {
			t.Fatalf("stale child result, valid=%t, failure count=%d", valid, len(failures))
		}
	}
}

func BenchmarkSharedResultViewVerification(b *testing.B) {
	for _, depth := range []int{8, 16, 20} {
		b.Run(fmt.Sprintf("depth=%d", depth), func(b *testing.B) {
			function, properties := repeatedStructViewFixture(0, 1)
			function.ResultType = sharedResultViewType(depth, function.ResultType)
			function.ParameterTypes = nil
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if failures := verifyStructViewTypes(&function, properties); len(failures) != 0 {
					b.Fatal(failures)
				}
			}
		})
	}
}
