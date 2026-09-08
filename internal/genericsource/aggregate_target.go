package genericsource

import (
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/ast"
)

// Resolve only exact source aliases; generated declarations remain with their
// defining package and references retain the caller's alias.
func (x *expander) aggregateTarget(ref ast.TypeReference) (callTarget, bool, error) {
	parts := strings.Split(ref.Path, "::")
	owner, name, prefix := x, parts[0], ""
	if len(parts) == 2 {
		owner, name, prefix = x.imports[parts[0]], parts[1], parts[0]+"::"
	} else if len(parts) != 1 {
		return callTarget{}, false, nil
	}
	if owner == nil {
		return callTarget{}, false, nil
	}
	aggregate, found := owner.aggregates[name]
	if !found {
		return callTarget{}, false, nil
	}
	if owner != x && !aggregate.shape().Public {
		return callTarget{}, false, x.at(ref.Span, FailurePrivate, fmt.Errorf("generic aggregate %q is private", ref.Path))
	}
	return callTarget{module: owner, name: name, prefix: prefix}, true, nil
}
