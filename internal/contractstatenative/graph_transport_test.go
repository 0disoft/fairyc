package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

// Authored wire writer: explicit offsets/site IDs and bytes, independent of the
// production plan, path walker, decoder and future C framing implementation.
func graphWireEvent(out []byte, operation uint64, site uint32, key, value []byte, present byte) []byte {
	out = binary.BigEndian.AppendUint64(out, operation)
	out = binary.BigEndian.AppendUint32(out, site)
	out = binary.BigEndian.AppendUint32(out, uint32(len(key)))
	out = binary.BigEndian.AppendUint32(out, uint32(len(value)))
	out = append(out, present)
	out = append(out, key...)
	return append(out, value...)
}

func graphWireResponse(input GraphInput, status, exit byte, trap uint32, cost, ops uint64, events uint32, trace []byte, writes []contractstate.PersistentWrite) []byte {
	request := input.Bytes()
	out := make([]byte, 186)
	copy(out, "FYGROUT1")
	copy(out[8:112], request[8:112])
	digest := sha256.Sum256(request)
	copy(out[112:144], digest[:])
	out[144], out[145] = status, exit
	binary.BigEndian.PutUint32(out[146:], trap)
	binary.BigEndian.PutUint64(out[150:], cost)
	binary.BigEndian.PutUint64(out[158:], ops)
	binary.BigEndian.PutUint32(out[166:], events)
	binary.BigEndian.PutUint32(out[170:], uint32(len(writes)))
	out = append(out, trace...)
	var size uint64
	for _, w := range writes {
		out = append(out, w.Key[:]...)
		out = append(out, 0)
		if w.Deleted {
			out[len(out)-1] = 1
		}
		out = binary.BigEndian.AppendUint32(out, uint32(len(w.Value)))
		out = append(out, w.Value...)
		size += 33 + uint64(len(w.Value))
	}
	binary.BigEndian.PutUint64(out[174:], size)
	binary.BigEndian.PutUint32(out[182:], uint32(len(out)-186))
	return out
}

func graphWireFixture(t *testing.T, exit contractstatemir.ExitKind, helper bool) (Handoff, GraphTransport, GraphInput, []byte) {
	t.Helper()
	source, coordinate := nativeStateInput(t, exit)
	if helper {
		f := source.Modules[0].Functions[0]
		f.Symbol, f.Name = 2, "helper"
		source.Modules[0].Functions = append(source.Modules[0].Functions, f)
		source.Modules[0].Functions[0].Body.Tail = &hir.Expression{Kind: hir.ExpressionCall, Type: hir.TypeVoid, Symbol: 2,
			Effects: f.Effects, Children: []hir.Expression{{Kind: hir.ExpressionPath, Symbol: 2}}}
		body := source.StateTransactions[0]
		body.Function, body.Exit, body.TrapCategory = 2, hir.StateTransactionReturn, ""
		source.StateTransactions = append(source.StateTransactions, body)
		source.StateTransactions[0].Accesses = []hir.StateAccess{{Kind: hir.StateAccessCall, Callee: 2, Cost: 1}}
	}
	h, err := Prepare(source, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	p, err := PrepareGraphTransport(h, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	in, err := p.Input(contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Version: math.MaxUint64}, overlayPersistentLimits())
	if err != nil {
		t.Fatal(err)
	}
	value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
	var trace []byte
	cost, ops := uint64(0), uint64(2)
	events := uint32(3)
	if helper {
		trace = graphWireEvent(trace, 1, 1, nil, nil, 0)
		trace = graphWireEvent(trace, 2, 2, nil, value, 1)
		trace = graphWireEvent(trace, 3, 3, nil, value, 1)
		trace = graphWireEvent(trace, 3, 0x80000002, nil, nil, 0)
		trace = graphWireEvent(trace, 3, 0x80000001, nil, nil, 0)
		cost, ops, events = 1, 3, 5
	} else {
		trace = graphWireEvent(trace, 1, 1, nil, value, 1)
		trace = graphWireEvent(trace, 2, 2, nil, value, 1)
		trace = graphWireEvent(trace, 2, 0x80000001, nil, nil, 0)
	}
	var writes []contractstate.PersistentWrite
	var wireExit byte
	var trap uint32
	switch exit {
	case contractstatemir.ExitCommit:
		writes = []contractstate.PersistentWrite{{Key: h.program.Layout.Fields[0].Prefix, Value: value}}
	case contractstatemir.ExitReject:
		wireExit = 1
	case contractstatemir.ExitTrap:
		wireExit, trap = 2, 1
	}
	return h, p, in, graphWireResponse(in, 0, wireExit, trap, cost, ops, events, trace, writes)
}

func TestGraphTransportLegacyAndHelper(t *testing.T) {
	for _, helper := range []bool{false, true} {
		for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap} {
			t.Run(string(exit)+map[bool]string{false: "/direct", true: "/helper"}[helper], func(t *testing.T) {
				h, p, in, frame := graphWireFixture(t, exit, helper)
				out, err := p.Decode(in, frame)
				if err != nil || out.Exit != exit || out.Version != math.MaxUint64 || len(out.Reads) != 1 || out.Reads[0].Value.Encoded[7] != 42 {
					t.Fatal("native evidence not admitted", out, err)
				}
				if helper && (out.Cost != 1 || out.Reads[0].Operation != 2) || !helper && (out.Cost != 0 || out.Reads[0].Operation != 1) {
					t.Fatal("helper operation/cost order lost", out)
				}
				if exit == contractstatemir.ExitCommit && (len(out.Writes) != 1 || out.Writes[0].Key != h.program.Layout.Fields[0].Prefix || out.WriteBytes != 41) {
					t.Fatal("final overlay not retained", out)
				}
				if exit == contractstatemir.ExitTrap && out.TrapCategory != "overflow" {
					t.Fatal("trap table not bound", out)
				}
				clear(frame)
				if out.Reads[0].Value.Encoded[7] != 42 || len(out.Writes) != 0 && out.Writes[0].Value[7] != 42 {
					t.Fatal("returned evidence aliases received bytes")
				}
			})
		}
	}
}

