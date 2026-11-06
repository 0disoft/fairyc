package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/target"
)

const (
	persistentInputMagic          = "FYSPIN01"
	persistentOutputMagic         = "FYSPOU01"
	persistentMaxOperations       = 1024
	persistentInputSize           = 137
	persistentOutputSize          = 135
	persistentReadSize            = 13
	persistentLayoutOffset        = 8
	persistentBodyOffset          = 40
	persistentProgramOffset       = 72
	persistentVersionOffset       = 104
	persistentWriteLimitOffset    = 112
	persistentValueLimitOffset    = 120
	persistentInputPresentOffset  = 128
	persistentInputValueOffset    = 129
	persistentStatusOffset        = 112
	persistentDirtyOffset         = 113
	persistentOutputPresentOffset = 114
	persistentOutputValueOffset   = 115
	persistentReadCountOffset     = 123
	persistentCostOffset          = 127
	persistentCellValueSize       = 8
)

// PersistentArtifact seals a compiler-owned native protocol unit. Its initial
// route retains Emit's single-u64-cell admission; broader state MIR stays pending.
type PersistentArtifact struct {
	unit        Artifact
	programHash [32]byte
	key         contractstate.Key
	reads       []uint32
	writes      bool
	exit        PersistentStatus
	trap        string
	valid       bool
}

func (artifact PersistentArtifact) Source() string { return artifact.unit.Source }
func (artifact PersistentArtifact) LayoutHash() contractstate.LayoutHash {
	return artifact.unit.LayoutHash
}
func (artifact PersistentArtifact) BodyHash() contractstatemir.StateBodyHash {
	return artifact.unit.BodyHash
}
func (artifact PersistentArtifact) ProgramHash() [32]byte { return artifact.programHash }

type PersistentStatus uint8

const (
	PersistentCommit PersistentStatus = iota
	PersistentReject
	PersistentTrap
	PersistentWriteBytesLimit
	PersistentValueLimit
	PersistentRuntimeFailure
)

// PersistentInput owns an immutable exact snapshot request. Bytes returns a copy.
type PersistentInput struct {
	packet [persistentInputSize]byte
	valid  bool
}

func (input PersistentInput) Bytes() []byte {
	if !input.valid {
		return nil
	}
	return slices.Clone(input.packet[:])
}

// PersistentResult describes native computation, never a durable publication.
// Version is the echoed snapshot version; only the persistence port may advance it.
type PersistentResult struct {
	Status       PersistentStatus
	Version      uint64
	Dirty        bool
	Present      bool
	Value        []byte
	Reads        []contractstatemir.Read
	TrapCategory string
}

func EmitPersistent(handoff Handoff, targetID target.ID) (PersistentArtifact, error) {
	if len(handoff.program.Operations) > persistentMaxOperations {
		return PersistentArtifact{}, fmt.Errorf("native state transport exceeds %d operations", persistentMaxOperations)
	}
	unit, err := emit(handoff, targetID, true)
	if err != nil {
		return PersistentArtifact{}, err
	}
	artifact := PersistentArtifact{unit: unit, key: handoff.program.Layout.Fields[0].Prefix, trap: handoff.program.TrapCategory, valid: true}
	artifact.programHash = sha256.Sum256([]byte("fairyc.native-state-program/v1\x00" + string(targetID) + "\x00" + unit.Source))
	switch handoff.program.Exit {
	case contractstatemir.ExitCommit:
		artifact.exit = PersistentCommit
	case contractstatemir.ExitReject:
		artifact.exit = PersistentReject
	case contractstatemir.ExitTrap:
		artifact.exit = PersistentTrap
	}
	for index, operation := range handoff.program.Operations {
		if operation.Kind == contractstatemir.OperationCellRead {
			artifact.reads = append(artifact.reads, uint32(index))
		} else {
			artifact.writes = true
		}
	}
	capacity := max(1, len(artifact.reads))
	var source strings.Builder
	fmt.Fprintf(&source, "#include <stddef.h>\n#include <stdint.h>\nint32_t fycs_persistent_before_v1(uint8_t);\nvoid fycs_persistent_written_v1(void);\nint32_t fycs_persistent_observe_v1(uint32_t, const uint8_t *, size_t, uint8_t);\n#define FYCS_PERSISTENT_READ_CAPACITY_V1 %du\n", capacity)
	source.WriteString(unit.Source)
	fmt.Fprintf(&source, "static const uint8_t fycs_persistent_program_hash_v1[32] = {%s};\n", cBytes(artifact.programHash[:]))
	source.WriteString(persistentTransportC)
	artifact.unit.Source = source.String()
	return artifact, nil
}

