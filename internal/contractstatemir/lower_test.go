package contractstatemir

import (
	"slices"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
)

func TestLowerHIRLayoutsIsPackageOrderedAndCanonical(t *testing.T) {
	input := hir.Package{StateLayouts: []hir.StateLayout{
		{Package: 9, ContractUID: "example.wallet", Fields: []hir.StateField{
			{Name: "owners", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.Byte20.TypeID()},
		}},
		{Package: 2, ContractUID: "example.token", Fields: []hir.StateField{
			{Name: "total_supply", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
			{Name: "balances", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
		}},
	}}
	got, err := LowerHIRLayouts(input)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(got, []PackageLayout{{Package: 2}, {Package: 9}}, func(left, right PackageLayout) bool {
		return left.Package == right.Package
	}) {
		t.Fatalf("package order = %+v", got)
	}
	expected, err := BuildLayout("example.token", []Declaration{
		{Name: "total_supply", Kind: contractstate.FieldCell, ValueTypeID: contractstate.U64.TypeID()},
		{Name: "balances", Kind: contractstate.FieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got[0].Layout.Hash != expected.Hash || got[0].Layout.ContractUID != expected.ContractUID {
		t.Fatalf("lowered token layout = %+v, want hash %s", got[0], expected.Hash)
	}
}

func TestLowerHIRLayoutsRejectsDuplicatePackageAndUnknownKind(t *testing.T) {
	duplicate := hir.Package{StateLayouts: []hir.StateLayout{
		{Package: 1, ContractUID: "example.one"},
		{Package: 1, ContractUID: "example.two"},
	}}
	if _, err := LowerHIRLayouts(duplicate); err == nil || !strings.Contains(err.Error(), "multiple") {
		t.Fatalf("duplicate error = %v", err)
	}

	unknown := hir.Package{StateLayouts: []hir.StateLayout{{
		Package: 1, ContractUID: "example.one",
		Fields: []hir.StateField{{Name: "value", Kind: "dynamic", ValueTypeID: contractstate.U64.TypeID()}},
	}}}
	if _, err := LowerHIRLayouts(unknown); err == nil || !strings.Contains(err.Error(), "unsupported field kind") {
		t.Fatalf("unknown kind error = %v", err)
	}
}

func TestLowerHIRTransactionsOrdersBindsAndExecutes(t *testing.T) {
	account := encodeHIR(t, contractstate.Byte20, contractstate.Bytes20{19: 7})
	ten := encodeHIR(t, contractstate.U64, uint64(10))
	seven := encodeHIR(t, contractstate.U64, uint64(7))
	expectedTen := slices.Clone(ten.Encoded)
	input := hir.Package{
		Modules: []hir.Module{
			{Package: 9, Functions: []hir.Function{{Symbol: 90}}},
			{Package: 2, Functions: []hir.Function{
				{Symbol: 20, Effects: []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}},
				{Symbol: 10},
			}},
		},
		StateLayouts: []hir.StateLayout{
			{Package: 9, ContractUID: "example.wallet", Fields: []hir.StateField{
				{Name: "owners", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.Byte20.TypeID()},
			}},
			{Package: 2, ContractUID: "example.token", Fields: []hir.StateField{
				{Name: "total_supply", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
				{Name: "balances", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Byte20.TypeID(), ValueTypeID: contractstate.U64.TypeID()},
			}},
		},
		StateTransactions: []hir.StateTransaction{
			{Package: 9, Function: 90, Exit: hir.StateTransactionTrap, TrapCategory: "assertion-failed"},
			{Package: 2, Function: 20, Exit: hir.StateTransactionCommit, Accesses: []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "total_supply", Value: ten},
				{Kind: hir.StateAccessCellRead, Field: "total_supply"},
				{Kind: hir.StateAccessMapWrite, Field: "balances", Key: account, Value: seven},
				{Kind: hir.StateAccessMapRead, Field: "balances", Key: account},
				{Kind: hir.StateAccessMapDelete, Field: "balances", Key: account},
			}},
			{Package: 2, Function: 10, Exit: hir.StateTransactionReject},
		},
	}

	got, err := LowerHIRTransactions(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("lowered transaction count = %d, want 3", len(got))
	}
	identities := [][2]int{{int(got[0].Package), int(got[0].Function)}, {int(got[1].Package), int(got[1].Function)}, {int(got[2].Package), int(got[2].Function)}}
	if !slices.Equal(identities, [][2]int{{2, 10}, {2, 20}, {9, 90}}) {
		t.Fatalf("transaction order = %v", identities)
	}
	if got[0].Program.Exit != ExitReject || got[1].Program.Exit != ExitCommit || got[2].Program.Exit != ExitTrap {
		t.Fatalf("lowered exits = %q, %q, %q", got[0].Program.Exit, got[1].Program.Exit, got[2].Program.Exit)
	}
	if got[0].Program.Capability == got[1].Program.Capability || got[0].Program.Capability == got[2].Program.Capability || got[1].Program.Capability == got[2].Program.Capability {
		t.Fatal("separate HIR transactions share one capability")
	}
	for index, operation := range got[1].Program.Operations {
		if operation.Capability != got[1].Program.Capability {
			t.Fatalf("operation %d capability differs from program", index)
		}
	}
	input.StateTransactions[1].Accesses[0].Value.Encoded[0] ^= 0xff
	if !slices.Equal(got[1].Program.Operations[0].Value.Encoded, expectedTen) {
		t.Fatal("lowered value aliases mutable HIR bytes")
	}

	store, err := NewMemoryStore(got[1].Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Execute(store, got[1].Program, testLimits())
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || len(outcome.Reads) != 2 {
		t.Fatalf("outcome = %+v", outcome)
	}
	assertRead(t, outcome.Reads[0], 1, true, Value{TypeID: contractstate.U64.TypeID(), Encoded: expectedTen})
	assertRead(t, outcome.Reads[1], 3, true, Value{TypeID: contractstate.U64.TypeID(), Encoded: seven.Encoded})
}

func TestLowerHIRTransactionsRejectsBoundaryDrift(t *testing.T) {
	tests := []struct {
		name     string
		mutate   func(*hir.Package)
		contains string
	}{
		{name: "missing layout", mutate: func(input *hir.Package) { input.StateLayouts = nil }, contains: "no owned layout"},
		{name: "missing function", mutate: func(input *hir.Package) { input.Modules = nil }, contains: "missing function"},
		{name: "missing symbol", mutate: func(input *hir.Package) { input.StateTransactions[0].Function = hir.NoSymbol }, contains: "no function symbol"},
		{name: "duplicate function", mutate: func(input *hir.Package) {
			input.Modules = append(input.Modules, hir.Module{Package: 1, Functions: []hir.Function{{Symbol: 7, Effects: []hir.Effect{hir.EffectStorageRead}}}})
		}, contains: "function symbol 7 is duplicated"},
		{name: "duplicate transaction", mutate: func(input *hir.Package) {
			input.StateTransactions = append(input.StateTransactions, input.StateTransactions[0])
		}, contains: "multiple state transactions"},
		{name: "unknown field", mutate: func(input *hir.Package) { input.StateTransactions[0].Accesses[0].Field = "missing" }, contains: "unknown field"},
		{name: "unknown access", mutate: func(input *hir.Package) { input.StateTransactions[0].Accesses[0].Kind = "dynamic" }, contains: "unsupported state access kind"},
		{name: "missing effect", mutate: func(input *hir.Package) { input.Modules[0].Functions[0].Effects = nil }, contains: "CSM1005"},
		{name: "host effect", mutate: func(input *hir.Package) { input.Modules[0].Functions[0].Effects = []hir.Effect{hir.EffectFilesystem} }, contains: "CSM1005"},
		{name: "unrepresented external call effect", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[0].Effects = []hir.Effect{hir.EffectExternalCall, hir.EffectStorageRead}
		}, contains: "has no contract-state transaction MIR operation"},
		{name: "owner call", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: "call"}
		}, contains: "contains a call; calls are not admitted"},
		{name: "nested owner call", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[0].Body.Statements = []hir.Statement{{For: &hir.ForLoop{
				Body: hir.Block{Tail: &hir.Expression{Kind: "group", Children: []hir.Expression{{Kind: "call"}}}},
			}}}
		}, contains: "contains a call; calls are not admitted"},
		{name: "noncanonical effects", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[0].Effects = []hir.Effect{hir.EffectStorageWrite, hir.EffectStorageRead}
		}, contains: "CSM1005"},
		{name: "wrong field kind", mutate: func(input *hir.Package) { input.StateTransactions[0].Accesses[0].Kind = hir.StateAccessMapRead }, contains: "CSM1002"},
		{name: "noncanonical value", mutate: func(input *hir.Package) {
			input.Modules[0].Functions[0].Effects = []hir.Effect{hir.EffectStorageWrite}
			input.StateTransactions[0].Accesses[0] = hir.StateAccess{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: []byte{1}}}
		}, contains: "CSM1002"},
		{name: "unknown exit", mutate: func(input *hir.Package) { input.StateTransactions[0].Exit = "suspend" }, contains: "unsupported state transaction exit"},
		{name: "invalid trap", mutate: func(input *hir.Package) { input.StateTransactions[0].Exit = hir.StateTransactionTrap }, contains: "CSM1003"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := baseHIRTransaction()
			test.mutate(&input)
			if _, err := LowerHIRTransactions(input); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want %q", err, test.contains)
			}
		})
	}
}

func baseHIRTransaction() hir.Package {
	return hir.Package{
		Modules: []hir.Module{{Package: 1, Functions: []hir.Function{{Symbol: 7, Effects: []hir.Effect{hir.EffectStorageRead}}}}},
		StateLayouts: []hir.StateLayout{{Package: 1, ContractUID: "example.state", Fields: []hir.StateField{
			{Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
		}}},
		StateTransactions: []hir.StateTransaction{{
			Package: 1, Function: 7, Exit: hir.StateTransactionCommit,
			Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}},
		}},
	}
}

func encodeHIR[T any](t *testing.T, codec contractstate.Codec[T], value T) hir.StateValue {
	t.Helper()
	encoded, err := codec.Encode(value)
	if err != nil {
		t.Fatal(err)
	}
	return hir.StateValue{TypeID: codec.TypeID(), Encoded: encoded}
}
