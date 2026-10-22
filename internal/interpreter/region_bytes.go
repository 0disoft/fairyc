package interpreter

import "github.com/0disoft/fairyc/internal/hir"

func (r *executionState) regionBytes(input runtimeValue) (runtimeValue, error) {
	length := input.u64()
	// Charge before allocating so source-controlled lengths cannot exhaust host memory.
	if length > uint64(r.steps) {
		return runtimeValue{}, capabilityErrorKind(LimitStep, "Region bytes exceed interpreter step budget")
	}
	if err := r.checkCancellation(); err != nil {
		return runtimeValue{}, err
	}
	r.steps -= int(length)
	backing := &arrayState{elements: make([]runtimeValue, int(length))}
	for i := range backing.elements {
		backing.elements[i] = runtimeValue{kind: valueU8, typ: hir.TypeU8}
	}
	return runtimeValue{kind: valueView, typ: hir.MutableBorrow(hir.TypeByteSlice), payload: &indexedViewState{array: backing, length: length, mutable: true}}, nil
}
