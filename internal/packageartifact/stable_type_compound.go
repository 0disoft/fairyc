package packageartifact

import "github.com/0disoft/fairyc/internal/hir"

// Rebuild parsed Option/Result children in postorder without rendering and
// reparsing each subtree. Push failure before success to retain error order.
func rebindStableCompound(root hir.Type, resolve packageResolver) (hir.Type, error) {
	type frame struct {
		value hir.Type
		ready bool
	}
	pending := []frame{{value: root}}
	var values []hir.Type
	for len(pending) != 0 {
		item := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		descriptor := item.value.Descriptor()
		switch descriptor.Kind {
		case hir.TypeKindOption:
			if item.ready {
				values[len(values)-1] = hir.Option(values[len(values)-1])
			} else {
				pending = append(pending, frame{item.value, true}, frame{value: descriptor.Element})
			}
		case hir.TypeKindResult:
			if item.ready {
				success, failure := values[len(values)-2], values[len(values)-1]
				values = values[:len(values)-1]
				values[len(values)-1] = hir.Result(success, failure)
			} else {
				pending = append(pending, frame{item.value, true}, frame{value: descriptor.Failure}, frame{value: descriptor.Element})
			}
		default:
			if builtinType(item.value) {
				values = append(values, item.value)
				continue
			}
			// Preserve stable pointer/array admission and nominal resolution. An
			// opaque child at the HIR depth limit is reparsed from this boundary.
			rebound, err := rebindValidatedStableType(item.value.String(), resolve)
			if err != nil {
				return hir.TypeInvalid, err
			}
			values = append(values, rebound)
		}
	}
	return values[0], nil
}
