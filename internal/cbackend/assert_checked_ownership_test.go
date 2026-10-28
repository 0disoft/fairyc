package cbackend

import (
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAssertionStateConsumptionPreservesEarlierProofs(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		f := assertionCheckedChainFixture(8)
		for i := range 8 {
			value := int32(0)
			if i%2 == 0 {
				value = math.MaxInt32
			}
			f.Blocks[i].Operations[0].Operands[0].I32 = value
			f.Blocks[i].Terminator = mir.Terminator{Kind: mir.TerminatorSwitch,
				SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI32, Operator: "+",
				SwitchOperands: []mir.Operand{{Kind: mir.OperandLocal, Type: hir.TypeI32, Local: mir.LocalID(i + 1)},
					{Kind: mir.OperandI32, Type: hir.TypeI32, I32: 1}},
				Successors: []mir.BlockID{mir.BlockID(i + 1), 9}, UnwindSuccessor: mir.NoBlock}
		}
		remap := func(id mir.BlockID) mir.BlockID {
			if reverse {
				return 9 - id
			}
			return id
		}
		if reverse {
			blocks := make([]mir.Block, len(f.Blocks))
			for _, block := range f.Blocks {
				block.ID = remap(block.ID)
				for i, successor := range block.Terminator.Successors {
					block.Terminator.Successors[i] = remap(successor)
				}
				blocks[block.ID] = block
			}
			f.Entry, f.Blocks = remap(f.Entry), blocks
		}
		if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
			t.Fatal(failures[0])
		}
		proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
		if len(proof) != 4 {
			t.Fatalf("reverse=%t: expected four overflow proofs, got %v", reverse, proof)
		}
		for i := range 8 {
			target, found := proof[remap(mir.BlockID(i))]
			if found != (i%2 == 0) || found && target != remap(9) {
				t.Fatalf("reverse=%t: proof for check %d changed: %v", reverse, i, proof)
			}
		}
	}
}

func TestAssertionStateConsumptionKeepsLoopBoundary(t *testing.T) {
	f := assertionCheckedChainFixture(4)
	f.Blocks[3].Terminator.Successors[0] = 1 // A backedge makes block 1 a merge.
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
		t.Fatal(failures[0])
	}
	proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
	if len(proof) != 1 || proof[3] != 5 {
		t.Fatalf("loop lost its local overflow proof: %v", proof)
	}
}
