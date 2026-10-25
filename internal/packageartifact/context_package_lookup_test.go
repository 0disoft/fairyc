package packageartifact

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestContextPackageLookupKeepsFreshAndAmbiguousMappings(t *testing.T) {
	descriptor, coordinates := contextPackageLookupFixture(64, 8)
	fresh := make(map[hir.PackageID]packagegraph.CoordinateID, len(coordinates)+1)
	for id, coordinate := range coordinates {
		fresh[id+900] = coordinate
	}
	// An ambiguous coordinate outside the used context was never an error.
	fresh[9999] = coordinates[101]
	program, err := RebindContextDescriptor(descriptor, fresh, nil, 50000)
	if err != nil {
		t.Fatal(err)
	}
	wantField := hir.RawPointer(hir.ParseType("$p1000::types::Type_0000"), false)
	for _, structure := range program.StructTypes {
		if structure.Span.Package != 1000 || structure.Fields[0].Type != wantField {
			t.Fatalf("type or nested pointer resolved outside its fresh owner: %+v", structure)
		}
	}
	fresh[9998] = coordinates[100]
	if _, err := RebindContextDescriptor(descriptor, fresh, nil, 50000); err == nil || !strings.Contains(err.Error(), "multiple local packages") {
		t.Fatalf("ambiguous used coordinate was not rejected: %v", err)
	}
	delete(fresh, 9998)
	delete(fresh, 1000)
	if _, err := RebindContextDescriptor(descriptor, fresh, nil, 50000); err == nil || !strings.Contains(err.Error(), "unknown PackageCoordinateId") {
		t.Fatalf("missing used coordinate was not rejected: %v", err)
	}
}

func BenchmarkRebindContextPackageLookup(b *testing.B) {
	for _, packages := range []int{1, 64, 1024} {
		b.Run(fmt.Sprint(packages), func(b *testing.B) {
			descriptor, coordinates := contextPackageLookupFixture(packages, 1024)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				program, err := RebindContextDescriptor(descriptor, coordinates, nil, 50000)
				if err != nil || len(program.StructTypes) != 1024 {
					b.Fatalf("rebind context across %d packages: %v", packages, err)
				}
			}
		})
	}
}

func contextPackageLookupFixture(packages, types int) (ContextDescriptor, map[hir.PackageID]packagegraph.CoordinateID) {
	coordinates := make(map[hir.PackageID]packagegraph.CoordinateID, packages)
	for i := range packages {
		ordinal := i + 1
		coordinates[hir.PackageID(100+i)] = packagegraph.CoordinateID{byte(ordinal), byte(ordinal >> 8)}
	}
	prefix := "pkg[" + coordinates[100].String() + "]::types::"
	descriptor := ContextDescriptor{Structs: make([]ContextStruct, types)}
	for i := range types {
		name := fmt.Sprintf("Type_%04d", i)
		descriptor.Structs[i] = ContextStruct{
			Type: prefix + name, Name: name,
			Fields: []ContextField{{Name: "next", Type: "ptr " + prefix + "Type_0000"}},
		}
	}
	return descriptor, coordinates
}
