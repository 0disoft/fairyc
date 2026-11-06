package contractstatenative

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	overlayInputHeader     = 152
	overlayOutputHeader    = 142
	overlaySnapshotBytes   = 16 << 20
	overlaySnapshotEntries = 65536
	overlayResponseBytes   = 64 << 20
)

type overlayReadSpec struct {
	operation uint32
	codec     uint8
	typeID    string
}

// OverlayTransport seals Go-side request/response authority for a literal MIR
// owner. The native framing entry and CAS executor are separate delivery steps.
type OverlayTransport struct {
	entry  OverlayArtifact
	known  map[contractstate.Key]string
	reads  []overlayReadSpec
	writes []contractstate.PersistentWrite
	exit   byte
	trap   string
}

type OverlayInput struct {
	packet []byte
	valid  bool
}

func (input OverlayInput) Bytes() []byte { return slices.Clone(input.packet) }

type OverlayResult struct {
	Status       byte
	Exit         contractstatemir.ExitKind
	TrapCategory string
	Version      uint64
	Reads        []contractstatemir.Read
	Writes       []contractstate.PersistentWrite
	WriteBytes   uint64
}

func PrepareOverlayTransport(handoff Handoff, id target.ID) (OverlayTransport, error) {
	entry, err := EmitOverlay(handoff, id)
	if err != nil {
		return OverlayTransport{}, err
	}
	prepared := OverlayTransport{entry: entry, known: make(map[contractstate.Key]string), trap: handoff.program.TrapCategory}
	final := make(map[contractstate.Key]contractstate.PersistentWrite)
	for index, op := range handoff.program.Operations {
		field := handoff.program.Layout.Fields[op.FieldIndex]
		key := overlayOperationKey(field, op.Key.Encoded)
		if old, found := prepared.known[key]; found && old != field.ValueTypeID {
			return OverlayTransport{}, fmt.Errorf("native overlay address has conflicting codecs")
		}
		prepared.known[key] = field.ValueTypeID
		switch op.Kind {
		case contractstatemir.OperationCellRead, contractstatemir.OperationMapRead:
			codec, err := overlayCodec(field.ValueTypeID)
			if err != nil {
				return OverlayTransport{}, err
			}
			prepared.reads = append(prepared.reads, overlayReadSpec{operation: uint32(index), codec: codec, typeID: field.ValueTypeID})
		case contractstatemir.OperationCellWrite, contractstatemir.OperationMapWrite:
			final[key] = contractstate.PersistentWrite{Key: key, Value: slices.Clone(op.Value.Encoded)}
		case contractstatemir.OperationCellDelete, contractstatemir.OperationMapDelete:
			final[key] = contractstate.PersistentWrite{Key: key, Deleted: true}
		}
	}
	for _, write := range final {
		prepared.writes = append(prepared.writes, write)
	}
	slices.SortFunc(prepared.writes, func(a, b contractstate.PersistentWrite) int { return bytes.Compare(a.Key[:], b.Key[:]) })
	if handoff.program.Exit == contractstatemir.ExitReject {
		prepared.exit = 1
	} else if handoff.program.Exit == contractstatemir.ExitTrap {
		prepared.exit = 2
	}
	return prepared, nil
}

func (prepared OverlayTransport) Input(snapshot contractstate.PersistentSnapshot, limits contractstate.Limits) (OverlayInput, error) {
	if !prepared.entry.valid {
		return OverlayInput{}, fmt.Errorf("overlay request requires a prepared owner")
	}
	if snapshot.LayoutHash != prepared.entry.layout {
		return OverlayInput{}, contractstate.ErrLayoutMismatch
	}
	if limits.MaxWrites <= 0 || limits.MaxWriteBytes == 0 || limits.MaxValueBytes <= 0 || limits.MaxMapKeyBytes <= 0 {
		return OverlayInput{}, fmt.Errorf("overlay request requires positive execution limits")
	}
	if len(snapshot.Entries) > overlaySnapshotEntries {
		return OverlayInput{}, fmt.Errorf("overlay snapshot exceeds entry limit")
	}
	var payload int
	for index, entry := range snapshot.Entries {
		if index > 0 && bytes.Compare(snapshot.Entries[index-1].Key[:], entry.Key[:]) >= 0 {
			return OverlayInput{}, fmt.Errorf("overlay snapshot keys must be sorted and unique")
		}
		if len(entry.Value) > overlaySnapshotBytes-36 || payload > overlaySnapshotBytes-36-len(entry.Value) {
			return OverlayInput{}, fmt.Errorf("overlay snapshot exceeds byte limit")
		}
		if id, found := prepared.known[entry.Key]; found {
			if err := contractstate.ValidateCanonicalEncoding(id, entry.Value); err != nil {
				return OverlayInput{}, fmt.Errorf("overlay snapshot value: %w", err)
			}
		}
		payload += 36 + len(entry.Value)
	}
	packet := make([]byte, overlayInputHeader, overlayInputHeader+payload)
	copy(packet, "FYOVIN01")
	copy(packet[8:40], prepared.entry.layout[:])
	copy(packet[40:72], prepared.entry.body[:])
	copy(packet[72:104], prepared.entry.programHash[:])
	binary.BigEndian.PutUint64(packet[104:], snapshot.Version)
	binary.BigEndian.PutUint64(packet[112:], uint64(limits.MaxWrites))
	binary.BigEndian.PutUint64(packet[120:], limits.MaxWriteBytes)
	binary.BigEndian.PutUint64(packet[128:], uint64(limits.MaxValueBytes))
	binary.BigEndian.PutUint64(packet[136:], uint64(limits.MaxMapKeyBytes))
	binary.BigEndian.PutUint32(packet[144:], uint32(len(snapshot.Entries)))
	binary.BigEndian.PutUint32(packet[148:], uint32(payload))
	for _, entry := range snapshot.Entries {
		packet = append(packet, entry.Key[:]...)
		packet = binary.BigEndian.AppendUint32(packet, uint32(len(entry.Value)))
		packet = append(packet, entry.Value...)
	}
	return OverlayInput{packet: packet, valid: true}, nil
}

