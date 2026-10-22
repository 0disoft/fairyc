package hir

import "testing"

func TestFloatingTransportTypes(t *testing.T) {
	for _, typ := range []Type{TypeF32, TypeF64} {
		if ParseType(typ.String()) != typ || typ.Descriptor().Kind != TypeKindBuiltin || !IsFloatScalar(typ) {
			t.Fatalf("floating builtin identity lost: %s", typ)
		}
		if IsCScalar(typ) || FixedArrayScalarElement(typ) || ScalarEqualityType(typ) || CheckedArithmeticType(typ) {
			t.Fatalf("transport type %s broadened another scalar operation family", typ)
		}
	}
	if TypeF32 == TypeCFloat || TypeF64 == TypeCDouble {
		t.Fatal("Fairy and C floating types were aliased")
	}
}
