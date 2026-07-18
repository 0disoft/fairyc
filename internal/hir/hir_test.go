package hir

import "testing"

func TestResultElementsPreservesNestedPayloads(t *testing.T) {
	t.Parallel()
	value := Result(Result(TypeI32, Type("main::InnerError")), Type("main::OuterError"))
	success, failure, ok := ResultElements(value)
	if !ok || success != Result(TypeI32, Type("main::InnerError")) || failure != Type("main::OuterError") {
		t.Fatalf("ResultElements(%q) = %q, %q, %v", value, success, failure, ok)
	}
}

func TestResultElementsRejectsNonResult(t *testing.T) {
	t.Parallel()
	if _, _, ok := ResultElements(TypeI32); ok {
		t.Fatal("ResultElements() accepted i32")
	}
}
