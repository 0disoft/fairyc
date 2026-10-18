package interpreter

// A copied aggregate does not acquire drop ownership when stored in a resource.
// Check the runtime shape too: caller-supplied MIR must not silently leak an
// owned payload hidden inside a struct. The bound also rejects cyclic state.
func inertDropValue(value runtimeValue, depth int) bool {
	if depth >= maxValueNesting {
		return false
	}
	switch value.kind {
	case valueI64, valueI32, valueU8, valueU32, valueUSize, valueNull:
		return true
	case valueStruct:
		if value.structure == nil || value.structure.definition == nil {
			return false
		}
		if len(value.structure.fields) != len(value.structure.definition.Fields) {
			return false
		}
		for index := range value.structure.definition.Fields {
			v, found := indexedField(value.structure.fields, index)
			if !found || !inertDropValue(v, depth+1) {
				return false
			}
		}
		return true
	case valueEnum:
		if value.enum == nil || value.enum.definition == nil {
			return false
		}
		for _, v := range value.enum.fields {
			if !inertDropValue(v, depth+1) {
				return false
			}
		}
		return true
	case valueArray:
		if value.array == nil {
			return false
		}
		for _, v := range value.array.elements {
			if !inertDropValue(v, depth+1) {
				return false
			}
		}
		return true
	case valueOption, valueResult:
		return value.tagged != nil && (value.tagged.payload == nil || inertDropValue(*value.tagged.payload, depth+1))
	default:
		return false
	}
}
