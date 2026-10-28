package interpreter

import "github.com/0disoft/fairyc/internal/mir"

// Definitions stay immutable throughout one execution. Keep lazy indexes on
// that execution, shared by its calls but never by concurrent prepared entries.
type aggregateFieldIndex struct {
	structs   map[*mir.StructType]map[string]int
	resources map[*mir.ResourceType]map[string]int
}

const linearFieldLookupLimit = 16

func (index *aggregateFieldIndex) structField(definition *mir.StructType, name string) int {
	if index == nil || definition == nil || len(definition.Fields) <= linearFieldLookupLimit {
		return structFieldIndex(definition, name)
	}
	fields := index.structs[definition]
	if fields == nil {
		fields = make(map[string]int, len(definition.Fields))
		// Reverse insertion preserves the first declaration for duplicate names.
		for slot := len(definition.Fields) - 1; slot >= 0; slot-- {
			fields[definition.Fields[slot].Name] = slot
		}
		if index.structs == nil {
			index.structs = make(map[*mir.StructType]map[string]int)
		}
		index.structs[definition] = fields
	}
	if slot, found := fields[name]; found {
		return slot
	}
	return -1
}

func (index *aggregateFieldIndex) resourceField(definition *mir.ResourceType, name string) int {
	if index == nil || definition == nil || len(definition.Fields) <= linearFieldLookupLimit {
		return resourceFieldIndex(definition, name)
	}
	fields := index.resources[definition]
	if fields == nil {
		fields = make(map[string]int, len(definition.Fields))
		for slot := len(definition.Fields) - 1; slot >= 0; slot-- {
			fields[definition.Fields[slot].Name] = slot
		}
		if index.resources == nil {
			index.resources = make(map[*mir.ResourceType]map[string]int)
		}
		index.resources[definition] = fields
	}
	if slot, found := fields[name]; found {
		return slot
	}
	return -1
}
