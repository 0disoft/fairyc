package contractstatemir

import (
	"bytes"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestStateGraphStablePayloadRebindAndExecution(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-c14ce14f76ca2cc42c89dfb70b90c251684bbb445d6c659c3956eeed097efd67")
	base := stateGraphPackage(t, 2, 17, false)
	artifacts, err := DescribeStableHIRStateBodies(base, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[2]
	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(StableStateGraphBodyVersion)) {
		t.Fatalf("graph payload does not use %q", StableStateGraphBodyVersion)
	}
	decoded, err := DecodeStableStateBody(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	fresh := stateGraphPackage(t, 9, 700, false)
	rebound, err := RebindStableStateBody(decoded, body.Hash, fresh, 9, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if len(rebound.Transactions) != 1 {
		t.Fatalf("rebound transactions = %d, want 1", len(rebound.Transactions))
	}
	program := rebound.Transactions[0].Program
	store, err := NewMemoryStore(program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Execute(store, program, contractstate.Limits{MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64})
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || outcome.Cost != 7 || outcome.Receipt.Writes != 3 || len(outcome.Reads) != 2 {
		t.Fatalf("graph outcome = %+v", outcome)
	}

	changed := stateGraphPackage(t, 2, 17, true)
	changedHashes, err := ComputeHIRStateBodyHashes(changed, []hir.PackageID{2}, map[hir.PackageID]packagegraph.CoordinateID{2: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if changedHashes[2] == body.Hash {
		t.Fatal("branch target change did not invalidate the state body hash")
	}
}

func TestLegacyProgramRejectsGraphOnlyOperationFields(t *testing.T) {
	layout, err := BuildLayout("fixture.legacy", []Declaration{{Name: "value", Kind: contractstate.FieldCell, ValueTypeID: contractstate.U64.TypeID()}})
	if err != nil {
		t.Fatal(err)
	}
	capability := newTransactionCapability()
	program := Program{
		Layout: layout, Capability: capability, Effects: []hir.Effect{hir.EffectStorageRead},
		Operations: []Operation{{Kind: OperationCellRead, Capability: capability, FieldIndex: 0, Result: 1}}, Exit: ExitCommit,
	}
	failures := Verify(program)
	if len(failures) == 0 || failures[0].Code != "CSM1002" {
		t.Fatalf("Verify() = %+v", failures)
	}
}

func stateGraphPackage(t *testing.T, packageID hir.PackageID, symbol hir.SymbolID, reverse bool) hir.Package {
	t.Helper()
	u64 := func(value uint64) hir.StateValue {
		encoded, err := contractstate.U64.Encode(value)
		if err != nil {
			t.Fatal(err)
		}
		return hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: encoded}
	}
	boolean := func(value bool) hir.StateValue {
		encoded, err := contractstate.Bool.Encode(value)
		if err != nil {
			t.Fatal(err)
		}
		return hir.StateValue{TypeID: contractstate.Bool.TypeID(), Encoded: encoded}
	}
	trueTarget, falseTarget := hir.StateBlockID(1), hir.StateBlockID(2)
	if reverse {
		trueTarget, falseTarget = falseTarget, trueTarget
	}
	return hir.Package{
		Modules: []hir.Module{{Package: packageID, Path: "ledger", Functions: []hir.Function{{
			Symbol: symbol, Name: "apply", ResultType: hir.TypeVoid,
			Effects: []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite},
		}}}},
		StateLayouts: []hir.StateLayout{{Package: packageID, ContractUID: "fixture.graph", Fields: []hir.StateField{
			{Name: "primary", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
			{Name: "secondary", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID()},
			{Name: "enabled", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bool.TypeID()},
		}}},
		StateTransactions: []hir.StateTransaction{{Package: packageID, Function: symbol, Entry: 0, Blocks: []hir.StateBlock{
			{ID: 0, Accesses: []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "primary", Value: u64(7), Cost: 1},
				{Kind: hir.StateAccessCellWrite, Field: "enabled", Value: boolean(true), Cost: 1},
				{Kind: hir.StateAccessCellRead, Field: "primary", Result: 1, FoundResult: 2, Cost: 1},
				{Kind: hir.StateAccessCellRead, Field: "enabled", Result: 3, FoundResult: 4, Cost: 1},
			}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorBranch, Condition: 3, TrueTarget: trueTarget, TrueArguments: []hir.StateValueID{1}, FalseTarget: falseTarget, FalseArguments: []hir.StateValueID{1}, Cost: 1}},
			{ID: 1, Parameters: []hir.StateBlockParameter{{ID: 5, TypeID: contractstate.U64.TypeID()}}, Accesses: []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "secondary", ValueInput: 5, Cost: 1},
			}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}},
			{ID: 2, Parameters: []hir.StateBlockParameter{{ID: 6, TypeID: contractstate.U64.TypeID()}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorReject, Cost: 1}},
		}}},
	}
}
