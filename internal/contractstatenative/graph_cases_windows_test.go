package contractstatenative

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatefixture"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
)

func dynamicGraphCases(t *testing.T) []contractstatefixture.HIRCase {
	t.Helper()
	var result []contractstatefixture.HIRCase
	ids := []string{contractstate.Bool.TypeID(), contractstate.U32.TypeID(), contractstate.U64.TypeID(), contractstate.Bytes.TypeID(), contractstate.Text.TypeID(), contractstate.Byte20.TypeID(), contractstate.Byte32.TypeID()}
	initial := [][]byte{{1}, {0, 255, 26, 13}, bytes.Repeat([]byte{255}, 8), {0, 255, 26, 13}, []byte("원래값"), bytes.Repeat([]byte{128}, 20), bytes.Repeat([]byte{204}, 32)}
	zero := [][]byte{{0}, make([]byte, 4), make([]byte, 8), nil, nil, make([]byte, 20), make([]byte, 32)}
	for index, id := range ids {
		for _, present := range []bool{true, false} {
			input, _ := nativeStateInput(t, contractstatemir.ExitCommit)
			input.StateLayouts[0].Fields = []hir.StateField{
				{Name: "key", Kind: hir.StateFieldCell, ValueTypeID: id},
				{Name: "source", Kind: hir.StateFieldCell, ValueTypeID: id},
				{Name: "target", Kind: hir.StateFieldCell, ValueTypeID: id},
				{Name: "items", Kind: hir.StateFieldMap, KeyTypeID: id, ValueTypeID: id},
			}
			value := hir.StateValue{TypeID: id, Encoded: initial[index]}
			entry := []hir.StateAccess{{Kind: hir.StateAccessCellWrite, Field: "key", Value: value, Cost: 1},
				{Kind: hir.StateAccessCellRead, Field: "key", Result: 1, FoundResult: 2, Cost: 1},
				{Kind: hir.StateAccessCellWrite, Field: "source", Value: value, Cost: 1},
				{Kind: hir.StateAccessCellRead, Field: "source", Result: 3, FoundResult: 4, Cost: 1},
				{Kind: hir.StateAccessMapWrite, Field: "items", KeyInput: 1, ValueInput: 3, Cost: 1},
				{Kind: hir.StateAccessCellWrite, Field: "source", Value: hir.StateValue{TypeID: id, Encoded: zero[index]}, Cost: 1},
				{Kind: hir.StateAccessMapRead, Field: "items", KeyInput: 1, Result: 5, FoundResult: 6, Cost: 1}}
			if !present {
				var reads []hir.StateAccess
				for _, op := range entry {
					if op.Kind == hir.StateAccessCellRead || op.Kind == hir.StateAccessMapRead {
						reads = append(reads, op)
					}
				}
				entry = reads
			}
			kind := []hir.StateTerminatorKind{hir.StateTerminatorCommit, hir.StateTerminatorReject, hir.StateTerminatorTrap}[index%3]
			trap := ""
			if kind == hir.StateTerminatorTrap {
				trap = "overflow"
			}
			input.StateTransactions[0] = hir.StateTransaction{Package: 1, Function: 1, Entry: 0, Blocks: []hir.StateBlock{
				{ID: 0, Accesses: entry, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorBranch, Condition: 6, TrueTarget: 1, TrueArguments: []hir.StateValueID{5, 1}, FalseTarget: 2, FalseArguments: []hir.StateValueID{3, 1}, Cost: 1}},
				{ID: 1, Parameters: []hir.StateBlockParameter{{ID: 10, TypeID: id}, {ID: 11, TypeID: id}}, Accesses: []hir.StateAccess{
					{Kind: hir.StateAccessMapDelete, Field: "items", KeyInput: 11, Cost: 1},
					{Kind: hir.StateAccessCellWrite, Field: "target", ValueInput: 10, Cost: 1},
				}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 3, Arguments: []hir.StateValueID{10}, Cost: 1}},
				{ID: 2, Parameters: []hir.StateBlockParameter{{ID: 12, TypeID: id}, {ID: 13, TypeID: id}}, Accesses: []hir.StateAccess{
					{Kind: hir.StateAccessCellWrite, Field: "target", ValueInput: 12, Cost: 1},
				}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 3, Arguments: []hir.StateValueID{12}, Cost: 1}},
				{ID: 3, Parameters: []hir.StateBlockParameter{{ID: 14, TypeID: id}}, Accesses: []hir.StateAccess{
					{Kind: hir.StateAccessMapWrite, Field: "items", Key: hir.StateValue{TypeID: id, Encoded: zero[index]}, ValueInput: 14, Cost: 1},
				}, Terminator: hir.StateTerminator{Kind: kind, TrapCategory: trap, Cost: 1}},
			}}
			result = append(result, contractstatefixture.HIRCase{Name: fmt.Sprintf("dynamic-codec-%d-present-%t", index+1, present), Input: input, Limits: overlayPersistentLimits()})
		}
	}
	return result
}
