package contractstatenative

import (
	"bytes"
	"encoding/binary"
	"math"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

func overlayWireFixture(t *testing.T, exit contractstatemir.ExitKind) (Handoff, OverlayTransport, OverlayInput, contractstate.Layout) {
	t.Helper()
	handoff, layout := overlayMIRFixture(t, exit, false)
	prepared, err := PrepareOverlayTransport(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	input, err := prepared.Input(contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: math.MaxUint64}, contractstate.Limits{MaxWrites: math.MaxInt, MaxWriteBytes: math.MaxUint64, MaxValueBytes: math.MaxInt, MaxMapKeyBytes: math.MaxInt})
	if err != nil {
		t.Fatal(err)
	}
	packet := input.Bytes()
	for index, want := range []uint64{uint64(math.MaxInt), math.MaxUint64, uint64(math.MaxInt), uint64(math.MaxInt)} {
		if got := binary.BigEndian.Uint64(packet[112+index*8 : 120+index*8]); got != want {
			t.Fatalf("execution limit %d truncated: %d/%d", index, got, want)
		}
	}
	return handoff, prepared, input, layout
}

// Independent wire construction from the Go execution oracle. No production
// encoder/decoder, read/write plan or native record writer constructs these bytes.
func overlayWireResponse(t *testing.T, handoff Handoff, layout contractstate.Layout, version uint64) []byte {
	t.Helper()
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	port := &overlayOraclePort{PersistencePort: store}
	program := handoff.program
	// A reject control exposes all read observations without hiding a trap panic.
	if program.Exit == contractstatemir.ExitTrap {
		program.Exit = contractstatemir.ExitReject
		program.TrapCategory = ""
	}
	out, err := contractstatemir.ExecutePersistent(port, program, contractstate.Limits{MaxWrites: 1024, MaxWriteBytes: 1 << 20, MaxValueBytes: 1 << 20, MaxMapKeyBytes: 1 << 20})
	if err != nil {
		t.Fatal(err)
	}
	layoutHash := layout.Hash()
	packet := append([]byte("FYOVOUT1"), layoutHash[:]...)
	packet = append(packet, handoff.bodyHash[:]...)
	artifact, err := EmitOverlay(handoff, target.X8664WindowsMSVC)
	if err != nil {
		t.Fatal(err)
	}
	hash := artifact.ProgramHash()
	packet = append(packet, hash[:]...)
	packet = binary.BigEndian.AppendUint64(packet, version)
	exit := byte(0)
	if handoff.program.Exit == contractstatemir.ExitReject {
		exit = 1
	} else if handoff.program.Exit == contractstatemir.ExitTrap {
		exit = 2
	}
	packet = append(packet, 0, exit)
	packet = binary.BigEndian.AppendUint32(packet, uint32(len(out.Reads)))
	packet = binary.BigEndian.AppendUint32(packet, uint32(len(port.writes)))
	packet = binary.BigEndian.AppendUint64(packet, out.Receipt.WriteBytes)
	packet = binary.BigEndian.AppendUint64(packet, 0)
	lengthOffset := len(packet)
	packet = binary.BigEndian.AppendUint32(packet, 0)
	ids := map[string]byte{"bool/v1": 1, "u32/be/v1": 2, "u64/be/v1": 3, "bytes/v1": 4, "utf8/v1": 5, "bytes20/v1": 6, "bytes32/v1": 7}
	for _, read := range out.Reads {
		packet = binary.BigEndian.AppendUint32(packet, uint32(read.Operation))
		packet = append(packet, ids[read.Value.TypeID], 0)
		if read.Found {
			packet[len(packet)-1] = 1
		}
		packet = binary.BigEndian.AppendUint32(packet, uint32(len(read.Value.Encoded)))
		packet = append(packet, read.Value.Encoded...)
	}
	for _, write := range port.writes {
		packet = append(packet, write.Key[:]...)
		if write.Deleted {
			packet = append(packet, 1)
		} else {
			packet = append(packet, 0)
		}
		packet = binary.BigEndian.AppendUint32(packet, uint32(len(write.Value)))
		packet = append(packet, write.Value...)
	}
	binary.BigEndian.PutUint32(packet[lengthOffset:], uint32(len(packet)-lengthOffset-4))
	return packet
}

func TestOverlayTransportInput(t *testing.T) {
	_, prepared, _, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	limits := contractstate.Limits{MaxWrites: 1, MaxWriteBytes: 1, MaxValueBytes: 1, MaxMapKeyBytes: 1}
	key := layout.Fields()[0].Prefix
	snapshot := contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Version: math.MaxUint64, Entries: []contractstate.Entry{{Key: key, Value: []byte{1}}}}
	input, err := prepared.Input(snapshot, limits)
	if err != nil {
		t.Fatal(err)
	}
	packet := input.Bytes()
	if string(packet[:8]) != "FYOVIN01" || len(packet) != 189 || binary.BigEndian.Uint64(packet[104:112]) != math.MaxUint64 || binary.BigEndian.Uint32(packet[144:148]) != 1 || binary.BigEndian.Uint32(packet[148:152]) != 37 || !bytes.Equal(packet[152:184], key[:]) || binary.BigEndian.Uint32(packet[184:188]) != 1 || packet[188] != 1 {
		t.Fatal("noncanonical input")
	}
	snapshot.Entries[0].Value[0] = 0
	packet[188] = 0
	if input.Bytes()[188] != 1 {
		t.Fatal("request aliases caller bytes")
	}
	for _, name := range []string{"zero", "layout", "duplicate", "unsorted", "codec", "writes", "write-bytes", "value-bytes", "key-bytes", "entry-bound"} {
		t.Run(name, func(t *testing.T) {
			selected := prepared
			state := contractstate.PersistentSnapshot{LayoutHash: layout.Hash()}
			policy := limits
			switch name {
			case "zero":
				selected = OverlayTransport{}
			case "layout":
				state.LayoutHash[0] ^= 1
			case "duplicate":
				state.Entries = []contractstate.Entry{{Key: key, Value: []byte{1}}, {Key: key, Value: []byte{1}}}
			case "unsorted":
				state.Entries = []contractstate.Entry{{Key: contractstate.Key{255}}, {Key: contractstate.Key{0}}}
			case "codec":
				state.Entries = []contractstate.Entry{{Key: key, Value: []byte{2}}}
			case "writes":
				policy.MaxWrites = 0
			case "write-bytes":
				policy.MaxWriteBytes = 0
			case "value-bytes":
				policy.MaxValueBytes = 0
			case "key-bytes":
				policy.MaxMapKeyBytes = 0
			case "entry-bound":
				state.Entries = make([]contractstate.Entry, 65537)
			}
			if _, err := selected.Input(state, policy); err == nil {
				t.Fatal("invalid input admitted")
			}
		})
	}
	// Unaccessed map addresses cannot be inverted into a field/codec. Preserve
	// their bytes, while still enforcing exact full-snapshot order and bounds.
	unknown := contractstate.Key{255}
	exact := bytes.Repeat([]byte{0xff}, (16<<20)-36)
	state := contractstate.PersistentSnapshot{LayoutHash: layout.Hash(), Entries: []contractstate.Entry{{Key: unknown, Value: exact}}}
	if input, err := prepared.Input(state, limits); err != nil || len(input.Bytes()) != (16<<20)+152 {
		t.Fatal("exact snapshot byte ceiling", err)
	}
	state.Entries[0].Value = append(exact, 0)
	if _, err := prepared.Input(state, limits); err == nil {
		t.Fatal("oversized snapshot admitted")
	}
	state.Entries = make([]contractstate.Entry, 65536)
	for index := range state.Entries {
		binary.BigEndian.PutUint32(state.Entries[index].Key[28:], uint32(index))
	}
	if _, err := prepared.Input(state, limits); err != nil {
		t.Fatal("exact entry ceiling", err)
	}
}

