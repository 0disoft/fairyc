package contractstatenative

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/0disoft/fairyc/internal/cdriver"
	"github.com/0disoft/fairyc/internal/contractstate"
	"github.com/0disoft/fairyc/internal/contractstatemir"
	"github.com/0disoft/fairyc/internal/cstdlib"
	"github.com/0disoft/fairyc/internal/target"
)

const overlayMIRProbe = `
#include <stdio.h>
#include <string.h>
#include <fcntl.h>
#include <io.h>
static uint8_t trace[8192];
static size_t trace_size;
static void put_number(uint64_t value, size_t count) {
    size_t i = count;
    while (i != 0u) { --i; (void)putchar((int)((value >> (i * 8u)) & 255u)); }
}
static int32_t observe(void *context, uint32_t operation, uint8_t codec, const uint8_t *value, size_t length, uint8_t found) {
    size_t i;
    (void)context;
    if (length > sizeof(trace) - trace_size || sizeof(trace) - trace_size - length < 10u) { return FYCS_OV_INVALID; }
    for (i = 4u; i != 0u; --i) { trace[trace_size++] = (uint8_t)(operation >> ((i-1u)*8u)); }
    trace[trace_size++] = codec;
    trace[trace_size++] = found;
    for (i = 4u; i != 0u; --i) { trace[trace_size++] = (uint8_t)(length >> ((i-1u)*8u)); }
    if (length != 0u) { memcpy(trace + trace_size, value, length); trace_size += length; }
    return 0;
}
int main(void) {
    fycs_overlay_v1 transaction;
    fycs_overlay_entry_v1 writes[32];
    uint8_t arena[4096], scratch[4096], input[10], exit_kind = 0u;
    uint32_t max_bytes;
    uint16_t max_value, max_key;
    int32_t status;
    size_t i;
    if (_setmode(_fileno(stdin), _O_BINARY) == -1 || _setmode(_fileno(stdout), _O_BINARY) == -1 || fread(input, 1u, sizeof(input), stdin) != sizeof(input) || getchar() != EOF) { return 2; }
    max_bytes = ((uint32_t)input[1] << 24u) | ((uint32_t)input[2] << 16u) | ((uint32_t)input[3] << 8u) | input[4];
    max_value = (uint16_t)((uint16_t)input[5] << 8u) | input[6];
    max_key = (uint16_t)((uint16_t)input[7] << 8u) | input[8];
    if (input[0] == 0u || input[0] > 32u || input[9] != 0u || max_bytes > sizeof(arena)) { return 3; }
    status = fycs_overlay_begin_v1(&transaction, NULL, 0u, writes, input[0], arena, scratch, sizeof(arena), max_bytes, max_value);
    if (status != 0) { return 4; }
    status = fycs_overlay_execute_v1(&transaction, max_key, observe, NULL, &exit_kind);
    if (status != 0 || exit_kind == 2u) { trace_size = 0u; }
    put_number((uint32_t)status, 1u); put_number(exit_kind, 1u);
    put_number(trace_size, 4u);
    if (trace_size != 0u && fwrite(trace, 1u, trace_size, stdout) != trace_size) { return 5; }
    put_number(transaction.count, 2u); put_number(transaction.write_bytes, 8u);
    for (i = 0u; i < transaction.count; ++i) {
        const fycs_overlay_entry_v1 *entry = &transaction.writes[i];
        if (fwrite(entry->key, 1u, 32u, stdout) != 32u) { return 6; }
        put_number(entry->deleted, 1u); put_number(entry->length, 4u);
        if (entry->length != 0u && fwrite(entry->value, 1u, entry->length, stdout) != entry->length) { return 7; }
    }
    return fflush(stdout) != 0 ? 8 : 0;
}
`

