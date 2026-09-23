package genericsource

import (
	"cmp"
	"encoding/json"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// NominalBinding retains declaration ownership for the private semantic bridge.
type NominalBinding struct {
	ID     genericplan.TypeID
	Owner  genericplan.PackageID
	Module string
	Name   string
}

func bindNominalContext(modules []*expander) error {
	type entry struct {
		Owner   genericplan.PackageID
		Module  *ast.Module
		Imports map[string]genericplan.PackageID
	}
	entries := make([]entry, 0, len(modules))
	for _, module := range modules {
		imports := make(map[string]genericplan.PackageID, len(module.imports))
		for alias, provider := range module.imports {
			imports[alias] = provider.owner
		}
		entries = append(entries, entry{module.owner, module.module, imports})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if order := cmp.Compare(a.Owner, b.Owner); order != 0 {
			return order
		}
		return cmp.Compare(a.Module.Path, b.Module.Path)
	})
	encoded, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	// Conservative private identity: every definition and exact import provider
	// in this closed request participates. This is not a reusable public interface.
	context := genericplan.DigestChunks("source-nominal-closed-graph-v0", encoded).String()
	for _, module := range modules {
		module.nominalContext = context
	}
	return nil
}

func bindNominalRegistry(modules []*expander) {
	registry := map[genericplan.TypeID]NominalBinding{}
	for _, module := range modules {
		if !module.packageNominal {
			continue
		}
		for id, name := range module.nominalNames {
			registry[id] = NominalBinding{ID: id, Owner: module.owner, Module: module.module.Path, Name: name}
		}
	}
	for _, module := range modules {
		if module.packageNominal {
			module.boundNominals = registry
		}
	}
}
