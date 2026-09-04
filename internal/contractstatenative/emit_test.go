package contractstatenative

import (
	"strings"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/packagegraph"
	"github.com/0disoft/fairyc/internal/target"
)

func TestEmitBindsVerifiedCellProgramAndIdentity(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := Emit(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	if artifact.LayoutHash != handoff.program.Layout.Hash || artifact.BodyHash != handoff.bodyHash ||
		!strings.Contains(artifact.Source, "fycs_native_begin_v0") ||
		!strings.Contains(artifact.Source, "fycs_native_cell_write_v0") ||
		!strings.Contains(artifact.Source, "fycs_native_cell_read_v0") ||
		!strings.Contains(artifact.Source, "fycs_native_commit_v0") {
		t.Fatalf("artifact does not retain verified transaction facts: %+v", artifact)
	}
}

func TestEmitFailsClosedOutsideFirstNativeSlice(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		handoff  Handoff
		target   target.ID
		contains string
	}{
		{name: "target", handoff: handoff, target: target.X8664LinuxGNU, contains: "does not admit target"},
		{name: "body identity", target: target.X8664WindowsMSVC, contains: "prepared exact body handoff"},
		{name: "unverified", handoff: corruptNativeHandoff(handoff), target: target.X8664WindowsMSVC, contains: "requires verified MIR"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Emit(test.handoff, test.target); err == nil || !strings.Contains(err.Error(), test.contains) {
				t.Fatalf("Emit() error = %v, want containing %q", err, test.contains)
			}
		})
	}
}

func nativeStateInput(t *testing.T, exit contractstatemir.ExitKind) (hir.Package, packagegraph.CoordinateID) {
	t.Helper()
	trapCategory := ""
	transactionExit := hir.StateTransactionExit(exit)
	if exit == contractstatemir.ExitTrap {
		trapCategory = "overflow"
	}
	input := hir.Package{
		Modules: []hir.Module{{Package: 1, Path: "native_state", Functions: []hir.Function{{
			Symbol: 1, Name: "entry", ResultType: hir.TypeVoid,
			Effects: []hir.Effect{hir.EffectStorageRead, hir.EffectStorageWrite},
		}}}},
		StateLayouts: []hir.StateLayout{{Package: 1, ContractUID: "native.state", Fields: []hir.StateField{{
			Name: "value", Kind: hir.StateFieldCell, ValueTypeID: contractstate.U64.TypeID(),
		}}}},
		StateTransactions: []hir.StateTransaction{{
			Package: 1, Function: 1,
			Accesses: []hir.StateAccess{
				{Kind: hir.StateAccessCellWrite, Field: "value", Value: hir.StateValue{TypeID: contractstate.U64.TypeID(), Encoded: []byte{0, 0, 0, 0, 0, 0, 0, 42}}},
				{Kind: hir.StateAccessCellRead, Field: "value"},
			},
			Exit: transactionExit, TrapCategory: trapCategory,
		}},
	}
	coordinate, err := packagegraph.ParseCoordinateID("fypc1-sha256-c14ce14f76ca2cc42c89dfb70b90c251684bbb445d6c659c3956eeed097efd67")
	if err != nil {
		t.Fatal(err)
	}
	return input, coordinate
}

func corruptNativeHandoff(handoff Handoff) Handoff {
	handoff.program.Operations = append([]contractstatemir.Operation(nil), handoff.program.Operations...)
	handoff.program.Operations[0].Value.Encoded = []byte{42}
	return handoff
}
