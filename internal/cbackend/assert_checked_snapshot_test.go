package cbackend

import (
	"fmt"
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func assertionCheckedChainFixture(blocks int) mir.Function {
	f := assertionLinearFixture(blocks)
	last := &f.Blocks[blocks-1]
	last.Operations[0].Operands[0].I32 = math.MaxInt32
	last.Terminator = mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI32, Operator: "+",
		SwitchOperands: []mir.Operand{{Kind: mir.OperandLocal, Type: hir.TypeI32, Local: mir.LocalID(blocks)}, {Kind: mir.OperandI32, Type: hir.TypeI32, I32: 1}},
		Successors:     []mir.BlockID{mir.BlockID(blocks), mir.BlockID(blocks + 1)}, UnwindSuccessor: mir.NoBlock}
	f.Blocks = append(f.Blocks,
		mir.Block{ID: mir.BlockID(blocks), Terminator: mir.Terminator{Kind: mir.TerminatorReturn, ReturnValue: mir.Operand{Kind: mir.OperandI32, Type: hir.TypeI32}, UnwindSuccessor: mir.NoBlock}},
		mir.Block{ID: mir.BlockID(blocks + 1), Terminator: mir.Terminator{Kind: mir.TerminatorTrap, TrapCategories: []string{"integer-overflow"}, UnwindSuccessor: mir.NoBlock}})
	return f
}

func TestAssertCheckedSnapshotsKeepSiblingFactsSeparate(t *testing.T) {
	local := func(id mir.LocalID, typ hir.Type) mir.Operand {
		return mir.Operand{Kind: mir.OperandLocal, Type: typ, Local: id}
	}
	f := mir.Function{Symbol: 1, Name: "main::test", Entry: 0, ResultType: hir.TypeVoid,
		Parameters: []mir.LocalID{1, 2}, ParameterTypes: []hir.Type{hir.TypeI64, hir.TypeBool}, Blocks: []mir.Block{
			{ID: 0, Operations: []mir.Operation{{Kind: mir.OperationBinary, Type: hir.TypeBool, OwnerType: hir.TypeI64, DestinationLocal: 3, Operator: "==", Operands: []mir.Operand{local(1, hir.TypeI64), {Kind: mir.OperandI64, Type: hir.TypeI64, I64: math.MaxInt64}}}},
				Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "bool", SwitchOperands: []mir.Operand{local(2, hir.TypeBool)}, Successors: []mir.BlockID{1, 3}, UnwindSuccessor: mir.NoBlock}},
			{ID: 1, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "assert", SwitchType: hir.TypeBool, Operator: "assert", SwitchOperands: []mir.Operand{local(3, hir.TypeBool)}, Successors: []mir.BlockID{2, 4}, UnwindSuccessor: mir.NoBlock}},
			{ID: 2, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI64, Operator: "+", SwitchOperands: []mir.Operand{local(1, hir.TypeI64), {Kind: mir.OperandI64, Type: hir.TypeI64, I64: 1}}, Successors: []mir.BlockID{5, 6}, UnwindSuccessor: mir.NoBlock}},
			{ID: 3, Terminator: mir.Terminator{Kind: mir.TerminatorSwitch, SwitchKind: "checked-arithmetic", SwitchType: hir.TypeI64, Operator: "+", SwitchOperands: []mir.Operand{local(1, hir.TypeI64), {Kind: mir.OperandI64, Type: hir.TypeI64, I64: 1}}, Successors: []mir.BlockID{5, 6}, UnwindSuccessor: mir.NoBlock}},
			{ID: 4, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, TrapCategories: []string{"assertion-failed"}, UnwindSuccessor: mir.NoBlock}},
			{ID: 5, Terminator: mir.Terminator{Kind: mir.TerminatorReturn, UnwindSuccessor: mir.NoBlock}},
			{ID: 6, Terminator: mir.Terminator{Kind: mir.TerminatorTrap, TrapCategories: []string{"integer-overflow"}, UnwindSuccessor: mir.NoBlock}},
		}}
	if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
		t.Fatal(failures[0])
	}
	proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
	if target, ok := proof[2]; !ok || target != 6 {
		t.Fatalf("asserted branch lost its overflow proof: %v", proof)
	}
	if _, leaked := proof[3]; leaked {
		t.Fatalf("assertion fact leaked into sibling branch: %v", proof)
	}
}

func BenchmarkAssertionProofCheckedChain(b *testing.B) {
	for _, blocks := range []int{64, 256, 512} {
		b.Run(fmt.Sprint(blocks), func(b *testing.B) {
			f := assertionCheckedChainFixture(blocks)
			if failures := mir.Verify(mir.Program{Functions: []mir.Function{f}}); len(failures) != 0 {
				b.Fatal(failures[0])
			}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				proof := assertedCheckedFailuresForC([]mir.Function{f})[f.Symbol]
				if target, ok := proof[mir.BlockID(blocks-1)]; !ok || target != mir.BlockID(blocks+1) {
					b.Fatal("missing overflow proof")
				}
			}
		})
	}
}