// Decode validates complete canonical native evidence before a caller may publish
// its copied bytes. It checks declared constant writes, never computes native
// reads or runs MIR to manufacture a result.
func (prepared OverlayTransport) Decode(input OverlayInput, output []byte) (OverlayResult, error) {
	bad := func() (OverlayResult, error) {
		return OverlayResult{}, fmt.Errorf("overlay response violates its prepared request or operation plan")
	}
	if !prepared.entry.valid || !input.valid || len(input.packet) < overlayInputHeader || len(output) < overlayOutputHeader || len(output) > overlayResponseBytes {
		return bad()
	}
	request := input.packet
	if string(request[:8]) != "FYOVIN01" || !bytes.Equal(request[8:40], prepared.entry.layout[:]) || !bytes.Equal(request[40:72], prepared.entry.body[:]) || !bytes.Equal(request[72:104], prepared.entry.programHash[:]) {
		return bad()
	}
	if string(output[:8]) != "FYOVOUT1" || !bytes.Equal(output[8:112], request[8:112]) || binary.BigEndian.Uint64(output[130:138]) != 0 {
		return bad()
	}
	status, exit := output[112], output[113]
	reads, writes := binary.BigEndian.Uint32(output[114:118]), binary.BigEndian.Uint32(output[118:122])
	payload := binary.BigEndian.Uint32(output[138:142])
	if status > 7 || exit > 2 || int(payload) != len(output)-overlayOutputHeader || reads > uint32(len(prepared.reads)) || writes > uint32(len(prepared.writes)) || uint64(reads)*10+uint64(writes)*37 > uint64(payload) {
		return bad()
	}
	if status == 0 && (exit != prepared.exit || int(reads) != len(prepared.reads)) {
		return bad()
	}
	writeBytes := binary.BigEndian.Uint64(output[122:130])
	if status != 0 && exit != 0 || (status != 0 || exit != 0) && (writes != 0 || writeBytes != 0) {
		return bad()
	}
	if status == 0 && exit == 0 && int(writes) != len(prepared.writes) {
		return bad()
	}
	result := OverlayResult{Status: status, Exit: contractstatemir.ExitCommit, Version: binary.BigEndian.Uint64(output[104:112]), WriteBytes: writeBytes}
	if exit == 1 {
		result.Exit = contractstatemir.ExitReject
	} else if exit == 2 {
		result.Exit = contractstatemir.ExitTrap
		result.TrapCategory = prepared.trap
	}
	at := overlayOutputHeader
	take := func(size int) ([]byte, bool) {
		if size < 0 || size > len(output)-at {
			return nil, false
		}
		part := output[at : at+size]
		at += size
		return part, true
	}
	for index := 0; index < int(reads); index++ {
		header, ok := take(10)
		if !ok {
			return bad()
		}
		spec := prepared.reads[index]
		length := binary.BigEndian.Uint32(header[6:10])
		value, ok := take(int(length))
		if !ok {
			return bad()
		}
		if binary.BigEndian.Uint32(header[:4]) != spec.operation || header[4] != spec.codec || header[5] > 1 || header[5] == 0 && length != 0 {
			return bad()
		}
		read := contractstatemir.Read{Operation: int(spec.operation), Found: header[5] == 1, Value: contractstatemir.Value{TypeID: spec.typeID}}
		if read.Found {
			if err := contractstate.ValidateCanonicalEncoding(spec.typeID, value); err != nil {
				return bad()
			}
			if len(value) != 0 {
				read.Value.Encoded = slices.Clone(value)
			}
		}
		result.Reads = append(result.Reads, read)
	}
	var total uint64
	for index := 0; index < int(writes); index++ {
		header, ok := take(37)
		if !ok {
			return bad()
		}
		value, ok := take(int(binary.BigEndian.Uint32(header[33:37])))
		if !ok {
			return bad()
		}
		expected := prepared.writes[index]
		if !bytes.Equal(header[:32], expected.Key[:]) || header[32] > 1 || (header[32] == 1) != expected.Deleted || !bytes.Equal(value, expected.Value) {
			return bad()
		}
		if uint64(len(value)) > binary.BigEndian.Uint64(request[128:136]) {
			return bad()
		}
		total += 33 + uint64(len(value))
		result.Writes = append(result.Writes, contractstate.PersistentWrite{Key: expected.Key, Deleted: expected.Deleted, Value: slices.Clone(value)})
	}
	if at != len(output) || total != writeBytes || writeBytes > binary.BigEndian.Uint64(request[120:128]) || uint64(writes) > binary.BigEndian.Uint64(request[112:120]) {
		return bad()
	}
	return result, nil
}
