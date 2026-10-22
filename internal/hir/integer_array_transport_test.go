package hir

import "testing"

func TestIntegerArrayTransportShapes(t *testing.T) {
	for _, scalar := range []Type{TypeI8, TypeI16, TypeU16, TypeU64, TypeISize} {
		for _, length := range []uint64{1, 2, 4096} {
			typ := FixedArray(scalar, length)
			if !IntegerTransportArray(typ) || !ReprCArrayField(typ) || ScalarArrayField(typ) || ExecutableFixedArrayShape(scalar, length) {
				t.Fatalf("transport array broadened element operations: %s", typ)
			}
		}
		for _, length := range []uint64{0, 4097} {
			if IntegerTransportArray(FixedArray(scalar, length)) || ReprCArrayField(FixedArray(scalar, length)) {
				t.Fatal("unbounded array admitted")
			}
		}
	}
}