func TestGraphTransportCorruption(t *testing.T) {
	_, p, in, original := graphWireFixture(t, contractstatemir.ExitCommit, false)
	mutations := map[string]func([]byte) []byte{
		"magic":               func(b []byte) []byte { b[0] ^= 1; return b },
		"layout":              func(b []byte) []byte { b[8] ^= 1; return b },
		"body":                func(b []byte) []byte { b[40] ^= 1; return b },
		"program":             func(b []byte) []byte { b[72] ^= 1; return b },
		"version":             func(b []byte) []byte { b[104] ^= 1; return b },
		"request":             func(b []byte) []byte { b[112] ^= 1; return b },
		"status":              func(b []byte) []byte { b[144] = 12; return b },
		"exit":                func(b []byte) []byte { b[145] = 3; return b },
		"trap":                func(b []byte) []byte { b[149] = 1; return b },
		"cost":                func(b []byte) []byte { b[157] = 1; return b },
		"operations":          func(b []byte) []byte { b[165] = 3; return b },
		"events":              func(b []byte) []byte { b[169] = 4; return b },
		"huge-count":          func(b []byte) []byte { binary.BigEndian.PutUint32(b[166:], math.MaxUint32); return b },
		"writes":              func(b []byte) []byte { b[173] = 2; return b },
		"write-cost":          func(b []byte) []byte { b[181] = 42; return b },
		"payload":             func(b []byte) []byte { b[185] ^= 1; return b },
		"operation-order":     func(b []byte) []byte { b[193] = 2; return b },
		"undeclared-site":     func(b []byte) []byte { b[197] = 9; return b },
		"skip-write":          func(b []byte) []byte { b[197] = 2; return b },
		"huge-key":            func(b []byte) []byte { binary.BigEndian.PutUint32(b[198:], math.MaxUint32); return b },
		"huge-value":          func(b []byte) []byte { binary.BigEndian.PutUint32(b[202:], math.MaxUint32); return b },
		"flag":                func(b []byte) []byte { b[206] = 2; return b },
		"literal-write":       func(b []byte) []byte { b[214] = 43; return b },
		"read":                func(b []byte) []byte { b[243] = 43; return b },
		"wrong-read-presence": func(b []byte) []byte { b[235] = 0; return b },
		"final-key":           func(b []byte) []byte { b[265] ^= 1; return b },
		"final-delete":        func(b []byte) []byte { b[297] = 1; return b },
		"final-value":         func(b []byte) []byte { b[len(b)-1] = 43; return b },
		"short":               func(b []byte) []byte { return b[:len(b)-1] },
		"trailing":            func(b []byte) []byte { return append(b, 0) },
	}
	for name, mutate := range mutations {
		t.Run(name, func(t *testing.T) {
			if out, err := p.Decode(in, mutate(slices.Clone(original))); err == nil || len(out.Writes) != 0 || len(out.Reads) != 0 {
				t.Fatal("corrupt evidence admitted", out, err)
			}
		})
	}
	for length := 0; length < len(original); length++ {
		if _, err := p.Decode(in, original[:length]); err == nil {
			t.Fatalf("short response %d accepted", length)
		}
	}
	if _, err := (GraphTransport{}).Decode(in, original); err == nil {
		t.Fatal("zero owner accepted")
	}
	if _, err := p.Decode(GraphInput{}, original); err == nil {
		t.Fatal("zero request accepted")
	}
	_, other, _, _ := graphWireFixture(t, contractstatemir.ExitReject, false)
	if _, err := other.Decode(in, original); err == nil {
		t.Fatal("foreign owner accepted")
	}
}

