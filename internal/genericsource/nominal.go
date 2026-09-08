package genericsource

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/diagnostic"
	"github.com/0disoft/fairyc/internal/genericplan"
)

func (x *expander) collectNominalTypes(builder *genericplan.Builder) error {
	if !x.allowNominal {
		return nil
	}
	// This private lane conservatively binds all local definitions and helper
	// bodies, including transitive field/cleanup dependencies, to the module AST.
	// It is not a minimal public-interface or cache identity algorithm.
	encoded, err := json.Marshal(x.module)
	if err != nil {
		return err
	}
	x.nominalTypes = map[string]genericplan.TypeID{}
	x.nominalNames = map[genericplan.TypeID]string{}
	for _, aggregate := range x.sourceAggregates() {
		shape := aggregate.shape()
		if len(shape.TypeParameters) != 0 {
			continue
		}
		id := genericplan.TypeID("source_nominal_v0::" + string(x.owner) + "::" + x.module.Path + "::" + shape.Name)
		if err := builder.AddType(genericplan.TypeDescriptor{ID: id, Owner: x.owner,
			Definition: genericplan.DigestText("source-nominal-module-v0", string(encoded)+"\n"+shape.Name)}); err != nil {
			return err
		}
		x.nominalTypes[shape.Name], x.nominalNames[id] = id, shape.Name
	}
	check := func(parameters []string) error {
		for _, parameter := range parameters {
			if _, found := x.nominalTypes[parameter]; found {
				return fmt.Errorf("type parameter %q shadows a local nominal type", parameter)
			}
		}
		return nil
	}
	for _, function := range x.module.Functions {
		if err := check(function.TypeParameters); err != nil {
			return err
		}
	}
	for _, aggregate := range x.sourceAggregates() {
		if err := check(aggregate.shape().TypeParameters); err != nil {
			return err
		}
	}
	return nil
}

func (x *expander) concreteType(name string) (genericplan.TypeID, bool) {
	if slices.Contains(scalarTypes, name) {
		return genericplan.TypeID(name), true
	}
	id, found := x.nominalTypes[name]
	return id, found
}

func (x *expander) boundType(id genericplan.TypeID, span diagnostic.Span) ast.TypeReference {
	name := string(id)
	if nominal, found := x.nominalNames[id]; found {
		name = nominal
	}
	return ast.NamedTypeReference(name, span)
}
