package mir

import "github.com/0disoft/fairyc/internal/hir"

// Definitions are immutable during one verification. Build indexes only for
// queried wide declarations; small declarations keep allocation-free lookup.
// Separate declaration keys preserve struct/resource collision diagnostics.
type aggregateFieldIndex struct {
	structs   map[*StructType]map[string]hir.Type
	resources map[*ResourceType]map[string]hir.Type
}

const linearFieldLookupLimit = 16

func (index *aggregateFieldIndex) structField(definition *StructType, name string) (hir.Type, bool) {
	if len(definition.Fields) <= linearFieldLookupLimit {
		field, found := structField(definition.Fields, name)
		return field.Type, found
	}
	fields := index.structs[definition]
	if fields == nil {
		fields = make(map[string]hir.Type, len(definition.Fields))
		for _, field := range definition.Fields {
			// Match the first declaration even when malformed MIR repeats a name.
			if _, exists := fields[field.Name]; !exists {
				fields[field.Name] = field.Type
			}
		}
		if index.structs == nil {
			index.structs = make(map[*StructType]map[string]hir.Type)
		}
		index.structs[definition] = fields
	}
	typ, found := fields[name]
	return typ, found
}

func (index *aggregateFieldIndex) resourceField(definition *ResourceType, name string) (hir.Type, bool) {
	if len(definition.Fields) <= linearFieldLookupLimit {
		field, found := resourceField(definition.Fields, name)
		return field.Type, found
	}
	fields := index.resources[definition]
	if fields == nil {
		fields = make(map[string]hir.Type, len(definition.Fields))
		for _, field := range definition.Fields {
			if _, exists := fields[field.Name]; !exists {
				fields[field.Name] = field.Type
			}
		}
		if index.resources == nil {
			index.resources = make(map[*ResourceType]map[string]hir.Type)
		}
		index.resources[definition] = fields
	}
	typ, found := fields[name]
	return typ, found
}
