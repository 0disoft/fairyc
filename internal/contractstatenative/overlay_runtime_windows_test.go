package contractstatenative

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/cstdlib"
)

//go:embed overlay_runtime_probe.c
var overlayRuntimeProbe string

type overlayOperation struct {
	kind       byte
	field      int
	key, value []byte
}

type overlayCase struct {
	name                string
	limits              contractstate.Limits
	initial, operations []overlayOperation
}

type overlayOraclePort struct {
	contractstate.PersistencePort
	writes []contractstate.PersistentWrite
}

func (port *overlayOraclePort) CompareAndPublish(hash contractstate.LayoutHash, version uint64, writes []contractstate.PersistentWrite) (contractstate.PersistentReceipt, error) {
	port.writes = writes
	return port.PersistencePort.CompareAndPublish(hash, version, writes)
}

func appendOverlayNumber(out []byte, value uint64, width int) []byte {
	for shift := (width - 1) * 8; shift >= 0; shift -= 8 {
		out = append(out, byte(value>>shift))
	}
	return out
}

func overlayPhysicalKey(field contractstate.Field, encoded []byte) contractstate.Key {
	if field.Kind == contractstate.FieldCell {
		return field.Prefix
	}
	payload := append([]byte("fairyc-contract-state-map-entry/v1"), field.Prefix[:]...)
	payload = binary.BigEndian.AppendUint64(payload, uint64(len(encoded)))
	return sha256.Sum256(append(payload, encoded...))
}

