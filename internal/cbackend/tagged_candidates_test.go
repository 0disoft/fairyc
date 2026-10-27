package cbackend

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/layout"
	"github.com/0disoft/fairyc/internal/mir"
)

// f64 is a valid HIR scalar but is not a supported tagged payload in this C
// transport. Interned Result nodes share their success and failure subgraphs.
func sharedUnsupportedTaggedType(depth int) hir.Type {
	value := hir.TypeF64
	for range depth {
		value = hir.Result(value, value)
	}
	return value
}

func taggedCandidateEngine(tb testing.TB) *layout.Engine {
	tb.Helper()
	spec, err := emissionTarget(Stage0Profile())
	if err != nil {
		tb.Fatal(err)
	}
	engine, err := layout.New(spec)
	if err != nil {
		tb.Fatal(err)
	}
	return engine
}

func TestTaggedCandidatesKeepSupportedSiblingOfRejectedGraph(t *testing.T) {
	supported := hir.Option(hir.TypeU8)
	unsupported := sharedUnsupportedTaggedType(12)
	functions := []mir.Function{{ParameterTypes: []hir.Type{
		unsupported, hir.Result(unsupported, supported), supported,
	}}}
	got, byType, err := prepareTaggedTypes(functions, nil, nil, nil, nil, nil, nil, nil, taggedCandidateEngine(t))
	if err != nil || len(got) != 1 || len(byType) != 1 || got[0].typ != supported {
		t.Fatalf("supported sibling: candidates=%d, index=%d, error=%v", len(got), len(byType), err)
	}
}

func TestTaggedCandidatesRejectCompactDeepGraph(t *testing.T) {
	// 33 distinct types represent more than four billion expanded leaf paths.
	// Do not format this type on failure: its textual spelling is also huge.
	functions := []mir.Function{{ResultType: sharedUnsupportedTaggedType(32)}}
	got, byType, err := prepareTaggedTypes(functions, nil, nil, nil, nil, nil, nil, nil, taggedCandidateEngine(t))
	if err != nil || len(got) != 0 || len(byType) != 0 {
		t.Fatalf("rejected graph: candidates=%d, index=%d, error=%v", len(got), len(byType), err)
	}
}

func TestTaggedCandidatesReconsiderChangedArraySupport(t *testing.T) {
	array := hir.FixedArray(hir.TypeU8, 4)
	functions := []mir.Function{{ResultType: hir.Option(array)}}
	engine := taggedCandidateEngine(t)
	for _, supported := range []bool{false, true, false} {
		var arrays map[hir.Type]int
		want := 0
		if supported {
			arrays = map[hir.Type]int{array: 0}
			want = 1
		}
		got, byType, err := prepareTaggedTypes(functions, nil, nil, nil, nil, nil, nil, arrays, engine)
		if err != nil || len(got) != want || len(byType) != want {
			t.Fatalf("array support=%v: candidates=%d, index=%d, error=%v", supported, len(got), len(byType), err)
		}
	}
}

func BenchmarkUnsupportedTaggedCandidates(b *testing.B) {
	for _, depth := range []int{12, 16} {
		b.Run(fmt.Sprintf("depth-%d", depth), func(b *testing.B) {
			functions := []mir.Function{{ResultType: sharedUnsupportedTaggedType(depth)}}
			engine := taggedCandidateEngine(b)
			b.ReportAllocs()
			for b.Loop() {
				got, byType, err := prepareTaggedTypes(functions, nil, nil, nil, nil, nil, nil, nil, engine)
				if err != nil || len(got) != 0 || len(byType) != 0 {
					b.Fatalf("unsupported graph: candidates=%d, index=%d, error=%v", len(got), len(byType), err)
				}
			}
		})
	}
}
