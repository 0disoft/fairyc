package contractstatenative

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"strings"

	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

// OverlayArtifact seals generated operation code, not a persistence transport.
// The existing single-cell runtime and its framing remain separate.
type OverlayArtifact struct {
	source      string
	layout      contractstate.LayoutHash
	body        contractstatemir.StateBodyHash
	programHash [32]byte
	valid       bool
}

func (artifact OverlayArtifact) Source() string                           { return artifact.source }
func (artifact OverlayArtifact) LayoutHash() contractstate.LayoutHash     { return artifact.layout }
func (artifact OverlayArtifact) BodyHash() contractstatemir.StateBodyHash { return artifact.body }
func (artifact OverlayArtifact) ProgramHash() [32]byte                    { return artifact.programHash }

// EmitOverlay lowers verified literal cell/map operations for all seven codecs.
// Graphs/helpers/borrows stay closed until their execution and framing are ready.
func EmitOverlay(handoff Handoff, targetID target.ID) (OverlayArtifact, error) {
	if targetID != target.X8664WindowsMSVC || handoff.bodyHash == (contractstatemir.StateBodyHash{}) {
		return OverlayArtifact{}, fmt.Errorf("overlay emission requires a prepared Windows native owner")
	}
	program := handoff.program
	if failures := contractstatemir.Verify(program); len(failures) != 0 {
		return OverlayArtifact{}, fmt.Errorf("overlay emission requires verified MIR: %s", failures[0])
	}
	if len(program.Blocks) != 0 || len(program.Helpers) != 0 || len(program.Operations) > persistentMaxOperations {
		return OverlayArtifact{}, fmt.Errorf("overlay emission requires a bounded literal operation body")
	}
	header, err := cstdlib.ContractStateOverlayDeclarations()
	if err != nil {
		return OverlayArtifact{}, err
	}
	var out strings.Builder
	out.WriteString(header)
	out.WriteString("\ntypedef int32_t (*fycs_overlay_observer_v1)(void *, uint32_t, uint8_t, const uint8_t *, size_t, uint8_t);\n")
	for index, op := range program.Operations {
		field := program.Layout.Fields[op.FieldIndex]
		key := field.Prefix
		if field.Kind == contractstate.FieldMap {
			payload := append([]byte("fairyc-contract-state-map-entry/v1"), field.Prefix[:]...)
			payload = binary.BigEndian.AppendUint64(payload, uint64(len(op.Key.Encoded)))
			key = sha256.Sum256(append(payload, op.Key.Encoded...))
		}
		fmt.Fprintf(&out, "static const uint8_t fycs_overlay_key_%d_v1[32] = {%s};\n", index, cBytes(key[:]))
		if op.Kind == contractstatemir.OperationCellWrite || op.Kind == contractstatemir.OperationMapWrite {
			value := op.Value.Encoded
			if len(value) == 0 {
				value = []byte{0}
			}
			fmt.Fprintf(&out, "static const uint8_t fycs_overlay_value_%d_v1[%d] = {%s};\n", index, len(value), cBytes(value))
		}
	}
	out.WriteString("int32_t fycs_overlay_execute_v1(fycs_overlay_v1 *transaction, size_t max_key, fycs_overlay_observer_v1 observe, void *observer, uint8_t *exit_kind) {\n")
	if len(program.Operations) != 0 {
		out.WriteString("    int32_t status;\n")
	}
	reads := false
	for _, op := range program.Operations {
		if op.Kind == contractstatemir.OperationCellRead || op.Kind == contractstatemir.OperationMapRead {
			reads = true
		}
	}
	if reads {
		out.WriteString("    const uint8_t *value; size_t length; uint8_t found;\n")
	}
	out.WriteString("    (void)observer;\n    if (transaction == NULL || exit_kind == NULL || observe == NULL || max_key == 0u) { return FYCS_OV_INVALID; }\n    if (transaction->state != 1u) { return FYCS_OV_CLOSED; }\n    *exit_kind = 0u;\n")
	for index, op := range program.Operations {
		field := program.Layout.Fields[op.FieldIndex]
		codec, err := overlayCodec(field.ValueTypeID)
		if err != nil {
			return OverlayArtifact{}, err
		}
		if field.Kind == contractstate.FieldMap && len(op.Key.Encoded) != 0 {
			fmt.Fprintf(&out, "    if (max_key < %du) { status = 7; goto failed; }\n", len(op.Key.Encoded))
		}
		switch op.Kind {
		case contractstatemir.OperationCellRead, contractstatemir.OperationMapRead:
			fmt.Fprintf(&out, "    status = fycs_overlay_read_v1(transaction, fycs_overlay_key_%d_v1, &value, &length, &found);\n    if (status != 0) { goto failed; }\n", index)
			fmt.Fprintf(&out, "    if (found != 0u) { status = fycs_codec_valid_v1(%du, value, length); if (status != 0) { goto failed; } }\n", codec)
			fmt.Fprintf(&out, "    status = observe(observer, %du, %du, value, length, found);\n    if (status != 0) { goto failed; }\n", index, codec)
		case contractstatemir.OperationCellWrite, contractstatemir.OperationMapWrite:
			fmt.Fprintf(&out, "    status = fycs_codec_valid_v1(%du, fycs_overlay_value_%d_v1, %du);\n    if (status != 0) { goto failed; }\n", codec, index, len(op.Value.Encoded))
			fmt.Fprintf(&out, "    status = fycs_overlay_write_v1(transaction, fycs_overlay_key_%d_v1, fycs_overlay_value_%d_v1, %du, 0u);\n    if (status != 0) { goto failed; }\n", index, index, len(op.Value.Encoded))
		case contractstatemir.OperationCellDelete, contractstatemir.OperationMapDelete:
			fmt.Fprintf(&out, "    status = fycs_overlay_write_v1(transaction, fycs_overlay_key_%d_v1, NULL, 0u, 1u);\n    if (status != 0) { goto failed; }\n", index)
		default:
			return OverlayArtifact{}, fmt.Errorf("overlay emission does not admit operation %q", op.Kind)
		}
	}
	switch program.Exit {
	case contractstatemir.ExitCommit:
		out.WriteString("    return fycs_overlay_commit_v1(transaction);\n")
	case contractstatemir.ExitReject, contractstatemir.ExitTrap:
		kind := 1
		if program.Exit == contractstatemir.ExitTrap {
			kind = 2
		}
		fmt.Fprintf(&out, "    *exit_kind = %du;\n    return fycs_overlay_abort_v1(transaction);\n", kind)
	default:
		return OverlayArtifact{}, fmt.Errorf("overlay emission does not admit exit %q", program.Exit)
	}
	if len(program.Operations) != 0 {
		out.WriteString("failed:\n    (void)fycs_overlay_abort_v1(transaction);\n    return status;\n")
	}
	out.WriteString("}\n")
	artifact := OverlayArtifact{source: out.String(), layout: program.Layout.Hash, body: handoff.bodyHash, valid: true}
	identity := append([]byte("fairyc.native-state-overlay-entry/v1\x00"+string(targetID)+"\x00"), artifact.layout[:]...)
	identity = append(identity, artifact.body[:]...)
	identity = append(identity, []byte(string(program.Exit)+"\x00"+program.TrapCategory+"\x00"+artifact.source)...)
	artifact.programHash = sha256.Sum256(identity)
	return artifact, nil
}

func overlayCodec(typeID string) (uint8, error) {
	for index, id := range []string{contractstate.Bool.TypeID(), contractstate.U32.TypeID(), contractstate.U64.TypeID(), contractstate.Bytes.TypeID(), contractstate.Text.TypeID(), contractstate.Byte20.TypeID(), contractstate.Byte32.TypeID()} {
		if typeID == id {
			return uint8(index + 1), nil
		}
	}
	return 0, fmt.Errorf("overlay emission does not admit codec %q", typeID)
}
