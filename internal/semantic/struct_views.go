package semantic

import "github.com/0disoft/fairyc/internal/hir"

func (a *analyzer) plainViewStruct(element hir.Type) bool {
	return hir.PlainViewStruct(element, a.plainStructFields)
}

func (a *analyzer) inlineArrayField(value hir.Type) bool {
	return hir.InlineArrayField(value, a.plainStructFields)
}

func (a *analyzer) plainStructFields(value hir.Type) ([]hir.Type, bool) {
	info := a.structTypes[value]
	if info == nil {
		return nil, false
	}
	fields := make([]hir.Type, len(info.orderedFields))
	for i, field := range info.orderedFields {
		fields[i] = field.typ
	}
	return fields, true
}

func (a *analyzer) validStructViewUse(value hir.Type) bool {
	if slice, _, borrowed := hir.BorrowElement(value); borrowed {
		if element, nominal := hir.StructSliceElement(slice); nominal {
			return a.plainViewStruct(element)
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
