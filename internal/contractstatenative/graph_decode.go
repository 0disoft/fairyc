package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
)

type graphEvidence struct {
	path       graphPath
	snapshot   map[contractstate.Key][]byte
	writes     map[contractstate.Key]contractstate.PersistentWrite
	writeBytes uint64
	limits     [4]uint64
	reads      []contractstatemir.Read
}

// Decode admits the received path, original encoded map keys, observations and
// final overlay. Branches/helper returns follow the sealed CFG; all counters are
// rederived from event sites, without interpreting SSA or generating results.
func (prepared GraphTransport) Decode(input GraphInput, output []byte) (GraphResult, error) {
	bad := func() (GraphResult, error) {
		return GraphResult{}, fmt.Errorf("graph response violates its prepared request or execution path")
	}
	request := input.packet
	if !prepared.entry.valid || !input.valid || len(request) < graphInputHeader ||
		len(output) < graphOutputHeader || len(output) > overlayResponseBytes {
		return bad()
	}
	if string(request[:8]) != "FYGRIN01" || !bytes.Equal(request[8:40], prepared.entry.layout[:]) ||
		!bytes.Equal(request[40:72], prepared.entry.body[:]) || !bytes.Equal(request[72:104], prepared.program[:]) {
		return bad()
	}
	digest := sha256.Sum256(request)
	if string(output[:8]) != "FYGROUT1" || !bytes.Equal(output[8:112], request[8:112]) || !bytes.Equal(output[112:144], digest[:]) {
		return bad()
	}
	status, exit := output[144], output[145]
	trap := binary.BigEndian.Uint32(output[146:150])
	cost, ops := binary.BigEndian.Uint64(output[150:158]), binary.BigEndian.Uint64(output[158:166])
	events, writes := binary.BigEndian.Uint32(output[166:170]), binary.BigEndian.Uint32(output[170:174])
	writeBytes, payload := binary.BigEndian.Uint64(output[174:182]), binary.BigEndian.Uint32(output[182:186])
	if status > 11 || exit > 2 || uint64(payload) != uint64(len(output)-graphOutputHeader) ||
		uint64(events)*graphEventHeader+uint64(writes)*37 > uint64(payload) ||
		(status != 0 || exit != 0) && (writes != 0 || writeBytes != 0) || status != 0 && (exit != 0 || trap != 0) {
		return bad()
	}
	evidence := graphEvidence{path: graphPath{next: []uint32{prepared.start}},
		snapshot: make(map[contractstate.Key][]byte), writes: make(map[contractstate.Key]contractstate.PersistentWrite)}
	for i := range evidence.limits {
		evidence.limits[i] = binary.BigEndian.Uint64(request[112+i*8 : 120+i*8])
	}
	// Input bytes are private and already validated/copied by Input. Keep views
	// only for the duration of admission; no caller-owned snapshot can change them.
	at := graphInputHeader
	for at < len(request) {
		if len(request)-at < 36 {
			return bad()
		}
		key := contractstate.Key(request[at : at+32])
		size := uint64(binary.BigEndian.Uint32(request[at+32 : at+36]))
		at += 36
		if size > uint64(len(request)-at) {
			return bad()
		}
		evidence.snapshot[key] = request[at : at+int(size)]
		at += int(size)
	}
	at = graphOutputHeader
	take := func(size uint64) ([]byte, bool) {
		if size > uint64(len(output)-at) {
			return nil, false
		}
		part := output[at : at+int(size)]
		at += int(size)
		return part, true
	}
	for range events {
		h, ok := take(graphEventHeader)
		if !ok {
			return bad()
		}
		site := binary.BigEndian.Uint32(h[8:12])
		n, exists := prepared.sites[site]
		key, okKey := take(uint64(binary.BigEndian.Uint32(h[12:16])))
		value, okValue := take(uint64(binary.BigEndian.Uint32(h[16:20])))
		if !exists || !okKey || !okValue || !evidence.path.advance(site, n) ||
			binary.BigEndian.Uint64(h[:8]) != evidence.path.ops || !evidence.admit(n, key, value, h[20]) {
			return bad()
		}
	}
	if !evidence.counters(prepared.sites, status, cost, ops) {
		return bad()
	}
	result := GraphResult{Status: status, Version: binary.BigEndian.Uint64(request[104:112]), Cost: cost, Operations: ops, Reads: evidence.reads, WriteBytes: writeBytes}
	if status == 0 {
		result.Exit = []contractstatemir.ExitKind{contractstatemir.ExitCommit, contractstatemir.ExitReject, contractstatemir.ExitTrap}[exit]
		if result.Exit != evidence.path.exit || trap != evidence.path.trap {
			return bad()
		}
		if exit == 2 {
			if trap == 0 || uint64(trap) > uint64(len(prepared.entry.traps)) {
				return bad()
			}
			result.TrapCategory = prepared.entry.traps[trap-1]
		}
		if exit == 0 && (int(writes) != len(evidence.writes) || writeBytes != evidence.writeBytes) {
			return bad()
		}
	}
	var previous contractstate.Key
	for i := uint32(0); i < writes; i++ {
		h, ok := take(37)
		if !ok {
			return bad()
		}
		key := contractstate.Key(h[:32])
		value, ok := take(uint64(binary.BigEndian.Uint32(h[33:37])))
		expected, exists := evidence.writes[key]
		if !ok || !exists || h[32] > 1 || (h[32] == 1) != expected.Deleted ||
			!bytes.Equal(value, expected.Value) || i != 0 && bytes.Compare(previous[:], key[:]) >= 0 {
			return bad()
		}
		previous = key
		result.Writes = append(result.Writes, contractstate.PersistentWrite{Key: key, Deleted: expected.Deleted, Value: slices.Clone(value)})
	}
	if at != len(output) {
		return bad()
	}
	return result, nil
}

