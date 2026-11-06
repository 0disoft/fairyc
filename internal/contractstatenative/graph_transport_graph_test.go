package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func graphBranchFixture(t *testing.T) (GraphTransport, contractstate.PersistentSnapshot, []byte, contractstate.Key) {
	t.Helper()
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateLayouts[0].Fields = []hir.StateField{
		{Name: "seed", Kind: hir.StateFieldCell, ValueTypeID: "u64/be/v1"},
		{Name: "items", Kind: hir.StateFieldMap, KeyTypeID: "u64/be/v1", ValueTypeID: "u64/be/v1"},
	}
	source.StateTransactions[0] = hir.StateTransaction{Package: 1, Function: 1, Blocks: []hir.StateBlock{
		{ID: 0, Accesses: []hir.StateAccess{{Kind: hir.StateAccessCellRead, Field: "seed", Result: 1, FoundResult: 2, Cost: 1}},
			Terminator: hir.StateTerminator{Kind: hir.StateTerminatorBranch, Condition: 2, TrueTarget: 1, TrueArguments: []hir.StateValueID{1}, FalseTarget: 2, Cost: 1}},
		{ID: 1, Parameters: []hir.StateBlockParameter{{ID: 3, TypeID: "u64/be/v1"}}, Accesses: []hir.StateAccess{{Kind: hir.StateAccessMapWrite, Field: "items", KeyInput: 3, ValueInput: 3, Cost: 1}},
			Terminator: hir.StateTerminator{Kind: hir.StateTerminatorCommit, Cost: 1}},
		{ID: 2, Terminator: hir.StateTerminator{Kind: hir.StateTerminatorReject, Cost: 1}},
	}}
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte{0, 0, 0, 0, 0, 0, 0, 7}
	snapshot := contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Version: 19,
		Entries: []contractstate.Entry{{Key: h.program.Layout.Fields[0].Prefix, Value: value}}}
	// Independent physical map address, without the production address helper.
	payload := append([]byte("fairyc-contract-state-map-entry/v1"), h.program.Layout.Fields[1].Prefix[:]...)
	payload = binary.BigEndian.AppendUint64(payload, 8)
	key := sha256.Sum256(append(payload, value...))
	trace := graphWireEvent(nil, 1, 1, nil, value, 1)
	trace = graphWireEvent(trace, 1, 0x80000001, nil, nil, 0)
	trace = graphWireEvent(trace, 2, 2, value, value, 1)
	trace = graphWireEvent(trace, 2, 0x80000002, nil, nil, 0)
	return p, snapshot, trace, key
}

func TestGraphTransportDynamicBranch(t *testing.T) {
	p, snapshot, trace, key := graphBranchFixture(t)
	limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 8, MaxMapKeyBytes: 8}
	in, err := p.Input(snapshot, limits)
	if err != nil {
		t.Fatal(err)
	}
	writes := []contractstate.PersistentWrite{{Key: key, Value: snapshot.Entries[0].Value}}
	frame := graphWireResponse(in, 0, 0, 0, 4, 2, 4, trace, writes)
	out, err := p.Decode(in, frame)
	if err != nil || out.Cost != 4 || out.Operations != 2 || out.Reads[0].Operation != 0 || out.Writes[0].Key != key {
		t.Fatal("dynamic map/branch witness", out, err)
	}
	// Repeated static sites are not an executable path, even if the header's
	// operation count and cost have been updated to agree with the forged event.
	duplicate := graphWireEvent(slices.Clone(trace[:29]), 2, 1, nil, snapshot.Entries[0].Value, 1)
	if _, err := p.Decode(in, graphWireResponse(in, 0, 0, 0, 2, 2, 2, duplicate, nil)); err == nil {
		t.Fatal("duplicated/reordered read site accepted")
	}
	for _, policy := range []contractstate.Limits{
		{MaxWrites: 1, MaxWriteBytes: 40, MaxValueBytes: 8, MaxMapKeyBytes: 8},
		{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 7, MaxMapKeyBytes: 8},
		{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 8, MaxMapKeyBytes: 7},
	} {
		request, err := p.Input(snapshot, policy)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.Decode(request, graphWireResponse(request, 0, 0, 0, 4, 2, 4, trace, writes)); err == nil {
			t.Fatal("successful trace bypassed write/value/original-map-key policy", policy)
		}
	}
	changed := snapshot
	changed.Entries = slices.Clone(snapshot.Entries)
	changed.Entries[0].Value = []byte{0, 0, 0, 0, 0, 0, 0, 8}
	other, err := p.Input(changed, limits)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Decode(other, frame); err == nil {
		t.Fatal("different same-version snapshot accepted the old response")
	}
	if _, err := p.Decode(other, graphWireResponse(other, 0, 0, 0, 4, 2, 4, trace, writes)); err == nil {
		t.Fatal("read differing from request snapshot accepted")
	}
	missing := contractstate.PersistentSnapshot{LayoutHash: snapshot.LayoutHash, Version: snapshot.Version}
	request, err := p.Input(missing, limits)
	if err != nil {
		t.Fatal(err)
	}
	falseTrace := graphWireEvent(nil, 1, 1, nil, nil, 0)
	falseTrace = graphWireEvent(falseTrace, 1, 0x80000001, nil, nil, 0)
	falseTrace = graphWireEvent(falseTrace, 1, 0x80000003, nil, nil, 0)
	if out, err := p.Decode(request, graphWireResponse(request, 0, 1, 0, 3, 1, 3, falseTrace, nil)); err != nil || out.Exit != contractstatemir.ExitReject {
		t.Fatal("alternate graph branch", out, err)
	}
}

