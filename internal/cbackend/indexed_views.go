package cbackend

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/hir"
)

func indexedViewCType(value hir.Type) (descriptor, pointer string, ok bool) {
	if _, structView := hir.StructViewElement(value); structView {
		if hir.MutableIndexedView(value) {
			return "fy_mut_struct_slice", "void *", true
		}
		return "fy_struct_slice", "const void *", true
	}
	element, ok := hir.IndexedViewElement(value)
	if !ok {
		return "", "", false
	}
	names := map[hir.Type]string{hir.TypeBool: "bool", hir.TypeU8: "byte", hir.TypeI64: "i64", hir.TypeI32: "i32", hir.TypeU32: "u32", hir.TypeUSize: "usize"}
	// Fairy bool uses the existing int32_t scalar/array representation, not C _Bool.
	types := map[hir.Type]string{hir.TypeBool: "int32_t", hir.TypeU8: "uint8_t", hir.TypeI64: "int64_t", hir.TypeI32: "int32_t", hir.TypeU32: "uint32_t", hir.TypeUSize: "uintptr_t"}
	if hir.MutableIndexedView(value) {
		return "fy_mut_" + names[element] + "_slice", types[element] + " *", true
	}
	return "fy_" + names[element] + "_slice", "const " + types[element] + " *", true
}

func (e *emitter) emitIntegerViewTypes(output *strings.Builder) {
	for _, slice := range []hir.Type{hir.TypeBoolSlice, hir.TypeI32Slice, hir.TypeU32Slice, hir.TypeUsizeSlice} {
		for _, value := range []hir.Type{hir.SharedBorrow(slice), hir.MutableBorrow(slice)} {
			if !e.analysis.integerViews[value] {
				continue
			}
			descriptor, pointer, _ := indexedViewCType(value)
			fmt.Fprintf(output, "typedef struct %s {\n    %sdata;\n    uintptr_t length;\n} %s;\n\n", descriptor, pointer, descriptor)
		}
	}
}
