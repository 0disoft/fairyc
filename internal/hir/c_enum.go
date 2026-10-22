package hir

// CEnumVariantCapacity bounds declaration-order discriminants by the minimum
// supported integer width, including either signedness of plain C char.
// Zero means the spelling is not an admitted C integer base.
func CEnumVariantCapacity(base string) uint64 {
	switch base {
	case "c.char", "c.schar":
		return 1 << 7
	case "c.uchar":
		return 1 << 8
	case "c.short":
		return 1 << 15
	case "c.ushort":
		return 1 << 16
	case "c.int", "c.long", "c.ssize_t":
		return 1 << 31
	case "c.uint", "c.ulong", "c.size_t":
		return 1 << 32
	case "c.longlong":
		return 1 << 63
	case "c.ulonglong":
		return ^uint64(0)
	}
	return 0
}
