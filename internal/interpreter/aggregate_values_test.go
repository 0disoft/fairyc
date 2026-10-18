package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAggregateFieldsRetainDeclarationOrder(t *testing.T) {
	definition := &mir.StructType{Type: hir.ParseType("test::Record"), Fields: []mir.StructField{{Name: "z", Type: hir.TypeI32}, {Name: "a", Type: hir.TypeI32}}}
	value, err := structValue(definition, []runtimeValue{i32Value(10), i32Value(20)})
	if err != nil {
		t.Fatal(err)
	}
	for name, expected := range map[string]int32{"z": 10, "a": 20} {
		field, err := value.structField(definition, name)
		if err != nil || field.i32 != expected {
			t.Fatalf("field %s: %+v %v", name, field, err)
		}
	}
	copy := value.cloneForValueCopy()
	copy.structure.fields[1] = i32Value(99)
	if value.structure.fields[1].i32 != 20 {
		t.Fatal("aggregate copy shares field storage")
	}
	enum := &mir.EnumType{Type: hir.ParseType("test::Message"), Variants: []mir.EnumVariant{{Name: "Pair", Fields: []mir.EnumField{{Name: "z", Type: hir.TypeI32}, {Name: "a", Type: hir.TypeI32}}}}}
	tag, err := enumValue(enum, "Pair", []string{"z", "a"}, []runtimeValue{i32Value(10), i32Value(20)})
	if err != nil {
		t.Fatal(err)
	}
	field, err := tag.enumPayload(enum, "Pair", "a")
	if err != nil || field.i32 != 20 {
		t.Fatalf("enum field order lost: %+v %v", field, err)
	}
}

func TestScalarFieldReferenceFollowsStoredOwner(t *testing.T) {
	definition := &mir.StructType{Type: hir.ParseType("test::Record"), Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32}}}
	owner, err := structValue(definition, []runtimeValue{i32Value(10)})
	if err != nil {
		t.Fatal(err)
	}
	reference := &scalarReference{fields: &owner.structure.fields, field: 0}
	old := owner.cloneForValueCopy()
	owner.structure.fields = copyValueFields([]runtimeValue{i32Value(42)})
	loaded, found := reference.load()
	if !found || loaded.i32 != 42 {
		t.Fatal("field reference retained obsolete field storage")
	}
	reference.store(i32Value(99))
	if owner.structure.fields[0].i32 != 99 || old.structure.fields[0].i32 != 10 {
		t.Fatal("field reference lost owner identity or mutated a detached copy")
	}
}

func TestI64TaggedValueCopiesDetachPayloadState(t *testing.T) {
	payload := i64Value(-9223372036854775808)
	value := runtimeValue{kind: valueResult, tagged: &taggedState{variant: "ok", payload: &payload}}
	copy := value.cloneForValueCopy()
	if copy.tagged == value.tagged || copy.tagged.payload == value.tagged.payload {
		t.Fatal("inert i64 payload shared linear tag state")
	}
	*copy.tagged.payload = i64Value(42)
	if value.tagged.payload.i64 != -9223372036854775808 {
		t.Fatal("i64 copy changed its source")
	}
}
