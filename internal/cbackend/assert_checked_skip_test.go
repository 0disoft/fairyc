package cbackend

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestAssertionProofWithoutSwitchPreservesReachability(t *testing.T) {
	function := assertionLinearFixture(64)
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{function}}); len(failures) != 0 {
		t.Fatal(failures[0])
	}
	proof := assertedCheckedFailuresForC([]mir.Function{function})[function.Symbol]
	if len(proof) != 0 {
		t.Fatal("switch-free function acquired a checked-failure proof")
	}
	for block, reachable := range reachableBlocksForC(function, nil, proof) {
		if !reachable {
			t.Fatalf("block %d lost reachability", block)
		}
	}
}

func BenchmarkAssertionProofWithoutSwitch(b *testing.B) {
	for _, blocks := range []int{64, 256, 512} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			function := assertionLinearFixture(blocks)
			if failures := mir.Verify(mir.Program{Functions: []mir.Function{function}}); len(failures) != 0 {
				b.Fatal(failures[0])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if proof := assertedCheckedFailuresForC([]mir.Function{function})[function.Symbol]; len(proof) != 0 {
					b.Fatal("unexpected failure proof")
				}
			}
		})
	}
}

func assertionLinearFixture(blocks int) mir.Function {
	function := mir.Function{Symbol: 1, Name: "main::main", ResultType: hir.TypeI32, Entry: 0}
	for i := range blocks {
		local := mir.LocalID(i + 1)
		block := mir.Block{ID: mir.BlockID(i), Operations: []mir.Operation{{
			Kind: mir.OperationAssign, Type: hir.TypeI32, DestinationLocal: local,
			Operands: []mir.Operand{{Kind: mir.OperandI32, Type: hir.TypeI32, I32: int32(i)}},
		}}, Terminator: mir.Terminator{UnwindSuccessor: mir.NoBlock}}
		if i+1 == blocks {
			block.Terminator.Kind = mir.TerminatorReturn
			block.Terminator.ReturnValue = mir.Operand{Kind: mir.OperandLocal, Type: hir.TypeI32, Local: local}
		} else {
			block.Terminator.Kind = mir.TerminatorGoto
			block.Terminator.Successors = []mir.BlockID{mir.BlockID(i + 1)}
		}
		function.Blocks = append(function.Blocks, block)
	}
	return function
}
