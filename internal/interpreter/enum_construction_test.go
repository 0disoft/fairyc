package interpreter

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestEnumConstructionFieldSourcesAndValidation(t *testing.T) {
	for _, count := range []int{0, 1, 8, 9, 64} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			definition, names, values := enumConstructionFixture(count)
			original := slices.Clone(names)
			value, err := enumValue(definition, "Data", names, values)
			if err != nil {
				t.Fatal(err)
			}
			for i, field := range value.enum().fields {
				if field.i32() != int32(i) {
					t.Fatalf("declaration field %d received value %d", i, field.i32())
				}
			}
			if !slices.Equal(names, original) {
				t.Fatal("construction reordered caller initializer names")
			}
			if count < 2 {
				return
			}
			duplicate := slices.Clone(names)
			duplicate[0] = duplicate[len(duplicate)-1]
			if _, err := enumValue(definition, "Data", duplicate, values); err == nil || !strings.Contains(err.Error(), "duplicate field") {
				t.Fatalf("duplicate initializer: %v", err)
			}
			unknown := slices.Clone(names)
			unknown[0] = "unknown"
			if _, err := enumValue(definition, "Data", unknown, values); err == nil || !strings.Contains(err.Error(), "mismatched fields") {
				t.Fatalf("unknown initializer: %v", err)
			}
			if _, err := enumValue(definition, "Data", names, values[:len(values)-1]); err == nil {
				t.Fatal("accepted mismatched name/value counts")
			}
		})
	}
}

func TestIndexedEnumConstructionDetachesNestedValues(t *testing.T) {
	definition, names, values := enumConstructionFixture(9)
	innerDefinition := &mir.StructType{Type: hir.ParseType("bench::Inner"), Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32}}}
	inner, err := structValue(innerDefinition, []runtimeValue{i32Value(42)})
	if err != nil {
		t.Fatal(err)
	}
	definition.Variants[0].Fields[0].Type = innerDefinition.Type
	values[len(values)-1] = inner
	value, err := enumValue(definition, "Data", names, values)
	if err != nil {
		t.Fatal(err)
	}
	inner.structure().fields[0] = i32Value(99)
	if value.enum().fields[0].structure().fields[0].i32() != 42 {
		t.Fatal("indexed constructor shared nested owned storage with its source")
	}
}

func enumConstructionFixture(count int) (*mir.EnumType, []string, []runtimeValue) {
	definition := &mir.EnumType{Type: hir.ParseType("bench::Packet"), Variants: []mir.EnumVariant{{Name: "Data", Fields: make([]mir.EnumField, count)}}}
	names := make([]string, count)
	values := make([]runtimeValue, count)
	for i := range count {
		name := fmt.Sprintf("field_%04d", i)
		definition.Variants[0].Fields[i] = mir.EnumField{Name: name, Type: hir.TypeI32}
		// Reverse written order to exercise declaration-slot mapping.
		names[count-1-i] = name
		values[count-1-i] = i32Value(int32(i))
	}
	return definition, names, values
}

func BenchmarkEnumConstructionFields(b *testing.B) {
	for _, count := range []int{1, 8, 64, 512, 4096} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			definition, names, values := enumConstructionFixture(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				value, err := enumValue(definition, "Data", names, values)
				if err != nil {
					b.Fatal(err)
				}
				benchmarkValue = value
			}
		})
	}
}
