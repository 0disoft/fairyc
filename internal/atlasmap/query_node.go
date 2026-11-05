package atlasmap

import "github.com/0disoft/fairyc/internal/semanticsnapshot"

// Query nodes use the snapshot representation reconstructed by map decoding:
// empty nested slices are nil. Source preparation may allocate empty slices;
// that backing-store detail must not change selected bytes or pack checksums.
// Normalize only detached query copies, never the snapshot or map transport.
func cloneQueryDeclaration(value semanticsnapshot.Declaration) semanticsnapshot.Declaration {
	value = cloneDeclarations([]semanticsnapshot.Declaration{value})[0]
	normalizeQueryType(&value.Type)
	if value.Owner != nil {
		normalizeQueryType(value.Owner)
	}
	if len(value.Parameters) == 0 {
		value.Parameters = nil
	}
	for i := range value.Parameters {
		normalizeQueryType(&value.Parameters[i].Type)
	}
	if len(value.Effects) == 0 {
		value.Effects = nil
	}
	normalizeQueryFields(&value.Fields)
	if len(value.Variants) == 0 {
		value.Variants = nil
	}
	for i := range value.Variants {
		normalizeQueryFields(&value.Variants[i].Fields)
	}
	if len(value.DropEffects) == 0 {
		value.DropEffects = nil
	}
	return value
}

func normalizeQueryFields(values *[]semanticsnapshot.Field) {
	if len(*values) == 0 {
		*values = nil
	}
	for i := range *values {
		normalizeQueryType(&(*values)[i].Type)
	}
}

func normalizeQueryType(value *semanticsnapshot.TypeRef) {
	if len(value.Arguments) == 0 {
		value.Arguments = nil
	}
	for i := range value.Arguments {
		normalizeQueryType(&value.Arguments[i])
	}
}
