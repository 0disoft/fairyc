package interpreter

import "github.com/0disoft/fairyc/internal/hir"

// Borrow the stored aggregate itself; ordinary indexedAt keeps copying values.
func (v runtimeValue) borrowStructElement(index uint64, typ hir.Type) (runtimeValue, error) {
	element, nominal := hir.StructViewElement(v.typ)
	referent, mutable, borrowed := hir.BorrowElement(typ)
	if !nominal || !borrowed || referent != element || mutable && (!hir.MutableIndexedView(v.typ) || v.view == nil || !v.view.mutable) || v.view == nil || v.view.array == nil || index >= v.view.length || v.view.offset+index >= uint64(len(v.view.array.elements)) {
		return runtimeValue{}, capabilityError("invalid indexed nominal reference")
	}
	value := v.view.array.elements[v.view.offset+index]
	if !indexedElementRuntimeMatches(value, element) {
		return runtimeValue{}, capabilityError("indexed nominal reference backing type mismatch")
	}
	value.typ = typ
	return value, nil
}
