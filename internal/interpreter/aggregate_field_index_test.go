package interpreter

import (
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAggregateFieldSlotsMatchLinearLookup(t *testing.T) {
	for _, count := range []int{0, 13, 14, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			var index aggregateFieldIndex
			structure := aggregateFieldReadProgram(count, false).StructTypes[0]
			// Include malformed declarations: first duplicate wins, including empty names.
			structure.Fields = append(structure.Fields, mir.StructField{Name: "duplicate"},
				mir.StructField{Name: "duplicate"}, mir.StructField{Name: ""})
			resource := &mir.ResourceType{Type: structure.Type}
			for _, field := range structure.Fields {
				resource.Fields = append(resource.Fields, mir.ResourceField(field))
			}
			other := structure
			other.Fields = slices.Clone(structure.Fields)
			slices.Reverse(other.Fields)
			for _, name := range []string{"field_0000", "field_0016", "field_0063", "duplicate", "", "absent"} {
				for range 2 { // Exercise both building and reusing an index.
					if got, want := index.structField(&structure, name), structFieldIndex(&structure, name); got != want {
						t.Fatalf("struct %q: slot %d, want %d", name, got, want)
					}
					if got, want := index.resourceField(resource, name), resourceFieldIndex(resource, name); got != want {
						t.Fatalf("resource %q: slot %d, want %d", name, got, want)
					}
					if got, want := index.structField(&other, name), structFieldIndex(&other, name); got != want {
						t.Fatalf("alternative definition %q: slot %d, want %d", name, got, want)
					}
				}
			}
			if index.structField(nil, "absent") != -1 || index.resourceField(nil, "absent") != -1 {
				t.Fatal("missing definition acquired a slot")
			}
			if len(structure.Fields) <= linearFieldLookupLimit && (index.structs != nil || index.resources != nil) {
				t.Fatal("small declarations allocated an index")
			}
		})
	}
}

func TestIndexedAggregateReadsPreserveValuesAndErrors(t *testing.T) {
	structure := aggregateFieldReadProgram(64, false).StructTypes[0]
	resource := aggregateFieldReadProgram(64, true).ResourceTypes[0]
	values := make([]runtimeValue, 64)
	for i := range values {
		values[i] = i32Value(int32(i))
	}
	owner, err := structValue(&structure, values)
	if err != nil {
		t.Fatal(err)
	}
	token, err := resourceValue(&resource, values)
	if err != nil {
		t.Fatal(err)
	}
	var index aggregateFieldIndex
	for i, field := range structure.Fields {
		left, err := owner.structField(&structure, field.Name, &index)
		if err != nil || left.i32() != int32(i) {
			t.Fatalf("struct field %s: %+v, %v", field.Name, left, err)
		}
		right, err := token.field(&resource, field.Name, &index)
		if err != nil || right.i32() != int32(i) {
			t.Fatalf("resource field %s: %+v, %v", field.Name, right, err)
		}
	}
	wrong := structure
	wrong.Type = hir.ParseType("other::Record")
	for _, cache := range []*aggregateFieldIndex{nil, &index} {
		for _, read := range []func() (runtimeValue, error){
			func() (runtimeValue, error) { return owner.structField(&structure, "absent", cache) },
			func() (runtimeValue, error) { return owner.structField(&wrong, "field_0000", cache) },
			func() (runtimeValue, error) { return token.field(&resource, "absent", cache) },
			func() (runtimeValue, error) { return i32Value(0).structField(&structure, "field_0000", cache) },
			func() (runtimeValue, error) { return i32Value(0).field(&resource, "field_0000", cache) },
		} {
			if _, err := read(); err == nil {
				t.Fatal("invalid field read succeeded")
			}
		}
	}
	token.resource().dropped = true
	if _, err := token.field(&resource, "field_0000", &index); err == nil {
		t.Fatal("cached read accepted a dropped resource")
	}
	owner.structure().fields = nil
	if _, err := owner.structField(&structure, "field_0063", &index); err == nil {
		t.Fatal("cached read exceeded the runtime field storage")
	}
}
