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

func TestFixedArrayPartsRoundTripsCanonicalScalarArray(t *testing.T) {
	t.Parallel()
	value := FixedArray(TypeU32, 3)
	element, length, ok := FixedArrayParts(value)
	if value != "[u32; 3]" || !ok || element != TypeU32 || length != 3 {
		t.Fatalf("FixedArray round trip = %q, %q, %d, %v", value, element, length, ok)
	}
	for _, invalid := range []Type{"[u32]", "[u32; 0]", "[u32; nope]", "[u32; 3; 4]"} {
		if _, _, accepted := FixedArrayParts(invalid); accepted {
			t.Fatalf("FixedArrayParts(%q) accepted invalid type", invalid)
		}
	}
}

func TestExecutableFixedArrayShapeEnforcesClosedElementAndLengthSet(t *testing.T) {
	t.Parallel()
	for _, test := range []struct {
		element Type
		length  uint64
		want    bool
	}{
		{element: TypeBool, length: 1, want: true},
		{element: TypeUSize, length: MaxExecutableFixedArrayElements, want: true},
		{element: TypeI32, length: 0, want: false},
		{element: TypeU8, length: MaxExecutableFixedArrayElements + 1, want: false},
		{element: Type("main::Resource"), length: 1, want: false},
	} {
		if got := ExecutableFixedArrayShape(test.element, test.length); got != test.want {
			t.Errorf("ExecutableFixedArrayShape(%q, %d) = %v, want %v", test.element, test.length, got, test.want)
		}
	}
}

func TestQualifiedSourcePathRoundTripsBuildLocalIdentity(t *testing.T) {
	t.Parallel()
	identity := QualifiedSourcePath(7, "src/util.fy")
	packageID, sourcePath := SourcePathParts(identity)
	if identity != "$p7/src/util.fy" || packageID != 7 || sourcePath != "src/util.fy" {
		t.Fatalf("source identity round trip = %q, %d, %q", identity, packageID, sourcePath)
	}
	for _, ordinary := range []string{"main.fy", "$p0/main.fy", "$p4294967296/main.fy", "$p7/"} {
		packageID, sourcePath = SourcePathParts(ordinary)
		if packageID != DefaultPackage || sourcePath != ordinary {
			t.Fatalf("SourcePathParts(%q) = %d, %q", ordinary, packageID, sourcePath)
		}
	}
}