// Input rejects wrong layouts, foreign physical keys and noncanonical snapshots
// before native execution. Positive limits match the transaction kernel policy.
func (artifact PersistentArtifact) Input(snapshot contractstate.PersistentSnapshot, limits contractstate.Limits) (PersistentInput, error) {
	if !artifact.valid {
		return PersistentInput{}, fmt.Errorf("native state transport requires a prepared artifact")
	}
	if snapshot.LayoutHash != artifact.unit.LayoutHash {
		return PersistentInput{}, contractstate.ErrLayoutMismatch
	}
	if limits.MaxWrites <= 0 || limits.MaxWriteBytes == 0 || limits.MaxValueBytes <= 0 || limits.MaxMapKeyBytes <= 0 {
		return PersistentInput{}, fmt.Errorf("native state transport requires positive execution limits")
	}
	if len(snapshot.Entries) > 1 {
		return PersistentInput{}, fmt.Errorf("native single-cell snapshot contains multiple entries")
	}
	var input PersistentInput
	if len(snapshot.Entries) == 1 {
		entry := snapshot.Entries[0]
		if entry.Key != artifact.key {
			return PersistentInput{}, fmt.Errorf("native single-cell snapshot contains a foreign key")
		}
		if err := contractstate.ValidateCanonicalEncoding(contractstate.U64.TypeID(), entry.Value); err != nil {
			return PersistentInput{}, fmt.Errorf("native snapshot value: %w", err)
		}
		input.packet[persistentInputPresentOffset] = 1
		copy(input.packet[persistentInputValueOffset:], entry.Value)
	}
	copy(input.packet[:], persistentInputMagic)
	copy(input.packet[persistentLayoutOffset:persistentBodyOffset], artifact.unit.LayoutHash[:])
	copy(input.packet[persistentBodyOffset:persistentProgramOffset], artifact.unit.BodyHash[:])
	copy(input.packet[persistentProgramOffset:persistentVersionOffset], artifact.programHash[:])
	binary.BigEndian.PutUint64(input.packet[persistentVersionOffset:], snapshot.Version)
	binary.BigEndian.PutUint64(input.packet[persistentWriteLimitOffset:], limits.MaxWriteBytes)
	binary.BigEndian.PutUint64(input.packet[persistentValueLimitOffset:], uint64(limits.MaxValueBytes))
	input.valid = true
	return input, nil
}

// Decode binds the complete bounded response to its request and prepared owner.
// It does not grant publication authority or reinterpret the state MIR in Go.
func (artifact PersistentArtifact) Decode(input PersistentInput, output []byte) (PersistentResult, error) {
	bad := func() (PersistentResult, error) {
		return PersistentResult{}, fmt.Errorf("native state response violates its exact request or program")
	}
	if !artifact.valid || !input.valid || !bytes.Equal(input.packet[persistentLayoutOffset:persistentBodyOffset], artifact.unit.LayoutHash[:]) || !bytes.Equal(input.packet[persistentBodyOffset:persistentProgramOffset], artifact.unit.BodyHash[:]) || !bytes.Equal(input.packet[persistentProgramOffset:persistentVersionOffset], artifact.programHash[:]) {
		return bad()
	}
	if len(output) < persistentOutputSize || len(output) > persistentOutputSize+persistentReadSize*len(artifact.reads) {
		return bad()
	}
	if string(output[:persistentLayoutOffset]) != persistentOutputMagic || !bytes.Equal(output[persistentLayoutOffset:persistentStatusOffset], input.packet[persistentLayoutOffset:persistentWriteLimitOffset]) {
		return bad()
	}
	status := PersistentStatus(output[persistentStatusOffset])
	if status > PersistentRuntimeFailure || output[persistentDirtyOffset] > 1 || output[persistentOutputPresentOffset] > 1 || binary.BigEndian.Uint64(output[persistentCostOffset:]) != 0 {
		return bad()
	}
	count := binary.BigEndian.Uint32(output[persistentReadCountOffset:])
	if uint64(count) > uint64(len(artifact.reads)) || len(output) != persistentOutputSize+int(count)*persistentReadSize {
		return bad()
	}
	if status <= PersistentTrap && (status != artifact.exit || int(count) != len(artifact.reads)) {
		return bad()
	}
	dirty := output[persistentDirtyOffset] != 0
	if dirty != (status == PersistentCommit && artifact.writes) {
		return bad()
	}
	present, value := output[persistentOutputPresentOffset] != 0, output[persistentOutputValueOffset:persistentOutputValueOffset+persistentCellValueSize]
	if !present && !allZero(value) {
		return bad()
	}
	if !dirty && (output[persistentOutputPresentOffset] != input.packet[persistentInputPresentOffset] || !bytes.Equal(value, input.packet[persistentInputValueOffset:])) {
		return bad()
	}
	result := PersistentResult{Status: status, Version: binary.BigEndian.Uint64(output[persistentVersionOffset:]), Dirty: dirty, Present: present}
	if present {
		result.Value = slices.Clone(value)
	}
	if status == PersistentTrap {
		result.TrapCategory = artifact.trap
	}
	for index := 0; index < int(count); index++ {
		at := persistentOutputSize + index*persistentReadSize
		if binary.BigEndian.Uint32(output[at:]) != artifact.reads[index] || output[at+4] > 1 || output[at+4] == 0 && !allZero(output[at+5:at+persistentReadSize]) {
			return bad()
		}
		read := contractstatemir.Read{Operation: int(artifact.reads[index]), Found: output[at+4] != 0, Value: contractstatemir.Value{TypeID: contractstate.U64.TypeID()}}
		if read.Found {
			read.Value.Encoded = slices.Clone(output[at+5 : at+persistentReadSize])
		}
		result.Reads = append(result.Reads, read)
	}
	return result, nil
}

func allZero(value []byte) bool {
	for _, b := range value {
		if b != 0 {
			return false
		}
	}
	return true
}
