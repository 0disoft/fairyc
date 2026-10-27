package semanticsnapshot

import (
	"fmt"
	"slices"
)

// LookupDeclarationDocumentation selects one frontend-attached documentation span
// for the exact declaration key and source span. Both tables are sorted during
// construction. Returned spans are values; no owned slices or type trees escape.
func (snapshot Snapshot) LookupDeclarationDocumentation(key DeclarationKey, declarationSpan Span) (Span, bool, error) {
	declarations := snapshot.data.declarations
	i, found := slices.BinarySearchFunc(declarations, key, func(value Declaration, key DeclarationKey) int {
		return compareDeclarationKey(value.Key, key)
	})
	if !found || (i+1 < len(declarations) && declarations[i+1].Key == key) {
		return Span{}, false, fmt.Errorf("snapshot does not contain exactly one query declaration")
	}
	if declarations[i].Span != declarationSpan {
		return Span{}, false, fmt.Errorf("snapshot declaration does not match the exact query subject")
	}
	documentation := snapshot.data.documentation
	i, found = slices.BinarySearchFunc(documentation, key, func(value DeclarationDocumentation, key DeclarationKey) int {
		return compareDeclarationKey(value.Declaration, key)
	})
	if !found {
		return Span{}, false, nil
	}
	if i+1 < len(documentation) && documentation[i+1].Declaration == key {
		return Span{}, false, fmt.Errorf("snapshot repeats the query declaration")
	}
	selected := documentation[i]
	if selected.DeclarationSpan != declarationSpan ||
		selected.Documentation.PackageInstance != declarationSpan.PackageInstance ||
		selected.Documentation.Source != declarationSpan.Source ||
		selected.Documentation.StartByte < 0 ||
		selected.Documentation.EndByte <= selected.Documentation.StartByte ||
		selected.Documentation.EndByte > declarationSpan.StartByte {
		return Span{}, false, fmt.Errorf("does not match the exact query declaration")
	}
	return selected.Documentation, true, nil
}
