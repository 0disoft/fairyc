package genericsource

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// One collector belongs to one declaration walk. Keep first-use order while
// bounding unique dependencies before the later catalog construction step.
type sourceDependencies struct {
	x    *expander
	span diagnostic.Span
	seen map[string]struct{}
}

func (d *sourceDependencies) add(declaration *genericplan.Declaration, dependency genericplan.Dependency) error {
	var key strings.Builder
	part := func(value string) {
		key.WriteString(strconv.Itoa(len(value)))
		key.WriteByte(':')
		key.WriteString(value)
	}
	part(string(dependency.Declaration))
	for _, argument := range dependency.Arguments {
		part(argument.Parameter)
		part(string(argument.Concrete))
	}
	identity := key.String()
	if _, exists := d.seen[identity]; exists {
		return nil
	}
	limit := genericplan.DefaultCatalogLimits().MaxDependencies
	if len(declaration.Dependencies) >= limit {
		return d.x.at(d.span, FailureDeclaration, &genericplan.Error{
			Code:    genericplan.CodeCatalogBudgetExceeded,
			Message: fmt.Sprintf("declaration %q dependency count exceeds %d", declaration.ID, limit),
		})
	}
	if d.seen == nil {
		d.seen = make(map[string]struct{})
	}
	d.seen[identity] = struct{}{}
	declaration.Dependencies = append(declaration.Dependencies, dependency)
	return nil
}
