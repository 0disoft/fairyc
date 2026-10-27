package contractstatemir

import (
	"fmt"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStableHelperRebindUsesCurrentPackageAndEveryRoot(t *testing.T) {
	input := helperRebindLookupFixture(8, 2)
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
	artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	for _, test := range []struct {
		name   string
		mutate func(*hir.Package)
		want   string
	}{
		{"fresh-symbols", func(input *hir.Package) {
			input.Modules[0].Functions[7].Symbol = 200000
			for i := 0; i < 2; i++ {
				input.Modules[0].Functions[i].Body.Tail.Symbol = 200000
				input.Modules[0].Functions[i].Body.Tail.Children[0].Symbol = 200000
			}
		}, ""},
		{"other-package-collision", func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 2, Path: "ledger", Functions: []hir.Function{{Symbol: 100000, Name: "write_value"}}})
		}, ""},
		{"second-root-effects", func(input *hir.Package) {
			input.Modules[0].Functions[1].Body.Tail.Effects = nil
		}, "call effects do not match"},
		{"helper-effects", func(input *hir.Package) {
			input.Modules[0].Functions[7].Effects = nil
		}, "call effects do not match"},
		{"missing-helper", func(input *hir.Package) {
			input.Modules[0].Functions = input.Modules[0].Functions[:7]
		}, "call effects do not match"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fresh := helperRebindLookupFixture(8, 2)
			fresh.StateTransactions = nil
			// Seed a separate successful operation before changing the next input.
			if _, err := RebindStableStateBody(body, body.Hash, fresh, 1, coordinates); err != nil {
				t.Fatal(err)
			}
			test.mutate(&fresh)
			result, err := RebindStableStateBody(body, body.Hash, fresh, 1, coordinates)
			if test.want != "" {
				if err == nil || !strings.Contains(err.Error(), test.want) {
					t.Fatalf("rebind error = %v, want %q", err, test.want)
				}
				return
			}
			if err != nil || len(result.Transactions) != 2 {
				t.Fatalf("rebind two roots: %v", err)
			}
		})
	}
}

func BenchmarkStableStateHelperRebindLookup(b *testing.B) {
	for _, roots := range []int{1, 16, 64} {
		b.Run(fmt.Sprint(roots), func(b *testing.B) {
			input := helperRebindLookupFixture(1024, roots)
			coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: {1}}
			artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, coordinates)
			if err != nil {
				b.Fatal(err)
			}
			body := artifacts[1]
			input.StateTransactions = nil
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				result, err := RebindStableStateBody(body, body.Hash, input, 1, coordinates)
				if err != nil || len(result.Transactions) != roots {
					b.Fatalf("rebind %d roots: %v", roots, err)
				}
			}
		})
	}
}

func helperRebindLookupFixture(functions, roots int) hir.Package {
	input := helperHIRPackage()
	root, helper := input.Modules[0].Functions[0], input.Modules[0].Functions[1]
	rootTransaction, helperTransaction := input.StateTransactions[0], input.StateTransactions[1]
	helper.Symbol = 100000
	input.Modules[0].Functions = make([]hir.Function, functions)
	for i := range functions {
		input.Modules[0].Functions[i] = hir.Function{Symbol: hir.SymbolID(1000 + i), Name: fmt.Sprintf("unused_%04d", i), ResultType: hir.TypeVoid}
	}
	input.Modules[0].Functions[functions-1] = helper
	input.StateTransactions = make([]hir.StateTransaction, roots+1)
	for i := range roots {
		function := root
		function.Symbol, function.Name = hir.SymbolID(i+1), fmt.Sprintf("root_%04d", i)
		function.Body.Tail = helperCallExpression(helper.Symbol, helper.Effects)
		input.Modules[0].Functions[i] = function
		transaction := rootTransaction
		transaction.Function = function.Symbol
		transaction.Accesses = []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: helper.Symbol, Cost: 1}}
		input.StateTransactions[i] = transaction
	}
	helperTransaction.Function = helper.Symbol
	input.StateTransactions[roots] = helperTransaction
	return input
}
