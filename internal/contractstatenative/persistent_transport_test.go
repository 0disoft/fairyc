package contractstatenative

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/hir"
	"github.com/0disoft/fairyc/internal/target"
)

func persistentFixture(t *testing.T, exit contractstatemir.ExitKind, deleting bool) (Handoff, PersistentArtifact, contractstate.PersistentSnapshot) {
	t.Helper()
	compiled, coordinate := nativeSourceResult(t, exit, deleting)
	owner := compiled.StateTransactions[0]
	handoff, err := PrepareCompiled(compiled, owner.Package, owner.Function, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := EmitPersistent(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	snapshot := contractstate.PersistentSnapshot{LayoutHash: handoff.program.Layout.Hash, Version: math.MaxUint64, Entries: []contractstate.Entry{{Key: handoff.program.Layout.Fields[0].Prefix, Value: []byte{0, 10, 13, 26, 255, 128, 1, 7}}}}
	return handoff, artifact, snapshot
}

func persistentLimits() contractstate.Limits {
	return contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 41, MaxValueBytes: 8, MaxMapKeyBytes: 1}
}

func TestPersistentTransportInput(t *testing.T) {
	_, artifact, snapshot := persistentFixture(t, contractstatemir.ExitCommit, false)
	input, err := artifact.Input(snapshot, persistentLimits())
	if err != nil {
		t.Fatal(err)
	}
	packet := input.Bytes()
	if len(packet) != 137 || string(packet[:8]) != "FYSPIN01" || binary.BigEndian.Uint64(packet[104:112]) != math.MaxUint64 || packet[128] != 1 || !bytes.Equal(packet[129:], snapshot.Entries[0].Value) {
		t.Fatal("input lost exact binary snapshot or version")
	}
	packet[129] ^= 255
	if bytes.Equal(packet, input.Bytes()) {
		t.Fatal("input bytes alias caller")
	}
	if _, err := (PersistentArtifact{}).Input(snapshot, persistentLimits()); err == nil {
		t.Fatal("zero artifact admitted")
	}
	for _, name := range []string{"layout", "foreign-key", "multiple-entries", "short-value", "long-value", "writes", "write-bytes", "value-bytes", "map-key-bytes"} {
		t.Run(name, func(t *testing.T) {
			bad := snapshot
			bad.Entries = slices.Clone(snapshot.Entries)
			limits := persistentLimits()
			switch name {
			case "layout":
				bad.LayoutHash[0] ^= 1
			case "foreign-key":
				bad.Entries[0].Key[0] ^= 1
			case "multiple-entries":
				bad.Entries = append(bad.Entries, bad.Entries[0])
			case "short-value":
				bad.Entries[0].Value = make([]byte, 7)
			case "long-value":
				bad.Entries[0].Value = make([]byte, 9)
			case "writes":
				limits.MaxWrites = 0
			case "write-bytes":
				limits.MaxWriteBytes = 0
			case "value-bytes":
				limits.MaxValueBytes = -1
			case "map-key-bytes":
				limits.MaxMapKeyBytes = 0
			}
			if _, err := artifact.Input(bad, limits); err == nil {
				t.Fatal("invalid snapshot or policy admitted")
			}
		})
	}
	snapshot.Entries = nil
	input, err = artifact.Input(snapshot, persistentLimits())
	if err != nil || !allZero(input.Bytes()[128:]) {
		t.Fatal("absent snapshot is not canonical", err)
	}
}

// This independent wire constructor follows the documented field sequence;
// it does not invoke production encoders or native serialization helpers.
func persistentResponse(artifact PersistentArtifact, version uint64, status, dirty, present byte, value []byte, reads []contractstatemir.Read) []byte {
	out := []byte("FYSPOU01")
	layout, body := artifact.LayoutHash(), artifact.BodyHash()
	out = append(out, layout[:]...)
	out = append(out, body[:]...)
	program := artifact.ProgramHash()
	out = append(out, program[:]...)
	out = binary.BigEndian.AppendUint64(out, version)
	out = append(out, status, dirty, present)
	var raw [8]byte
	copy(raw[:], value)
	out = append(out, raw[:]...)
	out = binary.BigEndian.AppendUint32(out, uint32(len(reads)))
	out = binary.BigEndian.AppendUint64(out, 0)
	for _, read := range reads {
		out = binary.BigEndian.AppendUint32(out, uint32(read.Operation))
		found := byte(0)
		if read.Found {
			found = 1
		}
		out = append(out, found)
		clear(raw[:])
		copy(raw[:], read.Value.Encoded)
		out = append(out, raw[:]...)
	}
	return out
}

func TestPersistentTransportDecode(t *testing.T) {
	_, artifact, snapshot := persistentFixture(t, contractstatemir.ExitCommit, false)
	input, err := artifact.Input(snapshot, persistentLimits())
	if err != nil {
		t.Fatal(err)
	}
	value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
	reads := []contractstatemir.Read{{Operation: 1, Found: true, Value: contractstatemir.Value{TypeID: contractstate.U64.TypeID(), Encoded: value}}}
	wire := persistentResponse(artifact, snapshot.Version, 0, 1, 1, value, reads)
	result, err := artifact.Decode(input, wire)
	if err != nil || !result.Dirty || !result.Present || result.Version != math.MaxUint64 || len(result.Reads) != 1 || !bytes.Equal(result.Value, value) {
		t.Fatalf("response: %+v %v", result, err)
	}
	for _, name := range []string{"short", "trailing", "magic", "layout", "body", "program", "version", "status", "dirty", "present", "absent-padding", "read-count", "read-index", "read-found", "read-absent-padding", "cost", "wrong-exit", "missing-read", "zero-input"} {
		t.Run(name, func(t *testing.T) {
			bad := slices.Clone(wire)
			selected := input
			switch name {
			case "short":
				bad = bad[:len(bad)-1]
			case "trailing":
				bad = append(bad, 0)
			case "magic":
				bad[0] ^= 1
			case "layout":
				bad[8] ^= 1
			case "body":
				bad[40] ^= 1
			case "program":
				bad[72] ^= 1
			case "version":
				bad[111] ^= 1
			case "status":
				bad[112] = 6
			case "dirty":
				bad[113] = 2
			case "present":
				bad[114] = 2
			case "absent-padding":
				bad[114] = 0
			case "read-count":
				binary.BigEndian.PutUint32(bad[123:], math.MaxUint32)
			case "read-index":
				binary.BigEndian.PutUint32(bad[135:], 0)
			case "read-found":
				bad[139] = 2
			case "read-absent-padding":
				bad[139] = 0
			case "cost":
				bad[134] = 1
			case "wrong-exit":
				bad[112] = 1
				bad[113] = 0
			case "missing-read":
				bad = bad[:135]
				binary.BigEndian.PutUint32(bad[123:], 0)
			case "zero-input":
				selected = PersistentInput{}
			}
			if _, err := artifact.Decode(selected, bad); err == nil {
				t.Fatal("malformed response admitted")
			}
		})
	}
	wire[122] ^= 1
	wire[len(wire)-1] ^= 1
	if !bytes.Equal(result.Value, value) || !bytes.Equal(result.Reads[0].Value.Encoded, value) {
		t.Fatal("decoded values alias native output")
	}
	_, other, _ := persistentFixture(t, contractstatemir.ExitReject, false)
	if _, err := other.Decode(input, wire); err == nil {
		t.Fatal("response reused under another body")
	}
}

func TestPersistentTransportAdmissionAndOperationBound(t *testing.T) {
	handoff, artifact, _ := persistentFixture(t, contractstatemir.ExitCommit, false)
	if _, err := EmitPersistent(Handoff{}, target.X8664WindowsMSVC); err == nil {
		t.Fatal("zero handoff admitted")
	}
	if _, err := EmitPersistent(handoff, target.X8664LinuxGNU); err == nil {
		t.Fatal("unimplemented Linux transport admitted")
	}
	if _, err := EmitPersistent(corruptNativeHandoff(handoff), target.X8664WindowsMSVC); err == nil {
		t.Fatal("corrupt handoff admitted")
	}
	if artifact.Source() == "" {
		t.Fatal("transport has no executable C")
	}
	for _, count := range []int{0, persistentMaxOperations, persistentMaxOperations + 1} {
		input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
		input.StateTransactions[0].Accesses = make([]hir.StateAccess, count)
		for i := range input.StateTransactions[0].Accesses {
			input.StateTransactions[0].Accesses[i] = hir.StateAccess{Kind: hir.StateAccessCellRead, Field: "value"}
		}
		selected, err := Prepare(input, 1, 1, coordinate)
		if count > persistentMaxOperations {
			if err == nil {
				t.Fatal("core admitted operation overflow")
			}
			selected = handoff
			selected.program.Operations = make([]contractstatemir.Operation, count)
		} else if err != nil {
			t.Fatal(count, err)
		}
		_, err = EmitPersistent(selected, target.X8664WindowsMSVC)
		if (err == nil) != (count <= persistentMaxOperations) {
			t.Fatalf("%d operation admission: %v", count, err)
		}
	}
}

func TestPersistentTransportRejectsOtherOwnerInSamePackage(t *testing.T) {
	input, coordinate := nativeStateInput(t, contractstatemir.ExitCommit)
	function := input.Modules[0].Functions[0]
	function.Symbol, function.Name = 2, "other"
	input.Modules[0].Functions = append(input.Modules[0].Functions, function)
	transaction := input.StateTransactions[0]
	transaction.Function = 2
	transaction.Accesses = slices.Clone(transaction.Accesses)
	transaction.Accesses[0].Value.Encoded = []byte{0, 0, 0, 0, 0, 0, 0, 99}
	input.StateTransactions = append(input.StateTransactions, transaction)
	first, err := Prepare(input, 1, 1, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Prepare(input, 1, 2, coordinate)
	if err != nil {
		t.Fatal(err)
	}
	a, err := EmitPersistent(first, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	b, err := EmitPersistent(second, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	if a.BodyHash() != b.BodyHash() {
		t.Fatal("fixture must share one package body")
	}
	snapshot := contractstate.PersistentSnapshot{LayoutHash: a.LayoutHash(), Version: 7}
	request, err := a.Input(snapshot, persistentLimits())
	if err != nil {
		t.Fatal(err)
	}
	value := []byte{0, 0, 0, 0, 0, 0, 0, 42}
	reads := []contractstatemir.Read{{Operation: 1, Found: true, Value: contractstatemir.Value{TypeID: contractstate.U64.TypeID(), Encoded: value}}}
	response := persistentResponse(a, 7, 0, 1, 1, value, reads)
	if _, err := b.Decode(request, response); err == nil {
		t.Fatal("accepted another transaction owner's result under the same package body")
	}
}
