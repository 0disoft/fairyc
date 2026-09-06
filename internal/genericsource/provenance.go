package genericsource

import (
	"cmp"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// DeclarationOrigin is source-declared provenance, not proof that an unused
// template body has been type checked. It contains no source text.
type DeclarationOrigin struct {
	ID         genericplan.DeclarationID
	Owner      genericplan.PackageID
	Module     string
	Name       string
	Span       diagnostic.Span
	NameSpan   diagnostic.Span
	Parameters []string
	Public     bool
}

type SpecializationOrigin struct {
	Declaration    genericplan.DeclarationID
	Specialization genericplan.SpecializationID
	GeneratedName  string
	Bindings       []genericplan.Binding
}

func sourceOrigins(modules []*expander, plan genericplan.Plan) ([]DeclarationOrigin, []SpecializationOrigin, error) {
	var declarations []DeclarationOrigin
	known := map[genericplan.DeclarationID]bool{}
	for _, module := range modules {
		for name, function := range module.templates {
			id := module.id(name)
			known[id] = true
			declarations = append(declarations, DeclarationOrigin{ID: id, Owner: module.owner,
				Module: module.module.Path, Name: name, Span: function.Span, NameSpan: function.NameSpan,
				Parameters: slices.Clone(function.TypeParameters), Public: function.Public})
		}
	}
	slices.SortFunc(declarations, func(a, b DeclarationOrigin) int { return cmp.Compare(a.ID, b.ID) })
	var origins []SpecializationOrigin
	for _, specialization := range plan.Specializations {
		if !known[specialization.Declaration] {
			return nil, nil, fmt.Errorf("specialization has no source declaration")
		}
		origins = append(origins, SpecializationOrigin{Declaration: specialization.Declaration,
			Specialization: specialization.ID, GeneratedName: generatedPrefix + string(specialization.ID),
			Bindings: slices.Clone(specialization.Bindings)})
	}
	return declarations, origins, nil
}
