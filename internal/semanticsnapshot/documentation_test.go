package semanticsnapshot

import (
	"slices"
	"strings"
	"testing"
)

func TestDocumentationLookupUsesTheCompleteDeclarationKey(t *testing.T) {
	var data snapshotData
	// Deliberately construct keys out of order across every identity component.
	for _, instance := range []string{"b", "a"} {
		for _, module := range []string{"z", "a"} {
			for _, kind := range []DeclarationKind{DeclarationStruct, DeclarationFunction} {
				for _, name := range []string{"z", "a"} {
					key := DeclarationKey{PackageInstance: instance, Module: module, Kind: kind, Name: name}
					data.declarations = append(data.declarations, Declaration{Key: key})
				}
			}
		}
	}
	slices.SortFunc(data.declarations, func(a, b Declaration) int { return compareDeclarationKey(a.Key, b.Key) })
	for i := range data.declarations {
		declaration := &data.declarations[i]
		declaration.Span = Span{PackageInstance: declaration.Key.PackageInstance, Source: declaration.Key.Module + ".fy", StartByte: 100 + i*10, EndByte: 105 + i*10}
		if i%3 != 0 {
			span := declaration.Span
			span.StartByte, span.EndByte = span.StartByte-9, span.StartByte-1
			data.documentation = append(data.documentation, DeclarationDocumentation{
				Declaration: declaration.Key, DeclarationSpan: declaration.Span, Documentation: span,
			})
		}
	}
	snapshot := Snapshot{data: data}
	for i, declaration := range data.declarations {
		got, found, err := snapshot.LookupDeclarationDocumentation(declaration.Key, declaration.Span)
		if err != nil || found != (i%3 != 0) {
			t.Fatalf("lookup %+v: %+v %t %v", declaration.Key, got, found, err)
		}
		if found && (got.PackageInstance != declaration.Key.PackageInstance || got.EndByte != declaration.Span.StartByte-1) {
			t.Fatalf("lookup selected a different declaration: %+v", got)
		}
	}
	for _, key := range []DeclarationKey{
		{}, {PackageInstance: "zz"},
		{PackageInstance: "a", Module: "b", Kind: DeclarationFunction, Name: "a"},
		{PackageInstance: "a", Module: "a", Kind: DeclarationFunction, Name: "m"},
	} {
		if _, found, err := snapshot.LookupDeclarationDocumentation(key, Span{}); err == nil || found {
			t.Fatalf("missing declaration %+v was admitted: %t %v", key, found, err)
		}
	}
	if _, found, err := (Snapshot{}).LookupDeclarationDocumentation(DeclarationKey{}, Span{}); err == nil || found {
		t.Fatal("empty snapshot admitted a declaration")
	}
}

func TestDocumentationLookupRejectsMismatchedAndDuplicateFacts(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*snapshotData)
		match  string
	}{
		{"declaration span", func(d *snapshotData) { d.declarations[0].Span.EndByte++ }, "snapshot declaration does not match"},
		{"duplicate declaration", func(d *snapshotData) { d.declarations = append(d.declarations, d.declarations[0]) }, "exactly one query declaration"},
		{"duplicate documentation", func(d *snapshotData) { d.documentation = append(d.documentation, d.documentation[0]) }, "snapshot repeats"},
		{"attachment owner", func(d *snapshotData) { d.documentation[0].DeclarationSpan.EndByte++ }, "does not match the exact query declaration"},
		{"package", func(d *snapshotData) { d.documentation[0].Documentation.PackageInstance = "other" }, "does not match"},
		{"source", func(d *snapshotData) { d.documentation[0].Documentation.Source = "other.fy" }, "does not match"},
		{"negative start", func(d *snapshotData) { d.documentation[0].Documentation.StartByte = -1 }, "does not match"},
		{"empty span", func(d *snapshotData) { d.documentation[0].Documentation.EndByte = 0 }, "does not match"},
		{"overlap", func(d *snapshotData) { d.documentation[0].Documentation.EndByte = 11 }, "does not match"},
	} {
		t.Run(test.name, func(t *testing.T) {
			key := DeclarationKey{PackageInstance: "instance", Module: "main", Kind: DeclarationFunction, Name: "f"}
			span := Span{PackageInstance: key.PackageInstance, Source: "main.fy", StartByte: 10, EndByte: 20}
			doc := span
			doc.StartByte, doc.EndByte = 0, 9
			data := snapshotData{
				declarations:  []Declaration{{Key: key, Span: span}},
				documentation: []DeclarationDocumentation{{Declaration: key, DeclarationSpan: span, Documentation: doc}},
			}
			test.mutate(&data)
			got, found, err := (Snapshot{data: data}).LookupDeclarationDocumentation(key, span)
			if err == nil || !strings.Contains(err.Error(), test.match) || found || got != (Span{}) {
				t.Fatalf("invalid lookup = %+v, %t, %v", got, found, err)
			}
		})
	}
}
