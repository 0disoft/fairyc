package interpreter

import "github.com/0disoft/fairyc/internal/hir"

func (r *executionState) regionBytes(input runtimeValue, copyInput bool) (runtimeValue, error) {
	length := input.u64()
	if copyInput {
		if input.view() == nil {
			return runtimeValue{}, capabilityError("Region copy requires a byte view")
		}
		length = input.view().length
	}
	// Charge before allocating so source-controlled lengths cannot exhaust host memory.
	if length > uint64(r.steps) {
		return runtimeValue{}, capabilityErrorKind(LimitStep, "Region bytes exceed interpreter step budget")
	}
	if err := r.checkCancellation(); err != nil {
		return runtimeValue{}, err
	}
	r.steps -= int(length)
	backing := &arrayState{byteElements: make([]byte, int(length))}
	if copyInput {
		if err := copyRegionByteView(backing.byteElements, input); err != nil {
			return runtimeValue{}, err
		}
	}
	return runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.TypeByteSlice), payload: &indexedViewState{array: backing, length: length, mutable: true}}, nil
}

// The caller owns destination and has already charged its full length. Empty
// copies read no source bytes. Packed/static backing needs one range check;
// typed array elements retain byteAt's per-element u8 validation.
func copyRegionByteView(destination []byte, input runtimeValue) error {
	if len(destination) == 0 {
		return nil
	}
	view := input.view()
	if input.kind != valueView || view == nil || uint64(len(destination)) > view.length {
		return capabilityError("byte-view read received an invalid runtime value")
	}
	if view.array == nil {
		if view.offset > uint64(len(view.staticData)) || uint64(len(destination)) > uint64(len(view.staticData))-view.offset {
			return capabilityError("byte-view read exceeds its static backing")
		}
		copy(destination, view.staticData[view.offset:view.offset+uint64(len(destination))])
		return nil
	}
	if backing := view.array.byteElements; backing != nil {
		if view.offset > uint64(len(backing)) || uint64(len(destination)) > uint64(len(backing))-view.offset {
			return capabilityError("byte-view read exceeds its array backing")
		}
		copy(destination, backing[view.offset:view.offset+uint64(len(destination))])
		return nil
	}
	for index := range destination {
		value, err := input.byteAt(uint64(index))
		if err != nil {
			return err
		}
		destination[index] = value
	}
	return nil
}
