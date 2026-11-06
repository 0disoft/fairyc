package cli

import (
	"bytes"
	"errors"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/contractstatenative"
	"github.com/0disoft/fairyc/internal/hir"
)

type graphNativeSemanticPort struct {
	*contractstate.ReferencePersistence
	writes    []contractstate.PersistentWrite
	publishes int
}

func (p *graphNativeSemanticPort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	p.publishes++
	for _, w := range writes {
		w.Value = bytes.Clone(w.Value)
		p.writes = append(p.writes, w)
	}
	return p.ReferencePersistence.CompareAndPublish(hash, version, writes)
}

// Fresh empty snapshots and limits match the audit driver. No Handoff or C
// emitter participates in this oracle; lower the original HIR independently.
func graphNativeSemanticExpect(t *testing.T, input hir.Package, got contractstatenative.GraphResult) {
	t.Helper()
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	expect := func(ok bool, format string, args ...any) {
		t.Helper()
		if !ok {
			t.Fatalf(format, args...)
		}
	}
	lowered, err := contractstatemir.LowerHIRTransactions(input)
	expect(err == nil && len(lowered) == 1, "semantic lowering: count=%d err=%v", len(lowered), err)
	program := lowered[0].Program
	builder, err := contractstate.NewLayoutBuilder(program.Layout.ContractUID)
	check(err)
	for _, f := range program.Layout.Fields {
		switch f.Kind {
		case contractstate.FieldCell:
			_, err = contractstate.DeclareEncodedCell(builder, f.Name, f.ValueTypeID)
		case contractstate.FieldMap:
			_, err = contractstate.DeclareEncodedMap(builder, f.Name, f.KeyTypeID, f.ValueTypeID)
		default:
			t.Fatalf("unknown semantic field kind %v", f.Kind)
		}
		check(err)
	}
	layout, err := builder.Build()
	expect(err == nil && layout.Hash() == program.Layout.Hash, "independent semantic layout: %v", err)
	store, err := contractstate.NewReferencePersistence(layout, nil)
	check(err)
	port := &graphNativeSemanticPort{ReferencePersistence: store}
	var want contractstatemir.Outcome
	var trapped string
	func() {
		defer func() {
			if r := recover(); r != nil {
				trapped = r.(contractstatemir.Trap).Category
			}
		}()
		want, err = contractstatemir.ExecutePersistent(port, program, graphNativeLimits())
	}()
	if trapped != "" {
		// A trap unwinds Outcome. Change only cloned trap exits to reject so
		// the same path retains read/cost evidence without publishing writes.
		reject := func(blocks []contractstatemir.Block) []contractstatemir.Block {
			blocks = slices.Clone(blocks)
			for i := range blocks {
				if blocks[i].Terminator.Kind == contractstatemir.TerminatorTrap {
					blocks[i].Terminator.Kind = contractstatemir.TerminatorReject
					blocks[i].Terminator.TrapCategory = ""
				}
			}
			return blocks
		}
		program.Blocks = reject(program.Blocks)
		program.Helpers = slices.Clone(program.Helpers)
		for i := range program.Helpers {
			program.Helpers[i].Blocks = reject(program.Helpers[i].Blocks)
		}
		want, err = contractstatemir.ExecutePersistent(port, program, graphNativeLimits())
		if err == nil {
			expect(want.Exit == contractstatemir.ExitReject, "trap replay exit=%s", want.Exit)
			want.Exit = contractstatemir.ExitTrap
		}
	}
	expect(got.Version == 0, "audit request version=%d, want fresh version 0", got.Version)
	if err != nil {
		expect(got.Status != 0 && got.Exit == "" && got.TrapCategory == "" && len(got.Writes) == 0 && got.WriteBytes == 0 && port.publishes == 0, "runtime error published or succeeded: err=%v got=%+v", err, got)
		// Compare only exported sentinel mappings; range/codec errors have no
		// stable typed Go identity and must still fail without publication.
		for _, code := range []struct {
			err    error
			status byte
		}{{contractstate.ErrValueTooLarge, 3}, {contractstate.ErrWriteLimit, 4}, {contractstate.ErrWriteBytesLimit, 5}, {contractstate.ErrMapKeyTooLarge, 7}, {contractstate.ErrBorrowInvalidated, 9}} {
			if errors.Is(err, code.err) && got.Status != code.status {
				t.Fatalf("runtime status=%d want=%d: %v", got.Status, code.status, err)
			}
		}
	} else {
		trap := trapped
		expect(got.Status == 0 && got.Exit == want.Exit && got.TrapCategory == trap && got.Cost == want.Cost && len(got.Reads) == len(want.Reads), "semantic outcome: got=%+v want=%+v trap=%q", got, want, trap)
		for i, r := range want.Reads {
			g := got.Reads[i]
			expect(g.Operation == r.Operation && g.Found == r.Found && g.Value.TypeID == r.Value.TypeID && bytes.Equal(g.Value.Encoded, r.Value.Encoded), "semantic read %d: got=%+v want=%+v", i, g, r)
		}
		if got.WriteBytes != want.Receipt.WriteBytes || len(got.Writes) != want.Receipt.Writes || !slices.EqualFunc(got.Writes, port.writes, func(a, b contractstate.PersistentWrite) bool {
			return a.Key == b.Key && a.Deleted == b.Deleted && bytes.Equal(a.Value, b.Value)
		}) {
			t.Fatalf("semantic writes: got=%+v bytes=%d want=%+v receipt=%+v", got.Writes, got.WriteBytes, port.writes, want.Receipt)
		}
		expect(want.Exit == contractstatemir.ExitCommit || port.publishes == 0, "non-commit semantic path published")
	}
	actual, errSnapshot := contractstate.NewReferencePersistence(layout, nil)
	check(errSnapshot)
	if got.Status == 0 && got.Exit == contractstatemir.ExitCommit {
		_, errSnapshot = actual.CompareAndPublish(layout.Hash(), got.Version, got.Writes)
		check(errSnapshot)
	}
	wantState, errSnapshot := store.Snapshot(layout.Hash())
	check(errSnapshot)
	gotState, errSnapshot := actual.Snapshot(layout.Hash())
	check(errSnapshot)
	if gotState.Version != wantState.Version || !slices.EqualFunc(gotState.Entries, wantState.Entries, func(a, b contractstate.Entry) bool { return a.Key == b.Key && bytes.Equal(a.Value, b.Value) }) {
		t.Fatalf("semantic final snapshot: got=%+v want=%+v", gotState, wantState)
	}
}
