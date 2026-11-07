package mir_test

import (
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestClosureValueMIRReservedIdentityAndPrivateFields(t *testing.T) {
	for _, tc := range []struct {
		name string
		edit func(*mir.Program)
	}{
		{"struct-shadow", func(p *mir.Program) {
			p.StructTypes = []mir.StructType{{Type: hir.TypeClosureI32, Name: "Shadow", Fields: []mir.StructField{{Name: "value", Type: hir.TypeI32}}}}
		}},
		{"resource-shadow", func(p *mir.Program) {
			p.ResourceTypes = []mir.ResourceType{{Type: hir.TypeClosureI32, DropKind: hir.ResourceDropEmpty}}
		}},
		{"enum-shadow", func(p *mir.Program) {
			p.EnumTypes = []mir.EnumType{{Type: hir.TypeClosureI32, Name: "Shadow", Variants: []mir.EnumVariant{{Name: "Only"}}}}
		}},
		{"private-field", func(p *mir.Program) {
			p.Functions[1].Blocks[0].Operations = append(p.Functions[1].Blocks[0].Operations, mir.Operation{Kind: mir.OperationFieldRead, DestinationLocal: 12, Type: hir.TypeI32, OwnerType: hir.TypeClosureI32, Field: "captures", Operands: []mir.Operand{{Kind: mir.OperandLocal, Local: 10}}})
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := closureMIRControl(0)
			tc.edit(&p)
			if !mir.HasCopiedClosureValues(p) {
				t.Fatal("feature format detection lost the closure identity")
			}
			for _, failure := range mir.Verify(p) {
				if failure.Code == mir.CodeOperationType {
					return
				}
			}
			t.Fatal("reserved closure identity or private representation was exposed")
		})
	}
	// A dormant declaration still selects the new feature format.
	for _, p := range []mir.Program{
		{StructTypes: []mir.StructType{{Type: hir.TypeClosureI32}}},
		{ResourceTypes: []mir.ResourceType{{Type: hir.TypeClosureI32}}},
		{EnumTypes: []mir.EnumType{{Type: hir.TypeClosureI32}}},
		{Functions: []mir.Function{{Resources: []mir.ResourceLocal{{Type: hir.TypeClosureI32}}}}},
	} {
		if !mir.HasCopiedClosureValues(p) {
			t.Fatal("dormant closure identity was accepted under a legacy format")
		}
	}
}
