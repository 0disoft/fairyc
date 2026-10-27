package hir

// ContainsTaggedType reports whether match accepts the root or a payload reached
// through Option/Result. It does not follow borrows, pointers, arrays or nominal
// fields. Each identity is tested once; match must stay stable during the search.
// Completed visits belong to this call so mutable caller registries stay fresh.
func ContainsTaggedType(value Type, match func(Type) bool) bool {
	inline := [16]Type{value}
	pending := inline[:1]
	seen := make(map[Type]struct{})
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if _, duplicate := seen[current]; duplicate {
			continue
		}
		seen[current] = struct{}{}
		if match(current) {
			return true
		}
		if element, option := OptionElement(current); option {
			pending = append(pending, element)
		} else if success, failure, result := ResultElements(current); result {
			pending = append(pending, failure, success)
		}
	}
	return false
}
