package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func copyTestStruct(t *testing.T) runtimeValue {
	t.Helper()
	v, err := structValue(&mir.StructType{Type: "main::State", Fields: []mir.StructField{{Name: "count", Type: hir.TypeI32}}}, []runtimeValue{i32Value(7)})
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func TestValueCopyDetachesNestedAggregateStorage(t *testing.T) {
	original := copyTestStruct(t)
	for name, wrap := range map[string]func(runtimeValue) runtimeValue{
		"struct": func(v runtimeValue) runtimeValue { return v },
		"array": func(v runtimeValue) runtimeValue {
			return runtimeValue{kind: valueArray, array: &arrayState{elements: []runtimeValue{v}}}
		},
		"enum": func(v runtimeValue) runtimeValue {
			return runtimeValue{kind: valueEnum, enum: &enumState{variant: "some", fields: map[string]runtimeValue{"value": v}}}
		},
		"option": func(v runtimeValue) runtimeValue {
			return runtimeValue{kind: valueOption, tagged: &taggedState{variant: "some", payload: &v}}
		},
		"result": func(v runtimeValue) runtimeValue {
			return runtimeValue{kind: valueResult, tagged: &taggedState{variant: "ok", payload: &v}}
		},
	} {
		t.Run(name, func(t *testing.T) {
			copy := wrap(original).cloneForValueCopy()
			child := copy
			switch copy.kind {
			case valueArray:
				child = copy.array.elements[0]
			case valueEnum:
				child = copy.enum.fields["value"]
			case valueOption, valueResult:
				child = *copy.tagged.payload
			}
			child.structure.fields["count"] = i32Value(42)
			if original.structure.fields["count"].i32 != 7 || child.structure.definition != original.structure.definition {
				t.Fatal("value copy changed original storage or declaration identity")
			}
		})
	}
}

func TestValueCopyPreservesBorrowAndResourceIdentity(t *testing.T) {
	value := copyTestStruct(t)
	for _, typ := range []hir.Type{hir.SharedBorrow(value.typ), hir.MutableBorrow(value.typ)} {
		borrowed := value
		borrowed.typ = typ
		copy := borrowed.cloneForValueCopy()
		if copy != borrowed {
			t.Fatal("copying a borrow detached its referent")
		}
	}
	resource := runtimeValue{kind: valueResource, resource: &resourceState{fields: map[string]runtimeValue{"value": value}}}
	owned := runtimeValue{kind: valueResult, tagged: &taggedState{variant: "ok", payload: &resource}}
	if owned.cloneForValueCopy().tagged != owned.tagged || resource.cloneForValueCopy().resource != resource.resource {
		t.Fatal("copy manufactured resource ownership")
	}
}
