package contractstatemir

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestLowerHIRTransactionsExecutesStaticStateHelperInRootOverlay(t *testing.T) {
	input := helperHIRPackage()
	lowered, err := LowerHIRTransactions(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(lowered) != 1 || lowered[0].Function != 7 || len(lowered[0].Program.Helpers) != 1 {
		t.Fatalf("lowered transactions = %+v", lowered)
	}
	if lowered[0].Program.Helpers[0].Name != "ledger::write_value" {
		t.Fatalf("helper = %+v", lowered[0].Program.Helpers[0])
	}
	store, err := NewMemoryStore(lowered[0].Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Execute(store, lowered[0].Program, contractstate.Limits{MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || outcome.Receipt.Version != 1 || outcome.Cost != 1 {
		t.Fatalf("outcome = %+v", outcome)
	}
	dump := store.Dump()
	if len(dump) != 1 || string(dump[0].Value) != string(mustEncodeU64(t, 42)) {
		t.Fatalf("dump = %+v", dump)
	}
}

func TestStableStateHelperBodyRoundTripsRebindsAndInvalidates(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-e8756812ba7956f1f997a35a522b7f1735d8679c99c48b140570f4cd690c1a91")
	input := helperHIRPackage()
	artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	if len(body.Transactions) != 1 || len(body.Transactions[0].Helpers) != 1 {
		t.Fatalf("artifact = %+v", body)
	}
	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(StableStateHelperBodySchema)) {
		t.Fatalf("helper body did not use v3 envelope: %s", encoded)
	}
	decoded, err := DecodeStableStateBody(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(decoded, body) {
		t.Fatalf("decoded helper artifact differs:\n got %#v\nwant %#v", decoded, body)
	}

	fresh := helperHIRPackage()
	fresh.Modules[0].Package = 9
	fresh.Modules[0].Functions[0].Symbol = 70
	fresh.Modules[0].Functions[0].Body.Tail = helperCallExpression(80, []hir.Effect{hir.EffectStorageWrite})
	fresh.Modules[0].Functions[1].Symbol = 80
	fresh.StateLayouts[0].Package = 9
	rebound, err := RebindStableStateBody(body, body.Hash, fresh, 9, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if len(rebound.Transactions) != 1 || rebound.Transactions[0].Function != 70 || len(rebound.Transactions[0].Program.Helpers) != 1 {
		t.Fatalf("rebound = %+v", rebound)
	}

	mutated := helperHIRPackage()
	mutated.StateTransactions[1].Accesses[0].Value.Encoded = mustEncodeU64(t, 43)
	hashes, err := ComputeHIRStateBodyHashes(mutated, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if hashes[1] == body.Hash {
		t.Fatal("helper body mutation did not invalidate package body hash")
	}
}

func TestLowerHIRTransactionsRejectsHelperGraphDrift(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*hir.Package)
		contains string
	}{
		{name: "missing transaction helper", mutate: func(input *hir.Package) { input.StateTransactions = input.StateTransactions[:1] }, contains: "missing or non-transaction helper"},
		{name: "mismatched call body", mutate: func(input *hir.Package) { input.StateTransactions[0].Accesses = nil }, contains: "helper calls do not match"},
		{name: "recursive helper graph", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[1].Body.Tail = helperCallExpression(7, []hir.Effect{hir.EffectStorageWrite})
			input.StateTransactions[1].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: 7, Cost: 1}}
		}, contains: "no root transaction"},
		{name: "root misses transitive effect", mutate: func(input *hir.Package) { input.Modules[0].Functions[0].Effects = nil }, contains: "missing required effects"},
		{name: "call expression effect drift", mutate: func(input *hir.Package) { input.Modules[0].Functions[0].Body.Tail.Effects = nil }, contains: "call effects do not match"},
		{name: "graph call remains closed", mutate: func(input *hir.Package) {
			input.StateTransactions[0].Accesses = nil
			input.StateTransactions[0].Exit = ""
			input.StateTransactions[0].Blocks = []hir.StateBlock{{ID: 0, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: 8, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}}
		}, contains: "graph helper calls are not admitted"},
		{name: "helper has parameter", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[1].Parameters = []hir.Parameter{{Type: hir.TypeU32}}
		}, contains: "must be void and have no parameters"},
		{name: "ambiguous stable owner", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[1].Name = "root"
		}, contains: "ambiguous owners"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := helperHIRPackage()
			test.mutate(&input)
			if _, err := LowerHIRTransactions(input); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func helperHIRPackage() hir.Package {
	return hir.Package{
		Modules: []hir.Module{{Package: 1, Path: "ledger", Functions: []hir.Function{
			{Symbol: 7, Name: "root", Effects: []hir.Effect{hir.EffectStorageWrite}, ResultType: hir.TypeVoid, Body: hir.Block{Tail: helperCallExpression(8, []hir.Effect{hir.EffectStorageWrite})}},
			{Symbol: 8, Name: "write_value", Effects: []hir.Effect{hir.EffectStorageWrite}, ResultType: hir.TypeVoid},
		}}},
		StateLayouts: []hir.StateLayout{{Package: 1, ContractUID: "helper.state", Fields: []hir.StateField{{Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()}}}},
		StateTransactions: []hir.StateTransaction{
			{Package: 1, Function: 7, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: 8, Cost: 1}}, Exit: hir.StateTransactionCommit},
			{Package: 1, Function: 8, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: mustEncodeU64(nil, 42)}}}, Exit: hir.StateTransactionReturn},
		},
	}
}

func helperCallExpression(symbol hir.SymbolID, effects []hir.Effect) *hir.Expression {
	return &hir.Expression{Kind: "call", Type: hir.TypeVoid, Symbol: symbol, Effects: effects, Children: []hir.Expression{{Kind: "path", Symbol: symbol}}}
}

func mustEncodeU64(t *testing.T, value uint64) []byte {
	encoded, err := contractstate.U64.Encode(value)
	if err != nil {
		if t != nil {
			t.Fatal(err)
		}
		panic(err)
	}
	return encoded
}
