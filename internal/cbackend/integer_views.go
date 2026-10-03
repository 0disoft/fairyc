package cbackend

import (
	"fmt"
	"github.com/0disoft/fairyc/internal/hir"
	"strings"
)

func integerViewCType(value hir.Type) (descriptor, pointer string, ok bool) {
	element, ok := hir.IndexedViewElement(value)
	if !ok {
		return "", "", false
	}
	names := map[hir.Type]string{hir.TypeU8: "byte", hir.TypeI64: "i64", hir.TypeI32: "i32", hir.TypeU32: "u32", hir.TypeUSize: "usize"}
	types := map[hir.Type]string{hir.TypeU8: "uint8_t", hir.TypeI64: "int64_t", hir.TypeI32: "int32_t", hir.TypeU32: "uint32_t", hir.TypeUSize: "uintptr_t"}
	if hir.MutableIndexedView(value) {
		return "fy_mut_" + names[element] + "_slice", types[element] + " *", true
	}
	return "fy_" + names[element] + "_slice", "const " + types[element] + " *", true
}

func (e *emitter) emitIntegerViewTypes(output *strings.Builder) {
	for _, slice := range []hir.Type{hir.TypeI32Slice, hir.TypeU32Slice, hir.TypeUsizeSlice} {
		for _, value := range []hir.Type{hir.SharedBorrow(slice), hir.MutableBorrow(slice)} {
			if !e.integerViews[value] {
				continue
			}
			descriptor, pointer, _ := integerViewCType(value)
			fmt.Fprintf(output, "typedef struct %s {\n    %sdata;\n    uintptr_t length;\n} %s;\n\n", descriptor, pointer, descriptor)
		}
	}
}
