package contractstatemir

import (
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func BenchmarkStateBodyFunctionLookup(b *testing.B) {
	for _, transactions := range []int{1, 64, 256} {
		b.Run(fmt.Sprint(transactions), func(b *testing.B) {
			input := stateFunctionLookupFixture(b, 4096, transactions)
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				hashes, err := ComputeHIRStateBodyHashes(input, []hir.PackageID{1}, coordinates)
				if err != nil || len(hashes) != 1 {
					b.Fatalf("hash %d transactions: %v", transactions, err)
				}
			}
		})
	}
}

func stateFunctionLookupFixture(t testing.TB, functions, transactions int) hir.Package {
	t.Helper()
	input := stateBodyPackage(t, 1, 1)
	function, transaction := input.Modules[0].Functions[0], input.StateTransactions[0]
	input.Modules[0].Functions = make([]hir.Function, functions)
	input.StateTransactions = make([]hir.StateTransaction, transactions)
	for i := range functions {
		item := function
		item.Symbol, item.Name = hir.SymbolID(i+1), fmt.Sprintf("apply_%04d", i)
		input.Modules[0].Functions[i] = item
	}
	for i := range transactions {
		item := transaction
		item.Function = hir.SymbolID(functions - i)
		input.StateTransactions[i] = item
	}
	return input
}
