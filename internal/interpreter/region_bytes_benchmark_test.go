package interpreter

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func BenchmarkRegionByteBacking(b *testing.B) {
	for _, length := range []int{4096, 65536} {
		for _, copyInput := range []bool{false, true} {
			b.Run(fmt.Sprintf("bytes=%d/copy=%v", length, copyInput), func(b *testing.B) {
				input := runtimeValue{kind: valueUSize, typ: hir.TypeUSize, scalar: uint64(length)}
				if copyInput {
					input = staticByteView(strings.Repeat("x", length), hir.SharedBorrow(hir.TypeByteSlice))
				}
				b.ReportAllocs()
				b.ResetTimer()
				for range b.N {
					r := executionState{steps: length}
					value, err := r.regionBytes(input, copyInput)
					if err != nil || value.view().length != uint64(length) || r.steps != 0 {
						b.Fatalf("Region byte allocation: %v", err)
					}
				}
			})
		}
	}
}