func TestOverlayTransportDecode(t *testing.T) {
	for _, exit := range []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap} {
		t.Run(string(exit), func(t *testing.T) {
			handoff, prepared, input, layout := overlayWireFixture(t, exit)
			packet := overlayWireResponse(t, handoff, layout, math.MaxUint64)
			out, err := prepared.Decode(input, packet)
			if err != nil || out.Version != math.MaxUint64 || out.Exit != exit || len(out.Reads) != 35 {
				t.Fatal("lost complete result", out, err)
			}
			if exit == contractstatemir.ExitCommit && (len(out.Writes) != 14 || out.WriteBytes == 0) {
				t.Fatal("lost full native overlay")
			}
			if exit != contractstatemir.ExitCommit && (len(out.Writes) != 0 || out.WriteBytes != 0) {
				t.Fatal("rollback escaped")
			}
			if exit == contractstatemir.ExitTrap && out.TrapCategory != "overflow" {
				t.Fatal("trap category lost")
			}
			original := slices.Clone(packet)
			if len(out.Reads[1].Value.Encoded) != 0 {
				out.Reads[1].Value.Encoded[0] ^= 255
			}
			if !bytes.Equal(original, packet) {
				t.Fatal("result aliases response")
			}
		})
	}
}

func TestOverlayTransportRejectsCorruption(t *testing.T) {
	handoff, prepared, input, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	original := overlayWireResponse(t, handoff, layout, math.MaxUint64)
	writeOffset := 142
	for index := 0; index < 35; index++ {
		writeOffset += 10 + int(binary.BigEndian.Uint32(original[writeOffset+6:writeOffset+10]))
	}
	for _, name := range []string{"zero-owner", "zero-input", "short", "trailing", "magic", "layout", "body", "program", "version", "status", "exit", "wrong-exit", "read-count", "write-count", "cost", "payload", "read-operation", "read-codec", "read-presence", "read-value", "write-key", "write-delete", "write-value", "write-bytes", "low-count", "low-bytes", "low-value", "foreign-owner"} {
		t.Run(name, func(t *testing.T) {
			packet := slices.Clone(original)
			request := OverlayInput{packet: input.Bytes(), valid: true}
			owner := prepared
			switch name {
			case "zero-owner":
				owner = OverlayTransport{}
			case "zero-input":
				request = OverlayInput{}
			case "short":
				packet = packet[:len(packet)-1]
			case "trailing":
				packet = append(packet, 0)
			case "magic":
				packet[0] ^= 1
			case "layout":
				packet[8] ^= 1
			case "body":
				packet[40] ^= 1
			case "program":
				packet[72] ^= 1
			case "version":
				packet[104] ^= 1
			case "status":
				packet[112] = 255
			case "exit":
				packet[113] = 3
			case "wrong-exit":
				packet[113] = 1
			case "read-count":
				binary.BigEndian.PutUint32(packet[114:118], math.MaxUint32)
			case "write-count":
				binary.BigEndian.PutUint32(packet[118:122], math.MaxUint32)
			case "cost":
				packet[137] = 1
			case "payload":
				binary.BigEndian.PutUint32(packet[138:142], math.MaxUint32)
			case "read-operation":
				packet[145] = 255
			case "read-codec":
				packet[146] = 7
			case "read-presence":
				packet[147] = 2
			case "read-value":
				packet[162] = 2
			case "write-key":
				packet[writeOffset] ^= 1
			case "write-delete":
				packet[writeOffset+32] = 2
			case "write-value":
				at := writeOffset
				for binary.BigEndian.Uint32(packet[at+33:at+37]) == 0 {
					at += 37
				}
				packet[at+37] ^= 1
			case "write-bytes":
				packet[129] ^= 1
			case "low-count":
				binary.BigEndian.PutUint64(request.packet[112:120], 1)
			case "low-bytes":
				binary.BigEndian.PutUint64(request.packet[120:128], 1)
			case "low-value":
				binary.BigEndian.PutUint64(request.packet[128:136], 1)
			case "foreign-owner":
				owner.entry.programHash[0] ^= 1
			}
			if _, err := owner.Decode(request, packet); err == nil {
				t.Fatal("corrupt result admitted")
			}
		})
	}
}

func TestOverlayTransportFailureCarriesNoWrites(t *testing.T) {
	handoff, prepared, input, layout := overlayWireFixture(t, contractstatemir.ExitCommit)
	complete := overlayWireResponse(t, handoff, layout, math.MaxUint64)
	for status := byte(1); status <= 7; status++ {
		packet := slices.Clone(complete[:142])
		packet[112], packet[113] = status, 0
		clear(packet[114:142])
		out, err := prepared.Decode(input, packet)
		if err != nil || out.Status != status || len(out.Reads) != 0 || len(out.Writes) != 0 || out.WriteBytes != 0 {
			t.Fatal("lost failure-only response", status, out, err)
		}
		packet[113] = 1
		if _, err := prepared.Decode(input, packet); err == nil {
			t.Fatal("failure has completed-exit authority")
		}
		packet[113] = 0
		binary.BigEndian.PutUint32(packet[118:122], 1)
		if _, err := prepared.Decode(input, packet); err == nil {
			t.Fatal("failure has write authority")
		}
	}
}
