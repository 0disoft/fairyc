package hir

// IndexedViewElement returns the element of an executable borrowed slice.
// The closed set preserves exclusive byte views and adds only shared i64 views.
func IndexedViewElement(value Type) (Type, bool) {
	switch value {
	case SharedBorrow(TypeByteSlice), MutableBorrow(TypeByteSlice):
		return TypeU8, true
	case SharedBorrow(TypeI64Slice):
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
