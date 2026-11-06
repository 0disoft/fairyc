package contractstatenative

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func overlayMIRFixture(t *testing.T, exit contractstatemir.ExitKind, empty bool) (Handoff, contractstate.Layout) {
	t.Helper()
	input, coordinate := nativeStateInput(t, exit)
	input.StateLayouts[0].Fields = nil
	input.StateTransactions[0].Accesses = nil
	builder, err := contractstate.NewLayoutBuilder(input.StateLayouts[0].ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	values := []struct {
		id    string
		value []byte
	}{{contractstate.Bool.TypeID(), []byte{1}}, {contractstate.U32.TypeID(), []byte{0, 255, 1, 2}}, {contractstate.U64.TypeID(), bytes.Repeat([]byte{255}, 8)}, {contractstate.Bytes.TypeID(), nil}, {contractstate.Text.TypeID(), []byte("새값")}, {contractstate.Byte20.TypeID(), bytes.Repeat([]byte{128}, 20)}, {contractstate.Byte32.TypeID(), bytes.Repeat([]byte{204}, 32)}}
	for i, value := range values {
		name := fmt.Sprintf("cell%d", i)
		mapName := fmt.Sprintf("map%d", i)
		input.StateLayouts[0].Fields = append(input.StateLayouts[0].Fields, hir.StateField{Name: name, Kind: hir.StateFieldCell, ValueTypeID: value.id}, hir.StateField{Name: mapName, Kind: hir.StateFieldMap, KeyTypeID: value.id, ValueTypeID: value.id})
		if _, err = contractstate.DeclareEncodedCell(builder, name, value.id); err != nil {
			t.Fatal(err)
		}
		if _, err = contractstate.DeclareEncodedMap(builder, mapName, value.id, value.id); err != nil {
			t.Fatal(err)
		}
		encoded := hir.StateValue{TypeID: value.id, Encoded: value.value}
		input.StateTransactions[0].Accesses = append(input.StateTransactions[0].Accesses,
			hir.StateAccess{Kind: hir.StateAccessCellRead, Field: name}, hir.StateAccess{Kind: hir.StateAccessCellWrite, Field: name, Value: encoded}, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: name},
			hir.StateAccess{Kind: hir.StateAccessMapRead, Field: mapName, Key: encoded}, hir.StateAccess{Kind: hir.StateAccessMapWrite, Field: mapName, Key: encoded, Value: encoded}, hir.StateAccess{Kind: hir.StateAccessMapRead, Field: mapName, Key: encoded}, hir.StateAccess{Kind: hir.StateAccessMapDelete, Field: mapName, Key: encoded}, hir.StateAccess{Kind: hir.StateAccessMapRead, Field: mapName, Key: encoded})
	}
	if empty {
		input.StateTransactions[0].Accesses = nil
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	handoff, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	return handoff, layout
}

func TestOverlayMIREmission(t *testing.T) {
	handoff, _ := overlayMIRFixture(t, contractstatemir.ExitCommit, false)
	artifact, err := EmitOverlay(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := EmitOverlay(handoff, target.X8664WindowsMSVC)
	if err != nil || !artifact.valid || artifact.Source() != repeat.Source() || artifact.ProgramHash() != repeat.ProgramHash() || artifact.LayoutHash() != handoff.program.Layout.Hash || artifact.BodyHash() != handoff.bodyHash {
		t.Fatal("lost sealed identity or deterministic code", err)
	}
	for _, name := range []string{"target", "zero", "invalid-value", "helper-call"} {
		t.Run(name, func(t *testing.T) {
			bad := handoff
			id := target.X8664WindowsMSVC
			switch name {
			case "target":
				id = target.X8664LinuxGNU
			case "zero":
				bad = Handoff{}
			case "invalid-value":
				bad = corruptNativeHandoff(handoff)
			case "helper-call":
				bad.program.Operations = append([]contractstatemir.Operation(nil), handoff.program.Operations...)
				bad.program.Operations[0].Kind = contractstatemir.OperationCall
			}
			if _, err := EmitOverlay(bad, id); err == nil {
				t.Fatal("unsupported MIR admitted")
			}
		})
	}
}

func TestOverlayMIRHashBindsTrapCategory(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitTrap)
	other := input.Modules[0].Functions[0]
	other.Symbol, other.Name = 2, "other"
	input.Modules[0].Functions = append(input.Modules[0].Functions, other)
	transaction := input.StateTransactions[0]
	transaction.Function, transaction.TrapCategory = 2, "assertion-failed"
	input.StateTransactions = append(input.StateTransactions, transaction)
	first, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Prepare(input, 1, 2, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	a, err := EmitOverlay(first, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EmitOverlay(second, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	if a.BodyHash() != b.BodyHash() || a.Source() != b.Source() || a.ProgramHash() == b.ProgramHash() {
		t.Fatal("same-package source-equivalent entries confused different trap categories")
	}
}
