package interpreter

// A copied aggregate does not acquire drop ownership when stored in a resource.
// Check the runtime shape too: caller-supplied MIR must not silently leak an
// owned payload hidden inside a struct. The bound also rejects cyclic state.
func (r *executionState) inertDropValue(value runtimeValue, depth int) (bool, error) {
	if err := r.consumeCopyUnits(1); err != nil {
		return false, err
	}
	if depth >= maxValueNesting {
		return false, nil
	}
	switch value.kind {
	case valueI64, valueI32, valueU8, valueU32, valueUSize, valueNull:
		return true, nil
	case valueStruct:
		if value.structure() == nil || value.structure().definition == nil {
			return false, nil
		}
		if len(value.structure().fields) != len(value.structure().definition.Fields) {
			return false, nil
		}
		for index := range value.structure().definition.Fields {
			v, found := indexedField(value.structure().fields, index)
			if !found {
				return false, nil
			}
			inert, err := r.inertDropValue(v, depth+1)
			if err != nil || !inert {
				return false, err
			}
		}
		return true, nil
	case valueEnum:
		if value.enum() == nil || value.enum().definition == nil {
			return false, nil
		}
		for _, v := range value.enum().fields {
			inert, err := r.inertDropValue(v, depth+1)
			if err != nil || !inert {
				return false, err
			}
		}
		return true, nil
	case valueArray:
		if value.array() == nil {
			return false, nil
		}
		for _, v := range value.array().elements {
			inert, err := r.inertDropValue(v, depth+1)
			if err != nil || !inert {
				return false, err
			}
		}
		return true, nil
	case valueOption, valueResult:
		if value.tagged() == nil {
			return false, nil
		}
		if value.tagged().payload == nil {
			return true, nil
		}
		return r.inertDropValue(*value.tagged().payload, depth+1)
	default:
		return false, nil
	}
}
