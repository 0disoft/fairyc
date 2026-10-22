package hir

import "testing"

func TestIntegerTransportTypes(t *testing.T) {
	for _, typ := range []Type{TypeI8, TypeI16, TypeU16, TypeU64, TypeISize} {
		if ParseType(typ.String()) != typ || typ.Descriptor().Kind != TypeKindBuiltin || !IsIntegerTransportScalar(typ) {
			t.Fatalf("integer builtin identity lost: %s", typ)
		}
		if IsCScalar(typ) || FixedArrayScalarElement(typ) || ScalarEqualityType(typ) || CheckedArithmeticType(typ) {
			t.Fatalf("transport type %s broadened another scalar operation family", typ)
		}
	}
}
