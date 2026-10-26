package cbackend

import (
	"fmt"
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func assertionBooleanChainFixture(blocks int) mir.Function {
	f := assertionLinearFixture(blocks)
	f.Parameters = []mir.LocalID{mir.LocalID(blocks + 1)}
	f.ParameterTypes = []hir.Type{hir.TypeBool}
	f.Blocks[blocks-1].Terminator = mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "bool",
		SwitchOperands: []mir.Operand{{Kind: mir.OperandLocal, Type: hir.TypeBool, Local: f.Parameters[0]}},
		Successors:     []mir.BlockID{mir.BlockID(blocks), mir.BlockID(blocks + 1)}, UnwindSuccessor: mir.NoBlock}
	for i := range 2 {
		f.Blocks = append(f.Blocks, mir.Block{ID: mir.BlockID(blocks + i), Terminator: mir.Terminator{Kind: mir.TerminatorReturn,
			ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32, I32: int32(i)}, UnwindSuccessor: mir.NoBlock}})
	}
	return f
}

func TestBooleanChainKeepsBothPaths(t *testing.T) {
	f := assertionBooleanChainFixture(64)
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
		t.Fatal(failures[0])
	}
	proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
	if len(proof) != 0 {
		t.Fatalf("ordinary boolean branch acquired failure proof: %v", proof)
	}
	if _, failed := constantCheckedFailure(f.Blocks[63]); failed {
		t.Fatal("ordinary boolean branch classified as a checked failure")
	}
	for block, reachable := range reachableBlocksForC(f, nil, proof) {
		if !reachable {
			t.Fatalf("block %d lost reachability", block)
		}
	}
}

func TestCheckedFailureKindsRemainProvable(t *testing.T) {
	for _, term := range []mir.Terminator{
		{SwitchKind: "checked-unary", SwitchType: hir.TypeI32, Operator: "-", SwitchOperands: []mir.Operand{{Kind: mir.OperandI32, Type: hir.TypeI32, I32: math.MinInt32}}},
		{SwitchKind: "checked-arithmetic", SwitchType: hir.TypeU32, Operator: "+", SwitchOperands: []mir.Operand{{Kind: mir.OperandU32, Type: hir.TypeU32, U64: math.MaxUint32}, {Kind: mir.OperandU32, Type: hir.TypeU32, U64: 1}}},
		{SwitchKind: "checked-view-index", Operator: "[]", SwitchOperands: []mir.Operand{{Kind: mir.OperandByteString, Type: hir.SharedBorrow(hir.TypeByteSlice)}, {Kind: mir.OperandUSize, Type: hir.TypeUSize}}},
		{SwitchKind: "checked-view-slice", Operator: "..", SwitchOperands: []mir.Operand{{Kind: mir.OperandByteString, Type: hir.SharedBorrow(hir.TypeByteSlice)}, {Kind: mir.OperandUSize, Type: hir.TypeUSize}, {Kind: mir.OperandUSize, Type: hir.TypeUSize, U64: 1}}},
	} {
		t.Run(term.SwitchKind, func(t *testing.T) {
			term.Kind, term.Successors, term.UnwindSuccessor = mir.TerminatorSwitch, []mir.BlockID{1, 2}, mir.NoBlock
			f := mir.Function{Symbol: 1, Name: "main::test", ResultType: hir.TypeVoid, Entry: 0, Blocks: []mir.Block{
				{ID: 0, Terminator: term},
				{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
				{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, UnwindSuccessor: mir.NoBlock}},
			}}
			if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
				t.Fatal(failures[0])
			}
			if target, failed := constantCheckedFailure(f.Blocks[0]); !failed || target != 2 {
				t.Fatal("constant failure proof lost")
			}
			if proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]; proof[0] != 2 {
				t.Fatalf("assertion failure proof lost: %v", proof)
			}
		})
	}
}

func BenchmarkAssertionProofBooleanChain(b *testing.B) {
	for _, blocks := range []int{64, 256, 512} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			f := assertionBooleanChainFixture(blocks)
			if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
				b.Fatal(failures[0])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]; len(proof) != 0 {
					b.Fatal("unexpected failure proof")
				}
			}
		})
	}
}
