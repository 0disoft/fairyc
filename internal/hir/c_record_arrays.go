package hir

// CRecordArray requires resolved, nonempty repr(C) definitions at every by-value
// record edge. Pointer targets and target-specific scalar availability remain
// owned by the existing C ABI checks. Array-of-array elements remain closed.
func CRecordArray(value Type, fields func(Type) ([]Type, bool), enumBase func(Type) string) bool {
	element, length, array := FixedArrayParts(value)
	if !array || length > MaxExecutableFixedArrayElements || fields == nil || element.Descriptor().Kind != TypeKindNominal {
		return false
	}
	type frame struct {
		typ      Type
		children []Type
		next     int
	}
	children, ok := fields(element)
	if !ok || len(children) == 0 {
		return false
	}
	stack := []frame{{typ: element, children: children}}
	state := map[Type]uint8{element: 1}
	for len(stack) != 0 {
		top := &stack[len(stack)-1]
		if top.next == len(top.children) {
			state[top.typ] = 2
			stack = stack[:len(stack)-1]
			continue
		}
		child := top.children[top.next]
		top.next++
		if _, _, pointer := RawPointerElement(child); pointer {
			continue
		}
		if ReprCArrayField(FixedArray(child, 1)) || enumBase != nil && CEnumVariantCapacity(enumBase(child)) != 0 {
			continue
		}
		if nested, n, fixed := FixedArrayParts(child); fixed {
			if n > MaxExecutableFixedArrayElements {
				return false
			}
			if ReprCArrayField(child) || CEnumArray(child, enumBase) {
				continue
			}
			child = nested
		}
		if child.Descriptor().Kind != TypeKindNominal || state[child] == 1 {
			return false
		}
		if state[child] == 2 {
			continue
		}
		nested, ok := fields(child)
		if !ok || len(nested) == 0 {
			return false
		}
		state[child] = 1
		stack = append(stack, frame{typ: child, children: nested})
	}
	return true
}