func TestGraphTransportOverlayReplacement(t *testing.T) {
	// Author a write/delete/read sequence over the same address. The emitted
	// body, rather than the response, declares the exact delete site.
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateTransactions[0].Accesses = append(source.StateTransactions[0].Accesses,
		hir.StateAccess{Kind: hir.StateAccessCellDelete, Field: "value"}, hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "value"})
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 8, MaxMapKeyBytes: 8}
	in, err := p.Input(contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash}, limits)
	if err != nil {
		t.Fatal(err)
	}
	value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
	trace := graphWireEvent(nil, 1, 1, nil, value, 1)
	trace = graphWireEvent(trace, 2, 2, nil, value, 1)
	trace = graphWireEvent(trace, 3, 3, nil, nil, 0)
	trace = graphWireEvent(trace, 4, 4, nil, nil, 0)
	trace = graphWireEvent(trace, 4, 0x80000001, nil, nil, 0)
	writes := []contractstate.PersistentWrite{{Key: h.program.Layout.Fields[0].Prefix, Deleted: true}}
	out, err := p.Decode(in, graphWireResponse(in, 0, 0, 0, 0, 4, 5, trace, writes))
	if err != nil || out.WriteBytes != 33 || len(out.Reads) != 2 || out.Reads[1].Found || !out.Writes[0].Deleted {
		t.Fatal("same-key replacement/delete accounting", out, err)
	}
	limits.MaxWriteBytes = 33
	in, err = p.Input(contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash}, limits)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := p.Decode(in, graphWireResponse(in, 0, 0, 0, 0, 4, 5, trace, writes)); err == nil {
		t.Fatal("final delete hid an intermediate oversized write")
	}
}

func TestGraphTransportResponseCeiling(t *testing.T) {
	source, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	source.StateLayouts[0].Fields = []hir.StateField{
		{Name: "large", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
		{Name: "small", Kind: hir.StateFieldCell, ValueTypeID: "bytes/v1"},
	}
	source.StateTransactions[0].Accesses = []hir.StateAccess{
		{Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "large"},
		{Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "large"}, {Kind: hir.StateAccessCellRead, Field: "small"},
	}
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	for _, delta := range []int{-1, 0, 1} {
		large, small := (16<<20)-78, 0
		if delta == -1 {
			large, small = large-1, 3
		} else if delta == 1 {
			small = 1
		}
		snapshot := contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Entries: []contractstate.Entry{
			{Key: h.program.Layout.Fields[0].Prefix, Value: bytes.Repeat([]byte{255}, large)},
			{Key: h.program.Layout.Fields[1].Prefix, Value: bytes.Repeat([]byte{0}, small)},
		}}
		slices.SortFunc(snapshot.Entries, func(a, b contractstate.Entry) int { return bytes.Compare(a.Key[:], b.Key[:]) })
		in, err := p.Input(snapshot, overlayPersistentLimits())
		if err != nil {
			t.Fatal(err)
		}
		var trace []byte
		for i := 1; i <= 5; i++ {
			key := h.program.Layout.Fields[0].Prefix
			if i == 5 {
				key = h.program.Layout.Fields[1].Prefix
			}
			for _, e := range snapshot.Entries {
				if e.Key == key {
					trace = graphWireEvent(trace, uint64(i), uint32(i), nil, e.Value, 1)
				}
			}
		}
		trace = graphWireEvent(trace, 5, 0x80000001, nil, nil, 0)
		frame := graphWireResponse(in, 0, 0, 0, 0, 5, 6, trace, nil)
		if len(frame) != (64<<20)+delta {
			t.Fatal("ceiling fixture has wrong size", len(frame), delta)
		}
		out, err := p.Decode(in, frame)
		if delta <= 0 && (err != nil || len(out.Reads) != 5) || delta > 0 && err == nil {
			t.Fatal("response ceiling", delta, err)
		}
	}
}
