package cbackend

// TrapCode returns the fixed Stage 0 runtime code for one canonical trap category.
func TrapCode(category string) (uint32, bool) {
	_, code, ok := cTrapSpec(category)
	return code, ok
}

func cTrapSpec(category string) (string, uint32, bool) {
	switch category {
	case "bounds":
		return "FY_TRAP_BOUNDS", 1, true
	case "integer-overflow":
		return "FY_TRAP_INTEGER_OVERFLOW", 2, true
	case "division-by-zero":
		return "FY_TRAP_DIVISION_BY_ZERO", 3, true
	case "signed-division-overflow":
		return "FY_TRAP_SIGNED_DIVISION_OVERFLOW", 4, true
	case "invalid-shift":
		return "FY_TRAP_INVALID_SHIFT", 5, true
	case "assertion-failed":
		return "FY_TRAP_ASSERTION_FAILED", 6, true
	case "unreachable":
		return "FY_TRAP_UNREACHABLE", 7, true
	default:
		return "", 0, false
	}
}
