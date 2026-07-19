package hir

import (
	"slices"
	"testing"
)

func TestCScalarVocabularyIsClosedAndOrdered(t *testing.T) {
	t.Parallel()
	want := []Type{
		"c::bool", "c::char", "c::double", "c::float", "c::int", "c::long", "c::longlong",
		"c::schar", "c::short", "c::size_t", "c::ssize_t", "c::uchar", "c::uint", "c::ulong",
		"c::ulonglong", "c::ushort",
	}
	got := CScalarTypes()
	if !slices.Equal(got, want) {
		t.Fatalf("CScalarTypes() = %v, want %v", got, want)
	}
	for _, value := range got {
		if !IsCScalar(value) {
			t.Fatalf("IsCScalar(%q) = false", value)
		}
	}
	got[0] = TypeInvalid
	if CScalarTypes()[0] != TypeCBool || IsCScalar(Type("c::unknown")) {
		t.Fatal("C scalar vocabulary leaked mutation or accepted an unknown type")
	}
}

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

func TestOptionElementPreservesNestedPayload(t *testing.T) {
	t.Parallel()
	value := Option(Result(TypeI32, Option(TypeBool)))
	element, ok := OptionElement(value)
	if !ok || element != Result(TypeI32, Option(TypeBool)) {
		t.Fatalf("OptionElement(%q) = %q, %v", value, element, ok)
	}
}

func TestOptionElementRejectsMultiplePayloads(t *testing.T) {
	t.Parallel()
	if _, ok := OptionElement(Type("Option<i32, bool>")); ok {
		t.Fatal("OptionElement() accepted two payload types")
	}
}
