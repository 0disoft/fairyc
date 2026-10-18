package interpreter

import "github.com/0disoft/fairyc/internal/hir"

// Value copies detach inert storage. Borrowed values and resource ownership
// retain their referents; copying an aggregate must never manufacture ownership.
func (v runtimeValue) cloneForValueCopy() runtimeValue {
	if _, _, borrowed := hir.BorrowElement(v.typ); borrowed {
		return v
	}
	copy := v
	switch v.kind {
	case valueArray:
		if v.array != nil {
			elements := make([]runtimeValue, len(v.array.elements))
			for index, element := range v.array.elements {
				elements[index] = element.cloneForValueCopy()
			}
			copy.array = &arrayState{elements: elements}
		}
	case valueStruct:
		if v.structure != nil {
			copy.structure = &structState{definition: v.structure.definition, fields: copyValueFields(v.structure.fields)}
		}
	case valueEnum:
		if v.enum != nil {
			copy.enum = &enumState{definition: v.enum.definition, variant: v.enum.variant, fields: copyValueFields(v.enum.fields)}
		}
	case valueOption, valueResult:
		// An owned resource payload uses the existing linear tagged state. Only
		// inert wrappers may acquire an independent copy of that state.
		if v.tagged != nil && inertDropValue(v, 0) {
			state := *v.tagged
			if state.payload != nil {
				payload := state.payload.cloneForValueCopy()
				state.payload = &payload
			}
			copy.tagged = &state
		}
	}
	return copy
}

func copyValueFields(fields []runtimeValue) []runtimeValue {
	copy := make([]runtimeValue, len(fields))
	for index, value := range fields {
		copy[index] = value.cloneForValueCopy()
	}
	return copy
}