func overlayMIROracle(t *testing.T, handoff Handoff, layout contractstate.Layout, limits contractstate.Limits) []byte {
	t.Helper()
	store, err := contractstate.NewReferencePersistence(layout, nil)
	if err != nil {
		t.Fatal(err)
	}
	port := &overlayOraclePort{PersistencePort: store}
	out, trap, err := persistentOracle(port, handoff.program, limits)
	status, exit := byte(0), byte(0)
	switch {
	case errors.Is(err, contractstate.ErrValueTooLarge):
		status = 3
	case errors.Is(err, contractstate.ErrWriteLimit):
		status = 4
	case errors.Is(err, contractstate.ErrWriteBytesLimit):
		status = 5
	case errors.Is(err, contractstate.ErrMapKeyTooLarge):
		status = 7
	case err != nil:
		t.Fatal(err)
	}
	if trap != "" {
		exit = 2
	} else if out.Exit == contractstatemir.ExitReject {
		exit = 1
	}
	var trace []byte
	for _, read := range out.Reads {
		codec, err := overlayCodec(read.Value.TypeID)
		if err != nil {
			t.Fatal(err)
		}
		trace = binary.BigEndian.AppendUint32(trace, uint32(read.Operation))
		trace = append(trace, codec, 0)
		if read.Found {
			trace[len(trace)-1] = 1
		}
		trace = binary.BigEndian.AppendUint32(trace, uint32(len(read.Value.Encoded)))
		trace = append(trace, read.Value.Encoded...)
	}
	result := []byte{status, exit}
	result = binary.BigEndian.AppendUint32(result, uint32(len(trace)))
	result = append(result, trace...)
	result = binary.BigEndian.AppendUint16(result, uint16(len(port.writes)))
	result = binary.BigEndian.AppendUint64(result, out.Receipt.WriteBytes)
	for _, write := range port.writes {
		result = append(result, write.Key[:]...)
		if write.Deleted {
			result = append(result, 1)
		} else {
			result = append(result, 0)
		}
		result = binary.BigEndian.AppendUint32(result, uint32(len(write.Value)))
		result = append(result, write.Value...)
	}
	return result
}

func TestOverlayMIRNative(t *testing.T) {
	toolchain, err := cdriver.ResolveWindowsHostMSVC()
	if err != nil {
		t.Fatal(err)
	}
	runtimeSource, err := cstdlib.ContractStateOverlayRuntimeSource()
	if err != nil {
		t.Fatal(err)
	}
	for _, family := range []string{"commit", "reject", "trap", "empty"} {
		t.Run(family, func(t *testing.T) {
			exit := contractstatemir.ExitCommit
			if family == "reject" {
				exit = contractstatemir.ExitReject
			}
			if family == "trap" {
				exit = contractstatemir.ExitTrap
			}
			handoff, layout := overlayMIRFixture(t, exit, family == "empty")
			artifact, err := EmitOverlay(handoff, target.X8664WindowsMSVC)
			if err != nil {
				t.Fatal(err)
			}
			for _, profile := range []cdriver.Profile{cdriver.LowProfile, cdriver.OptimizedProfile} {
				t.Run(profile.Name, func(t *testing.T) {
					image, err := cdriver.BuildWindowsHost(toolchain, profile, artifact.Source()+overlayMIRProbe, runtimeSource)
					if err != nil {
						t.Fatal(err)
					}
					for _, policy := range []struct {
						name       string
						count      int
						size       uint64
						value, key int
					}{{"exact", 14, 2048, 32, 32}, {"count-before", 13, 2048, 32, 32}, {"value-before", 14, 2048, 31, 32}, {"key-before", 14, 2048, 32, 31}, {"bytes-before-delete", 14, 32, 32, 32}, {"bytes-exact-delete", 14, 33, 32, 32}, {"after", 15, 2049, 33, 33}} {
						t.Run(policy.name, func(t *testing.T) {
							limits := contractstate.Limits{MaxWrites: policy.count, MaxWriteBytes: policy.size, MaxValueBytes: policy.value, MaxMapKeyBytes: policy.key}
							input := []byte{byte(policy.count)}
							input = binary.BigEndian.AppendUint32(input, uint32(policy.size))
							input = binary.BigEndian.AppendUint16(input, uint16(policy.value))
							input = binary.BigEndian.AppendUint16(input, uint16(policy.key))
							input = append(input, 0)
							want := overlayMIROracle(t, handoff, layout, limits)
							got, err := cdriver.ExecuteWindowsMSVCTargetWithInput(image, bytes.NewReader(input))
							if err != nil || got.ExitCode != 0 || got.Stderr != "" || !bytes.Equal([]byte(got.Stdout), want) {
								at := 0
								for at < len(got.Stdout) && at < len(want) && got.Stdout[at] == want[at] {
									at++
								}
								t.Fatalf("native/Go differ at %d: exit %d size %d/%d stderr %q error %v", at, got.ExitCode, len(got.Stdout), len(want), got.Stderr, err)
							}
						})
					}
				})
			}
		})
	}
}
