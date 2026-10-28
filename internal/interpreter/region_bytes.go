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
		for i := range backing.byteElements {
			value, err := input.byteAt(uint64(i))
			if err != nil {
				return runtimeValue{}, err
			}
			backing.byteElements[i] = value
		}
	}
	return runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.TypeByteSlice), payload: &indexedViewState{array: backing, length: length, mutable: true}}, nil
}
