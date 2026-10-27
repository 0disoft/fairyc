package compiler

import (
	"fmt"

	"github.com/0disoft/fairyc/internal/ast"
	"github.com/0disoft/fairyc/internal/genericsource"
)

type genericHeaderKey struct{ kind, name string }

type genericHeaderPosition struct {
	first, count int
	method       bool
}

// This index borrows one unchanged module for one provenance conversion. Keep
// counts and method flags so malformed same-name headers retain scan semantics.
type genericHeaderIndex struct {
	module    *ast.Module
	positions map[genericHeaderKey]genericHeaderPosition
}

func indexGenericHeaders(module *ast.Module) *genericHeaderIndex {
	index := &genericHeaderIndex{module: module, positions: make(map[genericHeaderKey]genericHeaderPosition,
		len(module.Functions)+len(module.Structs)+len(module.Resources))}
	add := func(kind, name string, position int, method bool) {
		key := genericHeaderKey{kind, name}
		entry := index.positions[key]
		if entry.count == 0 {
			entry.first = position
		}
		entry.count++
		entry.method = entry.method || method
		index.positions[key] = entry
	}
	for position, function := range module.Functions {
		add(genericsource.OriginFunction, function.Name, position, function.Owner != "")
	}
	for position, structure := range module.Structs {
		add(genericsource.OriginStruct, structure.Name, position, false)
	}
	for position, resource := range module.Resources {
		add(genericsource.OriginResource, resource.Name, position, false)
	}
	return index
}

func (index *genericHeaderIndex) original(origin genericsource.DeclarationOrigin) (genericHeader, error) {
	switch origin.Kind {
	case genericsource.OriginFunction, genericsource.OriginStruct, genericsource.OriginResource:
	default:
		return originalGenericHeader(index.module, origin)
	}
	entry := index.positions[genericHeaderKey{origin.Kind, origin.Name}]
	if entry.method {
		return genericHeader{}, fmt.Errorf("generic declaration cannot be a method")
	}
	if entry.count != 1 {
		return genericHeader{}, fmt.Errorf("generic declaration has no unique original header")
	}
	// Borrow only the matching declaration; the existing validator still checks
	// spans, visibility, binders, and constructs the complete written signature.
	var selected ast.Module
	switch origin.Kind {
	case genericsource.OriginFunction:
		selected.Functions = index.module.Functions[entry.first : entry.first+1]
	case genericsource.OriginStruct:
		selected.Structs = index.module.Structs[entry.first : entry.first+1]
	case genericsource.OriginResource:
		selected.Resources = index.module.Resources[entry.first : entry.first+1]
	}
	return originalGenericHeader(&selected, origin)
}
