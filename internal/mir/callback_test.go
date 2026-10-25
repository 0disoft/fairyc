package mir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
)

func callbackVerificationProgram(count int) (Program, map[hir.SymbolID]*Function, map[hir.SymbolID]*ForeignFunction) {
	pointer := hir.RawPointer(hir.TypeVoid, true)
	program := Program{
		ForeignFunctions: []ForeignFunction{{Symbol: 1, ABI: "C", ParameterTypes: []hir.Type{hir.TypeCCallbackI32UserData, pointer}, ResultType: hir.TypeVoid}},
		Functions: []Function{
			{Symbol: 2, Name: "callback", ParameterTypes: []hir.Type{hir.TypeI32, pointer}, ResultType: hir.TypeI32},
			{Symbol: 3, Name: "caller", ResultType: hir.TypeVoid, Blocks: make([]Block, count+1)},
		},
	}
	caller := &program.Functions[1]
	for i := range count {
		// Sparse IDs also exercise cached/forged MIR without dense allocation.
		local := LocalID(1000000 + i*7)
		caller.Blocks[i] = Block{
			ID: BlockID(i), Operations: []Operation{{Kind: OperationCCallback, Symbol: 2, DestinationLocal: local}},
			Terminator: Terminator{Kind: TerminatorCall, Callee: 1, ForeignABI: "C", DestinationLocal: NoLocal,
				Arguments: []Operand{{Kind: OperandLocal, Local: local}, {Kind: OperandNull, Type: pointer}}},
		}
	}
	caller.Blocks[count] = Block{ID: BlockID(count), Terminator: Terminator{Kind: TerminatorReturn, DestinationLocal: NoLocal}}
	return program, map[hir.SymbolID]*Function{2: &program.Functions[0], 3: caller}, map[hir.SymbolID]*ForeignFunction{1: &program.ForeignFunctions[0]}
}

func TestBoundedCallbackUsageChecksAllDefinitionAndUseSites(t *testing.T) {
	for _, mode := range []string{"valid", "parameter", "operation definition", "call definition", "operation use", "call use", "return use", "switch use", "duplicate origin"} {
		t.Run(mode, func(t *testing.T) {
			program, functions, foreign := callbackVerificationProgram(2)
			caller := &program.Functions[1]
			local := caller.Blocks[0].Operations[0].DestinationLocal
			operand := Operand{Kind: OperandLocal, Local: local}
			last := &caller.Blocks[2]
			switch mode {
			case "parameter":
				caller.Parameters = append(caller.Parameters, local)
			case "operation definition":
				last.Operations = []Operation{{DestinationLocal: local}}
			case "call definition":
				last.Terminator.DestinationLocal = local
			case "operation use":
				last.Operations = []Operation{{DestinationLocal: NoLocal, Operands: []Operand{operand}}}
			case "call use":
				last.Terminator.Arguments = []Operand{operand}
			case "return use":
				last.Terminator.ReturnValue = operand
			case "switch use":
				last.Terminator.SwitchOperands = []Operand{operand}
			case "duplicate origin":
				first := &caller.Blocks[0]
				first.Operations = append(first.Operations, first.Operations[0])
			}
			failures := verifyBoundedCallbacks(program, functions, foreign)
			if mode == "valid" {
				if len(failures) != 0 {
					t.Fatalf("valid callback usage: %+v", failures)
				}
				return
			}
			found := false
			for _, failure := range failures {
				found = found || failure.Code == CodeForeignDeclaration && failure.Detail == "callback address escapes or its temporary is overwritten"
			}
			if !found {
				t.Fatalf("%s was not rejected: %+v", mode, failures)
			}
		})
	}
}

func BenchmarkBoundedCallbackVerification(b *testing.B) {
	for _, count := range []int{512, 2048} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			program, functions, foreign := callbackVerificationProgram(count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				if failures := verifyBoundedCallbacks(program, functions, foreign); len(failures) != 0 {
					b.Fatalf("callback verification: %+v", failures)
				}
			}
		})
	}
}
