package hir

import "strings"

// IndexedViewElement returns the element of an executable borrowed slice.
// The closed set admits the executable integer elements and borrowed nominal shapes. Nominal elements require
// stage-owned scalar-leaf struct validation before execution.
func IndexedViewElement(value Type) (Type, bool) {
	switch value {
	case SharedBorrow(TypeBoolSlice), MutableBorrow(TypeBoolSlice):
		return TypeBool, true
	case SharedBorrow(TypeByteSlice), MutableBorrow(TypeByteSlice):
		return TypeU8, true
	case SharedBorrow(TypeI64Slice), MutableBorrow(TypeI64Slice):
		return TypeI64, true
	case SharedBorrow(TypeI32Slice), MutableBorrow(TypeI32Slice):
		return TypeI32, true
	case SharedBorrow(TypeU32Slice), MutableBorrow(TypeU32Slice):
		return TypeU32, true
	case SharedBorrow(TypeUsizeSlice), MutableBorrow(TypeUsizeSlice):
		return TypeUSize, true
	default:
		return StructViewElement(value)
	}
}

// ArrayViewBorrow matches the exact fixed-array element and borrow kind.
func ArrayViewBorrow(owner, view Type, operator string) bool {
	if array, ownerMutable, borrowed := BorrowElement(owner); borrowed {
		_, viewMutable, viewBorrowed := BorrowElement(view)
		element, length, fixed := FixedArrayParts(array)
		if !fixed || !ExecutableFixedArrayShape(element, length) || !viewBorrowed || ownerMutable != viewMutable {
			return false
		}
		owner = array
	}
	element, length, array := FixedArrayParts(owner)
	viewElement, indexed := IndexedViewElement(view)
	_, mutable, borrow := BorrowElement(view)
	return array && indexed && borrow && ExecutableFixedArrayShape(element, length) &&
		element == viewElement && (operator == "&" && !mutable || operator == "&mut" && mutable)
}

// MutableIndexedView reports executable exclusive element views.
func MutableIndexedView(value Type) bool {
	_, indexed := IndexedViewElement(value)
	_, mutable, borrow := BorrowElement(value)
	return indexed && borrow && mutable
}

// I64View includes both shared and exclusive i64 descriptors.
func I64View(value Type) bool {
	element, _, borrow := BorrowElement(value)
	return borrow && element == TypeI64Slice
}

// ScalarViewElement recognizes unsized bool/integer slices without admitting ownership.
func ScalarViewElement(value Type) (Type, bool) {
	element, ok := IndexedViewElement(SharedBorrow(value))
	return element, ok && FixedArrayScalarElement(element)
}

// NonByteIndexedView selects typed storage views rather than static byte data.
func NonByteIndexedView(value Type) bool {
	element, ok := IndexedViewElement(value)
	return ok && element != TypeU8
}

// StructSliceElement recognizes only canonical unsized nominal shapes. It is
// structural: semantic analysis and MIR must resolve the definition separately.
func StructSliceElement(value Type) (Type, bool) {
	text := string(value)
	if len(text) < 5 || text[0] != '[' || text[len(text)-1] != ']' {
		return TypeInvalid, false
	}
	element := text[1 : len(text)-1]
	if !strings.Contains(element, "::") || strings.ContainsAny(element, "[];&<> *\t\r\n") {
		return TypeInvalid, false
	}
	return Type(element), true
}

// StructViewElement recognizes nominal shared or exclusive descriptors.
// Each consumer must validate the actual plain struct definition separately.
func StructViewElement(value Type) (Type, bool) {
	slice, _, borrowed := BorrowElement(value)
	if !borrowed {
		return TypeInvalid, false
	}
	return StructSliceElement(slice)
}

func SharedStructViewElement(value Type) (Type, bool) {
	slice, mutable, borrowed := BorrowElement(value)
	if !borrowed || mutable {
		return TypeInvalid, false
	}
	return StructSliceElement(slice)
}

// PlainViewField admits scalar leaves and bounded inline scalar arrays without loans.
func PlainViewField(value Type) bool {
	return FixedArrayScalarElement(value) || ScalarArrayField(value)
}

// ScalarArrayField is the bounded inline, non-owning array field subset.
func ScalarArrayField(value Type) bool {
	element, length, array := FixedArrayParts(value)
	return array && FixedArrayScalarElement(element) && ExecutableFixedArrayShape(element, length)
}

// InlineArrayField also admits arrays of actual scalar-leaf structs. Array
// fields inside the element graph remain closed, keeping this subset acyclic
// and independent of array-field borrowing and iteration.
func InlineArrayField(value Type, fields func(Type) ([]Type, bool)) bool {
	if ScalarArrayField(value) {
		return true
	}
	element, length, array := FixedArrayParts(value)
	if !array || !ExecutableFixedArrayShape(element, length) {
		return false
	}
	return PlainViewStruct(element, func(typ Type) ([]Type, bool) {
		children, ok := fields(typ)
		if !ok {
			return nil, false
		}
		for _, child := range children {
			if _, _, array := FixedArrayParts(child); array {
				return nil, false
			}
		}
		return children, true
	})
}

// PlainViewStruct validates a nonempty, acyclic graph of structs with scalar
// leaves. The lookup admits only actual struct definitions. An explicit DFS
// stack and completed-node memoization bound work by the reachable type graph,
// including deep chains and repeated sibling types in untrusted MIR.
func PlainViewStruct(element Type, fields func(Type) ([]Type, bool)) bool {
	return plainStructGraph(element, fields, false)
}

// PlainStoredStruct uses the same acyclic graph for field replacement, but
// admits fieldless private values. View element admission remains nonempty.
func PlainStoredStruct(element Type, fields func(Type) ([]Type, bool)) bool {
	return plainStructGraph(element, fields, true)
}

func plainStructGraph(element Type, fields func(Type) ([]Type, bool), allowEmpty bool) bool {
	type frame struct {
		typ    Type
		fields []Type
		next   int
	}
	first, ok := fields(element)
	if !ok || !allowEmpty && len(first) == 0 {
		return false
	}
	stack := []frame{{typ: element, fields: first}}
	state := map[Type]uint8{element: 1}
	arrayFields := make(map[Type]bool)
	for len(stack) != 0 {
		top := &stack[len(stack)-1]
		if top.next == len(top.fields) {
			state[top.typ] = 2
			stack = stack[:len(stack)-1]
			continue
		}
		child := top.fields[top.next]
		top.next++
		if PlainViewField(child) {
			continue
		}
		if _, _, array := FixedArrayParts(child); array {
			admitted, checked := arrayFields[child]
			if !checked {
				admitted = InlineArrayField(child, fields)
				arrayFields[child] = admitted
			}
			if !admitted {
				return false
			}
			continue
		}
		switch state[child] {
		case 1:
			return false
		case 2:
			continue
		}
		nested, ok := fields(child)
		if !ok || !allowEmpty && len(nested) == 0 {
			return false
		}
		state[child] = 1
		stack = append(stack, frame{typ: child, fields: nested})
	}
	return true
}
