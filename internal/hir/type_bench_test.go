package hir

import "testing"

var benchmarkType Type
var benchmarkLength uint64

func BenchmarkTypeConstruction(b *testing.B) {
	benchmarkType = Result(Option(MutableBorrow(TypeI32Slice)), FixedArray(TypeBool, 32))
	b.ReportAllocs()
	for b.Loop() {
		benchmarkType = Result(Option(MutableBorrow(TypeI32Slice)), FixedArray(TypeBool, 32))
	}
}

func BenchmarkTypeDecomposition(b *testing.B) {
	value := Result(Option(MutableBorrow(FixedArray(TypeI32, 32))), TypeBool)
	b.ReportAllocs()
	for b.Loop() {
		success, _, _ := ResultElements(value)
		borrowed, _ := OptionElement(success)
		array, _, _ := BorrowElement(borrowed)
		benchmarkType, benchmarkLength, _ = FixedArrayParts(array)
	}
}
