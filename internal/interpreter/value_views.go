package interpreter

import (
	"github.com/0disoft/fairyc/internal/hir"
)

func staticByteView(data string, typ hir.Type) runtimeValue {
	return runtimeValue{kind: valueView, typ: typ, payload: &indexedViewState{staticData: data, length: uint64(len(data))}}
}

func (v runtimeValue) byteAt(index uint64) (byte, error) {
	if v.kind != valueView || v.view() == nil || index >= v.view().length {
		return 0, capabilityError("byte-view read received an invalid runtime value")
	}
	absolute := v.view().offset + index
	if v.view().array != nil {
		if absolute >= uint64(len(v.view().array.elements)) {
			return 0, capabilityError("byte-view read exceeds its array backing")
		}
		element := v.view().array.elements[absolute]
		if element.kind != valueU8 || element.typ != hir.TypeU8 || element.u64() > 255 {
			return 0, capabilityError("byte-view array backing contains a non-u8 value")
		}
		return byte(element.u64()), nil
	}
	if absolute >= uint64(len(v.view().staticData)) {
		return 0, capabilityError("byte-view read exceeds its static backing")
	}
	return v.view().staticData[absolute], nil
}

func (v runtimeValue) indexedAt(index uint64) (runtimeValue, error) {
	if hir.NonByteIndexedView(v.typ) {
		if v.kind != valueView || v.view() == nil || v.view().array == nil || index >= v.view().length ||
			v.view().offset+index >= uint64(len(v.view().array.elements)) {
			return runtimeValue{}, capabilityError("i64-view read exceeds its array backing")
		}
		value := v.view().array.elements[v.view().offset+index]
		element, _ := hir.IndexedViewElement(v.typ)
		if !indexedElementRuntimeMatches(value, element) {
			return runtimeValue{}, capabilityError("i64-view backing contains a non-i64 value")
		}
		return value.cloneForValueCopy(), nil
	}
	value, err := v.byteAt(index)
	return runtimeValue{kind: valueU8, scalar: uint64(value), typ: hir.TypeU8}, err
}

func (v runtimeValue) storeIndexed(index uint64, value runtimeValue) error {
	element, indexed := hir.IndexedViewElement(v.typ)
	if !indexed || v.kind != valueView || v.view() == nil || !v.view().mutable || !hir.MutableIndexedView(v.typ) ||
		v.view().array == nil || index >= v.view().length || !indexedElementRuntimeMatches(value, element) {
		return capabilityError("mutable byte-view store received an invalid runtime value")
	}
	absolute := v.view().offset + index
	if absolute >= uint64(len(v.view().array.elements)) {
		return capabilityError("mutable byte-view store exceeds its array backing")
	}
	v.view().array.elements[absolute] = value.cloneForValueCopy()
	return nil
}

func (v runtimeValue) sliceIndexedView(start, end uint64) (runtimeValue, error) {
	if v.kind != valueView || v.view() == nil || !isRuntimeIndexedViewType(v.typ) || start > end || end > v.view().length {
		return runtimeValue{}, capabilityError("byte-view slice received an invalid runtime value")
	}
	view := &indexedViewState{array: v.view().array, offset: v.view().offset + start, length: end - start, mutable: v.view().mutable}
	if v.view().array == nil {
		view.staticData = v.view().staticData
	}
	return runtimeValue{kind: valueView, typ: v.typ, payload: view}, nil
}
