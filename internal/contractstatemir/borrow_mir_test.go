package contractstatemir

import (
	"bytes"
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestHIRStateBorrowGraphExecutesNestedCellView(t *testing.T) {
	input := stateBorrowPackage(false)
	lowered, err := LowerHIRTransactions(input)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(lowered[0].Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := Execute(store, lowered[0].Program, borrowMIRLimits())
	if err != nil {
		t.Fatal(err)
	}
	if outcome.Exit != ExitCommit || len(outcome.Reads) != 1 || !outcome.Reads[0].Found || outcome.Cost != 6 {
		t.Fatalf("outcome = %+v", outcome)
	}
	dump := store.Dump()
	if len(dump) != 2 || string(dump[1].Value) != "bcd" {
		t.Fatalf("dump = %+v", dump)
	}
}

func TestHIRStateBorrowGraphPreservesUnrelatedMapEntry(t *testing.T) {
	input := stateMapBorrowPackage()
	lowered, err := LowerHIRTransactions(input)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(lowered[0].Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(store, lowered[0].Program, borrowMIRLimits()); err != nil {
		t.Fatal(err)
	}
	dump := store.Dump()
	counts := map[string]int{}
	for _, entry := range dump {
		counts[string(entry.Value)]++
	}
	if len(dump) != 3 || counts["alpha"] != 2 || counts["beta"] != 1 {
		t.Fatalf("dump = %+v", dump)
	}
}

func TestStateBorrowStablePayloadV4RebindAndHash(t *testing.T) {
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-c14ce14f76ca2cc42c89dfb70b90c251684bbb445d6c659c3956eeed097efd67")
	base := stateBorrowPackage(false)
	artifacts, err := DescribeStableHIRStateBodies(base, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	encoded, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(encoded, []byte(StableStateBorrowBodyVersion)) {
		t.Fatalf("borrow payload does not use %q", StableStateBorrowBodyVersion)
	}
	decoded, err := DecodeStableStateBody(encoded, 1<<20)
	if err != nil {
		t.Fatal(err)
	}

	fresh := stateBorrowPackage(false)
	fresh.Modules[0].Package = 9
	fresh.Modules[0].Functions[0].Symbol = 700
	fresh.StateLayouts[0].Package = 9
	fresh.StateTransactions[0].Package = 9
	fresh.StateTransactions[0].Function = 700
	rebound, err := RebindStableStateBody(decoded, body.Hash, fresh, 9, map[hir.PackageID]packagegraph.CoordinateID{9: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(rebound.Layout)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Execute(store, rebound.Transactions[0].Program, borrowMIRLimits()); err != nil {
		t.Fatal(err)
	}
	if dump := store.Dump(); len(dump) != 2 || string(dump[1].Value) != "bcd" {
		t.Fatalf("rebound dump = %+v", dump)
	}

	changed := stateBorrowPackage(false)
	changed.StateTransactions[0].Blocks[0].Accesses[2].ViewLength = 2
	hashes, err := ComputeHIRStateBodyHashes(changed, []hir.PackageID{1}, map[hir.PackageID]packagegraph.CoordinateID{1: coordinate})
	if err != nil {
		t.Fatal(err)
	}
	if hashes[1] == body.Hash {
		t.Fatal("borrow view length change did not invalidate the state body hash")
	}
}

func TestHIRStateBorrowGraphRejectsInvalidationAliasAndEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    hir.Package
		contains string
	}{
		{name: "exact mutation invalidates", input: stateBorrowPackage(true), contains: "non-invalidated borrow"},
		{name: "dynamic map alias", input: stateDynamicAliasPackage(), contains: "ambiguous alias"},
		{name: "edge escape", input: stateBorrowEscapePackage(), contains: "cannot cross a block edge"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := LowerHIRTransactions(test.input)
			if err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func TestStateBorrowVerifierRejectsHelperOwnership(t *testing.T) {
	lowered, err := LowerHIRTransactions(stateBorrowPackage(false))
	if err != nil {
		t.Fatal(err)
	}
	bound, err := bindLayout(lowered[0].Program.Layout)
	if err != nil {
		t.Fatal(err)
	}
	failures, _ := verifyGraph(lowered[0].Program, bound, nil, true)
	if len(failures) == 0 || failures[0].Code != "CSM1010" || !strings.Contains(failures[0].Detail, "helper body") {
		t.Fatalf("verification = %+v", failures)
	}
}

func stateBorrowPackage(invalidate bool) hir.Package {
	accesses := []hir.StateAccess{
		{Kind: hir.StateAccessCellWrite, Field: "source", Value: hir.StateValue{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("abcdef")}, Cost: 1},
		{Kind: hir.StateAccessCellBorrow, Field: "source", BorrowResult: 1, FoundResult: 1, Cost: 1},
		{Kind: hir.StateAccessBorrowView, BorrowInput: 1, BorrowResult: 2, ViewOffset: 1, ViewLength: 3, Cost: 1},
	}
	if invalidate {
		accesses = append(accesses, hir.StateAccess{Kind: hir.StateAccessCellWrite, Field: "source", Value: hir.StateValue{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("next")}, Cost: 1})
	}
	accesses = append(accesses,
		hir.StateAccess{Kind: hir.StateAccessBorrowCopy, BorrowInput: 2, Result: 2, Cost: 1},
		hir.StateAccess{Kind: hir.StateAccessCellWrite, Field: "target", ValueInput: 2, Cost: 1},
	)
	return borrowHIRPackage([]hir.StateField{
		{Name: "source", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
		{Name: "target", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
	}, []hir.StateBlock{{ID: 0, Accesses: accesses, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}})
}

func stateMapBorrowPackage() hir.Package {
	bytesValue := func(value string) hir.StateValue {
		return hir.StateValue{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte(value)}
	}
	return borrowHIRPackage([]hir.StateField{
		{Name: "values", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Bytes.TypeID(), ValueTypeID: contractstate.Bytes.TypeID()},
		{Name: "target", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
	}, []hir.StateBlock{{ID: 0, Accesses: []hir.StateAccess{
		{Kind: hir.StateAccessMapWrite, Field: "values", Key: bytesValue("a"), Value: bytesValue("alpha"), Cost: 1},
		{Kind: hir.StateAccessMapBorrow, Field: "values", Key: bytesValue("a"), BorrowResult: 1, FoundResult: 1, Cost: 1},
		{Kind: hir.StateAccessMapWrite, Field: "values", Key: bytesValue("b"), Value: bytesValue("beta"), Cost: 1},
		{Kind: hir.StateAccessBorrowCopy, BorrowInput: 1, Result: 2, Cost: 1},
		{Kind: hir.StateAccessCellWrite, Field: "target", ValueInput: 2, Cost: 1},
	}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}})
}

func stateDynamicAliasPackage() hir.Package {
	b := func(value string) hir.StateValue {
		return hir.StateValue{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte(value)}
	}
	return borrowHIRPackage([]hir.StateField{
		{Name: "key", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()},
		{Name: "values", Kind: hir.StateFieldMap, KeyTypeID: contractstate.Bytes.TypeID(), ValueTypeID: contractstate.Bytes.TypeID()},
	}, []hir.StateBlock{{ID: 0, Accesses: []hir.StateAccess{
		{Kind: hir.StateAccessCellWrite, Field: "key", Value: b("a"), Cost: 1},
		{Kind: hir.StateAccessCellRead, Field: "key", Result: 1, FoundResult: 2, Cost: 1},
		{Kind: hir.StateAccessMapBorrow, Field: "values", Key: b("a"), BorrowResult: 1, FoundResult: 3, Cost: 1},
		{Kind: hir.StateAccessMapWrite, Field: "values", KeyInput: 1, Value: b("next"), Cost: 1},
	}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}})
}

func stateBorrowEscapePackage() hir.Package {
	return borrowHIRPackage([]hir.StateField{{Name: "source", Kind: hir.StateFieldCell, ValueTypeID: contractstate.Bytes.TypeID()}}, []hir.StateBlock{
		{ID: 0, Accesses: []hir.StateAccess{
			{Kind: hir.StateAccessCellWrite, Field: "source", Value: hir.StateValue{TypeID: contractstate.Bytes.TypeID(), Encoded: []byte("x")}, Cost: 1},
			{Kind: hir.StateAccessCellBorrow, Field: "source", BorrowResult: 7, FoundResult: 1, Cost: 1},
		}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 1, Arguments: []hir.StateValueID{7}, Cost: 1}},
		{ID: 1, Parameters: []hir.StateBlockParameter{{ID: 8, TypeID: contractstate.Bytes.TypeID()}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}},
	})
}

func borrowHIRPackage(fields []hir.StateField, blocks []hir.StateBlock) hir.Package {
	return hir.Package{
		Modules:           []hir.Module{{Package: 1, Path: "borrow", Functions: []hir.Function{{Symbol: 1, Name: "apply", ResultType: hir.TypeVoid, Effects: []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}}}}},
		StateLayouts:      []hir.StateLayout{{Package: 1, ContractUID: "borrow.mir", Fields: fields}},
		StateTransactions: []hir.StateTransaction{{Package: 1, Function: 1, Entry: 0, Blocks: blocks}},
	}
}

func borrowMIRLimits() contractstate.Limits {
	return contractstate.Limits{MaxWrites: 8, MaxWriteBytes: 1024, MaxValueBytes: 64, MaxMapKeyBytes: 64}
}