func (e *graphEvidence) counters(sites map[uint32]graphSite, status byte, cost, ops uint64) bool {
	if cost == e.path.cost && ops == e.path.ops {
		return status != 0 || e.path.exit != ""
	}
	// A failed observer may lose exactly the event whose counters were already
	// charged. It cannot claim success or skip any other operation or cost.
	if status != 0 {
		for _, site := range e.path.next {
			n := sites[site]
			extra := uint64(0)
			if n.operation {
				extra = 1
			}
			if cost == e.path.cost+n.cost && ops == e.path.ops+extra {
				return true
			}
		}
	}
	return false
}

func (e *graphEvidence) admit(n graphSite, keyBytes, value []byte, found byte) bool {
	read, write, deleted := graphStorageKind(n.op.Kind)
	if !read && !write {
		return len(keyBytes) == 0 && len(value) == 0 && found == 0
	}
	if found > 1 {
		return false
	}
	key := n.field.Prefix
	if n.field.Kind == contractstate.FieldMap {
		if uint64(len(keyBytes)) > e.limits[3] || contractstate.ValidateCanonicalEncoding(n.field.KeyTypeID, keyBytes) != nil ||
			n.op.KeyInput == 0 && !bytes.Equal(keyBytes, n.op.Key.Encoded) {
			return false
		}
		key = overlayOperationKey(n.field, keyBytes)
	} else if len(keyBytes) != 0 {
		return false
	}
	if read {
		observed, present := e.snapshot[key]
		if w, changed := e.writes[key]; changed {
			observed, present = w.Value, !w.Deleted
		}
		if present != (found == 1) {
			return false
		}
		if !present && (n.op.Kind == contractstatemir.OperationCellBorrow || n.op.Kind == contractstatemir.OperationMapBorrow) {
			if !graphAbsentBorrow(n.field.ValueTypeID, value) {
				return false
			}
		} else if !bytes.Equal(observed, value) {
			return false
		}
		if present && contractstate.ValidateCanonicalEncoding(n.field.ValueTypeID, value) != nil {
			return false
		}
		e.reads = append(e.reads, contractstatemir.Read{Operation: int(e.path.ops - 1), Found: present,
			Value: contractstatemir.Value{TypeID: n.field.ValueTypeID, Encoded: slices.Clone(value)}})
		return true
	}
	if deleted {
		if found != 0 || len(value) != 0 {
			return false
		}
	} else if found != 1 || uint64(len(value)) > e.limits[2] ||
		contractstate.ValidateCanonicalEncoding(n.field.ValueTypeID, value) != nil ||
		n.op.ValueInput == 0 && !bytes.Equal(n.op.Value.Encoded, value) {
		return false
	}
	old, replaced := e.writes[key]
	if replaced {
		e.writeBytes -= 33 + uint64(len(old.Value))
	}
	e.writeBytes += 33 + uint64(len(value))
	e.writes[key] = contractstate.PersistentWrite{Key: key, Deleted: deleted, Value: value}
	return uint64(len(e.writes)) <= e.limits[0] && e.writeBytes <= e.limits[1]
}

func graphAbsentBorrow(id string, value []byte) bool {
	width := map[string]int{"bool/v1": 1, "u32/be/v1": 4, "u64/be/v1": 8, "bytes20/v1": 20, "bytes32/v1": 32}[id]
	if len(value) != width {
		return false
	}
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}
