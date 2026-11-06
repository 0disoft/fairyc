package contractstatemir

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
)

func TestGraphHelperCurrentOwnerAndStableRebind(t *testing.T) {
	input := helperHIRPackage()
	input.StateTransactions[0].Accesses = nil
	input.StateTransactions[0].Exit = ""
	input.StateTransactions[0].Blocks = []hir.StateBlock{{ID: 0, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: 8, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}}
	coordinate := stateBodyCoordinate(t, "fypc1-sha256-c14ce14f76ca2cc42c89dfb70b90c251684bbb445d6c659c3956eeed097efd67")
	coordinates := map[hir.PackageID]packagegraph.CoordinateID{1: coordinate}
	artifacts, err := DescribeStableHIRStateBodies(input, []hir.PackageID{1}, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	body := artifacts[1]
	wire, err := EncodeStableStateBody(body, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeStableStateBody(wire, 1<<20)
	if err != nil {
		t.Fatal(err)
	}
	rebound, err := RebindStableStateBody(decoded, body.Hash, input, 1, coordinates)
	if err != nil {
		t.Fatal(err)
	}
	store, err := NewMemoryStore(rebound.Layout)
	if err != nil {
		t.Fatal(err)
	}
	out, err := Execute(store, rebound.Transactions[0].Program, borrowMIRLimits())
	if err != nil || out.Exit != ExitCommit || out.Cost != 2 || len(store.Dump()) != 1 {
		t.Fatalf("rebound graph helper %+v %v", out, err)
	}
	input.Modules[0].Functions[0].Body.Tail = nil
	if _, err := RebindStableStateBody(decoded, body.Hash, input, 1, coordinates); err == nil {
		t.Fatal("stale graph call projection rebound")
	}
}

func TestGraphHelperBorrowMutationBoundary(t *testing.T) {
	for _, trial := range []struct {
		name                  string
		copiedFirst, readOnly bool
		valid                 bool
	}{
		{"write-invalidates", false, false, false}, {"read-preserves", false, true, true}, {"copy-survives-write", true, false, true},
	} {
		t.Run(trial.name, func(t *testing.T) {
			input := helperHIRPackage()
			input.Modules[0].Functions[0].Effects = []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite}
			if trial.readOnly {
				input.Modules[0].Functions[1].Effects = []hir.Effect{hir.EffectStorageRead}
				input.Modules[0].Functions[0].Body.Tail = helperCallExpression(8, []hir.Effect{hir.EffectStorageRead})
				input.StateTransactions[1].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value"}}
			}
			operations := []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: mustEncodeU64(t, 9)}, Cost: 1},
				{Kind: hir.StateAccessCellBorrow, Field: "value", BorrowResult: 1, FoundResult: 1, Cost: 1},
			}
			copyOp := hir.StateAccess{Kind: hir.StateAccessBorrowCopy, BorrowInput: 1, Result: 2, Cost: 1}
			if trial.copiedFirst {
				operations = append(operations, copyOp)
			}
			operations = append(operations, hir.StateAccess{Kind: hir.StateAccessCall, Callee: 8, Cost: 1})
			if !trial.copiedFirst {
				operations = append(operations, copyOp)
			}
			operations = append(operations, hir.StateAccess{Kind: hir.StateAccessCellWrite, Field: "value", ValueInput: 2, Cost: 1})
			input.StateTransactions[0].Accesses = nil
			input.StateTransactions[0].Exit = ""
			input.StateTransactions[0].Blocks = []hir.StateBlock{{ID: 0, Accesses: operations, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}}}
			lowered, err := LowerHIRTransactions(input)
			if !trial.valid {
				if err == nil || !strings.Contains(err.Error(), "non-invalidated borrow") {
					t.Fatal("writing helper retained borrowed authority", err)
				}
				return
			}
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
			value, err := contractstate.U64.Decode(store.Dump()[0].Value)
			if err != nil || value != 9 {
				t.Fatal("copied value changed across helper", value, err)
			}
		})
	}
}
