package hir

import "strings"

// IndexedViewElement returns the element of an executable borrowed slice.
// The closed set admits the executable integer elements and shared nominal shapes. Nominal elements require
// stage-owned scalar-field struct validation before execution.
func IndexedViewElement(value Type) (Type, bool) {
	switch value {
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
		return SharedStructViewElement(value)
	}
}

// ArrayViewBorrow matches the exact fixed-array element and borrow kind.
func ArrayViewBorrow(owner, view Type, operator string) bool {
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

// IntegerViewElement recognizes unsized integer slices without admitting ownership.
func IntegerViewElement(value Type) (Type, bool) {
	element, ok := IndexedViewElement(SharedBorrow(value))
	return element, ok && element != TypeBool && FixedArrayScalarElement(element)
}

// NonByteIndexedView selects array-backed integer views rather than static byte data.
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

func SharedStructViewElement(value Type) (Type, bool) {
	slice, mutable, borrowed := BorrowElement(value)
	if !borrowed || mutable {
		return TypeInvalid, false
	}
	return StructSliceElement(slice)
}

// PlainViewField admits only language scalar fields with no ownership or loans.
func PlainViewField(value Type) bool { return FixedArrayScalarElement(value) }