func overlayOracle(t *testing.T, trial overlayCase) (input, output []byte) {
	t.Helper()
	builder, err := contractstate.NewLayoutBuilder("native.overlay")
	if err != nil {
		t.Fatal(err)
	}
	cells := make([]contractstate.EncodedCell, 3)
	for i := range cells {
		cells[i], err = contractstate.DeclareEncodedCell(builder, fmt.Sprintf("value%d", i), contractstate.Bytes.TypeID())
		if err != nil {
			t.Fatal(err)
		}
	}
	mapped, err := contractstate.DeclareEncodedMap(builder, "items", contractstate.Bytes.TypeID(), contractstate.Bytes.TypeID())
	if err != nil {
		t.Fatal(err)
	}
	layout, err := builder.Build()
	if err != nil {
		t.Fatal(err)
	}
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	fields := layout.Fields()
	var seeded []contractstate.PersistentWrite
	for _, op := range trial.initial {
		seeded = append(seeded, contractstate.PersistentWrite{Key: overlayPhysicalKey(fields[op.field], op.key), Value: op.value})
	}
	if len(seeded) != 0 {
		if _, err := store.CompareAndPublish(layout.Hash(), 0, seeded); err != nil {
			t.Fatal(err)
		}
	}
	snapshot, err := store.Snapshot(layout.Hash())
	if err != nil {
		t.Fatal(err)
	}
	input = appendOverlayNumber(input, uint64(trial.limits.MaxWrites), 1)
	input = appendOverlayNumber(input, trial.limits.MaxWriteBytes, 4)
	input = appendOverlayNumber(input, uint64(trial.limits.MaxValueBytes), 2)
	input = append(input, byte(len(snapshot.Entries)), byte(len(trial.operations)))
	for _, entry := range snapshot.Entries {
		input = append(input, entry.Key[:]...)
		input = appendOverlayNumber(input, uint64(len(entry.Value)), 2)
		input = append(input, entry.Value...)
	}
	port := &overlayOraclePort{PersistencePort: store}
	var last []byte
	_, err = contractstate.ExecutePersistent(port, layout, trial.limits, func(tx *contractstate.Transaction) error {
		for _, op := range trial.operations {
			key := overlayPhysicalKey(fields[op.field], op.key)
			input = append(input, op.kind)
			input = append(input, key[:]...)
			input = appendOverlayNumber(input, uint64(len(op.value)), 2)
			input = append(input, op.value...)
			var value []byte
			var found bool
			var operationErr error
			switch op.kind {
			case 1:
				if op.field == 3 {
					value, found, operationErr = mapped.Get(tx, op.key)
				} else {
					value, found, operationErr = cells[op.field].Get(tx)
				}
				last = slices.Clone(value)
			case 2, 4:
				written := op.value
				if op.kind == 4 {
					written = last
				}
				if op.field == 3 {
					operationErr = mapped.Set(tx, op.key, written)
				} else {
					operationErr = cells[op.field].Set(tx, written)
				}
			case 3:
				if op.field == 3 {
					operationErr = mapped.Delete(tx, op.key)
				} else {
					operationErr = cells[op.field].Delete(tx)
				}
			default:
				t.Fatalf("bad test operation %d", op.kind)
			}
			status := byte(0)
			switch {
			case errors.Is(operationErr, contractstate.ErrClosed):
				status = 2
			case errors.Is(operationErr, contractstate.ErrValueTooLarge):
				status = 3
			case errors.Is(operationErr, contractstate.ErrWriteLimit):
				status = 4
			case errors.Is(operationErr, contractstate.ErrWriteBytesLimit):
				status = 5
			case operationErr != nil:
				t.Fatal(operationErr)
			}
			output = append(output, status, 0)
			if found {
				output[len(output)-1] = 1
			}
			output = appendOverlayNumber(output, uint64(len(value)), 2)
			output = append(output, value...)
			stats := tx.Stats()
			output = appendOverlayNumber(output, uint64(stats.Writes), 2)
			output = appendOverlayNumber(output, stats.WriteBytes, 4)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	var total uint64
	for _, write := range port.writes {
		total += 33 + uint64(len(write.Value))
	}
	output = appendOverlayNumber(output, uint64(len(port.writes)), 2)
	output = appendOverlayNumber(output, total, 4)
	for _, write := range port.writes {
		output = append(output, write.Key[:]...)
		if write.Deleted {
			output = append(output, 1)
		} else {
			output = append(output, 0)
		}
		output = appendOverlayNumber(output, uint64(len(write.Value)), 2)
		output = append(output, write.Value...)
	}
	return
}

func overlayTrials() []overlayCase {
	write := func(field int, value string) overlayOperation {
		return overlayOperation{kind: 2, field: field, value: []byte(value)}
	}
	read := func(field int) overlayOperation { return overlayOperation{kind: 1, field: field} }
	remove := func(field int) overlayOperation { return overlayOperation{kind: 3, field: field} }
	base := []overlayOperation{write(0, "old"), write(1, "")}
	mixed := []overlayOperation{read(0), read(1), read(2), write(2, "third"), write(0, "new"), read(0), remove(0), read(0), write(0, ""), read(0), {kind: 2, field: 3, key: []byte{0, 255}, value: []byte("mapped")}, {kind: 1, field: 3, key: []byte{0, 255}}, {kind: 3, field: 3, key: []byte{0, 255}}, {kind: 1, field: 3, key: []byte{0, 255}}, {kind: 2, field: 3, key: []byte("other"), value: []byte("other-value")}}
	limits := contractstate.Limits{MaxWrites: 8, MaxWriteBytes: 512, MaxValueBytes: 128, MaxMapKeyBytes: 64}
	trials := []overlayCase{{name: "mixed-cells-and-map", limits: limits, initial: base, operations: mixed}}
	for _, policy := range []struct {
		name  string
		count int
		size  uint64
		value int
	}{{"count-before", 1, 512, 128}, {"count-exact", 2, 512, 128}, {"bytes-before-delete", 8, 32, 128}, {"bytes-exact-delete", 8, 33, 128}, {"bytes-before-empty", 8, 32, 128}, {"bytes-before-write", 8, 35, 128}, {"bytes-exact-write", 8, 36, 128}, {"value-before", 1, 512, 2}, {"value-exact", 8, 512, 3}} {
		selected := limits
		selected.MaxWrites = policy.count
		selected.MaxWriteBytes = policy.size
		selected.MaxValueBytes = policy.value
		trials = append(trials, overlayCase{name: policy.name, limits: selected, initial: base, operations: []overlayOperation{write(0, "new"), write(2, "ok"), read(0), read(2), remove(0), read(0), write(0, ""), read(0)}})
	}
	replacing := limits
	replacing.MaxWrites = 1
	replacing.MaxWriteBytes = 65
	replacing.MaxValueBytes = 32
	var many []overlayOperation
	for i := 0; i < 80; i++ {
		many = append(many, overlayOperation{kind: 2, value: bytes.Repeat([]byte{byte(i)}, i%33)}, read(0))
	}
	trials = append(trials, overlayCase{name: "replacement-does-not-accumulate", limits: replacing, initial: base, operations: many})
	trials = append(trials, overlayCase{name: "current-view-write", limits: limits, operations: []overlayOperation{write(0, "source"), read(0), {kind: 4, field: 1}, read(1), {kind: 4, field: 1}, read(1)}})
	trials = append(trials, overlayCase{name: "empty-and-delete-distinct", limits: limits, operations: []overlayOperation{write(0, ""), read(0), remove(0), read(0), write(0, ""), read(0)}})
	return trials
}

func TestNativeOverlayRuntime(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	header, err := cstdlib.ContractStateOverlayDeclarations()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateOverlayRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
		t.Run(profile.Name, func(t *testing.T) {
			image, err := cdriver.BuildWindowsHost(toolchain, profile, header+"\n"+overlayRuntimeProbe, runtimeSource)
			if err != nil {
				t.Fatal(err)
			}
			t.Run("codecs", func(t *testing.T) {
				ids := []string{"", contractstate.Bool.TypeID(), contractstate.U32.TypeID(), contractstate.U64.TypeID(), contractstate.Bytes.TypeID(), contractstate.Text.TypeID(), contractstate.Byte20.TypeID(), contractstate.Byte32.TypeID(), "unknown"}
				input := []byte{0}
				var expected []byte
				add := func(codec byte, value []byte) {
					input = append(input, codec)
					input = binary.BigEndian.AppendUint32(input, uint32(len(value)))
					input = append(input, value...)
					valid := byte(0)
					if contractstate.ValidateCanonicalEncoding(ids[int(codec)], value) == nil {
						valid = 1
					}
					expected = append(expected, valid)
				}
				for codec := range ids {
					for _, length := range []int{0, 1, 3, 4, 7, 8, 19, 20, 21, 31, 32, 33, 64, 65536} {
						add(byte(codec), bytes.Repeat([]byte{0xff}, length))
					}
				}
				for lead := 0; lead < 256; lead++ {
					add(1, []byte{byte(lead)})
					add(5, []byte{byte(lead)})
					for second := 0; second < 256; second += 17 {
						add(5, []byte{byte(lead), byte(second), 0x80, 0xbf})
					}
				}
				for _, value := range []string{"", "\x00", "한글", "\u007f", "\u0080", "\u07ff", "\u0800", "\ud7ff", "\ue000", "\uffff", "\U00010000", "\U0010ffff"} {
					data := []byte(value)
					for size := 0; size <= len(data); size++ {
						add(5, data[:size])
					}
				}
				result, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
				if err != nil || result.ExitCode != 0 || result.Stderr != "" || !bytes.Equal([]byte(result.Stdout), expected) {
					t.Fatalf("codec %d native/Go results: exit %d streams %d/%q error %v", len(expected), result.ExitCode, len(result.Stdout), result.Stderr, err)
				}
				t.Logf("%d canonical codec comparisons", len(expected))
			})
			t.Run("overlay", func(t *testing.T) {
				trials := overlayTrials()
				input := []byte{1}
				input = binary.BigEndian.AppendUint16(input, uint16(len(trials)))
				var expected []byte
				for _, trial := range trials {
					request, response := overlayOracle(t, trial)
					input = append(input, request...)
					expected = append(expected, response...)
				}
				result, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
				if err != nil || result.ExitCode != 0 || result.Stderr != "" || !bytes.Equal([]byte(result.Stdout), expected) {
					at := 0
					for at < len(result.Stdout) && at < len(expected) && result.Stdout[at] == expected[at] {
						at++
					}
					t.Fatalf("overlay %d cases differ at byte %d, exit %d, length %d/%d, stderr %q error %v", len(trials), at, result.ExitCode, len(result.Stdout), len(expected), result.Stderr, err)
				}
				t.Logf("%d real multi-key overlay/Go comparisons", len(trials))
			})
			t.Run("failure-and-lifetime", func(t *testing.T) {
				result, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader([]byte{2}))
				if err != nil || result.ExitCode != 0 || result.Stdout != "" || result.Stderr != "" {
					t.Fatal(result, err)
				}
			})
		})
	}
}
