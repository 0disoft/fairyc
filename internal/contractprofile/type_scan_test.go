package contractprofile

import (
	"encoding/json"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAnalyzeMIRTypePolicy(t *testing.T) {
	t.Parallel()
	rawFloat := hir.RawPointer(hir.TypeF32, false)
	tests := []struct {
		name       string
		operations []mir.Operation
		terminator mir.Terminator
		codes      []string
	}{
		{name: "raw operation without type metadata", operations: []mir.Operation{{Kind: mir.OperationRawAddress}}, codes: []string{"CP1003"}},
		{name: "operation metadata", operations: []mir.Operation{{Kind: mir.OperationAssign, Type: rawFloat, OwnerType: hir.TypeCInt, IndexType: hir.TypeUSize}}, codes: []string{"CP1003", "CP1013", "CP1014", "CP1015"}},
		{name: "operation operands", operations: []mir.Operation{{Kind: mir.OperationAssign, Operands: []mir.Operand{{Type: rawFloat}, {Type: hir.TypeCInt}, {Type: hir.TypeUSize}}}}, codes: []string{"CP1003", "CP1013", "CP1014", "CP1015"}},
		{name: "return transport", terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: mir.Operand{Type: rawFloat}}, codes: []string{"CP1003", "CP1013"}},
		{name: "call arguments", terminator: mir.Terminator{Kind: mir.TerminatorReturn, Arguments: []mir.Operand{{Type: rawFloat}, {Type: hir.TypeCInt}, {Type: hir.TypeUSize}}}, codes: []string{"CP1003", "CP1013", "CP1014", "CP1015"}},
		{name: "switch type excludes raw transport", terminator: mir.Terminator{Kind: mir.TerminatorReturn, SwitchType: rawFloat}, codes: []string{"CP1013"}},
		{name: "switch operands only reject target dependence", terminator: mir.Terminator{Kind: mir.TerminatorReturn, SwitchOperands: []mir.Operand{{Type: rawFloat}, {Type: hir.TypeCInt}, {Type: hir.TypeUSize}}}, codes: []string{"CP1014", "CP1015"}},
		{name: "one violation per category", operations: []mir.Operation{{Kind: mir.OperationAssign, Type: hir.TypeCInt, OwnerType: hir.TypeCInt, IndexType: hir.TypeUSize, Operands: []mir.Operand{{Type: hir.TypeCInt}, {Type: hir.TypeUSize}}}}, codes: []string{"CP1014", "CP1015"}},
		{name: "region policy follows type policy", operations: []mir.Operation{{Kind: mir.OperationRegionPlace, Type: rawFloat}}, codes: []string{"CP1003", "CP1005", "CP1013"}},
		{name: "tokens within identifiers are not scalar types", operations: []mir.Operation{{Kind: mir.OperationAssign, Type: hir.ParseType("user::usize_box"), OwnerType: hir.ParseType("user::f32_box")}}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			terminator := test.terminator
			if terminator.Kind == "" {
				terminator.Kind = mir.TerminatorReturn
			}
			program := mir.Program{Functions: []mir.Function{{
				Name: "main::main", Symbol: 1, ResultType: hir.TypeI32, Entry: 0,
				Blocks: []mir.Block{{ID: 0, Operations: test.operations, Terminator: terminator}},
			}}}
			before, err := json.Marshal(program)
			if err != nil {
				t.Fatal(err)
			}
			report := Analyze(program, nil)
			var codes []string
			for _, violation := range report.Violations {
				codes = append(codes, violation.Code)
			}
			if !slices.Equal(codes, test.codes) {
				t.Fatalf("violations = %+v, want codes %v", report.Violations, test.codes)
			}
			after, err := json.Marshal(program)
			if err != nil {
				t.Fatal(err)
			}
			if string(before) != string(after) {
				t.Fatal("Analyze modified caller-owned MIR")
			}
		})
	}
}

func BenchmarkAnalyzeMIRTypeScan(b *testing.B) {
	for _, kind := range []string{"operations", "terminators"} {
		b.Run(kind, func(b *testing.B) {
			function := mir.Function{Name: "main::main", Symbol: 1, ResultType: hir.TypeI32, Entry: 0}
			if kind == "operations" {
				operations := make([]mir.Operation, 1024)
				for i := range operations {
					operations[i] = mir.Operation{Kind: mir.OperationBinary, Operator: "+", Type: hir.TypeI32, OwnerType: hir.TypeI32, Operands: []mir.Operand{{Type: hir.TypeI32}, {Type: hir.TypeI32}}}
				}
				function.Blocks = []mir.Block{{ID: 0, Operations: operations, Terminator: mir.Terminator{Kind: mir.TerminatorReturn}}}
			} else {
				function.Blocks = make([]mir.Block, 1024)
				for i := range function.Blocks {
					function.Blocks[i] = mir.Block{ID: mir.BlockID(i), Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchType: hir.TypeBool, SwitchOperands: []mir.Operand{{Type: hir.TypeBool}}, Successors: []mir.BlockID{mir.BlockID(i + 1), mir.BlockID(i + 1)}}}
				}
				function.Blocks[len(function.Blocks)-1].Terminator = mir.Terminator{Kind: mir.TerminatorReturn}
			}
			program := mir.Program{Functions: []mir.Function{function}}
			b.ReportAllocs()
			b.ResetTimer()
			for b.Loop() {
				if report := Analyze(program, nil); !report.Accepted {
					b.Fatal(report.Violations)
				}
			}
		})
	}
}
