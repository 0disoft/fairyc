package interpreter

import (
	"math"
	"testing"
)

func TestCompactScalarPayloadPreservesEveryBit(t *testing.T) {
	for _, number := range []int32{math.MinInt32, -1, 0, math.MaxInt32} {
		value := i32Value(number)
		if value.i32() != number || value.cloneForValueCopy().i32() != number || value.payload != nil {
			t.Fatalf("i32 bits or inline storage changed: %d", number)
		}
	}
	for _, number := range []int64{math.MinInt64, -1, 0, math.MaxInt64} {
		value := i64Value(number)
		if value.i64() != number || value.cloneForValueCopy().i64() != number || value.payload != nil {
			t.Fatalf("i64 bits or inline storage changed: %d", number)
		}
	}
	value := runtimeValue{kind: valueUSize, scalar: math.MaxUint64}
	if value.u64() != math.MaxUint64 || value.cloneForValueCopy().u64() != math.MaxUint64 {
		t.Fatal("pointer-width unsigned bits changed")
	}
}
