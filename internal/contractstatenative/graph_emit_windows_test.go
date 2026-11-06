package contractstatenative

import (
	"bytes"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatefixture"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

//go:embed graph_execute_probe.c
var graphExecuteProbe string

func graphOracleLayout(t *testing.T, p contractstatemir.Program) contractstate.Layout {
	t.Helper()
	b, err := contractstate.NewLayoutBuilder(p.Layout.ContractUID)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range p.Layout.Fields {
		if f.Kind == contractstate.FieldCell {
			_, err = contractstate.DeclareEncodedCell(b, f.Name, f.ValueTypeID)
		} else {
			_, err = contractstate.DeclareEncodedMap(b, f.Name, f.KeyTypeID, f.ValueTypeID)
		}
		if err != nil {
			t.Fatal(err)
		}
	}
	l, err := b.Build()
	if err != nil || l.Hash() != p.Layout.Hash {
		t.Fatal("independent layout", err)
	}
	return l
}

func graphExpected(t *testing.T, h Handoff, a GraphArtifact, limits contractstate.Limits) (input, output []byte) {
	t.Helper()
	for _, n := range []uint64{uint64(limits.MaxWrites), limits.MaxWriteBytes, uint64(limits.MaxValueBytes), uint64(limits.MaxMapKeyBytes)} {
		input = binary.BigEndian.AppendUint64(input, n)
	}
	l := graphOracleLayout(t, h.program)
	store, err := contractstate.NewReferencePersistence(l, nil)
	if err != nil {
		t.Fatal(err)
	}
	port := &overlayOraclePort{PersistencePort: store}
	out, trap, err := persistentOracle(port, h.program, limits)
	status := byte(0)
	switch {
	case errors.Is(err, contractstate.ErrValueTooLarge):
		status = 3
	case errors.Is(err, contractstate.ErrWriteLimit):
		status = 4
	case errors.Is(err, contractstate.ErrWriteBytesLimit):
		status = 5
	case errors.Is(err, contractstate.ErrMapKeyTooLarge):
		status = 7
	case errors.Is(err, contractstate.ErrBorrowInvalidated):
		status = 9
	case err != nil:
		t.Fatal(err)
	}
	exit := byte(0)
	trapID := 0
	if trap != "" {
		exit = 2
		trapID = slices.Index(a.traps, trap) + 1
		// Replacing explicit trap exits by reject preserves the executed path,
		// bytes and cost while making Go's otherwise panicking trace observable.
		p := h.program
		p.Blocks = slices.Clone(p.Blocks)
		p.Helpers = slices.Clone(p.Helpers)
		if p.Exit == contractstatemir.ExitTrap {
			p.Exit = contractstatemir.ExitReject
			p.TrapCategory = ""
		}
		for i := range p.Blocks {
			if p.Blocks[i].Terminator.Kind == contractstatemir.TerminatorTrap {
				p.Blocks[i].Terminator.Kind = contractstatemir.TerminatorReject
				p.Blocks[i].Terminator.TrapCategory = ""
			}
		}
		for i := range p.Helpers {
			if p.Helpers[i].Exit == contractstatemir.ExitTrap {
				p.Helpers[i].Exit = contractstatemir.ExitReject
				p.Helpers[i].TrapCategory = ""
			}
		}
		fresh, _ := contractstate.NewReferencePersistence(l, nil)
		var category string
		out, category, err = persistentOracle(fresh, p, limits)
		if err != nil || category != "" {
			t.Fatal("independent trap control", category, err)
		}
	} else if out.Exit == contractstatemir.ExitReject {
		exit = 1
	}
	output = append(output, status, exit)
	output = binary.BigEndian.AppendUint32(output, uint32(trapID))
	output = binary.BigEndian.AppendUint64(output, out.Cost)
	var reads []byte
	for _, read := range out.Reads {
		reads = binary.BigEndian.AppendUint64(reads, uint64(read.Operation))
		codec, _ := overlayCodec(read.Value.TypeID)
		reads = append(reads, codec, 0)
		if read.Found {
			reads[len(reads)-1] = 1
		}
		reads = binary.BigEndian.AppendUint32(reads, uint32(len(read.Value.Encoded)))
		reads = append(reads, read.Value.Encoded...)
	}
	output = binary.BigEndian.AppendUint32(output, uint32(len(reads)))
	output = binary.BigEndian.AppendUint16(output, uint16(len(port.writes)))
	output = binary.BigEndian.AppendUint64(output, out.Receipt.WriteBytes)
	output = append(output, reads...)
	for _, w := range port.writes {
		output = append(output, w.Key[:]...)
		output = append(output, 0)
		if w.Deleted {
			output[len(output)-1] = 1
		}
		output = binary.BigEndian.AppendUint32(output, uint32(len(w.Value)))
		output = append(output, w.Value...)
	}
	return
}

func TestNativeStateGraphEmission(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateGraphRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	_, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	for _, version := range []int{0, 1, 2, 3, 4} {
		path := fmt.Sprintf("../../spec/fixtures/contract-state/manifest-v%d.json", version)
		cases, err := contractstatefixture.LoadHIRCases(path)
		if err != nil {
			t.Fatal(err)
		}
		if version == 4 {
			cases = append(cases, dynamicGraphCases(t)...)
		}
		// Retain the authored fixed expectations as an independent prerequisite.
		report, err := contractstatefixture.Run(path)
		if err != nil || len(report.Failures) != 0 {
			t.Fatal("original fixture oracle", report, err)
		}
		for _, trial := range cases {
			t.Run(fmt.Sprintf("v%d-%s", version, trial.Name), func(t *testing.T) {
				h, err := Prepare(trial.Input, 1, 1, coordinate)
				if err != nil {
					t.Fatal(err)
				}
				a, err := EmitGraph(h, target.X8664WindowsMSVC)
				if err != nil {
					t.Fatal(err)
				}
				repeat, err := EmitGraph(h, target.X8664WindowsMSVC)
				if err != nil || repeat.Source() != a.Source() || repeat.ProgramHash() != a.ProgramHash() {
					t.Fatal("unstable graph identity", err)
				}
				input, want := graphExpected(t, h, a, trial.Limits)
				for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
					t.Run(profile.Name, func(t *testing.T) {
						image, err := cdriver.BuildWindowsMSVCTarget(toolchain, profile, a.Source()+graphExecuteProbe, runtimeSource)
						if err != nil {
							t.Fatal(err)
						}
						out, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
						got := []byte(out.Stdout)
						if err != nil || out.ExitCode != 0 || out.Stderr != "" || len(got) < 28 {
							t.Fatal("native graph process", out.ExitCode, err, out.Stderr)
						}
						if want[0] != 0 {
							if got[0] != want[0] || got[1] != 0 || binary.BigEndian.Uint16(got[18:20]) != 0 || binary.BigEndian.Uint64(got[20:28]) != 0 {
								t.Fatalf("failed graph published: %x", got)
							}
						} else if !bytes.Equal(got, want) {
							t.Fatalf("native %x; Go %x", got, want)
						}
					})
				}
			})
		}
	}
}