func TestGraphTransportFailedPrefix(t *testing.T) {
	_, p, in, _ := graphWireFixture(t, contractstatemir.ExitCommit, false)
	for status := byte(1); status <= 11; status++ {
		for _, ops := range []uint64{0, 1} {
			// The first storage operation can fail before its charge, or its
			// observer can fail after the charge without returning an event.
			frame := graphWireResponse(in, status, 0, 0, 0, ops, 0, nil, nil)
			out, err := p.Decode(in, frame)
			if err != nil || out.Status != status || len(out.Writes) != 0 || out.Exit != "" {
				t.Fatal("write-free failure prefix", status, ops, out, err)
			}
		}
	}
	for _, test := range []struct {
		status, exit byte
		trap         uint32
		cost, ops    uint64
	}{
		{0, 0, 0, 0, 0}, {1, 1, 0, 0, 0}, {1, 0, 1, 0, 0}, {1, 0, 0, 1, 0}, {1, 0, 0, 0, 2},
	} {
		if _, err := p.Decode(in, graphWireResponse(in, test.status, test.exit, test.trap, test.cost, test.ops, 0, nil, nil)); err == nil {
			t.Fatal("impossible completion or failed counters accepted", test)
		}
	}
}

func TestGraphTransportRequest(t *testing.T) {
	h, p, _, _ := graphWireFixture(t, contractstatemir.ExitCommit, false)
	limits := contractstate.Limits{MaxWrites: math.MaxInt, MaxWriteBytes: math.MaxUint64, MaxValueBytes: math.MaxInt, MaxMapKeyBytes: math.MaxInt}
	value := []byte{0, 255, 13, 10, 0, 0, 0, 7}
	snapshot := contractstate.PersistentSnapshot{LayoutHash: h.program.Layout.Hash, Version: math.MaxUint64,
		Entries: []contractstate.Entry{{Key: h.program.Layout.Fields[0].Prefix, Value: value}}}
	in, err := p.Input(snapshot, limits)
	if err != nil {
		t.Fatal(err)
	}
	packet := in.Bytes()
	if string(packet[:8]) != "FYGRIN01" || len(packet) != 196 || binary.BigEndian.Uint64(packet[104:112]) != math.MaxUint64 || !bytes.Equal(packet[188:], value) {
		t.Fatal("graph request framing changed")
	}
	for i, want := range []uint64{uint64(math.MaxInt), math.MaxUint64, uint64(math.MaxInt), uint64(math.MaxInt)} {
		if binary.BigEndian.Uint64(packet[112+i*8:120+i*8]) != want {
			t.Fatal("request policy truncated", i)
		}
	}
	value[0], packet[0] = 1, 0
	if in.Bytes()[188] != 0 || in.Bytes()[0] != 'F' {
		t.Fatal("graph request aliases caller bytes")
	}
	for _, mutate := range []func(*contractstate.PersistentSnapshot, *contractstate.Limits){
		func(s *contractstate.PersistentSnapshot, _ *contractstate.Limits) { s.LayoutHash[0] ^= 1 },
		func(s *contractstate.PersistentSnapshot, _ *contractstate.Limits) {
			s.Entries = append(s.Entries, s.Entries[0])
		},
		func(s *contractstate.PersistentSnapshot, _ *contractstate.Limits) { s.Entries[0].Value = []byte{1} },
		func(_ *contractstate.PersistentSnapshot, l *contractstate.Limits) { l.MaxWrites = 0 },
		func(_ *contractstate.PersistentSnapshot, l *contractstate.Limits) { l.MaxWriteBytes = 0 },
		func(_ *contractstate.PersistentSnapshot, l *contractstate.Limits) { l.MaxValueBytes = 0 },
		func(_ *contractstate.PersistentSnapshot, l *contractstate.Limits) { l.MaxMapKeyBytes = 0 },
	} {
		s, l := snapshot, limits
		s.Entries = slices.Clone(snapshot.Entries)
		mutate(&s, &l)
		if _, err := p.Input(s, l); err == nil {
			t.Fatal("invalid snapshot/policy accepted")
		}
	}
	if _, err := (GraphTransport{}).Input(snapshot, limits); err == nil {
		t.Fatal("zero graph owner accepted")
	}
}
