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

// Integer references retain the backing cell, including offset subviews. Static
// byte data can yield a shared reference only; it never acquires write authority.
func (v runtimeValue) borrowViewElement(index uint64, typ hir.Type) (runtimeValue, error) {
	if _, nominal := hir.StructViewElement(v.typ); nominal {
		return v.borrowStructElement(index, typ)
	}
	element, indexed := hir.IndexedViewElement(v.typ)
	referent, mutable, borrowed := hir.BorrowElement(typ)
	if !indexed || !borrowed || referent != element || v.kind != valueView || v.view == nil || index >= v.view.length || mutable && (!hir.MutableIndexedView(v.typ) || !v.view.mutable || v.view.array == nil) {
		return runtimeValue{}, capabilityError("invalid indexed integer reference")
	}
	value, err := v.indexedAt(index)
	if err != nil || !scalarStoreValueMatches(value, element) {
		return runtimeValue{}, capabilityError("indexed integer reference backing type mismatch")
	}
	value.typ = typ
	if v.view.array != nil {
		value.reference = &scalarReference{array: v.view.array, index: v.view.offset + index}
	} else {
		value.reference = &scalarReference{static: v.view, index: v.view.offset + index}
	}
	return value, nil
}
