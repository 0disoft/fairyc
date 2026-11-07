package genericsource

import (
	"encoding/json"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
)

func validStaticGroupLabel(label string) bool {
	if len(label) == 0 || len(label) > 64 {
		return false
	}
	for i := 0; i < len(label); i++ {
		c := label[i]
		if c != '_' && !(c >= 'A' && c <= 'Z') && !(c >= 'a' && c <= 'z') && !(i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

// Structural and duplicate-tuple validation has already completed. Keep source
// order for diagnostics, and sort only detached names for the identity closure.
func (x *expander) defineStaticGroups() error {
	groups := map[string][]string{}
	var order []string
	for _, fn := range x.module.Functions {
		if op := x.operations[fn.Name]; op != nil && op.group != "" {
			if len(groups[op.group]) == 0 {
				order = append(order, op.group)
			}
			groups[op.group] = append(groups[op.group], fn.Name)
		}
	}
	for _, label := range order {
		members := groups[label]
		if len(members) < 2 || len(members) > 16 {
			return x.at(x.templates[members[0]].Span, FailureDeclaration, fmt.Errorf("@interface(%s) requires 2..16 operations", label))
		}
	}
	// Walking providers globally also selects the earliest incomplete tuple when
	// groups and scalar types are interleaved in the written module.
	for _, fn := range x.module.Functions {
		a, supplied, _ := marker(fn, "implements")
		if !supplied {
			continue
		}
		op := x.operations[a.Arguments[0].Raw]
		if op.group == "" {
			continue
		}
		typ := a.Arguments[1].Raw
		for _, name := range groups[op.group] {
			if _, exists := x.operations[name].providers[typ]; !exists {
				return x.at(fn.Span, FailureDeclaration, fmt.Errorf("@interface(%s) requires a complete provider bundle for %s", op.group, typ))
			}
		}
	}
	for _, label := range order {
		members := groups[label]
		slices.Sort(members)
		var definitions []ast.Function
		for _, name := range members {
			definitions = append(definitions, x.templates[name])
			var types []string
			for typ := range x.operations[name].providers {
				types = append(types, typ)
			}
			slices.Sort(types)
			for _, typ := range types {
				definitions = append(definitions, x.operations[name].providers[typ])
			}
		}
		raw, err := json.Marshal(definitions)
		if err != nil {
			return err
		}
		definition := genericplan.DigestText("source-static-operation-group-v1", string(raw))
		for _, name := range members {
			x.operations[name].definition = definition
		}
	}
	return nil
}
