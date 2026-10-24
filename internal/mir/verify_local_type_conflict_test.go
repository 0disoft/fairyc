package mir

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func TestVerifyRejectsRegionReturnWithUnrelatedLocalTypeConflict(t *testing.T) {
	t.Parallel()
	for _, conflict := range []bool{false, true} {
		name := "without conflict"
		if conflict {
			name = "with unrelated conflict"
		}
		t.Run(name, func(t *testing.T) {
			function := Function{
				Symbol: 1, Name: "test::region_return", Entry: 0,
				Parameters: []LocalID{0, 1}, ParameterTypes: []hir.Type{hir.TypeRegion, hir.TypeI32},
				ResultType: hir.TypeI32,
				Blocks: []Block{{ID: 0, Terminator: Terminator{
					Kind: TerminatorReturn, UnwindSuccessor: NoBlock,
					ReturnValue: Operand{Kind: OperandLocal, Local: 0, Type: hir.TypeI32},
				}}},
			}
			if conflict {
				function.Blocks[0].Operations = []Operation{{
					Kind: OperationAssign, DestinationLocal: 1,
					Operands: []Operand{{Kind: OperandU32, Type: hir.TypeU32}},
				}}
			}
			failures := Verify(Program{Functions: []Function{function}})
			code := "V1403"
			if conflict {
				code = CodeOperationType
			}
			if len(failures) != 1 || failures[0].Code != code {
				t.Fatalf("Verify() = %+v, want %s", failures, code)
			}
			if conflict && !strings.Contains(failures[0].Detail, "local 1 has conflicting types") {
				t.Fatalf("Verify() lost the conflicting local: %+v", failures)
			}
		})
	}
}

func TestVerifyAcceptsConsistentLocalReassignmentWithRegionParameter(t *testing.T) {
	t.Parallel()
	function := Function{
		Symbol: 1, Name: "test::scalar_return", Entry: 0,
		Parameters: []LocalID{0, 1}, ParameterTypes: []hir.Type{hir.TypeRegion, hir.TypeI32},
		ResultType: hir.TypeI32,
		Blocks: []Block{{
			ID: 0,
			Operations: []Operation{{
				Kind: OperationAssign, DestinationLocal: 1,
				Operands: []Operand{{Kind: OperandI32, Type: hir.TypeI32, I32: 7}},
			}},
			Terminator: Terminator{
				Kind: TerminatorReturn, UnwindSuccessor: NoBlock,
				ReturnValue: Operand{Kind: OperandLocal, Local: 1, Type: hir.TypeI32},
			},
		}},
	}
	if failures := Verify(Program{Functions: []Function{function}}); len(failures) != 0 {
		t.Fatalf("Verify rejected a consistent scalar reassignment: %+v", failures)
	}
}
