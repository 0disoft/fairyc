package hir

import "testing"

func TestCScalarArrayTransportShapes(t *testing.T) {
	for _, scalar := range CScalarTypes() {
		for _, length := range []uint64{1, 3, MaxExecutableFixedArrayElements} {
			value := FixedArray(scalar, length)
			if !TransportOnlyArray(value) || !ReprCArrayField(value) || ScalarArrayField(value) || ExecutableFixedArrayShape(scalar, length) {
				t.Fatalf("incorrect transport/element boundary: %s", value)
			}
		}
		for _, length := range []uint64{0, MaxExecutableFixedArrayElements + 1} {
			if value := FixedArray(scalar, length); TransportOnlyArray(value) || ReprCArrayField(value) {
				t.Fatalf("invalid array length admitted: %s", value)
			}
		}
	}
	for _, scalar := range []Type{TypeBool, TypeVoid, TypeInvalid, RawPointer(TypeU8, true), ParseType("c::unknown")} {
		if value := FixedArray(scalar, 3); TransportOnlyArray(value) || ReprCArrayField(value) {
			t.Fatalf("non-C-scalar array admitted: %s", value)
		}
	}
}
