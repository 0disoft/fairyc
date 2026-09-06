package genericsource

import (
	"cmp"
	"encoding/json"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericplan"
)

// PublicTemplate identifies an exported source declaration and all reachable
// generic definitions, including private helpers and uninstantiated dependencies.
type PublicTemplate struct {
	Owner   genericplan.PackageID
	Module  string
	Name    string
	Closure genericplan.Digest
}

// PackageExpansion owns detached metadata for one completed expansion. Template
// hashes do not contain source text and generated names are not source exports.
type PackageExpansion struct {
	Modules         []*ast.Module
	Plan            genericplan.Plan
	Templates       []PublicTemplate
	GeneratedNames  []string
	Specializations map[genericplan.PackageID][]genericplan.SpecializationID
}

func publicTemplates(modules []*expander, catalog *genericplan.Catalog) ([]PublicTemplate, error) {
	var result []PublicTemplate
	for _, module := range modules {
		for name, function := range module.templates {
			if !function.Public {
				continue
			}
			digest, err := templateClosure(catalog, module.id(name))
			if err != nil {
				return nil, err
			}
			result = append(result, PublicTemplate{Owner: module.owner, Module: module.module.Path, Name: name, Closure: digest})
		}
	}
	slices.SortFunc(result, func(a, b PublicTemplate) int {
		if order := cmp.Compare(a.Owner, b.Owner); order != 0 {
			return order
		}
		if order := cmp.Compare(a.Module, b.Module); order != 0 {
			return order
		}
		return cmp.Compare(a.Name, b.Name)
	})
	return result, nil
}

func templateClosure(catalog *genericplan.Catalog, root genericplan.DeclarationID) (genericplan.Digest, error) {
	seen := map[genericplan.DeclarationID]bool{}
	pending := []genericplan.DeclarationID{root}
	var declarations []genericplan.Declaration
	for len(pending) != 0 {
		id := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if seen[id] {
			continue
		}
		seen[id] = true
		declaration, exists := catalog.Declaration(id)
		if !exists {
			return genericplan.Digest{}, fmt.Errorf("missing public template dependency %q", id)
		}
		declarations = append(declarations, declaration)
		for _, dependency := range declaration.Dependencies {
			pending = append(pending, dependency.Declaration)
		}
	}
	// A visited set makes unused recursive declarations finite to describe.
	// Actual instantiation still goes through the planner's cycle rejection.
	slices.SortFunc(declarations, func(a, b genericplan.Declaration) int { return cmp.Compare(a.ID, b.ID) })
	encoded, err := json.Marshal(struct {
		Root         genericplan.DeclarationID
		Declarations []genericplan.Declaration
	}{root, declarations})
	if err != nil {
		return genericplan.Digest{}, err
	}
	return genericplan.DigestText("source-public-template-closure-v0", string(encoded)), nil
}
