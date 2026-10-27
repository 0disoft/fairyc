package packageartifact

import "github.com/0disoft/fairyc/internal/hir"

// Walk parsed compound children without rendering and reparsing every subtree.
// Push failure before success to preserve callback order and duplicate visits.
func collectStableCompoundNominals(root hir.Type, collect func(string)) {
	pending := []hir.Type{root}
	for len(pending) != 0 {
		value := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		descriptor := value.Descriptor()
		switch descriptor.Kind {
		case hir.TypeKindOption:
			pending = append(pending, descriptor.Element)
		case hir.TypeKindResult:
			pending = append(pending, descriptor.Failure, descriptor.Element)
		default:
			if !builtinType(value) {
				// Stable pointer/array spellings differ from HIR spellings. Keep
				// their existing admission, including opaque depth-limit children.
				collectStableNominalTypes(value.String(), collect)
			}
		}
	}
}
