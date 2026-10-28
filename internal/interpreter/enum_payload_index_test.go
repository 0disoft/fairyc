package interpreter

import (
	"fmt"
	"slices"
	"testing"
)

func TestEnumPayloadSlotsPreserveFieldsCopiesAndDefinitions(t *testing.T) {
	for _, count := range []int{8, 9, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			definition, names, values := enumConstructionFixture(count)
			owner, err := enumValue(definition, "Data", names, values)
			if err != nil {
				t.Fatal(err)
			}
			copy := owner.cloneForValueCopy()
			for i, field := range definition.Variants[0].Fields {
				for _, value := range []runtimeValue{owner, copy} {
					payload, err := value.enumPayload(definition, "Data", field.Name)
					if err != nil || payload.i32() != int32(i) {
						t.Fatalf("field %s: %+v, %v", field.Name, payload, err)
					}
				}
			}
			copy.enum().fields[0] = i32Value(42)
			payload, err := owner.enumPayload(definition, "Data", definition.Variants[0].Fields[0].Name)
			if err != nil || payload.i32() != 0 {
				t.Fatal("copied enum mutated its source's owned payload")
			}
			// Equal nominal types do not make different definition pointers interchangeable
			// as cached layouts. Preserve the original scan for a caller-supplied definition.
			other := *definition
			other.Variants = slices.Clone(definition.Variants)
			other.Variants[0].Fields = slices.Clone(other.Variants[0].Fields)
			slices.Reverse(other.Variants[0].Fields)
			field := other.Variants[0].Fields[0].Name
			payload, err = owner.enumPayload(&other, "Data", field)
			if err != nil || payload.i32() != 0 {
				t.Fatalf("alternative definition reused the wrong slots: %+v, %v", payload, err)
			}
			if _, err := owner.enumPayload(definition, "Data", "unknown"); err == nil {
				t.Fatal("accepted an unknown field")
			}
			if _, err := owner.enumPayload(definition, "Wrong", field); err == nil {
				t.Fatal("accepted a field from an inactive variant")
			}
			if _, err := owner.enumPayload(nil, "Data", field); err == nil {
				t.Fatal("accepted a missing definition")
			}
			if _, err := i32Value(0).enumPayload(definition, "Data", field); err == nil {
				t.Fatal("accepted a non-enum receiver")
			}
			copy.enum().fields = nil
			if _, err := copy.enumPayload(definition, "Data", field); err == nil {
				t.Fatal("cached slot read beyond the runtime payload")
			}
		})
	}
}

func BenchmarkEnumPayloadFields(b *testing.B) {
	for _, count := range []int{8, 64, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			definition, names, values := enumConstructionFixture(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				// Include construction, so retaining an index cannot hide setup cost.
				owner, err := enumValue(definition, "Data", names, values)
				if err != nil {
					b.Fatal(err)
				}
				for _, field := range definition.Variants[0].Fields {
					value, err := owner.enumPayload(definition, "Data", field.Name)
					if err != nil {
						b.Fatal(err)
					}
					benchmarkValue = value
				}
			}
		})
	}
}
