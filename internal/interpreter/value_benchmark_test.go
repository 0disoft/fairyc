package interpreter

import (
	"fmt"
	"testing"
	"unsafe"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

var benchmarkValue runtimeValue

func BenchmarkScalarValueCopy(b *testing.B) {
	value := i32Value(42)
	b.ReportAllocs()
	b.ReportMetric(float64(unsafe.Sizeof(value)), "value-bytes")
	for range b.N {
		benchmarkValue = value.cloneForValueCopy()
	}
}

func BenchmarkStructValueCopy(b *testing.B) {
	definition := &mir.StructType{Type: hir.ParseType("bench::Record")}
	values := make([]runtimeValue, 8)
	for index := range values {
		definition.Fields = append(definition.Fields, mir.StructField{Name: fmt.Sprintf("field%d", index), Type: hir.TypeI32})
		values[index] = i32Value(int32(index))
	}
	value, err := structValue(definition, values)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		benchmarkValue = value.cloneForValueCopy()
	}
}
