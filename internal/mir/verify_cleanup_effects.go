package mir

import "github.com/0disoft/fairyc/internal/hir"

// resourceCleanupEffects includes implicit field destruction as well as the
// declared user cleanup. Definitions remain unchanged: each linked user body
// retains exactly its declaration's effect set.
func resourceCleanupEffects(value hir.Type, definitions map[hir.Type]*ResourceType) []hir.Effect {
	var result []hir.Effect
	seen := make(map[hir.Type]struct{})
	var visit func(hir.Type)
	visit = func(current hir.Type) {
		if _, duplicate := seen[current]; duplicate {
			return
		}
		seen[current] = struct{}{}
		if resource := definitions[current]; resource != nil {
			result = append(result, resource.DropEffects...)
			for _, field := range resource.Fields {
				visit(field.Type)
			}
			return
		}
		if element, option := hir.OptionElement(current); option {
			visit(element)
			return
		}
		if success, failure, tagged := hir.ResultElements(current); tagged {
			visit(success)
			visit(failure)
		}
	}
	visit(value)
	return hir.CanonicalEffects(result)
}
