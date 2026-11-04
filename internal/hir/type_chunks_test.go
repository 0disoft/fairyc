package hir

import (
	"strings"
	"testing"
)

func TestVisitStringChunksMatchesString(t *testing.T) {
	values := []Type{{}, TypeI32, ParseType("malformed<&"), ParseType("[i32;   0001]"), SharedBorrow(TypeI32), MutableBorrow(TypeI32), RawPointer(TypeU8, false), RawPointer(TypeI32, true), Slice(TypeI32), FixedArray(TypeI32, 123456), Option(TypeI32), Result(TypeI32, TypeU8), CCallback(TypeI32, []Type{TypeU8}), Option(Result(Slice(TypeI32), TypeU8))}
	for _, value := range values {
		var b strings.Builder
		if !value.VisitStringChunks(func(chunk string) bool { b.WriteString(chunk); return true }) {
			t.Fatal("unexpected stop")
		}
		if got, want := b.String(), value.String(); got != want {
			t.Fatalf("chunks %q, String %q", got, want)
		}
	}
}

func TestVisitStringChunksStops(t *testing.T) {
	calls := 0
	completed := Option(Result(TypeI32, TypeU8)).VisitStringChunks(func(string) bool { calls++; return false })
	if completed || calls != 1 {
		t.Fatalf("completed=%v calls=%d", completed, calls)
	}
}
