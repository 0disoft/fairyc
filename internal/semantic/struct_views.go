package semantic

import "github.com/0disoft/fairyc/internal/hir"

func (a *analyzer) plainViewStruct(element hir.Type) bool {
	info := a.structTypes[element]
	if info == nil || len(info.orderedFields) == 0 {
		return false
	}
	for _, field := range info.orderedFields {
		if !hir.PlainViewField(field.typ) {
			return false
		}
	}
	return true
}

func (a *analyzer) validStructViewUse(value hir.Type) bool {
	if slice, mutable, borrowed := hir.BorrowElement(value); borrowed {
		if element, nominal := hir.StructSliceElement(slice); nominal {
			return !mutable && a.plainViewStruct(element)
		}
		return a.validStructViewUse(slice)
	}
	if element, ok := hir.OptionElement(value); ok {
		return a.validStructViewUse(element)
	}
	if success, failure, ok := hir.ResultElements(value); ok {
		return a.validStructViewUse(success) && a.validStructViewUse(failure)
	}
	if element, _, ok := hir.RawPointerElement(value); ok {
		return a.validStructViewUse(element)
	}
	_, unsized := hir.StructSliceElement(value)
	return !unsized
}
