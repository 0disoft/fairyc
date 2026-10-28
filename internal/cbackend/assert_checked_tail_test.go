package cbackend

import (
	"fmt"
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func assertionEarlyCheckFixture(blocks int, knownFailure bool) mir.Function {
	f := assertionLinearFixture(blocks)
	f.Parameters = []mir.LocalID{0}
	f.ParameterTypes = []hir.Type{hir.TypeI32}
	operand := mir.Operand{Kind: mir.OperandLocal, Local: 0, Type: hir.TypeI32}
	if knownFailure {
		f.Blocks[0].Operations[0].Operands[0].I32 = math.MaxInt32
		operand.Local = 1
	}
	f.Blocks[0].Terminator = mir.Terminator{Kind: mir.TerminatorSwitch,
		SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI32, Operator: "+",
		SwitchOperands: []mir.Operand{operand, {Kind: mir.OperandI32, I32: 1, Type: hir.TypeI32}},
		Successors:     []mir.BlockID{1, mir.BlockID(blocks)}, UnwindSuccessor: mir.NoBlock}
	f.Blocks = append(f.Blocks, mir.Block{ID: mir.BlockID(blocks), Terminator: mir.Terminator{
		Kind: mir.TerminatorTrap, TrapCategories: []string{"integer-overflow"}, UnwindSuccessor: mir.NoBlock}})
	return f
}

func TestAssertionProofSkipsUnneededTail(t *testing.T) {
	for _, known := range []bool{false, true} {
		f := assertionEarlyCheckFixture(64, known)
		if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
			t.Fatal(failures[0])
		}
		proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
		if known && (len(proof) != 1 || proof[0] != 64) || !known && len(proof) != 0 {
			t.Fatalf("known=%t: unexpected proof %v", known, proof)
		}
		reachable := reachableBlocksForC(f, nil, proof)
		if reachable[1] == known || !reachable[64] {
			t.Fatalf("known=%t: reachability changed: %v", known, reachable)
		}
	}
}

func TestAssertionProofRetainsLaterNumberedPredecessors(t *testing.T) {
	f := assertionCheckedChainFixture(4)
	last := mir.BlockID(len(f.Blocks) - 1)
	blocks := make([]mir.Block, len(f.Blocks))
	for _, block := range f.Blocks {
		block.ID = last - block.ID
		for i, successor := range block.Terminator.Successors {
			block.Terminator.Successors[i] = last - successor
		}
		if block.Terminator.UnwindSuccessor != mir.NoBlock {
			block.Terminator.UnwindSuccessor = last - block.Terminator.UnwindSuccessor
		}
		blocks[block.ID] = block
	}
	f.Entry, f.Blocks = last-f.Entry, blocks
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
		t.Fatal(failures[0])
	}
	proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
	if target, found := proof[last-3]; !found || target != 0 || len(proof) != 1 {
		t.Fatalf("proof lost facts from higher block IDs: %v", proof)
	}
}

func BenchmarkAssertionProofEarlyCheck(b *testing.B) {
	for _, blocks := range []int{64, 256, 512} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			f := assertionEarlyCheckFixture(blocks, false)
			if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
				b.Fatal(failures[0])
			}
			b.ReportAllocs()
			for b.Loop() {
				if proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]; len(proof) != 0 {
					b.Fatal("unknown parameter acquired an overflow proof")
				}
			}
		})
	}
}
