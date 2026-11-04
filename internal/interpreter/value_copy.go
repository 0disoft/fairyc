package interpreter

import "github.com/0disoft/fairyc/internal/hir"

// Value copies detach inert storage. Borrowed values and resource ownership
// retain their referents; copying an aggregate must never manufacture ownership.
// Each visited value, detached container, and inert-proof node consumes one unit.
func (r *executionState) cloneForValueCopy(v runtimeValue) (runtimeValue, error) {
	return r.cloneWithInertProof(v, false, 0)
}

// knownInert is a traversal-local proof, never a cache on mutable runtime state.
// Once a tagged subtree passes inertDropValue, its descendants need no rescan.
func (r *executionState) cloneWithInertProof(v runtimeValue, knownInert bool, depth int) (runtimeValue, error) {
	if depth >= maxValueNesting {
		return runtimeValue{}, capabilityErrorKind(LimitCopy, "interpreter copy nesting limit exceeded")
	}
	if err := r.consumeCopyUnits(1); err != nil {
		return runtimeValue{}, err
	}
	if _, _, borrowed := hir.BorrowElement(v.typ); borrowed {
		return v, nil
	}
	copy := v
	switch v.kind {
	case valueArray:
		if v.array() != nil {
			if err := r.consumeCopyUnits(1); err != nil {
				return runtimeValue{}, err
			}
			elements, err := r.copyFieldsWithInertProof(v.array().elements, knownInert, depth+1)
			if err != nil {
				return runtimeValue{}, err
			}
			copy.payload = &arrayState{elements: elements}
		}
	case valueStruct:
		if v.structure() != nil {
			if err := r.consumeCopyUnits(1); err != nil {
				return runtimeValue{}, err
			}
			fields, err := r.copyFieldsWithInertProof(v.structure().fields, knownInert, depth+1)
			if err != nil {
				return runtimeValue{}, err
			}
			copy.payload = &structState{definition: v.structure().definition, fields: fields}
		}
	case valueEnum:
		if v.enum() != nil {
			if err := r.consumeCopyUnits(1); err != nil {
				return runtimeValue{}, err
			}
			fields, err := r.copyFieldsWithInertProof(v.enum().fields, knownInert, depth+1)
			if err != nil {
				return runtimeValue{}, err
			}
			copy.payload = &enumState{definition: v.enum().definition, variant: v.enum().variant, fields: fields, fieldSlots: v.enum().fieldSlots}
		}
	case valueOption, valueResult:
		// An owned resource payload uses the existing linear tagged state. Only
		// inert wrappers may acquire an independent copy of that state.
		inert := knownInert
		if v.tagged() != nil && !inert {
			var err error
			inert, err = r.inertDropValue(v, 0)
			if err != nil {
				return runtimeValue{}, err
			}
		}
		if v.tagged() != nil && inert {
			if err := r.consumeCopyUnits(1); err != nil {
				return runtimeValue{}, err
			}
			state := *v.tagged()
			if state.payload != nil {
				if err := r.consumeCopyUnits(1); err != nil {
					return runtimeValue{}, err
				}
				payload, err := r.cloneWithInertProof(*state.payload, true, depth+1)
				if err != nil {
					return runtimeValue{}, err
				}
				state.payload = &payload
			}
			copy.payload = &state
		}
	}
	return copy, nil
}

func (r *executionState) copyValueFields(fields []runtimeValue) ([]runtimeValue, error) {
	return r.copyFieldsWithInertProof(fields, false, 0)
}

func (r *executionState) newCopyValueSlice(length int) ([]runtimeValue, error) {
	if err := r.consumeCopyUnits(1 + length); err != nil {
		return nil, err
	}
	return make([]runtimeValue, length), nil
}

func (r *executionState) copyFieldsWithInertProof(fields []runtimeValue, knownInert bool, depth int) ([]runtimeValue, error) {
	copy, err := r.newCopyValueSlice(len(fields))
	if err != nil {
		return nil, err
	}
	for index, value := range fields {
		cloned, err := r.cloneWithInertProof(value, knownInert, depth)
		if err != nil {
			return nil, err
		}
		copy[index] = cloned
	}
	return copy, nil
}
