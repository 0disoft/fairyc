package hir

import "testing"

func TestFloatingArrayTransportShapes(t *testing.T) {
	for _, scalar := range []Type{TypeF32, TypeF64} {
		for _, length := range []uint64{1, 2, 4096} {
			typ := FixedArray(scalar, length)
			if !TransportOnlyArray(typ) || !ReprCArrayField(typ) || ScalarArrayField(typ) || ExecutableFixedArrayShape(scalar, length) {
				t.Fatalf("transport array broadened element operations: %s", typ)
			}
		}
		for _, length := range []uint64{0, 4097} {
			if TransportOnlyArray(FixedArray(scalar, length)) || ReprCArrayField(FixedArray(scalar, length)) {
				t.Fatal("unbounded array admitted")
			}
		}
	}
}
