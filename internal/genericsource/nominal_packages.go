package genericsource

import (
	"cmp"
	"encoding/json"
	"slices"

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
		Module  json.RawMessage
		Imports map[string]genericplan.PackageID
		path    string
	}
	entries := make([]entry, 0, len(modules))
	for _, module := range modules {
		encoded, err := json.Marshal(module.module)
		if err != nil {
			return err
		}
		module.nominalModuleJSON = encoded
		imports := make(map[string]genericplan.PackageID, len(module.imports))
		for alias, provider := range module.imports {
			imports[alias] = provider.owner
		}
		entries = append(entries, entry{Owner: module.owner, Module: json.RawMessage(encoded), Imports: imports, path: module.module.Path})
	}
	slices.SortFunc(entries, func(a, b entry) int {
		if order := cmp.Compare(a.Owner, b.Owner); order != 0 {
			return order
		}
		return cmp.Compare(a.path, b.path)
	})
	encoded, err := json.Marshal(entries)
	if err != nil {
		return err
	}
	// Conservative private identity: every definition and exact import provider
	// in this closed request participates. This is not a reusable public interface.
	context := genericplan.DigestText("source-nominal-closed-graph-v0", string(encoded)).String()
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
