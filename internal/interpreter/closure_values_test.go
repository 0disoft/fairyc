package interpreter

import (
	"math"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/mir"
)

func TestClosureValueSnapshotCopyAndZeroSlots(t *testing.T) {
	for count := 0; count <= 4; count++ {
		target := &mir.Function{Symbol: 7, ClosureTarget: true, ClosureCaptures: uint8(count), ResultType: hir.TypeI32}
		for i := 0; i <= count; i++ {
			target.ParameterTypes = append(target.ParameterTypes, hir.TypeI32)
		}
		r := executionState{bySymbol: map[hir.SymbolID]*mir.Function{7: target}, copyUnits: 1000}
		locals := map[mir.LocalID]runtimeValue{1: i32Value(math.MinInt32), 2: i32Value(-1), 3: i32Value(0), 4: i32Value(math.MaxInt32)}
		op := mir.Operation{Kind: mir.OperationClosureCreate, Type: hir.TypeClosureI32, Symbol: 7, DestinationLocal: 8}
		for i := 0; i < count; i++ {
			op.Operands = append(op.Operands, mir.Operand{Kind: mir.OperandLocal, Local: mir.LocalID(i + 1), Type: hir.TypeI32})
		}
		value, err := r.createClosure(locals, op)
		if err != nil {
			t.Fatal(err)
		}
		before := r.closures[0]
		for i := 1; i <= 4; i++ {
			locals[mir.LocalID(i)] = i32Value(123)
		}
		copied, err := r.cloneForValueCopy(value)
		if err != nil || copied != value || len(r.closures) != 1 || copied.payload != nil {
			t.Fatalf("closure copy changed immutable handle: %+v %v", copied, err)
		}
		if r.closures[0] != before {
			t.Fatal("creator local change altered snapshot")
		}
		for i := count; i < 4; i++ {
			if before.captures[i] != 0 {
				t.Fatal("unused capture slot was not zero")
			}
		}
		if n := testing.AllocsPerRun(20, func() {
			r.copyUnits = 1000
			_, err = r.cloneForValueCopy(value)
			if err != nil {
				panic(err)
			}
		}); n != 0 {
			t.Fatalf("closure copy allocated: %v", n)
		}
	}
}

func TestClosureValueRejectsForgedRuntimeState(t *testing.T) {
	r := executionState{}
	term := mir.Terminator{Kind: mir.TerminatorClosureCall, Arguments: []mir.Operand{{Kind: mir.OperandLocal, Local: 1}, {Kind: mir.OperandI32, I32: 1}}}
	for _, value := range []runtimeValue{{kind: valueClosure, typ: hir.TypeClosureI32}, {kind: valueClosure, typ: hir.TypeClosureI32, scalar: 1}, i32Value(1)} {
		if _, err := r.callClosure(map[mir.LocalID]runtimeValue{1: value}, term, 0); err == nil {
			t.Fatal("forged callable executed")
		}
	}
	for _, count := range []int{0, 5} {
		op := mir.Operation{Type: hir.TypeClosureI32, Symbol: 9, Operands: make([]mir.Operand, count)}
		if _, err := r.createClosure(nil, op); err == nil {
			t.Fatal("missing target/invalid count accepted")
		}
	}
}
