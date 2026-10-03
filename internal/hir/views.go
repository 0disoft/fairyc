package hir

// IndexedViewElement returns the element of an executable borrowed slice.
// The closed set admits the executable integer elements; bool and aggregates remain excluded.
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
		return TypeInvalid, false
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
	return IndexedViewElement(SharedBorrow(value))
}

// NonByteIndexedView selects array-backed integer views rather than static byte data.
func NonByteIndexedView(value Type) bool {
	element, ok := IndexedViewElement(value)
	return ok && element != TypeU8
}