func TestNativeStateGraphAdmission(t *testing.T) {
	h, _ := overlayMIRFixture(t, contractstatemir.ExitCommit, false)
	for _, name := range []string{"zero", "target", "bad-MIR"} {
		t.Run(name, func(t *testing.T) {
			owner, id := h, target.X8664WindowsMSVC
			if name == "zero" {
				owner = Handoff{}
			} else if name == "target" {
				id = target.X8664LinuxGNU
			} else {
				owner = corruptNativeHandoff(h)
			}
			if _, err := EmitGraph(owner, id); err == nil {
				t.Fatal("unverified graph admitted")
			}
		})
	}
}

func TestNativeStateGraphSlotBoundary(t *testing.T) {
	for _, count := range []int{4096, 4097} {
		input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
		input.StateLayouts[0].Fields[0].ValueTypeID = contractstate.Bool.TypeID()
		parameters := make([]hir.StateBlockParameter, count-2)
		arguments := make([]hir.StateValueID, len(parameters))
		for index := range parameters {
			parameters[index] = hir.StateBlockParameter{ID: hir.StateValueID(index + 3), TypeID: contractstate.Bool.TypeID()}
			arguments[index] = 1
		}
		input.StateTransactions[0] = hir.StateTransaction{Package: 1, Function: 1, Entry: 0, Blocks: []hir.StateBlock{
			{ID: 0, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "value", Result: 1, FoundResult: 2, Cost: 1}}, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorJump, Target: 1, Arguments: arguments, Cost: 1}},
			{ID: 1, Parameters: parameters, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}},
		}}
		h, err := Prepare(input, 1, 1, coordinate)
		if err != nil {
			t.Fatal("valid graph slot control", count, err)
		}
		_, err = EmitGraph(h, target.X8664WindowsMSVC)
		if (count == 4096) != (err == nil) {
			t.Fatal("wrong native slot boundary", count, err)
		}
	}
}

func TestNativeStateGraphHashBindsTrap(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitTrap)
	other := input.Modules[0].Functions[0]
	other.Symbol, other.Name = 2, "other"
	input.Modules[0].Functions = append(input.Modules[0].Functions, other)
	tx := input.StateTransactions[0]
	tx.Function, tx.TrapCategory = 2, "assertion-failed"
	input.StateTransactions = append(input.StateTransactions, tx)
	a, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Prepare(input, 1, 2, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	first, err := EmitGraph(a, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	second, err := EmitGraph(b, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	if first.BodyHash() != second.BodyHash() || first.Source() != second.Source() || first.ProgramHash() == second.ProgramHash() {
		t.Fatal("same C confused distinct trap identities")
	}
}
