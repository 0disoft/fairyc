package interpreter

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAggregateConstructionRetainsBorrowTypeAndIdentity(t *testing.T) {
	typ := hir.Type(hir.QualifiedModulePath(2, "owned") + "::Value")
	for name, construct := range map[string]func() (runtimeValue, error){
		"struct": func() (runtimeValue, error) {
			return structValue(&mir.StructType{Type: typ, Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32}}}, []runtimeValue{i32Value(42)})
		},
		"resource": func() (runtimeValue, error) {
			return resourceValue(&mir.ResourceType{Type: typ, DropKind: hir.ResourceDropEmpty,
				Fields: []mir.ResourceField{{Name: "value", Type: hir.TypeI32}}}, []runtimeValue{i32Value(42)})
		},
		"enum": func() (runtimeValue, error) {
			return enumValue(&mir.EnumType{Type: typ, Variants: []mir.EnumVariant{{Name: "value"}}}, "value", nil, nil)
		},
	} {
		t.Run(name, func(t *testing.T) {
			value, err := construct()
			if err != nil || value.typ != typ {
				t.Fatalf("constructed type: %q %v", value.typ, err)
			}
			locals := map[mir.LocalID]runtimeValue{0: value}
			for _, mutable := range []bool{false, true} {
				operator, borrowedType := "&", hir.SharedBorrow(typ)
				if mutable {
					operator, borrowedType = "&mut", hir.MutableBorrow(typ)
				}
				op := mir.Operation{Operator: operator, OwnerType: typ, Type: borrowedType,
					Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 0}}}
				borrowed, trap, err := evaluateUnary(locals, op)
				if err != nil || trap != "" || borrowed.typ != borrowedType || locals[0] != value ||
					borrowed.resource != value.resource || borrowed.structure != value.structure || borrowed.enum != value.enum {
					t.Fatalf("borrow changed owner/type/state: %v %q", err, trap)
				}
				wrong := hir.Type(hir.QualifiedModulePath(3, "owned") + "::Value")
				op.OwnerType, op.Type = wrong, hir.SharedBorrow(wrong)
				if mutable {
					op.Type = hir.MutableBorrow(wrong)
				}
				if _, _, err := evaluateUnary(locals, op); err == nil {
					t.Fatal("same-named different owner accepted")
				}
			}
		})
	}
}
