package hir

// IndexedViewElement returns the element of an executable borrowed slice.
// The closed set admits only byte and i64 element views.
func IndexedViewElement(value Type) (Type, bool) {
	switch value {
	case SharedBorrow(TypeByteSlice), MutableBorrow(TypeByteSlice):
		return TypeU8, true
	case SharedBorrow(TypeI64Slice), MutableBorrow(TypeI64Slice):
		return TypeI64, true
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
